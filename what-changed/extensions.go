// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package what_changed

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"strings"
)

type ExtensionChanges struct {
	PropertyChanges[any]
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

	var changes []*Change[any]
	for i := range seenLeft {
		if seenRight[i] == nil {
			// deleted
			CreateChange[any](&changes, PropertyRemoved, i, seenLeft[i].ValueNode, nil, false, l, nil)

		}
		if seenRight[i] != nil {
			// potentially modified and or moved
			var changeType int
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
				CreateChange[any](&changes, changeType, i, seenLeft[i].ValueNode, seenRight[i].ValueNode, false, l, r)
			}
		}
	}
	for i := range seenRight {
		if seenLeft[i] == nil {
			// added
			CreateChange[any](&changes, PropertyAdded, i, nil, seenRight[i].ValueNode, false, nil, r)
		}
	}

	if len(changes) <= 0 {
		return nil
	}
	ex := new(ExtensionChanges)
	ex.Changes = changes
	return ex
}
