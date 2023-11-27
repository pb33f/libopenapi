// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"fmt"
	"sort"

	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/datamodel/high"
	lowbase "github.com/pb33f/libopenapi/datamodel/low"
	low "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

// Responses represents a high-level OpenAPI 3+ Responses object that is backed by a low-level one.
//
// It's a container for the expected responses of an operation. The container maps a HTTP response code to the
// expected response.
//
// The specification is not necessarily expected to cover all possible HTTP response codes because they may not be
// known in advance. However, documentation is expected to cover a successful operation response and any known errors.
//
// The default MAY be used as a default response object for all HTTP codes that are not covered individually by
// the Responses Object.
//
// The Responses Object MUST contain at least one response code, and if only one response code is provided it SHOULD
// be the response for a successful operation call.
//   - https://spec.openapis.org/oas/v3.1.0#responses-object
type Responses struct {
	Codes      orderedmap.Map[string, *Response] `json:"-" yaml:"-"`
	Default    *Response                         `json:"default,omitempty" yaml:"default,omitempty"`
	Extensions map[string]any                    `json:"-" yaml:"-"`
	low        *low.Responses
}

// NewResponses will create a new high-level Responses instance from a low-level one. It operates asynchronously
// internally, as each response may be considerable in complexity.
func NewResponses(responses *low.Responses) *Responses {
	r := new(Responses)
	r.low = responses
	r.Extensions = high.ExtractExtensions(responses.Extensions)
	if !responses.Default.IsEmpty() {
		r.Default = NewResponse(responses.Default.Value)
	}
	codes := orderedmap.New[string, *Response]()

	translateFunc := func(pair orderedmap.Pair[lowbase.KeyReference[string], lowbase.ValueReference[*low.Response]]) (asyncResult[*Response], error) {
		return asyncResult[*Response]{
			key:    pair.Key().Value,
			result: NewResponse(pair.Value().Value),
		}, nil
	}
	resultFunc := func(value asyncResult[*Response]) error {
		codes.Set(value.key, value.result)
		return nil
	}
	_ = datamodel.TranslateMapParallel[lowbase.KeyReference[string], lowbase.ValueReference[*low.Response]](responses.Codes, translateFunc, resultFunc)
	r.Codes = codes
	return r
}

// FindResponseByCode is a shortcut for looking up code by an integer vs. a string
func (r *Responses) FindResponseByCode(code int) *Response {
	return r.Codes.GetOrZero(fmt.Sprintf("%d", code))
}

// GoLow returns the low-level Response object used to create the high-level one.
func (r *Responses) GoLow() *low.Responses {
	return r.low
}

// GoLowUntyped will return the low-level Responses instance that was used to create the high-level one, with no type
func (r *Responses) GoLowUntyped() any {
	return r.low
}

// Render will return a YAML representation of the Responses object as a byte slice.
func (r *Responses) Render() ([]byte, error) {
	return yaml.Marshal(r)
}

func (r *Responses) RenderInline() ([]byte, error) {
	d, _ := r.MarshalYAMLInline()
	return yaml.Marshal(d)
}

// MarshalYAML will create a ready to render YAML representation of the Responses object.
func (r *Responses) MarshalYAML() (interface{}, error) {
	// map keys correctly.
	m := utils.CreateEmptyMapNode()
	type responseItem struct {
		resp *Response
		code string
		line int
		ext  *yaml.Node
	}
	var mapped []*responseItem

	for pair := orderedmap.First(r.Codes); pair != nil; pair = pair.Next() {
		ln := 9999 // default to a high value to weight new content to the bottom.
		if r.low != nil {
			for lPair := orderedmap.First(r.low.Codes); lPair != nil; lPair = lPair.Next() {
				if lPair.Key().Value == pair.Key() {
					ln = lPair.Key().KeyNode.Line
				}
			}
		}
		mapped = append(mapped, &responseItem{pair.Value(), pair.Key(), ln, nil})
	}

	// extract extensions
	nb := high.NewNodeBuilder(r, r.low)
	extNode := nb.Render()
	if extNode != nil && extNode.Content != nil {
		var label string
		for u := range extNode.Content {
			if u%2 == 0 {
				label = extNode.Content[u].Value
				continue
			}
			mapped = append(mapped, &responseItem{
				nil, label,
				extNode.Content[u].Line, extNode.Content[u],
			})
		}
	}

	sort.Slice(mapped, func(i, j int) bool {
		return mapped[i].line < mapped[j].line
	})
	for j := range mapped {
		if mapped[j].resp != nil {
			rendered, _ := mapped[j].resp.MarshalYAML()
			m.Content = append(m.Content, utils.CreateStringNode(mapped[j].code))
			m.Content = append(m.Content, rendered.(*yaml.Node))
		}
		if mapped[j].ext != nil {
			m.Content = append(m.Content, utils.CreateStringNode(mapped[j].code))
			m.Content = append(m.Content, mapped[j].ext)
		}

	}
	return m, nil
}

func (r *Responses) MarshalYAMLInline() (interface{}, error) {
	// map keys correctly.
	m := utils.CreateEmptyMapNode()
	type responseItem struct {
		resp *Response
		code string
		line int
		ext  *yaml.Node
	}
	var mapped []*responseItem

	for pair := orderedmap.First(r.Codes); pair != nil; pair = pair.Next() {
		ln := 9999 // default to a high value to weight new content to the bottom.
		if r.low != nil {
			for lPair := orderedmap.First(r.low.Codes); lPair != nil; lPair = lPair.Next() {
				if lPair.Key().Value == pair.Key() {
					ln = lPair.Key().KeyNode.Line
				}
			}
		}
		mapped = append(mapped, &responseItem{pair.Value(), pair.Key(), ln, nil})
	}

	// extract extensions
	nb := high.NewNodeBuilder(r, r.low)
	nb.Resolve = true
	extNode := nb.Render()
	if extNode != nil && extNode.Content != nil {
		var label string
		for u := range extNode.Content {
			if u%2 == 0 {
				label = extNode.Content[u].Value
				continue
			}
			mapped = append(mapped, &responseItem{
				nil, label,
				extNode.Content[u].Line, extNode.Content[u],
			})
		}
	}

	sort.Slice(mapped, func(i, j int) bool {
		return mapped[i].line < mapped[j].line
	})
	for j := range mapped {
		if mapped[j].resp != nil {
			rendered, _ := mapped[j].resp.MarshalYAMLInline()
			m.Content = append(m.Content, utils.CreateStringNode(mapped[j].code))
			m.Content = append(m.Content, rendered.(*yaml.Node))

		}
		if mapped[j].ext != nil {
			m.Content = append(m.Content, utils.CreateStringNode(mapped[j].code))
			m.Content = append(m.Content, mapped[j].ext)
		}

	}
	return m, nil
}
