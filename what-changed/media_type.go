// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package what_changed

type MediaTypeChanges struct {
	PropertyChanges
	SchemaChanges    *SchemaChanges
	ExtensionChanges *ExtensionChanges
	ExampleChanges   map[string]*ExampleChanges
	EncodingChanges  *EncodingChanges
}
