// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	lowmodel "github.com/pb33f/libopenapi/datamodel/low"
	low "github.com/pb33f/libopenapi/datamodel/low/v3"
	"sync"
)

// MediaType represents a high-level OpenAPI MediaType object that is backed by a low-level one.
//  - https://spec.openapis.org/oas/v3.1.0#media-type-object
type MediaType struct {
	Schema     *base.SchemaProxy
	Example    any
	Examples   map[string]*base.Example
	Encoding   map[string]*Encoding
	Extensions map[string]any
	low        *low.MediaType
}

func NewMediaType(mediaType *low.MediaType) *MediaType {
	m := new(MediaType)
	m.low = mediaType
	if !mediaType.Schema.IsEmpty() {
		m.Schema = base.NewSchemaProxy(&mediaType.Schema)
	}
	m.Example = mediaType.Example
	m.Examples = base.ExtractExamples(mediaType.Examples.Value)
	m.Extensions = high.ExtractExtensions(mediaType.Extensions)
	m.Encoding = ExtractEncoding(mediaType.Encoding.Value)
	return m
}

func (m *MediaType) GoLow() *low.MediaType {
	return m.low
}

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
