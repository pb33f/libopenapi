// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package what_changed

import (
	"github.com/pb33f/libopenapi/datamodel/low"
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

type PropertyCheck[T any] struct {
	Original  T
	New       T
	Label     string
	LeftNode  *yaml.Node
	RightNode *yaml.Node
	Breaking  bool
	Changes   *[]*Change[T]
}

func CheckForAdditionOrRemoval[T any](l, r map[string]*low.ValueReference[T], label string, changes *[]*Change[T],
	breakingAdd, breakingRemove bool) {
	var left, right T
	if CheckObjectRemoved(l, r) {
		left = l[label].GetValue()
		CreateChange[T](changes, ObjectRemoved, label, l[label].GetValueNode(), nil,
			breakingRemove, left, right)
	}
	if added, key := CheckObjectAdded(l, r); added {
		right = r[key].GetValue()
		CreateChange[T](changes, ObjectAdded, label, nil, r[key].GetValueNode(),
			breakingAdd, left, right)
	}
}

func CheckObjectRemoved[T any](l, r map[string]*T) bool {
	for i := range l {
		if r[i] == nil {
			return true
		}
	}
	return false
}

func CheckObjectAdded[T any](l, r map[string]*T) (bool, string) {
	for i := range r {
		if l[i] == nil {
			return true, i
		}
	}
	return false, ""
}

func CheckProperties[T any](properties []*PropertyCheck[T]) {
	for _, n := range properties {
		CheckPropertyAdditionOrRemoval(n.LeftNode, n.RightNode, n.Label, n.Changes, n.Breaking, n.Original, n.New)
		CheckForModification(n.LeftNode, n.RightNode, n.Label, n.Changes, n.Breaking, n.Original, n.New)
		CheckForMove(n.LeftNode, n.RightNode, n.Label, n.Changes, n.Breaking, n.Original, n.New)
	}
}

func CheckPropertyAdditionOrRemoval[T any](l, r *yaml.Node,
	label string, changes *[]*Change[T], breaking bool, orig, new T) {
	CheckForRemoval[T](l, r, label, changes, breaking, orig, new)
	CheckForAddition[T](l, r, label, changes, breaking, orig, new)
}

func CheckForRemoval[T any](l, r *yaml.Node, label string, changes *[]*Change[T], breaking bool, orig, new T) {
	if l != nil && l.Value != "" && (r == nil || r.Value == "") {
		CreateChange[T](changes, PropertyRemoved, label, l, r, breaking, orig, new)
	}
}

func CheckForAddition[T any](l, r *yaml.Node, label string, changes *[]*Change[T], breaking bool, orig, new T) {
	if (l == nil || l.Value == "") && r != nil && r.Value != "" {
		CreateChange[T](changes, PropertyAdded, label, l, r, breaking, orig, new)
	}
}

func CheckForModification[T any](l, r *yaml.Node, label string, changes *[]*Change[T], breaking bool, orig, new T) {
	if l != nil && l.Value != "" && r != nil && r.Value != "" && r.Value != l.Value {
		changeType := Modified
		ctx := CreateContext(l, r)
		if ctx.HasChanged() {
			changeType = ModifiedAndMoved
		}
		CreateChange[T](changes, changeType, label, l, r, breaking, orig, new)
	}
}

func CheckForMove[T any](l, r *yaml.Node, label string, changes *[]*Change[T], breaking bool, orig, new T) {
	if l != nil && l.Value != "" && r != nil && r.Value != "" && r.Value == l.Value { // everything is equal
		ctx := CreateContext(l, r)
		if ctx.HasChanged() {
			CreateChange[T](changes, Moved, label, l, r, breaking, orig, new)
		}
	}
}

func CheckExtensions[T low.HasExtensions[T]](l, r T) *ExtensionChanges {
	var lExt, rExt map[low.KeyReference[string]]low.ValueReference[any]
	if len(l.GetExtensions()) > 0 {
		lExt = l.GetExtensions()
	}
	if len(r.GetExtensions()) > 0 {
		rExt = r.GetExtensions()
	}
	return CompareExtensions(lExt, rExt)
}
