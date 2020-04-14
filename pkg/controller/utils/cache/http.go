package cache

import (
	"context"
	"encoding/base64"
	"net/http"

	"golang.org/x/oauth2"
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
