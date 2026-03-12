package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/guitmz/n26"
	"golang.org/x/oauth2"
)

const apiURL = "https://api.tech26.de"

// Client wraps the n26.Client with a cleaner interface.
// Also provides raw HTTP access for endpoints the library doesn't fully support.
type Client struct {
	inner      *n26.Client
	httpClient *http.Client
}

// NewClientFromToken creates an authenticated Client from a saved access token,
// bypassing the MFA flow entirely.
func NewClientFromToken(accessToken string) *Client {
	tokenSource := &n26.TokenSource{AccessToken: accessToken}
	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	return &Client{
		inner:      (*n26.Client)(oauthClient),
		httpClient: oauthClient,
	}
}

// get makes an authenticated GET request to the N26 API and decodes JSON into dst.
func (c *Client) get(endpoint string, params map[string]string, dst interface{}) error {
	u, _ := url.ParseRequestURI(apiURL)
	u.Path = endpoint
	if len(params) > 0 {
		q := u.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
	}

	resp, err := c.httpClient.Get(u.String())
	if err != nil {
		return fmt.Errorf("GET %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GET %s: status %d: %s", endpoint, resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(dst)
}
