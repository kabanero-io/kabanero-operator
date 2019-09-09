package kabaneroplatform

import (
	"context"
	"errors"
	"fmt"
	"strings"

	kabanerov1alpha1 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha1"
	mf "github.com/kabanero-io/manifestival"
	routev1 "github.com/openshift/api/route/v1"
	yaml "gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	rlog "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var kllog = rlog.Log.WithName("kabanero-landing")
var landingImage = "kabanero/landing"
var landingImageTag = "0.1.0"

// Deploys resources and customizes to the Openshift web console.
func deployLandingPage(k *kabanerov1alpha1.Kabanero, c client.Client) error {
	rev, err := resolveSoftwareRevision(k, "landing", k.Spec.AppsodyOperator.Version)
	if err != nil {
		return err
	}

	//The context which will be used to render any templates
	templateContext := rev.Identifiers

	image, err := imageUriWithOverrides("", "", "", rev)
	if err != nil {
		return err
	}
	templateContext["image"] = image

	f, err := rev.OpenOrchestration("kabanero-landing.yaml")
	if err != nil {
		return err
	}

	s, err := renderOrchestration(f, templateContext)
	if err != nil {
		return err
	}

	m, err := mf.FromReader(strings.NewReader(s), c)
	if err != nil {
		return err
	}

	transforms := []mf.Transformer{mf.InjectOwner(k), mf.InjectNamespace(k.GetNamespace())}
	err = m.Transform(transforms...)
	if err != nil {
		return err
	}

	err = m.ApplyAll()
	if err != nil {
		return err
	}

	// Create a clientset to drive API operations on resources.
	config, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		return err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	// Retrieve the kabanero landing URL.
	landingURL, err := getLandingURL(k, config)
	if err != nil {
		return err
	}

	// Gather Kabanero operator ownerReference information.
	ownerRef, err := getOwnerReference(k, c)
	if err != nil {
		return err
	}

	// Create a Deployment. The landing application requires knowledge of the landingURL
	// post route creation.
	err = createDeployment(k, clientset, c, landingURL, ownerRef)
	if err != nil {
		return err
	}

	// Update the web console's ConfigMap with custom data.
	err = customizeWebConsole(k, clientset, config, landingURL)

	return err
}

// Retrieves the landing URL from the landing Route.
func getLandingURL(k *kabanerov1alpha1.Kabanero, config *restclient.Config) (string, error) {
	landingURL := ""

	// Get the Route instance.
	myScheme := runtime.NewScheme()
	cl, _ := client.New(config, client.Options{Scheme: myScheme})
	routev1.AddToScheme(myScheme)
	landingRoute := &routev1.Route{}
	landingRouteName := types.NamespacedName{Namespace: k.ObjectMeta.Namespace, Name: "kabanero-landing"}
	err := cl.Get(context.TODO(), landingRouteName, landingRoute)

	if err != nil {
		return landingURL, err
	}

	// Look for the ingress entry with the status of admitted.
	// There should only be one URL and it should be the one that is auto generated.
	for _, ingress := range landingRoute.Status.Ingress {
		for _, condition := range ingress.Conditions {
			if condition.Type == routev1.RouteAdmitted && condition.Status == corev1.ConditionTrue {
				landingURL = ingress.Host
				break
			}
		}
	}

	// If the URL is invalid, return an error.
	if len(landingURL) == 0 {
		err = errors.New("Invalid kabanero landing URL")
	} else {
		landingURL = "https://" + landingURL
	}

	kllog.Info(fmt.Sprintf("getLandingURL: URL: %v", landingURL))

	return landingURL, err
}

// Retrieves an OwnerRereference object populated with the Kabanero operator information.
func getOwnerReference(k *kabanerov1alpha1.Kabanero, c client.Client) (metav1.OwnerReference, error) {
	ownerIsController := true
	kInstance := &kabanerov1alpha1.Kabanero{}
	err := c.Get(context.Background(), types.NamespacedName{
		Name:      k.ObjectMeta.Name,
		Namespace: k.ObjectMeta.Namespace}, kInstance)

	if err != nil {
		return metav1.OwnerReference{}, err
	}

	ownerRef := metav1.OwnerReference{
		APIVersion: kInstance.TypeMeta.APIVersion,
		Kind:       kInstance.TypeMeta.Kind,
		Name:       kInstance.ObjectMeta.Name,
		UID:        kInstance.ObjectMeta.UID,
		Controller: &ownerIsController,
	}

	kllog.Info(fmt.Sprintf("getOwnerReference: OwnerReference: %v", ownerRef))

	return ownerRef, err
}

// Creates and deploys a Deployment resource.
//
// apiVersion: apps/v1
// kind: Deployment
// metadata:
//   name: kabanero-landing
// spec:
//   selector:
//     matchLabels:
//       app: kabanero-landing
//   replicas: 1
//   template:
//     metadata:
//       labels:
//         app: kabanero-landing
//     spec:
//       serviceAccountName: kabanero-landing
//       containers:
//         - name: kabanero-landing
//           image: kabanero/landing:0.0.1
//           imagePullPolicy: Always
//           ports:
//             - containerPort: 9443
//           env:
//             - name: LANDING_URL
//               value: <FILL_IN>>
//
func createDeployment(k *kabanerov1alpha1.Kabanero, clientset *kubernetes.Clientset, c client.Client, landingURL string, ownerRef metav1.OwnerReference) error {
	// Check if the Deployment resource already exists.
	dInstance := &appsv1.Deployment{}
	err := c.Get(context.Background(), types.NamespacedName{
		Name:      "kabanero-landing",
		Namespace: k.ObjectMeta.Namespace}, dInstance)

	if err == nil {
		return nil
	}

	if !apierrors.IsNotFound(err) {
		return err
	}

	// The Deployment resource does not exist. Create it.
	cl := clientset.AppsV1().Deployments(k.ObjectMeta.Namespace)

	var repCount int32 = 1
	image := landingImage + ":" + landingImageTag
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kabanero-landing",
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: ownerRef.APIVersion,
					Kind:       ownerRef.Kind,
					Name:       ownerRef.Name,
					UID:        ownerRef.UID,
					Controller: ownerRef.Controller,
				},
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &repCount,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "kabanero-landing",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "kabanero-landing",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "kabanero-landing",
					Containers: []corev1.Container{
						{
							Name:            "kabanero-landing",
							Image:           image,
							ImagePullPolicy: "Always",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 9443,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "LANDING_URL",
									Value: landingURL,
								},
							},
						},
					},
				},
			},
		},
	}

	kllog.Info(fmt.Sprintf("createDeployment: Deployment for create: %v", deployment))

	_, err = cl.Create(deployment)

	return err
}

