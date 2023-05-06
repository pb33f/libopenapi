// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"fmt"
	"sort"

	"github.com/pb33f/libopenapi/datamodel/high"
	low "github.com/pb33f/libopenapi/datamodel/low/v3"
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
	Codes      map[string]*Response `json:"-" yaml:"-"`
	Default    *Response            `json:"default,omitempty" yaml:"default,omitempty"`
	Extensions map[string]any       `json:"-" yaml:"-"`
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
	codes := make(map[string]*Response)

	// struct to hold response and code sent over chan.
	type respRes struct {
		code string
		resp *Response
	}

	// build each response async for speed
	rChan := make(chan respRes)
	var buildResponse = func(code string, resp *low.Response, c chan respRes) {
		c <- respRes{code: code, resp: NewResponse(resp)}
	}
	for k, v := range responses.Codes {
		go buildResponse(k.Value, v.Value, rChan)
	}
	totalCodes := len(responses.Codes)
	codesParsed := 0
	for codesParsed < totalCodes {
		select {
		case re := <-rChan:
			codesParsed++
			codes[re.code] = re.resp
		}
	}
	r.Codes = codes
	return r
}

// FindResponseByCode is a shortcut for looking up code by an integer vs. a string
func (r *Responses) FindResponseByCode(code int) *Response {
	return r.Codes[fmt.Sprintf("%d", code)]
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

	for k, re := range r.Codes {
		ln := 9999 // default to a high value to weight new content to the bottom.
		if r.low != nil {
			for lKey := range r.low.Codes {
				if lKey.Value == k {
					ln = lKey.KeyNode.Line
				}
			}
		}
		mapped = append(mapped, &responseItem{re, k, ln, nil})
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
			mapped = append(mapped, &responseItem{nil, label,
				extNode.Content[u].Line, extNode.Content[u]})
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

	for k, re := range r.Codes {
		ln := 9999 // default to a high value to weight new content to the bottom.
		if r.low != nil {
			for lKey := range r.low.Codes {
				if lKey.Value == k {
					ln = lKey.KeyNode.Line
				}
			}
		}
		mapped = append(mapped, &responseItem{re, k, ln, nil})
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
			mapped = append(mapped, &responseItem{nil, label,
				extNode.Content[u].Line, extNode.Content[u]})
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
