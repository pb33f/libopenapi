// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package what_changed

import (
	"github.com/pb33f/libopenapi/datamodel/low/v2"
	"github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/what-changed/model"
)

func CompareOpenAPIDocuments(original, updated *v3.Document) *model.DocumentChanges {
	return model.CompareDocuments(original, updated)
}

func CompareSwaggerDocuments(original, updated *v2.Swagger) *model.DocumentChanges {
	return model.CompareDocuments(original, updated)
}
