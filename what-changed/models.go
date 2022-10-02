// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package what_changed

import "gopkg.in/yaml.v3"

// Definitions of the possible changes between two items
const (

	// Modified means that was a modification of a value was made
	Modified = iota + 1

	// PropertyAdded means that a new property to an object was added
	PropertyAdded

	// ObjectAdded means that a new object was added
	ObjectAdded

	// ObjectRemoved means that an object was removed
	ObjectRemoved

	// PropertyRemoved means that a property of an object was removed
	PropertyRemoved

	// Moved means that a property or an object was moved, but unchanged (line/columns changed)
	Moved

	// ModifiedAndMoved means that a property was modified, and it was also moved.
	ModifiedAndMoved
)

// WhatChanged is a summary object that contains a high level summary of everything changed.
type WhatChanged struct {
	Added            int
	Removed          int
	ModifiedAndMoved int
	Modified         int
	Moved            int
	TotalChanges     int
	Changes          *Changes
}

// ChangeContext holds a reference to the line and column positions of original and new change.
type ChangeContext struct {
	OriginalLine   int
	OriginalColumn int
	NewLine        int
	NewColumn      int
}

// HasChanged determines if the line and column numbers of the original and new values have changed.
func (c *ChangeContext) HasChanged() bool {
	return c.NewLine != c.OriginalLine || c.NewColumn != c.OriginalColumn
}

// Change represents a change between two different elements inside an OpenAPI specification.
type Change[T any] struct {

	// Context represents the lines and column numbers of the original and new changes.
	Context *ChangeContext

	// ChangeType represents the type of change that occurred. stored as an integer, defined by constants above.
	ChangeType int

	// Property is the property name key being changed.
	Property string

	// Original is the original value represented as a string.
	Original string

	// New is the new value represented as a string.
	New string

	// Breaking determines if the change is a breaking one or not.
	Breaking bool

	// OriginalObject represents the original object that was changed.
	OriginalObject T

	// NewObject represents the new object that has been modified.
	NewObject T
}

// PropertyChanges holds a slice of Change[T] change pointers
type PropertyChanges[T any] struct {
	Changes []*Change[T]
}

// PropertyCheck is used by functions to check the state of left and right values.
type PropertyCheck[T any] struct {

	// Original is the property we're checking on the left
	Original T

	// New is s the property we're checking on the right
	New T

	// Label is the identifier we're looking for on the left and right hand sides
	Label string

	// LeftNode is the yaml.Node pointer that holds the original node structure of the value
	LeftNode *yaml.Node

	// RightNode is the yaml.Node pointer that holds the new node structure of the value
	RightNode *yaml.Node

	// Breaking determines if the check is a breaking change (modifications or removals etc.)
	Breaking bool

	// Changes represents a pointer to the slice to contain all changes found.
	Changes *[]*Change[T]
}

type Changes struct {
	TagChanges *TagChanges
}
