package main

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/go-resty/resty/v2"
)

// Client wraps Resty for the QPS Tagging API
type Client struct{ r *resty.Client }

// NewClient points Resty at your QPS server
func NewClient(baseURL, username, password string) *Client {
	r := resty.New().
		SetHostURL(baseURL).
		SetBasicAuth(username, password).
		SetHeader("Content-Type", "text/xml").
		SetHeader("Accept", "application/xml")
	return &Client{r: r}
}

// QualysTag is one tag’s ID + name
type QualysTag struct {
	ID   int    `xml:"id"`
	Name string `xml:"name"`
}

// tagsResponse matches the <ServiceResponse><data><Tag>… XML
type tagsResponse struct {
	XMLName xml.Name `xml:"ServiceResponse"`
	Data    struct {
		Tags []QualysTag `xml:"Tag"`
	} `xml:"data"`
}

func xmlEscape(s string) string {
	return strings.ReplaceAll(s, "&", "&amp;")
}

const searchTagPath = "/qps/rest/2.0/search/am/tag"

// ListTags retrieves *all* tags via a POST to the Search Tags endpoint
func (qc *Client) ListTags() ([]QualysTag, error) {
	// Build a body that says “no filters, give me up to 10 000 tags”
	reqBody := `
      <ServiceRequest>
        <preferences>
          <startFromOffset>1</startFromOffset>
          <limitResults>1000</limitResults>
        </preferences>
      </ServiceRequest>`
	resp, err := qc.r.R().
		SetBody(reqBody).
		SetResult(&tagsResponse{}).
		Post(searchTagPath)
	if err != nil {
		return nil, fmt.Errorf("Qualys ListTags failed: %w", err)
	}
	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("unexpected HTTP %d from Qualys:\n%s",
			resp.StatusCode(), resp.String())
	}
	result := resp.Result().(*tagsResponse)
	return result.Data.Tags, nil
}

// UpsertTag creates or updates a tag in Qualys via the QPS Update endpoint.
func (qc *Client) UpsertTag(t Tag) error {
	// Build the ServiceRequest XML body.
	// QPS expects <ServiceRequest><data><Tag>…</Tag></data></ServiceRequest>
	reqBody := fmt.Sprintf(`
<ServiceRequest>
  <data>
    <Tag>
      <name>%s</name>
    </Tag>
  </data>
</ServiceRequest>`, xmlEscape(t.Name))

	resp, err := qc.r.R().
		SetBody(reqBody).
		Post("/qps/rest/2.0/update/am/tag")
	if err != nil {
		return fmt.Errorf("Qualys UpsertTag request failed: %w", err)
	}
	if resp.StatusCode() != 200 {
		return fmt.Errorf("Qualys UpsertTag unexpected HTTP %d: %s",
			resp.StatusCode(), resp.String())
	}
	return nil
}
