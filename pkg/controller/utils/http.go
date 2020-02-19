package utils

import (
	"context"
	"net/http"

	"golang.org/x/oauth2"
)

// Retrieves a HTTP client that uses the unencoded input access token.
func GetOauth2HTTPCLient(accessToken []byte) (*http.Client, error) {
	ctx := context.Background()

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: string(accessToken)},
	)

	return oauth2.NewClient(ctx, ts), nil
}
