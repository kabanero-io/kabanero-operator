package client

import (
	"bytes"
	"context"
	"fmt"
	error_util "github.com/pkg/errors"
	"io"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/jsonmergepatch"
	"k8s.io/apimachinery/pkg/util/mergepatch"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	cr_client "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type EncoderDecoder interface {
	Encoder
	Decoder
}

type EncodeOptions struct {
	//json or yaml
	Format string

	//Pretty print json
	Pretty bool
}

type Encoder interface {
	//Marshals the provided objects into json or yaml
	Encode(w io.Writer, o runtime.Object, format string, options *EncodeOptions) error
}

type Decoder interface {
	Decode(r io.ReadCloser, format string) ([]runtime.Object, error)
}

var DefaultEncoderDecoder EncoderDecoder

type Config struct {
	RestConfig       *rest.Config
	KubernetesClient *kubernetes.Clientset
	Scheme           *runtime.Scheme
	Discovery        discovery.DiscoveryInterface
}

func init() {
	DefaultClient = NewClient(nil)
}

var DefaultClient Client

type Client interface {
	//Parses the given yaml/json into the static or dynamic object model
	Unmarshal(r io.ReadCloser, format string) ([]metav1.Object, error)

	//Marshals the provided objects into json or yaml
	Marshal(w io.Writer, o []metav1.Object, format string) error

	// Determines whether the server has the provided Group/Version/Kind
	HasGVK(schema.GroupVersionKind) (bool, error)

	//Calculates a patch
	Diff(runtime.Object) error

	//Applies the provided object
	Apply(o metav1.Object, options *ApplyOptions) error

	//Applies the provided object
	ApplyAll(o runtime.Object, options *ApplyOptions) error

	//Applies the provided yaml text
	ApplyText(r io.Reader, o *ApplyOptions) ([]runtime.Object, error)

	//Deletes the provided object
	Delete(context.Context, runtime.Object, *DeleteOptions) error

	DeleteAll(context.Context, []runtime.Object, *DeleteOptions) error
}

func NewClient(c *Config) Client {
	if c == nil {
		c = &Config{RestConfig: config.GetConfigOrDie()}
	}
	if c.Scheme == nil {
		c.Scheme = scheme.Scheme
	}

	clientset, err := kubernetes.NewForConfig(c.RestConfig)
	if err != nil {
		//TODO: add error to return args and return instead of panic
		panic(err)
	}
	c.KubernetesClient = clientset

	return &client{Config: *c}
}

type client struct {
	Config
	Client
}

func (c *client) Unmarshal(r io.ReadCloser, s string) ([]metav1.Object, error) {
	d := yaml.NewYAMLOrJSONDecoder(r, 4096)

	objs := make([]metav1.Object, 0)
	for {
		ext := runtime.RawExtension{}
		if err := d.Decode(&ext); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		versions := &runtime.VersionedObjects{}
		_, gvk, err := unstructured.UnstructuredJSONScheme.Decode(ext.Raw, nil, versions)

		var obj runtime.Object
		if c.Scheme.IsGroupRegistered(gvk.Group) {
			var err error
			obj, err = c.Scheme.New(*gvk)
			if err != nil {
				return nil, err
			}
			scheme.Codecs.UniversalDeserializer().Decode(ext.Raw, nil, obj)
		} else {
			obj = versions.Objects[0]
			if err != nil {
				return nil, err
			}
		}

		objs = append(objs, obj.(metav1.Object))
	}

	return objs, nil
}

