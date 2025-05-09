// ninjaone_client.go
// NinjaOne Public API tagging integration (package main)
// Uses Resty for HTTP requests and OAuth2 authentication

package main

import (
    "encoding/json"
    "fmt"
    "time"

    "github.com/go-resty/resty/v2"
)

// NinjaOneClient holds Resty client and auth token
type NinjaOneClient struct {
    r *resty.Client
}

// NinjaTag is the asset tag model
type NinjaTag struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}

// NewNinjaOneClient authenticates via OAuth2 and returns a Resty-based client
func NewNinjaOneClient(baseURL, clientID, clientSecret string) (*NinjaOneClient, error) {
    r := resty.New().
        SetBaseURL(baseURL).
        SetHeader("Content-Type", "application/json").
        SetRetryCount(3).
        SetRetryWaitTime(2 * time.Second)

    // Authenticate
    var tok struct { AccessToken string `json:"access_token"` }
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

    // Set bearer token for subsequent requests
    r.SetAuthToken(tok.AccessToken)
    return &NinjaOneClient{r: r}, nil
}

// ListTags retrieves all asset tags
func (c *NinjaOneClient) ListTags() ([]NinjaTag, error) {
    var result struct { Data []NinjaTag `json:"data"` }
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

// CreateTag creates a new tag by name
func (c *NinjaOneClient) CreateTag(name string) (*NinjaTag, error) {
    var tag NinjaTag
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

// UpdateTag renames an existing tag
func (c *NinjaOneClient) UpdateTag(id, newName string) error {
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

// DeleteTag removes a tag by ID
func (c *NinjaOneClient) DeleteTag(id string) error {
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

