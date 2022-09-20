// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"fmt"
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/utils"
	"gopkg.in/yaml.v3"
)

const (
	LinksLabel   = "links"
	DefaultLabel = "default"
)

type Responses struct {
	Codes   map[low.KeyReference[string]]low.ValueReference[*Response]
	Default low.NodeReference[*Response]
}

func (r *Responses) Build(root *yaml.Node, idx *index.SpecIndex) error {
	if utils.IsNodeMap(root) {
		codes, err := low.ExtractMapNoLookup[*Response](root, idx)
		if err != nil {
			return err
		}
		if codes != nil {
			r.Codes = codes
		}

		def, derr := low.ExtractObject[*Response](DefaultLabel, root, idx)
		if derr != nil {
			return derr
		}
		if def.Value != nil {
			r.Default = def
		}
	} else {
		return fmt.Errorf("responses build failed: vn node is not a map! line %d, col %d", root.Line, root.Column)
	}
	return nil
}

func (r *Responses) FindResponseByCode(code string) *low.ValueReference[*Response] {
	return low.FindItemInMap[*Response](code, r.Codes)
}

type Response struct {
	Description low.NodeReference[string]
	Headers     low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Header]]
	Content     low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*MediaType]]
	Extensions  map[low.KeyReference[string]]low.ValueReference[any]
	Links       low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Link]]
}

func (r *Response) FindExtension(ext string) *low.ValueReference[any] {
	return low.FindItemInMap[any](ext, r.Extensions)
}

func (r *Response) FindContent(cType string) *low.ValueReference[*MediaType] {
	return low.FindItemInMap[*MediaType](cType, r.Content.Value)
}

func (r *Response) FindHeader(hType string) *low.ValueReference[*Header] {
	return low.FindItemInMap[*Header](hType, r.Headers.Value)
}

func (r *Response) FindLink(hType string) *low.ValueReference[*Link] {
	return low.FindItemInMap[*Link](hType, r.Links.Value)
}

func (r *Response) Build(root *yaml.Node, idx *index.SpecIndex) error {
	r.Extensions = low.ExtractExtensions(root)

	//extract headers
	headers, lN, kN, err := low.ExtractMap[*Header](HeadersLabel, root, idx)
	if err != nil {
		return err
	}
	if headers != nil {
		r.Headers = low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Header]]{
			Value:     headers,
			KeyNode:   lN,
			ValueNode: kN,
		}
	}

	con, clN, cN, cErr := low.ExtractMap[*MediaType](ContentLabel, root, idx)
	if cErr != nil {
		return cErr
	}
	if con != nil {
		r.Content = low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*MediaType]]{
			Value:     con,
			KeyNode:   clN,
			ValueNode: cN,
		}
	}

	// handle links if set
	links, linkLabel, linkValue, lErr := low.ExtractMap[*Link](LinksLabel, root, idx)
	if lErr != nil {
		return lErr
	}
	if links != nil {
		r.Links = low.NodeReference[map[low.KeyReference[string]]low.ValueReference[*Link]]{
			Value:     links,
			KeyNode:   linkLabel,
			ValueNode: linkValue,
		}
	}

	return nil
}
