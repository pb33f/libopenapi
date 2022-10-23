// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"github.com/pb33f/libopenapi/datamodel"
	lowv3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/what-changed/core"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCompareTags(t *testing.T) {

	left := `openapi: 3.0.1
tags:
  - name: a tag
    description: a lovely tag
    x-tag: something
    externalDocs:
      url: https://quobix.com
      description: cool`

	right := `openapi: 3.0.1
tags:
  - name: a tag
    description: a lovelier tag description
    x-tag: something else
    externalDocs:
      url: https://pb33f.io
      description: cooler`

	// create document (which will create our correct tags low level structures)
	lInfo, _ := datamodel.ExtractSpecInfo([]byte(left))
	rInfo, _ := datamodel.ExtractSpecInfo([]byte(right))
	lDoc, _ := lowv3.CreateDocument(lInfo)
	rDoc, _ := lowv3.CreateDocument(rInfo)

	// compare.
	changes := CompareTags(lDoc.Tags.Value, rDoc.Tags.Value)

	// evaluate.
	assert.Len(t, changes.Changes, 1)
	assert.Len(t, changes.ExternalDocs.Changes, 2)
	assert.Len(t, changes.ExtensionChanges.Changes, 1)
	assert.Equal(t, 4, changes.TotalChanges())

	descChange := changes.Changes[0]
	assert.Equal(t, "a lovelier tag description", descChange.New)
	assert.Equal(t, "a lovely tag", descChange.Original)
	assert.Equal(t, core.Modified, descChange.ChangeType)
	assert.False(t, descChange.Context.HasChanged())
}

func TestCompareTags_AddNewTag(t *testing.T) {

	left := `openapi: 3.0.1
tags:
  - name: a tag
    description: a lovelier tag description
    x-tag: something else
    externalDocs:
      url: https://pb33f.io
      description: cooler`

	right := `openapi: 3.0.1
tags:
  - name: a tag
    description: a lovelier tag description
    x-tag: something else
    externalDocs:
      url: https://pb33f.io
      description: cooler
  - name: a new tag
    description: a cool new tag`

	// create document (which will create our correct tags low level structures)
	lInfo, _ := datamodel.ExtractSpecInfo([]byte(left))
	rInfo, _ := datamodel.ExtractSpecInfo([]byte(right))
	lDoc, _ := lowv3.CreateDocument(lInfo)
	rDoc, _ := lowv3.CreateDocument(rInfo)

	// compare.
	changes := CompareTags(lDoc.Tags.Value, rDoc.Tags.Value)

	// evaluate.
	assert.Len(t, changes.Changes, 1)
	assert.Equal(t, 1, changes.TotalChanges())

	descChange := changes.Changes[0]
	assert.Equal(t, core.ObjectAdded, descChange.ChangeType)
}

func TestCompareTags_AddDeleteTag(t *testing.T) {

	left := `openapi: 3.0.1
tags:
  - name: a tag
    description: a lovelier tag description
    x-tag: something else
    externalDocs:
      url: https://pb33f.io
      description: cooler`

	right := `openapi: 3.0.1
tags:
  - name: a new tag
    description: a cool new tag`

	// create document (which will create our correct tags low level structures)
	lInfo, _ := datamodel.ExtractSpecInfo([]byte(left))
	rInfo, _ := datamodel.ExtractSpecInfo([]byte(right))
	lDoc, _ := lowv3.CreateDocument(lInfo)
	rDoc, _ := lowv3.CreateDocument(rInfo)

	// compare.
	changes := CompareTags(lDoc.Tags.Value, rDoc.Tags.Value)

	// evaluate.
	assert.Len(t, changes.Changes, 2)
	assert.Equal(t, 2, changes.TotalChanges())

	assert.Equal(t, core.ObjectRemoved, changes.Changes[0].ChangeType)
	assert.Equal(t, core.ObjectAdded, changes.Changes[1].ChangeType)
	assert.Equal(t, 1, changes.TotalBreakingChanges())
}

