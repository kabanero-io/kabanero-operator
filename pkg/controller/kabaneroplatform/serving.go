package kabaneroplatform

import (
	"context"
	"github.com/go-logr/logr"
	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"

	corev1 "k8s.io/api/core/v1"

	mf "github.com/manifestival/manifestival"
	mfc "github.com/manifestival/controller-runtime-client"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

func reconcileServing(ctx context.Context, k *kabanerov1alpha2.Kabanero, c client.Client, reqLogger logr.Logger) error {

	// Figure out what version of the orchestration we are going to use.
	rev, err := resolveSoftwareRevision(k, "serving", k.Spec.Serving.Version)
	if err != nil {
		return err
	}

	templateContext := rev.Identifiers

	image, err := imageUriWithOverrides(k.Spec.Serving.Repository, k.Spec.Serving.Tag, k.Spec.Serving.Image, rev)
	if err != nil {
		return err
	}
	
	templateContext["image"] = image
	templateContext["instance"] = k.ObjectMeta.UID
	templateContext["version"] = rev.Version

	f, err := rev.OpenOrchestration("serving.yaml")
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


func cleanupServing(k *kabanerov1alpha2.Kabanero, c client.Client, reqLogger logr.Logger) error {

	rev, err := resolveSoftwareRevision(k, "serving", k.Spec.Serving.Version)
	if err != nil {
		return err
	}

	//The context which will be used to render any templates
	templateContext := rev.Identifiers

	// TODO
	image, err := imageUriWithOverrides(k.Spec.Serving.Repository, k.Spec.Serving.Tag, k.Spec.Serving.Image, rev)
	if err != nil {
		return err
	}
	templateContext["image"] = image
	templateContext["instance"] = k.ObjectMeta.UID
	templateContext["version"] = rev.Version

	f, err := rev.OpenOrchestration("serving.yaml")
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
	secretInstance.Name = "kabanero-operator-serving-cert"
	secretInstance.Namespace = k.GetNamespace()
	err = c.Delete(context.TODO(), secretInstance)

	if (err != nil) && (errors.IsNotFound(err) == false) {
		return err
	}

	return nil
}
