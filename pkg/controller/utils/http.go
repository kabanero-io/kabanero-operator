package utils

import (
	"context"
	"encoding/base64"
	"net/http"

	"golang.org/x/oauth2"
)

// Retrieves a HTTP client from a base64 encoded input access token.
func GetOauth2HTTPCLient(accessToken []byte) (*http.Client, error) {
	ctx := context.Background()
	encodedToken := base64.StdEncoding.EncodeToString([]byte(accessToken))
	decodedTokenBytes, err := base64.StdEncoding.DecodeString(encodedToken)
	if err != nil {
		return nil, err
	}

	sts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: string(decodedTokenBytes)},
	)

	return oauth2.NewClient(ctx, sts), nil
}