func TestCompareTags_DescriptionMoved(t *testing.T) {

	left := `openapi: 3.0.1
tags:
  - description: a lovelier tag description
    name: a tag
    x-tag: something else
    externalDocs:
      url: https://pb33f.io
      description: cooler`

	right := `openapi: 3.0.1
tags:
  - name: a tag
    x-tag: something else
    description: a lovelier tag description
    externalDocs:
      url: https://pb33f.io
      description: cooler`

	// create document (which will create our correct tags low level structures)
	lInfo, _ := datamodel.ExtractSpecInfo([]byte(left))
	rInfo, _ := datamodel.ExtractSpecInfo([]byte(right))
	lDoc, _ := lowv3.CreateDocument(lInfo)
	rDoc, _ := lowv3.CreateDocument(rInfo)

	// compare.
	changes := CompareTags(lDoc.Tags.Value, rDoc.Tags.Value)

	// evaluate.
	assert.Nil(t, changes)

}

func TestCompareTags_NameMoved(t *testing.T) {

	left := `openapi: 3.0.1
tags:
  - description: a lovelier tag description
    name: a tag
    x-tag: something else
    externalDocs:
      url: https://pb33f.io
      description: cooler`

	right := `openapi: 3.0.1
tags:
  - description: a lovelier tag description
    x-tag: something else
    externalDocs:
      url: https://pb33f.io
      description: cooler
    name: a tag`

	// create document (which will create our correct tags low level structures)
	lInfo, _ := datamodel.ExtractSpecInfo([]byte(left))
	rInfo, _ := datamodel.ExtractSpecInfo([]byte(right))
	lDoc, _ := lowv3.CreateDocument(lInfo)
	rDoc, _ := lowv3.CreateDocument(rInfo)

	// compare.
	changes := CompareTags(lDoc.Tags.Value, rDoc.Tags.Value)

	// evaluate.
	assert.Nil(t, changes)
}

func TestCompareTags_ModifiedAndMoved(t *testing.T) {

	left := `openapi: 3.0.1
tags:
  - description: a lovelier tag description
    name: a tag
    x-tag: something else
    externalDocs:
      url: https://pb33f.io
      description: cooler`

	right := `openapi: 3.0.1
tags:
  - name: a tag
    x-tag: something else
    description: a different tag description
    externalDocs:
      url: https://pb33f.io
      description: cooler`

	// create document (which will create our correct tags low level structures)
	lInfo, _ := datamodel.ExtractSpecInfo([]byte(left))
	rInfo, _ := datamodel.ExtractSpecInfo([]byte(right))
	lDoc, _ := lowv3.CreateDocument(lInfo)
	rDoc, _ := lowv3.CreateDocument(rInfo)

	// compare.
	changes := CompareTags(lDoc.Tags.Value, rDoc.Tags.Value)

	// evaluate.
	assert.Len(t, changes.Changes, 1)
	assert.Equal(t, 1, changes.TotalChanges())

	descChange := changes.Changes[0]
	assert.Equal(t, core.Modified, descChange.ChangeType)
	assert.Equal(t, "a lovelier tag description", descChange.Original)
	assert.Equal(t, "a different tag description", descChange.New)
	assert.True(t, descChange.Context.HasChanged())
}

func TestCompareTags_Identical(t *testing.T) {

	left := `openapi: 3.0.1
tags:
  - description: a lovelier tag description
    name: a tag
    x-tag: something else
    externalDocs:
      url: https://pb33f.io
      description: cooler`

	right := `openapi: 3.0.1
tags:
  - description: a lovelier tag description
    name: a tag
    x-tag: something else
    externalDocs:
      url: https://pb33f.io
      description: cooler`

	// create document (which will create our correct tags low level structures)
	lInfo, _ := datamodel.ExtractSpecInfo([]byte(left))
	rInfo, _ := datamodel.ExtractSpecInfo([]byte(right))
	lDoc, _ := lowv3.CreateDocument(lInfo)
	rDoc, _ := lowv3.CreateDocument(rInfo)

	// compare.
	changes := CompareTags(lDoc.Tags.Value, rDoc.Tags.Value)

	// evaluate.
	assert.Nil(t, changes)

}
