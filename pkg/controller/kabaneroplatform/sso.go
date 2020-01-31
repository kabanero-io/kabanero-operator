package kabaneroplatform

import (
	"context"
	//"fmt"
	"strings"

	"github.com/go-logr/logr"
	kabanerov1alpha2 "github.com/kabanero-io/kabanero-operator/pkg/apis/kabanero/v1alpha2"

	mf "github.com/kabanero-io/manifestival"
	//"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func reconcileSso(ctx context.Context, k *kabanerov1alpha2.Kabanero, c client.Client, reqLogger logr.Logger) error {
	// Figure out what version of the orchestration we are going to use.
	noOverrideVersion := ""
	rev, err := resolveSoftwareRevision(k, "sso", noOverrideVersion)
	if err != nil {
		return err
	}
	
	//The context which will be used to render any templates
	templateContext := make(map[string]interface{})

	f, err := rev.OpenOrchestration("sso.yaml")
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

	transforms := []mf.Transformer{
		mf.InjectOwner(k),
		mf.InjectNamespace(k.GetNamespace()),
	}

	err = m.Transform(transforms...)
	if err != nil {
		return err
	}

	// TODO: This enable/disable logic probably needs to be better.
	if k.Spec.Sso.Enable == true {
		err = m.ApplyAll()
		if err != nil {
			return err
		}
	} else {
		_ = m.DeleteAll()
	}
	
	return nil
}

func getSsoStatus(k *kabanerov1alpha2.Kabanero, c client.Client, reqLogger logr.Logger) (bool, error) {
	// TODO: Implement status
	return true, nil
}

