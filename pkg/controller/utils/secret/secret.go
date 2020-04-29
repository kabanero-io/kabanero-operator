package secret

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Customizable filter.
type filter func(secretList *corev1.SecretList, filterStrings ...string) (*corev1.Secret, error)

// Retrieves Secret Objects matching the input annotation key in the specified namespace.
func GetMatchingSecret(c client.Client, namespace string, f filter, filterStrings ...string) (*corev1.Secret, error) {
	secretList := &corev1.SecretList{}
	err := c.List(context.Background(), secretList, client.InNamespace(namespace))
	if err != nil {
		return nil, err
	}

	secret, err := f(secretList, filterStrings...)
	if err != nil {
		return nil, err
	}

	return secret, nil
}