// Adds customizations to the OpenShift web console.
func customizeWebConsole(k *kabanerov1alpha1.Kabanero, clientset *kubernetes.Clientset, config *restclient.Config, landingURL string) error {

	// Get the Route instance.
	myScheme := runtime.NewScheme()
	cl, _ := client.New(config, client.Options{Scheme: myScheme})
	corev1.AddToScheme(myScheme)
	configmap := &corev1.ConfigMap{}
	err := cl.Get(context.TODO(), types.NamespacedName{
		Namespace: "openshift-web-console", Name: "webconsole-config"}, configmap)

	if err != nil {
		return err
	}

	// Create a copy of the original config map.
	newcm := configmap.DeepCopy()

	// Get the embedded yaml entry in the web console ConfigMap yaml.
	wccyaml := newcm.Data["webconsole-config.yaml"]

	m := make(map[string]interface{})
	err = yaml.Unmarshal([]byte(wccyaml), &m)
	if err != nil {
		return err
	}

	// Update the extensions section of the embedded webconsole-config.yaml entry
	scripts, ssheets := getCustomizationURLs(landingURL)

	for k, v := range m {
		if k == "extensions" {
			extmap := v.(map[interface{}]interface{})
			for kk, vv := range extmap {
				if kk == "scriptURLs" {
					su := vv.([]interface{})
					eScripts := getEffectiveCustomizationURLs(su, scripts)
					sun := make([]interface{}, (len(su) + len(eScripts)))
					var ix = 0
					for i := 0; i < len(su); i++ {
						sun[ix] = su[ix]
						ix++
					}
					for _, u := range eScripts {
						sun[ix] = u
						ix++
					}
					extmap[kk] = sun
				}
				if kk == "stylesheetURLs" {
					su := vv.([]interface{})
					eSsheets := getEffectiveCustomizationURLs(su, ssheets)
					sun := make([]interface{}, (len(su) + len(eSsheets)))
					var ix = 0
					for i := 0; i < len(su); i++ {
						sun[ix] = su[ix]
						ix++
					}

					for _, u := range eSsheets {
						sun[ix] = u
						ix++
					}
					extmap[kk] = sun
				}
			}
			m[k] = extmap
			break
		}
	}

	// Update our copy of the web console yaml and update it.
	upatedCMBytes, err := yaml.Marshal(m)
	newcm.Data["webconsole-config.yaml"] = string(upatedCMBytes)
	_, err = yaml.Marshal(newcm)

	if err != nil {
		return err
	}

	kllog.Info(fmt.Sprintf("customizeWebConsole: ConfigMap for update: %v", newcm))

	_, err = clientset.CoreV1().ConfigMaps("openshift-web-console").Update(newcm)

	return err
}

