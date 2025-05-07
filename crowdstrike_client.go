// crowdstrike_client.go
// CrowdStrike Falcon Tagging API integration in package main
// Implements ListTags and UpsertTag using Resty and OAuth2

package main

import (
	//    "encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

// CrowdstrikeClient wraps a Resty client with OAuth2 auth for Falcon API
type CrowdstrikeClient struct {
	r *resty.Client
}

// NewCrowdstrikeClient initializes the client and fetches an OAuth2 token
// baseURL: e.g. "https://api.us-2.crowdstrike.com"
func NewCrowdstrikeClient(baseURL, clientID, clientSecret string) (*CrowdstrikeClient, error) {
	r := resty.New().
		SetHostURL(baseURL).
		SetHeader("Accept", "application/json").
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetRetryCount(3).
		SetRetryWaitTime(2 * time.Second)

	// OAuth2 token request
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
	return &CrowdstrikeClient{r: r}, nil
}

// ListTags fetches all tags (resource names) and returns normalized Tag list
func (c *CrowdstrikeClient) ListTags() ([]Tag, error) {
	// API returns { resources: ["tag1", "tag2", ...] }
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

	var tags []Tag
	for _, name := range out.Resources {
		n := strings.ToLower(strings.TrimSpace(name))
		n = strings.ReplaceAll(n, " ", "-")
		tags = append(tags, Tag{Name: n})
	}
	return tags, nil
}

// UpsertTag ensures the tag exists by POSTing to the tags endpoint
func (c *CrowdstrikeClient) UpsertTag(t Tag) error {
	body := struct {
		Resources []string `json:"resources"`
	}{Resources: []string{t.Name}}
	resp, err := c.r.R().SetBody(body).Post("/tags/entities/tags/v1")
	if err != nil {
		return fmt.Errorf("UpsertTag error: %w", err)
	}
	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return fmt.Errorf("UpsertTag HTTP %d: %s", resp.StatusCode(), resp.String())
	}
	return nil
}
