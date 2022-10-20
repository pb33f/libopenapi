// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package what_changed

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"gopkg.in/yaml.v3"
)

// CreateChange is a generic function that will create a Change of type T, populate all properties if set, and then
// add a pointer to Change[T] in the slice of Change pointers provided
func CreateChange(changes *[]*Change, changeType int, property string, leftValueNode, rightValueNode *yaml.Node,
	breaking bool, originalObject, newObject any) *[]*Change {

	// create a new context for the left and right nodes.
	ctx := CreateContext(leftValueNode, rightValueNode)
	c := &Change{
		Context:    ctx,
		ChangeType: changeType,
		Property:   property,
		Breaking:   breaking,
	}
	// if the left is not nil, we have an original value
	if leftValueNode != nil && leftValueNode.Value != "" {
		c.Original = leftValueNode.Value
	}
	// if the right is not nil, then we have a new value
	if rightValueNode != nil && rightValueNode.Value != "" {
		c.New = rightValueNode.Value
	}
	// original and new objects
	c.OriginalObject = originalObject
	c.NewObject = newObject

	// add the change to supplied changes slice
	*changes = append(*changes, c)
	return changes
}

// CreateContext will return a pointer to a ChangeContext containing the original and new line and column numbers
// of the left and right value nodes.
func CreateContext(l, r *yaml.Node) *ChangeContext {
	ctx := new(ChangeContext)
	if l != nil {
		ctx.OriginalLine = l.Line
		ctx.OriginalColumn = l.Column
	} else {
		ctx.OriginalLine = -1
		ctx.OriginalColumn = -1
	}
	if r != nil {
		ctx.NewLine = r.Line
		ctx.NewColumn = r.Column
	} else {
		ctx.NewLine = -1
		ctx.NewColumn = -1
	}
	return ctx
}

func FlattenLowLevelMap[T any](
	lowMap map[low.KeyReference[string]]low.ValueReference[T]) map[string]*low.ValueReference[T] {
	flat := make(map[string]*low.ValueReference[T])
	for i := range lowMap {
		l := lowMap[i]
		flat[i.Value] = &l
	}
	return flat
}

// CountBreakingChanges counts the number of changes in a slice that are breaking
func CountBreakingChanges(changes []*Change) int {
	b := 0
	for i := range changes {
		if changes[i].Breaking {
			b++
		}
	}
	return b
}

// CheckForObjectAdditionOrRemoval will check for the addition or removal of an object from left and right maps.
// The label is the key to look for in the left and right maps.
//
// To determine this a breaking change for an addition then set breakingAdd to true (however I can't think of many
// scenarios that adding things should break anything). Removals are generally breaking, except for non contract
// properties like descriptions, summaries and other non-binding values, so a breakingRemove value can be tuned for
// these circumstances.
func CheckForObjectAdditionOrRemoval[T any](l, r map[string]*low.ValueReference[T], label string, changes *[]*Change,
	breakingAdd, breakingRemove bool) {
	var left, right T
	if CheckSpecificObjectRemoved(l, r, label) {
		left = l[label].GetValue()
		CreateChange(changes, ObjectRemoved, label, l[label].GetValueNode(), nil,
			breakingRemove, left, right)
	}
	if CheckSpecificObjectAdded(l, r, label) {
		right = r[label].GetValue()
		CreateChange(changes, ObjectAdded, label, nil, r[label].GetValueNode(),
			breakingAdd, left, right)
	}
}

// CheckSpecificObjectRemoved returns true if a specific value is not in both maps.
func CheckSpecificObjectRemoved[T any](l, r map[string]*T, label string) bool {
	return l[label] != nil && r[label] == nil
}

// CheckSpecificObjectAdded returns true if a specific value is not in both maps.
func CheckSpecificObjectAdded[T any](l, r map[string]*T, label string) bool {
	return l[label] == nil && r[label] != nil
}

// CheckProperties will iterate through a slice of PropertyCheck pointers of type T. The method is a convenience method
// for running checks on the following methods in order:
//   CheckPropertyAdditionOrRemoval
//   CheckForModification
func CheckProperties(properties []*PropertyCheck) {

	// todo: make this async to really speed things up.
	for _, n := range properties {
		CheckPropertyAdditionOrRemoval(n.LeftNode, n.RightNode, n.Label, n.Changes, n.Breaking, n.Original, n.New)
		CheckForModification(n.LeftNode, n.RightNode, n.Label, n.Changes, n.Breaking, n.Original, n.New)
	}
}

// CheckPropertyAdditionOrRemoval will run both CheckForRemoval (first) and CheckForAddition (second)
func CheckPropertyAdditionOrRemoval[T any](l, r *yaml.Node,
	label string, changes *[]*Change, breaking bool, orig, new T) {
	CheckForRemoval[T](l, r, label, changes, breaking, orig, new)
	CheckForAddition[T](l, r, label, changes, breaking, orig, new)
}

// CheckForRemoval will check left and right yaml.Node instances for changes. Anything that is found missing on the
// right, but present on the left, is considered a removal. A new Change[T] will be created with the type
//
//  PropertyRemoved
//
// The Change is then added to the slice of []Change[T] instances provided as a pointer.
func CheckForRemoval[T any](l, r *yaml.Node, label string, changes *[]*Change, breaking bool, orig, new T) {
	if l != nil && l.Value != "" && (r == nil || r.Value == "") {
		CreateChange(changes, PropertyRemoved, label, l, r, breaking, orig, new)
	}
}

// CheckForAddition will check left and right yaml.Node instances for changes. Anything that is found missing on the
// left, but present on the left, is considered an addition. A new Change[T] will be created with the type
//
//  PropertyAdded
//
// The Change is then added to the slice of []Change[T] instances provided as a pointer.
func CheckForAddition[T any](l, r *yaml.Node, label string, changes *[]*Change, breaking bool, orig, new T) {
	if (l == nil || l.Value == "") && r != nil && r.Value != "" {
		CreateChange(changes, PropertyAdded, label, l, r, breaking, orig, new)
	}
}

// CheckForModification will check left and right yaml.Node instances for changes. Anything that is found in both
// sides, but vary in value is considered a modification.
//
// If there is a change in value the function adds a change type of Modified.
//
// The Change is then added to the slice of []Change[T] instances provided as a pointer.
func CheckForModification[T any](l, r *yaml.Node, label string, changes *[]*Change, breaking bool, orig, new T) {
	if l != nil && l.Value != "" && r != nil && r.Value != "" && r.Value != l.Value && r.Tag == l.Tag {
		CreateChange(changes, Modified, label, l, r, breaking, orig, new)
	}
	// the values may have not changed, but the tag (node type) type may have
	if l != nil && l.Value != "" && r != nil && r.Value != "" && r.Value != l.Value && r.Tag != l.Tag {
		CreateChange(changes, Modified, label, l, r, breaking, orig, new)
	}
}

// CheckExtensions is a helper method to un-pack a left and right model that contains extensions. Once unpacked
// the extensions are compared and returns a pointer to ExtensionChanges. If nothing changed, nil is returned.
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
