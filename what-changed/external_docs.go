// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package what_changed

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	lowv3 "github.com/pb33f/libopenapi/datamodel/low/v3"
)

type ExternalDocChanges struct {
	PropertyChanges[*lowbase.ExternalDoc]
	ExtensionChanges *ExtensionChanges
}

func (e *ExternalDocChanges) TotalChanges() int {
	c := len(e.Changes)
	if e.ExtensionChanges != nil {
		c += len(e.ExtensionChanges.Changes)
	}
	return c
}

func CompareExternalDocs(l, r *lowbase.ExternalDoc) *ExternalDocChanges {
	var changes []*Change[*lowbase.ExternalDoc]

	skipURL := false
	skipDescription := false

	// check if url was removed
	if r.URL.Value == "" && l.URL.Value != "" {
		CreateChange[*lowbase.ExternalDoc](&changes, PropertyRemoved, lowv3.URLLabel, l.URL.ValueNode,
			nil, false, l, nil)
		skipURL = true
	}

	// check if description was removed
	if r.Description.Value == "" && l.Description.Value != "" {
		CreateChange[*lowbase.ExternalDoc](&changes, PropertyRemoved, lowv3.DescriptionLabel, l.Description.ValueNode,
			nil, false, l, nil)
		skipDescription = true
	}

	// check if url was added
	if r.URL.Value != "" && l.URL.Value == "" {
		CreateChange[*lowbase.ExternalDoc](&changes, PropertyAdded, lowv3.URLLabel, nil,
			r.URL.ValueNode, false, nil, r)
		skipURL = true
	}

	// check if description was added
	if r.Description.Value != "" && l.Description.Value == "" {
		CreateChange[*lowbase.ExternalDoc](&changes, PropertyAdded, lowv3.DescriptionLabel, nil,
			r.Description.ValueNode, false, nil, r)
		skipDescription = true
	}

	// if left and right URLs are set but not equal
	if !skipURL && l != nil && r != nil && l.URL.Value != r.URL.Value {
		var changeType int
		changeType = Modified
		ctx := CreateContext(l.URL.ValueNode, r.URL.ValueNode)
		if ctx.HasChanged() {
			changeType = ModifiedAndMoved
		}
		CreateChange[*lowbase.ExternalDoc](&changes, changeType, lowv3.URLLabel, l.URL.ValueNode,
			r.URL.ValueNode, false, l, r)
	}

	// if left and right descriptions are set, but not equal
	if !skipDescription && l != nil && r != nil && l.Description.Value != r.Description.Value {
		var changeType int
		changeType = Modified
		ctx := CreateContext(l.Description.ValueNode, r.Description.ValueNode)
		if ctx.HasChanged() {
			changeType = ModifiedAndMoved
		}
		CreateChange[*lowbase.ExternalDoc](&changes, changeType, lowv3.DescriptionLabel, l.Description.ValueNode,
			r.Description.ValueNode, false, l, r)
	}
	dc := new(ExternalDocChanges)
	dc.Changes = changes
	var lExt, rExt map[low.KeyReference[string]]low.ValueReference[any]
	if l != nil && len(l.Extensions) > 0 {
		lExt = l.Extensions
	}
	if r != nil && len(r.Extensions) > 0 {
		rExt = r.Extensions
	}
	dc.ExtensionChanges = CompareExtensions(lExt, rExt)
	if len(dc.Changes) <= 0 && dc.ExtensionChanges == nil {
		return nil
	}
	return dc
}
