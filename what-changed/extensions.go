// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package what_changed

import (
	"fmt"
	"github.com/pb33f/libopenapi/datamodel/low"
	"strings"
)

type ExtensionChanges struct {
	PropertyChanges
}

func CompareExtensions(l, r map[low.KeyReference[string]]low.ValueReference[any]) *ExtensionChanges {

	// look at the original and then look through the new.
	seenLeft := make(map[string]*low.ValueReference[any])
	seenRight := make(map[string]*low.ValueReference[any])
	for i := range l {
		h := l[i]
		seenLeft[strings.ToLower(i.Value)] = &h
	}
	for i := range r {
		h := r[i]
		seenRight[strings.ToLower(i.Value)] = &h
	}

	var changes []*Change
	var changeType int
	for i := range seenLeft {
		changeType = 0
		if seenRight[i] == nil {
			// deleted
			changeType = PropertyRemoved
			ctx := CreateContext(seenLeft[i].ValueNode, nil)
			changes = append(changes, &Change{
				Context:    ctx,
				ChangeType: changeType,
				Property:   i,
				Original:   fmt.Sprintf("%v", seenLeft[i].Value),
			})

		}
		if seenRight[i] != nil {
			// potentially modified and or moved
			ctx := CreateContext(seenLeft[i].ValueNode, seenRight[i].ValueNode)
			if seenLeft[i].Value != seenRight[i].Value {
				changeType = Modified
			}
			if ctx.HasChanged() {
				if changeType == Modified {
					changeType = ModifiedAndMoved
				} else {
					changeType = Moved
				}
			}
			if changeType != 0 {
				changes = append(changes, &Change{
					Context:    ctx,
					ChangeType: changeType,
					Property:   i,
					Original:   fmt.Sprintf("%v", seenLeft[i].Value),
					New:        fmt.Sprintf("%v", seenRight[i].Value),
				})
			}
		}
	}
	for i := range seenRight {
		if seenLeft[i] == nil {
			// added
			ctx := CreateContext(nil, seenRight[i].ValueNode)
			changes = append(changes, &Change{
				Context:    ctx,
				ChangeType: PropertyAdded,
				Property:   i,
				New:        fmt.Sprintf("%v", seenRight[i].Value),
			})
		}
	}

	if len(changes) <= 0 {
		return nil
	}
	ex := new(ExtensionChanges)
	ex.Changes = changes
	return ex
}
