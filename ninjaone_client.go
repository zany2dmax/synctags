// ninjaone_client.go
// NinjaOne Public API tagging integration (package main)
// Implements Asset Tag CRUD, batch operations, and asset-to-tag assignments.
// Endpoints correspond to operations createTag, getTags, updateTag, deleteTag,
// deleteTagsBatch, mergeTags, batchTagAssets from the core-resources API.

package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"
    "time"
)

// NinjaOneClient holds auth and HTTP client
type NinjaOneClient struct {
    BaseURL      string
    ClientID     string
    ClientSecret string
    Token        string
    HTTPClient   *http.Client
}

// NinjaTag is the core tag model
type NinjaTag struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}

// NewNinjaOneClient authenticates via OAuth2 and returns a client
func NewNinjaOneClient(baseURL, clientID, clientSecret string) (*NinjaOneClient, error) {
    c := &NinjaOneClient{
        BaseURL:      baseURL,
        ClientID:     clientID,
        ClientSecret: clientSecret,
        HTTPClient:   &http.Client{Timeout: 10 * time.Second},
    }
    if err := c.authenticate(); err != nil {
        return nil, err
    }
    return c, nil
}

// authenticate obtains a bearer token
func (c *NinjaOneClient) authenticate() error {
    url := fmt.Sprintf("%s/oauth2/token", c.BaseURL)
    creds := map[string]string{
        "client_id":     c.ClientID,
        "client_secret": c.ClientSecret,
    }
    body, _ := json.Marshal(creds)
    req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
    req.Header.Set("Content-Type", "application/json")

    resp, err := c.HTTPClient.Do(req)
    if err != nil {
        return fmt.Errorf("auth request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        data, _ := ioutil.ReadAll(resp.Body)
        return fmt.Errorf("auth HTTP %d: %s", resp.StatusCode, string(data))
    }

    var tok struct{ AccessToken string `json:"access_token"` }
    if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
        return fmt.Errorf("auth decode error: %w", err)
    }
    c.Token = tok.AccessToken
    return nil
}

// ListTags retrieves all asset tags
func (c *NinjaOneClient) ListTags() ([]NinjaTag, error) {
    url := fmt.Sprintf("%s/core-resources/assetTags", c.BaseURL)
    req, _ := http.NewRequest("GET", url, nil)
    req.Header.Set("Authorization", "Bearer "+c.Token)

    resp, err := c.HTTPClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("ListTags failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        data, _ := ioutil.ReadAll(resp.Body)
        return nil, fmt.Errorf("ListTags HTTP %d: %s", resp.StatusCode, string(data))
    }

    var result struct{ Data []NinjaTag `json:"data"` }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("ListTags decode error: %w", err)
    }
    return result.Data, nil
}

// CreateTag creates a new asset tag
func (c *NinjaOneClient) CreateTag(name string) (*NinjaTag, error) {
    url := fmt.Sprintf("%s/core-resources/assetTags", c.BaseURL)
    payload := map[string]string{"name": name}
    body, _ := json.Marshal(payload)
    req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
    req.Header.Set("Authorization", "Bearer "+c.Token)
    req.Header.Set("Content-Type", "application/json")

    resp, err := c.HTTPClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("CreateTag failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusCreated {
        data, _ := ioutil.ReadAll(resp.Body)
        return nil, fmt.Errorf("CreateTag HTTP %d: %s", resp.StatusCode, string(data))
    }

    var tag NinjaTag
    if err := json.NewDecoder(resp.Body).Decode(&tag); err != nil {
        return nil, fmt.Errorf("CreateTag decode error: %w", err)
    }
    return &tag, nil
}

// UpdateTag renames an existing tag
func (c *NinjaOneClient) UpdateTag(id, newName string) error {
    url := fmt.Sprintf("%s/core-resources/assetTags/%s", c.BaseURL, id)
    payload := map[string]string{"name": newName}
    body, _ := json.Marshal(payload)
    req, _ := http.NewRequest("PUT", url, bytes.NewBuffer(body))
    req.Header.Set("Authorization", "Bearer "+c.Token)
    req.Header.Set("Content-Type", "application/json")

    resp, err := c.HTTPClient.Do(req)
    if err != nil {
        return fmt.Errorf("UpdateTag failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusNoContent {
        data, _ := ioutil.ReadAll(resp.Body)
        return fmt.Errorf("UpdateTag HTTP %d: %s", resp.StatusCode, string(data))
    }
    return nil
}

// DeleteTag removes a tag by ID
func (c *NinjaOneClient) DeleteTag(id string) error {
    url := fmt.Sprintf("%s/core-resources/assetTags/%s", c.BaseURL, id)
    req, _ := http.NewRequest("DELETE", url, nil)
    req.Header.Set("Authorization", "Bearer "+c.Token)

    resp, err := c.HTTPClient.Do(req)
    if err != nil {
        return fmt.Errorf("DeleteTag failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusNoContent {
        data, _ := ioutil.ReadAll(resp.Body)
        return fmt.Errorf("DeleteTag HTTP %d: %s", resp.StatusCode, string(data))
    }
    return nil
}

// Additional operations (batch delete, merge, batch tag assets) can be added similarly
