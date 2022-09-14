// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

// Package high contains a set of high-level models that represent OpenAPI 2 and 3 documents.
// These high-level models (porcelain) are used by applications directly, rather than the low-level models
// plumbing) that are used to compose high level models.
//
// High level models are simple to navigate, strongly typed, precise representations of the OpenAPI schema
// that are created from an OpenAPI specification.
//
// All high level objects contains a 'GoLow' method. This 'GoLow' method will return the low-level model that
// was used to create it, which provides an engineer as much low level detail about the raw spec used to create
// those models, things like key/value breakdown of each value, lines, column, source comments etc.
package high

import "github.com/pb33f/libopenapi/datamodel/low"

// GoesLow is used to represent any high-level model. All high level models meet this interface and can be used to
// extract low-level models from any high-level model.
type GoesLow[T any] interface {

	// GoLow returns the low-level object that was used to create the high-level object. This allows consumers
	// to dive-down into the plumbing API at any point in the model.
	GoLow() T
}

// ExtractExtensions is a convenience method for converting low-level extension definitions, to a high level map[string]any
// definition that is easier to consume in applications.
func ExtractExtensions(extensions map[low.KeyReference[string]]low.ValueReference[any]) map[string]any {
	extracted := make(map[string]any)
	for k, v := range extensions {
		extracted[k.Value] = v.Value
	}
	return extracted
}
