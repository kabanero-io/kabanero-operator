package kabaneroplatform

import (
	"context"
	"github.com/go-logr/logr"
	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"
	"github.com/kabanero-io/kabanero-operator/pkg/versioning"

	corev1 "k8s.io/api/core/v1"

	mf "github.com/manifestival/manifestival"
	mfc "github.com/manifestival/controller-runtime-client"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

func reconcileDevfileRegistry(ctx context.Context, k *kabanerov1alpha2.Kabanero, c client.Client, reqLogger logr.Logger) error {

	// Figure out what version of the orchestration we are going to use.  If this version doesn't have a devfile
	// registry, then make sure the devfile registry is stopped.
	rev, err := resolveSoftwareRevision(k, "devfile-registry-controller", k.Spec.DevfileRegistry.Version)
	if err != nil {
		// We'll need to re-evaluate this down the road, but for now assume the 0.10.0 version of the devfile
		// registry controller might be installed, so lets try to remove it.
		rev, err2 := resolveSoftwareRevision(k, "devfile-registry-controller", "0.10.0")
		if err2 != nil {
			return err
		}

		err = cleanupDevfileRegistryForRevision(rev, k, c, reqLogger)
		if err != nil {
			return err
		}
		
		return nil
	}

	templateContext := rev.Identifiers

	image, err := imageUriWithOverrides(k.Spec.DevfileRegistry.Repository, k.Spec.DevfileRegistry.Tag, k.Spec.DevfileRegistry.Image, rev)
	if err != nil {
		return err
	}
	
	templateContext["image"] = image
	templateContext["instance"] = k.ObjectMeta.UID
	templateContext["version"] = rev.Version

	f, err := rev.OpenOrchestration("devfile-registry-controller.yaml")
	if err != nil {
		return err
	}

	s, err := renderOrchestration(f, templateContext)
	if err != nil {
		return err
	}

	mOrig, err := mf.ManifestFrom(mf.Reader(strings.NewReader(s)), mf.UseClient(mfc.NewClient(c)), mf.UseLogger(reqLogger.WithName("manifestival")))
	if err != nil {
		return err
	}

	transforms := []mf.Transformer{
		mf.InjectOwner(k),
		mf.InjectNamespace(k.GetNamespace()),
	}

	m, err := mOrig.Transform(transforms...)
	if err != nil {
		return err
	}

	err = m.Apply()
	if err != nil {
		return err
	}

	return nil
}


func cleanupDevfileRegistry(k *kabanerov1alpha2.Kabanero, c client.Client, reqLogger logr.Logger) error {

	rev, err := resolveSoftwareRevision(k, "devfile-registry-controller", k.Spec.DevfileRegistry.Version)
	if err != nil {
		// It may be that this version of kabanero doesn't have a devfile registry, so just exit.
		reqLogger.Error(err, "Error encountered when cleanup up the devfile registry")
		return nil
	}

	return cleanupDevfileRegistryForRevision(rev, k, c, reqLogger)
}

func cleanupDevfileRegistryForRevision(rev versioning.SoftwareRevision, k *kabanerov1alpha2.Kabanero, c client.Client, reqLogger logr.Logger) error {
	
	//The context which will be used to render any templates
	templateContext := rev.Identifiers

	// TODO
	image, err := imageUriWithOverrides(k.Spec.DevfileRegistry.Repository, k.Spec.DevfileRegistry.Tag, k.Spec.DevfileRegistry.Image, rev)
	if err != nil {
		return err
	}
	templateContext["image"] = image
	templateContext["instance"] = k.ObjectMeta.UID
	templateContext["version"] = rev.Version

	f, err := rev.OpenOrchestration("devfile-registry-controller.yaml")
	if err != nil {
		return err
	}

	s, err := renderOrchestration(f, templateContext)
	if err != nil {
		return err
	}

	mOrig, err := mf.ManifestFrom(mf.Reader(strings.NewReader(s)), mf.UseClient(mfc.NewClient(c)), mf.UseLogger(reqLogger.WithName("manifestival")))
	if err != nil {
		return err
	}

	transforms := []mf.Transformer{mf.InjectNamespace(k.GetNamespace())}
	m, err := mOrig.Transform(transforms...)
	if err != nil {
		return err
	}

	// Manifestival ignores the "NotFound" error for us.
	err = m.Delete()
	if err != nil {
		return err
	}


	// Now, clean up the things that the controller-runtime created on
	// our behalf.
	secretInstance := &corev1.Secret{}
	secretInstance.Name = "kabanero-operator-devfile-registry-cert"
	secretInstance.Namespace = k.GetNamespace()
	err = c.Delete(context.TODO(), secretInstance)

	if (err != nil) && (errors.IsNotFound(err) == false) {
		return err
	}

	return nil
}
