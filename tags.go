package main

import (
	"strings"
	"gopkg.in/yaml.v3"
	"io"
)

// Tag is the normalized representation
type Tag struct {
	ID   int    `yaml:"id"`
	Name string `yaml:"name"`
}

// ReadTagsFromYAML reads tags from a YAML file
func ReadTagsFromYAML(path string) ([]Tag, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc struct{ Tags []Tag `yaml:"tags"` }
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return doc.Tags, nil
}

// WriteTagsToYAML writes the slice of Tag into a YAML file
func WriteTagsToYAML(tags []Tag, path string) error {
	data, err := yaml.Marshal(map[string][]Tag{"tags": tags})
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, data, 0644)
}

// NormalizeQualysTag converts a QualysTag to the canonical Tag
func NormalizeQualysTag(qt QualysTag) Tag {
	// import "strings"
	n := strings.TrimSpace(qt.Name)
	n = strings.ToLower(n)
	n = strings.ReplaceAll(n, " ", "-" )
	return Tag{ID: qt.ID, Name: n}
}
