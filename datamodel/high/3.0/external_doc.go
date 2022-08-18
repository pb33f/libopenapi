// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	low "github.com/pb33f/libopenapi/datamodel/low/3.0"
)

type ExternalDoc struct {
	Description string
	URL         string
	Extensions  map[string]any
	low         *low.ExternalDoc
}

func NewExternalDoc(extDoc *low.ExternalDoc) *ExternalDoc {
	d := new(ExternalDoc)
	d.low = extDoc
	d.Description = extDoc.Description.Value
	d.URL = extDoc.URL.Value
	d.Extensions = high.ExtractExtensions(extDoc.Extensions)
	return d
}

func (e *ExternalDoc) GoLow() *low.ExternalDoc {
	return e.low
}