func (c *client) Apply(obj metav1.Object, options *ApplyOptions) error {
	m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return error_util.Wrap(err, "Could not convert to unstructured")
	}

	u := unstructured.Unstructured{Object: m}

	if options != nil {
		options.setControllerReference(&u, &c.Config)

		if options.Namespace != "" {
			u.SetNamespace(options.Namespace)
		}
	}

	cl, err := cr_client.New(c.RestConfig, cr_client.Options{})

	key := types.NamespacedName{Name: u.GetName(), Namespace: u.GetNamespace()}
	var currentObj unstructured.Unstructured
	currentObj.SetGroupVersionKind(u.GroupVersionKind())

	var w bytes.Buffer
	err = Marshal(&u, &w, "json")
	modified := w.Bytes()

	mapper, err := apiutil.NewDiscoveryRESTMapper(c.RestConfig)
	mapping, err := mapper.RESTMapping(u.GroupVersionKind().GroupKind(), u.GroupVersionKind().Version)

	var namespaced bool
	namespaceScope := mapping.Scope.Name()

	switch namespaceScope {
	case meta.RESTScopeNameNamespace:
		namespaced = true
	case meta.RESTScopeNameRoot:
		namespaced = false
	}

	err = cl.Get(context.Background(), key, &currentObj)
	if errors.IsNotFound(err) {
		annotations := u.GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string)
		}
		annotations["kubectl.kubernetes.io/last-applied-configuration"] = string(modified)
		u.SetAnnotations(annotations)

		err = cl.Create(context.Background(), &u)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		fmt.Println("Object exists, attempting patch")

		//Object already exists
		if lastAppliedConfiguration, exists := currentObj.GetAnnotations()["kubectl.kubernetes.io/last-applied-configuration"]; exists {
			original := []byte(lastAppliedConfiguration)

			var w bytes.Buffer
			err := Marshal(&currentObj, &w, "json")
			if err != nil {
				return err
			}
			current := w.Bytes()

			preconditions := []mergepatch.PreconditionFunc{
				mergepatch.RequireKeyUnchanged("apiVersion"),
				mergepatch.RequireKeyUnchanged("kind"),
				mergepatch.RequireMetadataKeyUnchanged("name"),
			}

			patchType := types.MergePatchType
			data, err := jsonmergepatch.CreateThreeWayJSONMergePatch(original, modified, current, preconditions...)
			if err != nil {
				if mergepatch.IsPreconditionFailed(err) {
					return fmt.Errorf("%s", "At least one of apiVersion, kind and name was changed")
				}
				return err
			}

			rclient, err := apiutil.RESTClientForGVK(u.GroupVersionKind(), c.RestConfig, scheme.Codecs)

			if bytes.Equal(data, []byte("{}")) {
				//Empty patch, there is nothing to apply
				return nil
			}

			fmt.Println("Apllying patch ", string(data))

			_, err = rclient.Patch(patchType).
				NamespaceIfScoped(u.GetNamespace(), namespaced).
				Resource(mapping.Resource.Resource).
				Name(u.GetName()).
				Body(data).
				Do().
				Get()

			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *client) ApplyText(r io.Reader, o *ApplyOptions) ([]runtime.Object, error) {
	//TODO: align reader/readcloser across this package
	objs, err := c.Unmarshal(ioutil.NopCloser(r), "yaml")
	if err != nil {
		return nil, err
	}

	for _, obj := range objs {
		err := c.Apply(obj, o)
		if err != nil {
			return nil, err
		}
	}

	//TODO: align metav1 vs runtime.Objects across this package
	//then drop this conversion
	_objs := make([]runtime.Object, len(objs))
	for i, obj := range objs {
		_objs[i] = obj.(runtime.Object)
	}

	return _objs, nil
}

//Deletes the provided object
func (c *client) Delete(ctx context.Context, o runtime.Object, options *DeleteOptions) error {
	cl, err := cr_client.New(c.RestConfig, cr_client.Options{})
	if err != nil {
		return err
	}

	if options != nil {
		if options.Namespace != "" {
			o.(metav1.Object).SetNamespace(options.Namespace)
		}
	}

	err = cl.Delete(ctx, o)
	if errors.IsNotFound(err) {
		//Not found, skip object
	} else if err != nil {
		return err
	}

	return nil
}

//Deletes the provided object
func (c *client) DeleteAll(ctx context.Context, o []runtime.Object, options *DeleteOptions) error {
	for _, obj := range o {
		err := c.Delete(ctx, obj, options)
		if err != nil {
			return err
		}
	}

	return nil
}