// Gets the customization URLs.
func getCustomizationURLs(landingURL string) ([]string, []string) {
	scripts := []string{
		"LANDING_URL/appnav/openshift/featuredApp.js",
		"LANDING_URL/appnav/openshift/appLauncher.js",
		"LANDING_URL/appnav/openshift/projectNavigation.js",
	}

	ssheets := []string{
		"LANDING_URL/appnav/openshift/appNavIcon.css",
	}

	// Replace script URLs with the correct host name.
	for i, url := range scripts {
		scripts[i] = strings.Replace(url, "LANDING_URL", landingURL, -1)
	}

	// Replace stylesheet URLs with the correct host name.
	for i, url := range ssheets {
		ssheets[i] = strings.Replace(url, "LANDING_URL", landingURL, -1)
	}

	return scripts, ssheets
}

// Returns the customization URLs that are not currently defined.
func getEffectiveCustomizationURLs(extUrls []interface{}, urls []string) []string {
	var eUrls []string

	for _, url := range urls {
		if !contains(extUrls, url) {
			eUrls = append(eUrls, url)
		}
	}
	return eUrls

}

// Checks if the given string is contained in the given array.
func contains(urls []interface{}, s string) bool {
	for _, url := range urls {
		if url == s {
			return true
		}
	}
	return false
}

// Retrieves the current kabanero landing page status.
func getKabaneroLandingPageStatus(k *kabanerov1alpha1.Kabanero, c client.Client) (bool, error) {
	k.Status.Landing.ErrorMessage = ""
	k.Status.Landing.Ready = "False"

	// Create a clientset to drive API operations on resources.
	config, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		k.Status.Landing.Ready = "False"
		k.Status.Landing.ErrorMessage = "Failed to build configuration to retrieve status."
		return false, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		k.Status.Landing.Ready = "False"
		k.Status.Landing.ErrorMessage = "Failed to create clientset to retrieve status."
		return false, err
	}

	options := metav1.ListOptions{LabelSelector: "app=kabanero-landing"}
	pods, err := clientset.CoreV1().Pods(k.ObjectMeta.Namespace).List(options)

	if err != nil {
		k.Status.Landing.Ready = "False"
		k.Status.Landing.ErrorMessage = "Pod instance(s) with label kabanero-landing under the namespace of " + k.ObjectMeta.Namespace + " could not be retrieved."
		return false, err
	}

	// By default there should only be one instance of the landing pod (replica count = 1).
	// Aggregate the results if that changes.
	ready := true
	finalErrorMessage := ""
	k.Status.Landing.Version = landingImageTag
	for _, pod := range pods.Items {
		for _, condition := range pod.Status.Conditions {
			if strings.ToLower(string(condition.Type)) == "ready" {
				status := string(condition.Status)
				if strings.ToLower(status) == "false" {
					ready = false
					finalErrorMessage += "Pod " + pod.Name + " not ready: " + condition.Message + ". "
				}
				break
			}
		}
	}

	if ready {
		k.Status.Landing.Ready = "True"
	} else {
		k.Status.Landing.Ready = "False"
	}

	k.Status.Landing.ErrorMessage = finalErrorMessage

	return ready, err
}
