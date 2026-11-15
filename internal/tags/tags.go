// Package tags holds the vendor-neutral tag type and its YAML file format.
package tags

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Tag is the normalized representation shared by every vendor.
type Tag struct {
	ID   int    `yaml:"id"`
	Name string `yaml:"name"`
}

// ReadYAML reads tags from a YAML file shaped as {tags: [{id, name}, ...]}.
func ReadYAML(path string) ([]Tag, error) {
	// #nosec G304 -- the path is the operator's own --input flag or config value.
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Tags []Tag `yaml:"tags"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return doc.Tags, nil
}

// WriteYAML writes tags to a YAML file, readable only by the owner.
func WriteYAML(tags []Tag, path string) error {
	data, err := yaml.Marshal(map[string][]Tag{"tags": tags})
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// Normalize turns a vendor tag name into the canonical form: trimmed,
// lower-case, spaces replaced by hyphens.
func Normalize(name string) string {
	n := strings.ToLower(strings.TrimSpace(name))
	return strings.ReplaceAll(n, " ", "-")
}
