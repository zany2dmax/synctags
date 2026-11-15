// Package crowdstrike is a client for the CrowdStrike Falcon API (OAuth2).
package crowdstrike

import (
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/zany2dmax/synctags/internal/tags"
)

// Client wraps a Resty client with OAuth2 auth for the Falcon API.
type Client struct {
	r *resty.Client
}

// NewClient initializes the client and fetches an OAuth2 token.
// baseURL: e.g. "https://api.us-2.crowdstrike.com".
func NewClient(baseURL, clientID, clientSecret string) (*Client, error) {
	r := resty.New().
		SetBaseURL(baseURL).
		SetHeader("Accept", "application/json").
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetRetryCount(3).
		SetRetryWaitTime(2 * time.Second)

	var tok struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		TokenType   string `json:"token_type"`
	}
	resp, err := r.R().
		SetFormData(map[string]string{
			"client_id":     clientID,
			"client_secret": clientSecret,
		}).
		SetResult(&tok).
		Post("/oauth2/token")
	if err != nil {
		return nil, fmt.Errorf("crowdstrike token error: %w", err)
	}
	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return nil, fmt.Errorf("token HTTP %d: %s", resp.StatusCode(), resp.String())
	}

	r.SetAuthToken(tok.AccessToken)
	return &Client{r: r}, nil
}

// ListTags fetches tag names and returns them normalized.
// Known bug: Falcon has no "list tags" endpoint; tags must be read off hosts.
func (c *Client) ListTags() ([]tags.Tag, error) {
	var out struct {
		Resources []string `json:"resources"`
	}
	resp, err := c.r.R().SetResult(&out).Get("/tags/entities/tags/v1")
	if err != nil {
		return nil, fmt.Errorf("ListTags error: %w", err)
	}
	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return nil, fmt.Errorf("ListTags HTTP %d: %s", resp.StatusCode(), resp.String())
	}

	list := make([]tags.Tag, 0, len(out.Resources))
	for _, name := range out.Resources {
		list = append(list, tags.Tag{Name: tags.Normalize(name)})
	}
	return list, nil
}

// UpsertTag ensures the tag exists by POSTing to the tags endpoint.
// Known bug: same nonexistent endpoint as ListTags.
func (c *Client) UpsertTag(name string) error {
	body := struct {
		Resources []string `json:"resources"`
	}{Resources: []string{name}}
	resp, err := c.r.R().SetBody(body).Post("/tags/entities/tags/v1")
	if err != nil {
		return fmt.Errorf("UpsertTag error: %w", err)
	}
	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return fmt.Errorf("UpsertTag HTTP %d: %s", resp.StatusCode(), resp.String())
	}
	return nil
}
