// Package ninjaone is a client for the NinjaOne public API (OAuth2).
//
// Not wired into the CLI yet: NinjaOne comes after v1. Known bug: the
// endpoints and the login call below are wrong and must be fixed first.
package ninjaone

import (
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
)

// Client holds the Resty client and auth token.
type Client struct {
	r *resty.Client
}

// Tag is the NinjaOne asset tag model.
type Tag struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// NewClient authenticates via OAuth2 and returns a Resty-based client.
func NewClient(baseURL, clientID, clientSecret string) (*Client, error) {
	r := resty.New().
		SetBaseURL(baseURL).
		SetHeader("Content-Type", "application/json").
		SetRetryCount(3).
		SetRetryWaitTime(2 * time.Second)

	var tok struct {
		AccessToken string `json:"access_token"`
	}
	resp, err := r.R().
		SetBody(map[string]string{
			"client_id":     clientID,
			"client_secret": clientSecret,
		}).
		SetResult(&tok).
		Post("/oauth2/token")
	if err != nil {
		return nil, fmt.Errorf("NinjaOne auth request failed: %w", err)
	}
	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("NinjaOne auth HTTP %d: %s", resp.StatusCode(), resp.String())
	}

	r.SetAuthToken(tok.AccessToken)
	return &Client{r: r}, nil
}

// ListTags retrieves all asset tags.
func (c *Client) ListTags() ([]Tag, error) {
	var result struct {
		Data []Tag `json:"data"`
	}
	resp, err := c.r.R().
		SetResult(&result).
		Get("/core-resources/assetTags")
	if err != nil {
		return nil, fmt.Errorf("ListTags failed: %w", err)
	}
	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("ListTags HTTP %d: %s", resp.StatusCode(), resp.String())
	}
	return result.Data, nil
}

// CreateTag creates a new tag by name.
func (c *Client) CreateTag(name string) (*Tag, error) {
	var tag Tag
	resp, err := c.r.R().
		SetBody(map[string]string{"name": name}).
		SetResult(&tag).
		Post("/core-resources/assetTags")
	if err != nil {
		return nil, fmt.Errorf("CreateTag failed: %w", err)
	}
	if resp.StatusCode() != 201 {
		return nil, fmt.Errorf("CreateTag HTTP %d: %s", resp.StatusCode(), resp.String())
	}
	return &tag, nil
}

// UpdateTag renames an existing tag.
func (c *Client) UpdateTag(id, newName string) error {
	resp, err := c.r.R().
		SetBody(map[string]string{"name": newName}).
		Put(fmt.Sprintf("/core-resources/assetTags/%s", id))
	if err != nil {
		return fmt.Errorf("UpdateTag failed: %w", err)
	}
	if resp.StatusCode() != 204 {
		return fmt.Errorf("UpdateTag HTTP %d: %s", resp.StatusCode(), resp.String())
	}
	return nil
}

// DeleteTag removes a tag by ID.
func (c *Client) DeleteTag(id string) error {
	resp, err := c.r.R().
		Delete(fmt.Sprintf("/core-resources/assetTags/%s", id))
	if err != nil {
		return fmt.Errorf("DeleteTag failed: %w", err)
	}
	if resp.StatusCode() != 204 {
		return fmt.Errorf("DeleteTag HTTP %d: %s", resp.StatusCode(), resp.String())
	}
	return nil
}
