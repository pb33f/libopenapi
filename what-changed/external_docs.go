// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package what_changed

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	lowv3 "github.com/pb33f/libopenapi/datamodel/low/v3"
)

type ExternalDocChanges struct {
	PropertyChanges
	ExtensionChanges *ExtensionChanges
}

func CompareExternalDocs(l, r *lowbase.ExternalDoc) *ExternalDocChanges {
	var changes []*Change
	changeType := 0
	if l != nil && r != nil && l.URL.Value != r.URL.Value {
		changeType = Modified
		ctx := CreateContext(l.URL.ValueNode, r.URL.ValueNode)
		if ctx.HasChanged() {
			changeType = ModifiedAndMoved
		}
		changes = append(changes, &Change{
			Context:    ctx,
			ChangeType: changeType,
			Property:   lowv3.URLLabel,
			Original:   l.URL.Value,
			New:        r.URL.Value,
		})
	}
	if l != nil && r != nil && l.Description.Value != r.Description.Value {
		changeType = Modified
		ctx := CreateContext(l.Description.ValueNode, r.Description.ValueNode)
		if ctx.HasChanged() {
			changeType = ModifiedAndMoved
		}
		changes = append(changes, &Change{
			Context:    ctx,
			ChangeType: changeType,
			Property:   lowv3.DescriptionLabel,
			Original:   l.Description.Value,
			New:        r.Description.Value,
		})
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
