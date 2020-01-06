package transforms

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	mf "github.com/kabanero-io/manifestival"
)

func InjectNamespace(namespace string) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		kind := u.GetKind()

		// Presently, TriggerBinding and TriggerTemplate objects are created
		// in a different namespace, so we should not enforce our own.
		if (kind != "TriggerBinding") && (kind != "TriggerTemplate") {
			return mf.InjectNamespace(namespace)(u)
		}

		return nil
	}
}
