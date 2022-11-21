// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

// Package what_changed
//
// The low level (or plumbing) models are designed to capture every single detail about specification, including
// all lines, columns, positions, tags, comments and essentially everything you would ever want to know.
// Positions of every key, value and meta-data that is lost when blindly un-marshaling JSON/YAML into a struct.
//
// The high model (porcelain) is a much simpler representation of the low model, keys are simple strings and indices
// are numbers. When developing consumers of the model, the high model is really what you want to use instead of the
// low model, it's much easier to navigate and is designed for easy consumption.
//
// The high model requires the low model to be built. Every high model has a 'GoLow' method that allows the consumer
// to 'drop down' from the porcelain API to the plumbing API, which gives instant access to everything low.
package what_changed

import (
	"github.com/pb33f/libopenapi/datamodel/low/v2"
	"github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/what_changed/model"
)

func CompareOpenAPIDocuments(original, updated *v3.Document) *model.DocumentChanges {
	return model.CompareDocuments(original, updated)
}

func CompareSwaggerDocuments(original, updated *v2.Swagger) *model.DocumentChanges {
	return model.CompareDocuments(original, updated)
}
