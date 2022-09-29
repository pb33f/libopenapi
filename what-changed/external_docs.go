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
	changeType := 0
	if l != nil && r != nil && l.URL.Value != r.URL.Value {
		changeType = Modified
		ctx := CreateContext(l.URL.ValueNode, r.URL.ValueNode)
		if ctx.HasChanged() {
			changeType = ModifiedAndMoved
		}
		CreateChange[*lowbase.ExternalDoc](&changes, changeType, lowv3.URLLabel, l.URL.ValueNode,
			r.URL.ValueNode, false, l, r)
	}
	if l != nil && r != nil && l.Description.Value != r.Description.Value {
		changeType = Modified
		ctx := CreateContext(l.Description.ValueNode, r.Description.ValueNode)
		if ctx.HasChanged() {
			changeType = ModifiedAndMoved
		}
		CreateChange[*lowbase.ExternalDoc](&changes, changeType, lowv3.DescriptionLabel, l.Description.ValueNode,
			r.Description.ValueNode, false, l, r)
	}
	if changeType == 0 {
		// no change, return nothing.
		return nil
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
	return dc
}
