// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"testing"

	"github.com/pb33f/libopenapi/datamodel"
	lowv3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/stretchr/testify/assert"
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
	lDoc, _ := lowv3.CreateDocumentFromConfig(lInfo, datamodel.NewDocumentConfiguration())
	rDoc, _ := lowv3.CreateDocumentFromConfig(rInfo, datamodel.NewDocumentConfiguration())

	// compare.
	changes := CompareTags(lDoc.Tags.Value, rDoc.Tags.Value)

	// evaluate.
	assert.Len(t, changes[0].Changes, 1)
	assert.Len(t, changes[0].ExternalDocs.Changes, 2)
	assert.Len(t, changes[0].ExtensionChanges.Changes, 1)
	assert.Equal(t, 4, changes[0].TotalChanges())
	assert.Len(t, changes[0].GetAllChanges(), 4)

	descChange := changes[0].Changes[0]
	assert.Equal(t, "a lovelier tag description", descChange.New)
	assert.Equal(t, "a lovely tag", descChange.Original)
	assert.Equal(t, Modified, descChange.ChangeType)
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
	lDoc, _ := lowv3.CreateDocumentFromConfig(lInfo, datamodel.NewDocumentConfiguration())
	rDoc, _ := lowv3.CreateDocumentFromConfig(rInfo, datamodel.NewDocumentConfiguration())

	// compare.
	changes := CompareTags(lDoc.Tags.Value, rDoc.Tags.Value)

	// evaluate.
	assert.Len(t, changes[0].Changes, 1)
	assert.Equal(t, 1, changes[0].TotalChanges())
	assert.Len(t, changes[0].GetAllChanges(), 1)

	descChange := changes[0].Changes[0]
	assert.Equal(t, ObjectAdded, descChange.ChangeType)
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
	lDoc, _ := lowv3.CreateDocumentFromConfig(lInfo, datamodel.NewDocumentConfiguration())
	rDoc, _ := lowv3.CreateDocumentFromConfig(rInfo, datamodel.NewDocumentConfiguration())

	// compare.
	changes := CompareTags(lDoc.Tags.Value, rDoc.Tags.Value)

	// evaluate.
	assert.Len(t, changes, 2)
	assert.Equal(t, 1, changes[0].TotalChanges())
	assert.Len(t, changes[0].GetAllChanges(), 1)
	assert.Equal(t, 1, changes[1].TotalChanges())
	assert.Equal(t, 1, changes[0].TotalBreakingChanges())
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
	lDoc, _ := lowv3.CreateDocumentFromConfig(lInfo, datamodel.NewDocumentConfiguration())
	rDoc, _ := lowv3.CreateDocumentFromConfig(rInfo, datamodel.NewDocumentConfiguration())

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
	lDoc, _ := lowv3.CreateDocumentFromConfig(lInfo, datamodel.NewDocumentConfiguration())
	rDoc, _ := lowv3.CreateDocumentFromConfig(rInfo, datamodel.NewDocumentConfiguration())

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
	lDoc, _ := lowv3.CreateDocumentFromConfig(lInfo, datamodel.NewDocumentConfiguration())
	rDoc, _ := lowv3.CreateDocumentFromConfig(rInfo, datamodel.NewDocumentConfiguration())
	// compare.
	changes := CompareTags(lDoc.Tags.Value, rDoc.Tags.Value)

	// evaluate.
	assert.Len(t, changes[0].Changes, 1)
	assert.Equal(t, 1, changes[0].TotalChanges())
	assert.Len(t, changes[0].GetAllChanges(), 1)

	descChange := changes[0].Changes[0]
	assert.Equal(t, Modified, descChange.ChangeType)
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
	lDoc, _ := lowv3.CreateDocumentFromConfig(lInfo, datamodel.NewDocumentConfiguration())
	rDoc, _ := lowv3.CreateDocumentFromConfig(rInfo, datamodel.NewDocumentConfiguration())

	// compare.
	changes := CompareTags(lDoc.Tags.Value, rDoc.Tags.Value)

	// evaluate.
	assert.Nil(t, changes)
}

func TestCompareTags_AddExternalDocs(t *testing.T) {
	left := `openapi: 3.0.1
tags:
  - name: something else`

	right := `openapi: 3.0.1
tags:
  - name: something else
    externalDocs:
      url: https://pb33f.io
      description: cooler`

	// create document (which will create our correct tags low level structures)
	lInfo, _ := datamodel.ExtractSpecInfo([]byte(left))
	rInfo, _ := datamodel.ExtractSpecInfo([]byte(right))
	lDoc, _ := lowv3.CreateDocumentFromConfig(lInfo, datamodel.NewDocumentConfiguration())
	rDoc, _ := lowv3.CreateDocumentFromConfig(rInfo, datamodel.NewDocumentConfiguration())

	// compare.
	changes := CompareTags(lDoc.Tags.Value, rDoc.Tags.Value)

	// evaluate.
	assert.Equal(t, 1, changes[0].TotalChanges())
	assert.Len(t, changes[0].GetAllChanges(), 1)
	assert.Equal(t, ObjectAdded, changes[0].Changes[0].ChangeType)
}

func TestCompareTags_RemoveExternalDocs(t *testing.T) {
	left := `openapi: 3.0.1
tags:
  - name: something else`

	right := `openapi: 3.0.1
tags:
  - name: something else
    externalDocs:
      url: https://pb33f.io
      description: cooler`

	// create document (which will create our correct tags low level structures)
	lInfo, _ := datamodel.ExtractSpecInfo([]byte(left))
	rInfo, _ := datamodel.ExtractSpecInfo([]byte(right))
	lDoc, _ := lowv3.CreateDocumentFromConfig(lInfo, datamodel.NewDocumentConfiguration())
	rDoc, _ := lowv3.CreateDocumentFromConfig(rInfo, datamodel.NewDocumentConfiguration())

	// compare.
	changes := CompareTags(rDoc.Tags.Value, lDoc.Tags.Value)

	// evaluate.
	assert.Equal(t, 1, changes[0].TotalChanges())
	assert.Len(t, changes[0].GetAllChanges(), 1)
	assert.Equal(t, ObjectRemoved, changes[0].Changes[0].ChangeType)
}
