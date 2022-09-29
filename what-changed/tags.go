// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package what_changed

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	lowv3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"strings"
)

type TagChanges struct {
	PropertyChanges[*lowbase.Tag]
	ExternalDocs     *ExternalDocChanges
	ExtensionChanges *ExtensionChanges
}

func (t *TagChanges) TotalChanges() int {
	c := len(t.Changes)
	if t.ExternalDocs != nil {
		c += t.ExternalDocs.TotalChanges()
	}
	if t.ExtensionChanges != nil {
		c += len(t.ExtensionChanges.Changes)
	}
	return c
}

func CompareTags(l, r []low.ValueReference[*lowbase.Tag]) *TagChanges {
	tc := new(TagChanges)

	// look at the original and then look through the new.
	seenLeft := make(map[string]*low.ValueReference[*lowbase.Tag])
	seenRight := make(map[string]*low.ValueReference[*lowbase.Tag])
	for i := range l {
		h := l[i]
		seenLeft[strings.ToLower(l[i].Value.Name.Value)] = &h
	}
	for i := range r {
		h := r[i]
		seenRight[strings.ToLower(r[i].Value.Name.Value)] = &h
	}

	var changes []*Change[*lowbase.Tag]

	// check for removals, modifications and moves
	for i := range seenLeft {
		if seenRight[i] == nil {
			// deleted
			CreateChange[*lowbase.Tag](&changes, ObjectRemoved, i, seenLeft[i].ValueNode, nil,
				false, seenLeft[i].Value, nil)
			continue
		}

		// if the existing tag exists, let's check it.
		if seenRight[i] != nil {

			// check if name has moved
			ctx := CreateContext(seenLeft[i].Value.Name.ValueNode, seenRight[i].Value.Name.ValueNode)
			if ctx.HasChanged() {
				CreateChange[*lowbase.Tag](&changes, Moved, lowv3.NameLabel,
					seenLeft[i].Value.Name.ValueNode, seenRight[i].Value.Name.ValueNode,
					false, seenLeft[i].Value, seenRight[i].Value)

			}

			// check if description has been modified
			if seenLeft[i].Value.Description.Value != seenRight[i].Value.Description.Value {
				var changeType int
				changeType = Modified
				ctx = CreateContext(seenLeft[i].Value.Description.ValueNode, seenRight[i].Value.Description.ValueNode)
				if ctx.HasChanged() {
					changeType = ModifiedAndMoved
				}
				CreateChange[*lowbase.Tag](&changes, changeType, lowv3.DescriptionLabel,
					seenLeft[i].Value.Description.ValueNode, seenRight[i].Value.Description.ValueNode,
					false, seenLeft[i].Value, seenRight[i].Value)
			}

			// check if description has moved
			if seenLeft[i].Value.Description.Value == seenRight[i].Value.Description.Value {
				ctx = CreateContext(seenLeft[i].Value.Description.ValueNode, seenRight[i].Value.Description.ValueNode)
				if ctx.HasChanged() {
					CreateChange[*lowbase.Tag](&changes, Moved, lowv3.DescriptionLabel,
						seenLeft[i].Value.Description.ValueNode, seenRight[i].Value.Description.ValueNode,
						false, seenLeft[i].Value, seenRight[i].Value)
				}
			}

			// compare extensions
			var lExt, rExt map[low.KeyReference[string]]low.ValueReference[any]
			if l != nil && len(seenLeft[i].Value.Extensions) > 0 {
				lExt = seenLeft[i].Value.Extensions
			}
			if r != nil && len(seenRight[i].Value.Extensions) > 0 {
				rExt = seenRight[i].Value.Extensions
			}
			tc.ExtensionChanges = CompareExtensions(lExt, rExt)

			// compare external docs
			tc.ExternalDocs = CompareExternalDocs(seenLeft[i].Value.ExternalDocs.Value,
				seenRight[i].Value.ExternalDocs.Value)
		}
	}

	// check for additions
	for i := range seenRight {
		if seenLeft[i] == nil {
			// added
			CreateChange[*lowbase.Tag](&changes, ObjectAdded, i,
				nil, seenRight[i].ValueNode,
				false, nil, seenRight[i].Value)
		}
	}
	if len(changes) <= 0 {
		return nil
	}
	tc.Changes = changes
	return tc
}
