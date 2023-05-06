// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"sync"

	"github.com/pb33f/libopenapi/datamodel/high"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	lowmodel "github.com/pb33f/libopenapi/datamodel/low"
	low "github.com/pb33f/libopenapi/datamodel/low/v3"
	"gopkg.in/yaml.v3"
)

// MediaType represents a high-level OpenAPI MediaType object that is backed by a low-level one.
//
// Each Media Type Object provides schema and examples for the media type identified by its key.
//   - https://spec.openapis.org/oas/v3.1.0#media-type-object
type MediaType struct {
	Schema     *base.SchemaProxy        `json:"schema,omitempty" yaml:"schema,omitempty"`
	Example    any                      `json:"example,omitempty" yaml:"example,omitempty"`
	Examples   map[string]*base.Example `json:"examples,omitempty" yaml:"examples,omitempty"`
	Encoding   map[string]*Encoding     `json:"encoding,omitempty" yaml:"encoding,omitempty"`
	Extensions map[string]any           `json:"-" yaml:"-"`
	low        *low.MediaType
}

// NewMediaType will create a new high-level MediaType instance from a low-level one.
func NewMediaType(mediaType *low.MediaType) *MediaType {
	m := new(MediaType)
	m.low = mediaType
	if !mediaType.Schema.IsEmpty() {
		m.Schema = base.NewSchemaProxy(&mediaType.Schema)
	}
	m.Example = mediaType.Example.Value
	m.Examples = base.ExtractExamples(mediaType.Examples.Value)
	m.Extensions = high.ExtractExtensions(mediaType.Extensions)
	m.Encoding = ExtractEncoding(mediaType.Encoding.Value)
	return m
}

// GoLow will return the low-level instance of MediaType used to create the high-level one.
func (m *MediaType) GoLow() *low.MediaType {
	return m.low
}

// GoLowUntyped will return the low-level MediaType instance that was used to create the high-level one, with no type
func (m *MediaType) GoLowUntyped() any {
	return m.low
}

// Render will return a YAML representation of the MediaType object as a byte slice.
func (m *MediaType) Render() ([]byte, error) {
	return yaml.Marshal(m)
}

func (m *MediaType) RenderInline() ([]byte, error) {
	d, _ := m.MarshalYAMLInline()
	return yaml.Marshal(d)
}

// MarshalYAML will create a ready to render YAML representation of the MediaType object.
func (m *MediaType) MarshalYAML() (interface{}, error) {
	nb := high.NewNodeBuilder(m, m.low)
	return nb.Render(), nil
}

func (m *MediaType) MarshalYAMLInline() (interface{}, error) {
	nb := high.NewNodeBuilder(m, m.low)
	nb.Resolve = true
	return nb.Render(), nil
}

// ExtractContent takes in a complex and hard to navigate low-level content map, and converts it in to a much simpler
// and easier to navigate high-level one.
func ExtractContent(elements map[lowmodel.KeyReference[string]]lowmodel.ValueReference[*low.MediaType]) map[string]*MediaType {
	// extract everything async
	doneChan := make(chan bool)

	var extLock sync.RWMutex
	extractContentItem := func(k lowmodel.KeyReference[string],
		v lowmodel.ValueReference[*low.MediaType], c chan bool, e map[string]*MediaType) {
		extLock.Lock()
		e[k.Value] = NewMediaType(v.Value)
		extLock.Unlock()
		c <- true

	}
	extracted := make(map[string]*MediaType)
	for k, v := range elements {
		go extractContentItem(k, v, doneChan, extracted)
	}
	n := 0
	for n < len(elements) {
		select {
		case <-doneChan:
			n++
		}
	}
	return extracted
}
