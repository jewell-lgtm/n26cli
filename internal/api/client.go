package api

import (
	"github.com/guitmz/n26"
	"golang.org/x/oauth2"
)

// Client wraps the n26.Client with a cleaner interface.
type Client struct {
	inner *n26.Client
}

// NewClientFromToken creates an authenticated Client from a saved access token,
// bypassing the MFA flow entirely.
func NewClientFromToken(accessToken string) *Client {
	tokenSource := &n26.TokenSource{AccessToken: accessToken}
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	return &Client{inner: (*n26.Client)(oauthClient)}
}
