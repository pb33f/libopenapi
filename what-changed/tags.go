// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package what_changed

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/datamodel/low/v3"
	"strings"
)

// TagChanges represents changes made to the Tags object of an OpenAPI document.
type TagChanges struct {
	PropertyChanges[*base.Tag]
	ExternalDocs     *ExternalDocChanges
	ExtensionChanges *ExtensionChanges
}

// TotalChanges returns a count of everything that changed within tags.
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

// CompareTags will compare a left (original) and a right (new) slice of ValueReference nodes for
// any changes between them. If there are changes, a pointer to TagChanges is returned, if not then
// nil is returned instead.
func CompareTags(l, r []low.ValueReference[*base.Tag]) *TagChanges {
	tc := new(TagChanges)

	// look at the original and then look through the new.
	seenLeft := make(map[string]*low.ValueReference[*base.Tag])
	seenRight := make(map[string]*low.ValueReference[*base.Tag])
	for i := range l {
		h := l[i]
		seenLeft[strings.ToLower(l[i].Value.Name.Value)] = &h
	}
	for i := range r {
		h := r[i]
		seenRight[strings.ToLower(r[i].Value.Name.Value)] = &h
	}

	var changes []*Change[*base.Tag]

	// check for removals, modifications and moves
	for i := range seenLeft {

		CheckForObjectAdditionOrRemoval[*base.Tag](seenLeft, seenRight, i, &changes, false, true)

		// if the existing tag exists, let's check it.
		if seenRight[i] != nil {

			var props []*PropertyCheck[*base.Tag]

			// Name
			props = append(props, &PropertyCheck[*base.Tag]{
				LeftNode:  seenLeft[i].Value.Name.ValueNode,
				RightNode: seenRight[i].Value.Name.ValueNode,
				Label:     v3.NameLabel,
				Changes:   &changes,
				Breaking:  true,
				Original:  seenLeft[i].Value,
				New:       seenRight[i].Value,
			})

			// Description
			props = append(props, &PropertyCheck[*base.Tag]{
				LeftNode:  seenLeft[i].Value.Description.ValueNode,
				RightNode: seenRight[i].Value.Description.ValueNode,
				Label:     v3.DescriptionLabel,
				Changes:   &changes,
				Breaking:  true,
				Original:  seenLeft[i].Value,
				New:       seenRight[i].Value,
			})

			// check properties
			CheckProperties(props)

			// check extensions
			tc.ExtensionChanges = CheckExtensions(seenLeft[i].GetValue(), seenRight[i].GetValue())

			// compare external docs
			tc.ExternalDocs = CompareExternalDocs(seenLeft[i].Value.ExternalDocs.Value,
				seenRight[i].Value.ExternalDocs.Value)
		}
	}

	if len(changes) <= 0 {
		return nil
	}
	tc.Changes = changes
	return tc
}
