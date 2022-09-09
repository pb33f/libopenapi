// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package main

import (
	"github.com/pb33f/libopenapi/datamodel"
	v2high "github.com/pb33f/libopenapi/datamodel/high/2.0"
	v3high "github.com/pb33f/libopenapi/datamodel/high/3.0"
)

type Document[T any] struct {
	version string
	info    *datamodel.SpecInfo
	Model   T
}

func (d *Document[T]) GetVersion() string {
	return d.version
}

func (d *Document[T]) BuildV2Document() (*v2high.Swagger, error) {
	return nil, nil
}

func (d *Document[T]) BuildV3Document() (*v3high.Document, error) {
	return nil, nil
}
