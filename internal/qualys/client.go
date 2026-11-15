// Package qualys is a client for the Qualys QPS Tagging API.
package qualys

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/go-resty/resty/v2"
)

// Client wraps Resty for the QPS Tagging API.
type Client struct{ r *resty.Client }

// NewClient points Resty at the QPS server.
func NewClient(baseURL, username, password string) *Client {
	r := resty.New().
		SetBaseURL(baseURL).
		SetBasicAuth(username, password).
		SetHeader("Content-Type", "text/xml").
		SetHeader("Accept", "application/xml")
	return &Client{r: r}
}

// Tag is one Qualys tag's ID and name.
type Tag struct {
	ID   int    `xml:"id"`
	Name string `xml:"name"`
}

// tagsResponse matches the <ServiceResponse><data><Tag>... XML.
type tagsResponse struct {
	XMLName xml.Name `xml:"ServiceResponse"`
	Data    struct {
		Tags []Tag `xml:"Tag"`
	} `xml:"data"`
}

func xmlEscape(s string) string {
	return strings.ReplaceAll(s, "&", "&amp;")
}

const searchTagPath = "/qps/rest/2.0/search/am/tag"

// ListTags retrieves tags via a POST to the Search Tags endpoint.
// Known bug: stops at 1,000 tags (no paging yet).
func (qc *Client) ListTags() ([]Tag, error) {
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
		return nil, fmt.Errorf("qualys ListTags failed: %w", err)
	}
	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("unexpected HTTP %d from Qualys:\n%s",
			resp.StatusCode(), resp.String())
	}
	result, ok := resp.Result().(*tagsResponse)
	if !ok {
		return nil, fmt.Errorf("qualys ListTags: unexpected result type %T", resp.Result())
	}
	return result.Data.Tags, nil
}

// UpsertTag creates or updates a tag in Qualys via the QPS Update endpoint.
// Known bug: the update endpoint needs a tag ID, so this cannot create.
func (qc *Client) UpsertTag(name string) error {
	reqBody := fmt.Sprintf(`
<ServiceRequest>
  <data>
    <Tag>
      <name>%s</name>
    </Tag>
  </data>
</ServiceRequest>`, xmlEscape(name))

	resp, err := qc.r.R().
		SetBody(reqBody).
		Post("/qps/rest/2.0/update/am/tag")
	if err != nil {
		return fmt.Errorf("qualys UpsertTag request failed: %w", err)
	}
	if resp.StatusCode() != 200 {
		return fmt.Errorf("qualys UpsertTag unexpected HTTP %d: %s",
			resp.StatusCode(), resp.String())
	}
	return nil
}
