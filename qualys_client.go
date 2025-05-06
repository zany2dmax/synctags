package main

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/go-resty/resty/v2"
)

// Client wraps the Resty client for Qualys Tagging API v2
type Client struct { r *resty.Client }

// NewClient creates a Qualys API client with basic auth
func NewClient(baseURL, username, password string) *Client {
	r := resty.New().
		SetHostURL(baseURL).
		SetBasicAuth(username, password).
		SetHeader("Content-Type", "application/xml")
	return &Client{r: r}
}

// QualysTag models a single tag from Qualys
type QualysTag struct { ID int `xml:"TAG_ID"`; Name string `xml:"TAG_NAME"` }

// tagsResponse wraps the XML from Qualys
type tagsResponse struct {
	XMLName xml.Name `xml:"ServiceResponse"`
	Data    struct{ Tags []QualysTag `xml:"data>ROW"` } `xml:"data"`
}

// ListTags retrieves all tags from Qualys
func (qc *Client) ListTags() ([]QualysTag, error) {
	resp, err := qc.r.R().
		EnableTrace().
		SetQueryParams(map[string]string{"action":"list","truncation_limit":"10000"}).
		SetResult(&tagsResponse{}).
		Get("/api/2.0/fo/asset/tag/")
	if err != nil { return nil, fmt.Errorf("Qualys ListTags failed: %w", err) }

 	fmt.Printf("Qualys replied (HTTP %d):\n%s\n\n", resp.StatusCode(), resp.String())

  result := resp.Result().(*tagsResponse)
	return result.Data.Tags, nil
}

// UpsertTag creates or updates a tag in Qualys
type upsertResponse struct{
	XMLName xml.Name
	Data    struct{ Count int `xml:"count,attr"` } `xml:"data"`
}

func (qc *Client) UpsertTag(t Tag) error {
	// Qualys add: action=add
	payload := fmt.Sprintf("<ServiceRequest><data><ROW><TAG_NAME>%s</TAG_NAME></ROW></data></ServiceRequest>", xmlEscape(t.Name))
	_, err := qc.r.R().
		SetQueryParam("action", "add").
		SetBody(payload).
		Post("/api/2.0/fo/asset/tag/")
	return err
}

func xmlEscape(s string) string { return strings.ReplaceAll(s, "&", "&amp;") }
