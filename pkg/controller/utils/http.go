package utils

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Retrieves a HTTP client. If the input access token is specified, an oauth2 generated http client is returned.
// If the access token is not specified a default http client is returned. The default http client will contain
// the input transport if specified.
func GetHTTPClient(accessToken []byte, transport *http.Transport) (*http.Client, error) {
	if accessToken != nil {
		encodedToken := base64.StdEncoding.EncodeToString([]byte(accessToken))
		decodedTokenBytes, err := base64.StdEncoding.DecodeString(encodedToken)
		if err != nil {
			return nil, err
		}

		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: string(decodedTokenBytes)},
		)
		ctx := context.Background()
		return oauth2.NewClient(ctx, ts), nil
	}

	if transport != nil {
		return &http.Client{Transport: transport}, nil
	}

	return http.DefaultClient, nil
}

// Populates a TLS config struct based specified input.
func GetTLSCConfig(c client.Client, skipCertVerify bool) (*tls.Config, error) {
	var tlsConfig *tls.Config
	if skipCertVerify {
		return &tls.Config{InsecureSkipVerify: skipCertVerify}, nil
	}

	ingressRouterCACert, err := getIngressRouterCACert(c)
	if err != nil {
		return nil, err
	}

	systemCertPool, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}

	ok := systemCertPool.AppendCertsFromPEM(ingressRouterCACert)
	if !ok {
		return nil, fmt.Errorf("Unable to append ingress router certificate to system cert pool.")
	}
	tlsConfig = &tls.Config{RootCAs: systemCertPool}

	return tlsConfig, nil
}

// Retrieve the ingress operator CA cert.
func getIngressRouterCACert(c client.Client) ([]byte, error) {
	secretName := "router-ca"
	secretNamespace := "openshift-ingress-operator"
	caRouterSecret, err := GetUnstructuredSecret(c, secretName, secretNamespace)
	if err != nil {
		return nil, fmt.Errorf("Unable to retrieve a secret object. Secret name: %v. Namespace: %v. Error: %v", secretName, secretNamespace, err)
	}

	tlsCrtI, found, err := unstructured.NestedFieldCopy(caRouterSecret.Object, "data", "tls.crt")
	if err != nil {
		return nil, fmt.Errorf("Unable to retrieve data.tls.crt from the secret %v. Namespace: %v. Error: %v", secretName, secretNamespace, err)
	}
	if !found {
		return nil, fmt.Errorf("The value of data.tls.crt in secret %v. Namespace: %v. Error: %v", secretName, secretNamespace, err)
	}

	tlscrt, ok := tlsCrtI.(string)
	if !ok {
		return nil, fmt.Errorf("The data.tls.crt entry under secret %v could not be casted as string. Namespace: %v", secretName, secretNamespace)
	}

	decodedCrtString, err := base64.StdEncoding.DecodeString(tlscrt)
	if err != nil {
		return nil, fmt.Errorf("Unable to decode secret %v tls.crt. Namespace: %v. Error: %v", secretName, secretNamespace, err)
	}

	return decodedCrtString, nil
}
