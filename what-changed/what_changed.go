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

type Change[T any] struct {
	Context        *ChangeContext
	ChangeType     int
	Property       string
	Original       string
	New            string
	Breaking       bool
	OriginalObject T
	NewObject      T
}

type PropertyChanges[T any] struct {
	Changes []*Change[T]
}

type Changes struct {
	TagChanges *TagChanges
}

func CreateChange[T any](changes *[]*Change[T], changeType int, property string, leftValueNode, rightValueNode *yaml.Node,
	breaking bool, originalObject, newObject T) *[]*Change[T] {

	ctx := CreateContext(leftValueNode, rightValueNode)
	c := &Change[T]{
		Context:    ctx,
		ChangeType: changeType,
		Property:   property,
		Breaking:   breaking,
	}
	if leftValueNode != nil && leftValueNode.Value != "" {
		c.Original = leftValueNode.Value
	}
	if rightValueNode != nil && rightValueNode.Value != "" {
		c.New = rightValueNode.Value
	}
	c.OriginalObject = originalObject
	c.NewObject = newObject
	*changes = append(*changes, c)
	return changes
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
