// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	low "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
	"sort"
)

// Callback represents a high-level Callback object for OpenAPI 3+.
//
// A map of possible out-of band callbacks related to the parent operation. Each value in the map is a
// PathItem Object that describes a set of requests that may be initiated by the API provider and the expected
// responses. The key value used to identify the path item object is an expression, evaluated at runtime,
// that identifies a URL to use for the callback operation.
//  - https://spec.openapis.org/oas/v3.1.0#callback-object
type Callback struct {
	Expression map[string]*PathItem `json:"-" yaml:"-"`
	Extensions map[string]any       `json:"-" yaml:"-"`
	low        *low.Callback
}

// NewCallback creates a new high-level callback from a low-level one.
func NewCallback(lowCallback *low.Callback) *Callback {
	n := new(Callback)
	n.low = lowCallback
	n.Expression = make(map[string]*PathItem)
	for i := range lowCallback.Expression.Value {
		n.Expression[i.Value] = NewPathItem(lowCallback.Expression.Value[i].Value)
	}
	n.Extensions = make(map[string]any)
	for k, v := range lowCallback.Extensions {
		n.Extensions[k.Value] = v.Value
	}
	return n
}

// GoLow returns the low-level Callback instance used to create the high-level one.
func (c *Callback) GoLow() *low.Callback {
	return c.low
}

// GoLowUntyped will return the low-level Callback instance that was used to create the high-level one, with no type
func (c *Callback) GoLowUntyped() any {
	return c.low
}

// Render will return a YAML representation of the Callback object as a byte slice.
func (c *Callback) Render() ([]byte, error) {
	return yaml.Marshal(c)
}

// MarshalYAML will create a ready to render YAML representation of the Callback object.
func (c *Callback) MarshalYAML() (interface{}, error) {
	// map keys correctly.
	m := utils.CreateEmptyMapNode()
	type cbItem struct {
		cb   *PathItem
		exp  string
		line int
		ext  *yaml.Node
	}
	var mapped []*cbItem

	for k, ex := range c.Expression {
		ln := 999 // default to a high value to weight new content to the bottom.
		if c.low != nil {
			for lKey := range c.low.Expression.Value {
				if lKey.Value == k {
					ln = lKey.KeyNode.Line
				}
			}
		}
		mapped = append(mapped, &cbItem{ex, k, ln, nil})
	}

	// extract extensions
	nb := high.NewNodeBuilder(c, c.low)
	extNode := nb.Render()
	if extNode != nil && extNode.Content != nil {
		var label string
		for u := range extNode.Content {
			if u%2 == 0 {
				label = extNode.Content[u].Value
				continue
			}
			mapped = append(mapped, &cbItem{nil, label,
				extNode.Content[u].Line, extNode.Content[u]})
		}
	}

	sort.Slice(mapped, func(i, j int) bool {
		return mapped[i].line < mapped[j].line
	})
	for j := range mapped {
		if mapped[j].cb != nil {
			rendered, _ := mapped[j].cb.MarshalYAML()
			m.Content = append(m.Content, utils.CreateStringNode(mapped[j].exp))
			m.Content = append(m.Content, rendered.(*yaml.Node))
		}
		if mapped[j].ext != nil {
			m.Content = append(m.Content, utils.CreateStringNode(mapped[j].exp))
			m.Content = append(m.Content, mapped[j].ext)
		}
	}

	return m, nil
}
