// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package what_changed

import (
	"gopkg.in/yaml.v3"
)

const (
	Modified = iota + 1
	PropertyAdded
	ObjectAdded
	ObjectRemoved
	PropertyRemoved
	Moved
	ModifiedAndMoved
)

type WhatChanged struct {
	Added        int
	Removed      int
	Modified     int
	Moved        int
	TotalChanges int
	Changes      *Changes
}

type ChangeContext struct {
	OrigLine int
	OrigCol  int
	NewLine  int
	NewCol   int
}

func (c *ChangeContext) HasChanged() bool {
	return c.NewLine != c.OrigLine || c.NewCol != c.OrigCol
}

type Change struct {
	Context    *ChangeContext
	ChangeType int
	Property   string
	Original   string
	New        string
}

type PropertyChanges struct {
	Changes []*Change
}

type TagChanges struct {
	PropertyChanges
	ExternalDocs *ExternalDocChanges
}

type Changes struct {
	TagChanges *TagChanges
}

//func WhatChangedBetweenDocuments(leftDocument, rightDocument *lowv3.Document) *WhatChanged {
//
//	// compare tags
//	//leftTags := leftDocument.Tags.Value
//	//rightTags := rightDocument.Tags.Value
//
//	return nil
//}

func CreateContext(l, r *yaml.Node) *ChangeContext {
	ctx := new(ChangeContext)
	if l != nil {
		ctx.OrigLine = l.Line
		ctx.OrigCol = l.Column
	} else {
		ctx.OrigLine = -1
		ctx.OrigCol = -1
	}
	if r != nil {
		ctx.NewLine = r.Line
		ctx.NewCol = r.Column
	} else {
		ctx.NewLine = -1
		ctx.NewCol = -1
	}
	return ctx
}

//
//func compareTags(l, r []low.ValueReference[*lowbase.Tag]) *TagChanges {
//
//	tc := new(TagChanges)
//
//	// look at the original and then look through the new.
//	seenLeft := make(map[string]*low.ValueReference[*lowbase.Tag])
//	seenRight := make(map[string]*low.ValueReference[*lowbase.Tag])
//	for i := range l {
//		seenLeft[strings.ToLower(l[i].Value.Name.Value)] = &l[i]
//	}
//	for i := range r {
//		seenRight[strings.ToLower(l[i].Value.Name.Value)] = &l[i]
//	}
//
//	for i := range seenLeft {
//		if seenRight[i] == nil {
//			// deleted
//			//ctx := CreateContext(seenLeft[i].ValueNode, nil)
//			//tc.Changes =
//
//		}
//		if seenRight[i] != nil {
//
//			// potentially modified and or moved
//		}
//	}
//
//	for i := range seenRight {
//		if seenLeft[i] == nil {
//			// added
//		}
//	}
//
//	for i := range r {
//		// if we find a match
//		t := r[i]
//		name := r[i].Value.Name.Value
//		found := seenLeft[strings.ToLower(name)]
//		if found.Value != nil {
//
//			// check values
//			if found.Value.Description.Value != t.Value.Description.Value {
//				ctx := CreateContext(found.ValueNode, t.ValueNode)
//				changeType := Modified
//				if ctx.HasChanged() {
//					changeType = ModifiedAndMoved
//				}
//				tc.Changes = append(tc.Changes, &Change{
//					Context:    ctx,
//					ChangeType: changeType,
//					Property:   lowv3.DescriptionLabel,
//					Original:   found.Value.Description.Value,
//					New:        t.Value.Description.Value,
//				})
//			}
//
//		} else {
//
//			// new stuff
//
//		}
//
//	}
//
//	// more tags in right hand-side
//	if len(r) > len(l) {
//
//	}
//
//	// less tags in right hand-side
//	if len(r) < len(l) {
//
//	}
//
//	//for i := range a {
//	//	eq, l, c := comparePositions(a)
//	//
//	//}
//
//	return nil
//}
//
//func comparePositions(left, right *yaml.Node) (bool, int, int) {
//	if left.Line == right.Line && left.Column == right.Column {
//		return true, 0, 0
//	}
//	return false, right.Line, right.Column
//}
