// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	lowmodel "github.com/pb33f/libopenapi/datamodel/low"
	low "github.com/pb33f/libopenapi/datamodel/low/3.0"
)

type MediaType struct {
	Schema     *Schema
	Example    any
	Examples   map[string]*Example
	Encoding   map[string]*Encoding
	Extensions map[string]any
	low        *low.MediaType
}

func NewMediaType(mediaType *low.MediaType) *MediaType {
	m := new(MediaType)
	m.low = mediaType
	if !mediaType.Schema.IsEmpty() {
		// check if schema has been seen or not.
		if v := getSeenSchema(mediaType.Schema.GenerateMapKey()); v != nil {
			m.Schema = v
		} else {
			m.Schema = NewSchema(mediaType.Schema.Value)
			addSeenSchema(mediaType.Schema.GenerateMapKey(), m.Schema)
		}
	}
	m.Example = mediaType.Example
	m.Examples = ExtractExamples(mediaType.Examples.Value)
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
	extractContentItem := func(k lowmodel.KeyReference[string],
		v lowmodel.ValueReference[*low.MediaType], c chan bool, e map[string]*MediaType) {
		e[k.Value] = NewMediaType(v.Value)
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
