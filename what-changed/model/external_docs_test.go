// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"context"
	"github.com/pb33f/libopenapi/datamodel/low"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	lowv3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestCompareExternalDocs(t *testing.T) {

	left := `url: https://pb33f.io
description: this is a test
x-testing: hello`

	right := `url: https://quobix.com
description: this is another test
x-testing: hiya!`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc lowbase.ExternalDoc
	var rDoc lowbase.ExternalDoc
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareExternalDocs(&lDoc, &rDoc)
	assert.Len(t, extChanges.ExtensionChanges.Changes, 1)
	assert.Len(t, extChanges.Changes, 2)
	assert.Len(t, extChanges.GetAllChanges(), 3)
	assert.Equal(t, 3, extChanges.TotalChanges())

	// validate property changes
	urlChange := extChanges.Changes[0]
	assert.Equal(t, Modified, urlChange.ChangeType)
	assert.False(t, urlChange.Context.HasChanged())
	assert.Equal(t, "https://pb33f.io", urlChange.Original)
	assert.Equal(t, "https://quobix.com", urlChange.New)
	assert.Equal(t, 1, *urlChange.Context.OriginalLine)
	assert.Equal(t, lowv3.URLLabel, urlChange.Property)

	descChange := extChanges.Changes[1]
	assert.Equal(t, Modified, descChange.ChangeType)
	assert.False(t, descChange.Context.HasChanged())
	assert.Equal(t, "this is another test", descChange.New)
	assert.Equal(t, "this is a test", descChange.Original)
	assert.Equal(t, 2, *descChange.Context.OriginalLine)
	assert.Equal(t, 14, *descChange.Context.OriginalColumn)

	// validate extensions
	extChange := extChanges.ExtensionChanges.Changes[0]
	assert.Equal(t, Modified, extChange.ChangeType)
	assert.False(t, extChange.Context.HasChanged())
	assert.Equal(t, "hiya!", extChange.New)
	assert.Equal(t, "hello", extChange.Original)
	assert.Equal(t, 3, *extChange.Context.OriginalLine)
	assert.Equal(t, 12, *extChange.Context.OriginalColumn)

}

func TestCompareExternalDocs_Moved(t *testing.T) {

	left := `url: https://pb33f.io
description: this is a test
x-testing: hello`

	right := `description: this is another test
x-testing: hiya!
url: https://quobix.com`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc lowbase.ExternalDoc
	var rDoc lowbase.ExternalDoc
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareExternalDocs(&lDoc, &rDoc)
	assert.Len(t, extChanges.ExtensionChanges.Changes, 1)
	assert.Len(t, extChanges.Changes, 2)
	assert.Len(t, extChanges.GetAllChanges(), 3)

	// validate property changes
	urlChange := extChanges.Changes[0]
	assert.Equal(t, Modified, urlChange.ChangeType)
	assert.True(t, urlChange.Context.HasChanged())
	assert.Equal(t, "https://pb33f.io", urlChange.Original)
	assert.Equal(t, "https://quobix.com", urlChange.New)
	assert.Equal(t, lowv3.URLLabel, urlChange.Property)

	descChange := extChanges.Changes[1]
	assert.Equal(t, Modified, descChange.ChangeType)
	assert.True(t, descChange.Context.HasChanged())
	assert.Equal(t, "this is another test", descChange.New)
	assert.Equal(t, "this is a test", descChange.Original)

	// validate extensions
	extChange := extChanges.ExtensionChanges.Changes[0]
	assert.Equal(t, Modified, extChange.ChangeType)
	assert.True(t, extChange.Context.HasChanged())
	assert.Equal(t, "hiya!", extChange.New)
	assert.Equal(t, "hello", extChange.Original)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
}

func TestCompareExternalDocs_Identical(t *testing.T) {

	left := `url: https://pb33f.io
description: this is a test
x-testing: hello`

	right := `url: https://pb33f.io
description: this is a test
x-testing: hello`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc lowbase.ExternalDoc
	var rDoc lowbase.ExternalDoc
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareExternalDocs(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareExternalDocs_DescriptionAdded(t *testing.T) {

	left := `url: https://pb33f.io
x-testing: hello`

	right := `url: https://pb33f.io
description: this is a test
x-testing: hello`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc lowbase.ExternalDoc
	var rDoc lowbase.ExternalDoc
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareExternalDocs(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, PropertyAdded, extChanges.Changes[0].ChangeType)
}

func TestCompareExternalDocs_URLAdded(t *testing.T) {

	left := `description: hi!`

	right := `description: hi!
url: https://pb33f.io`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc lowbase.ExternalDoc
	var rDoc lowbase.ExternalDoc
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareExternalDocs(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, PropertyAdded, extChanges.Changes[0].ChangeType)
}

func TestCompareExternalDocs_DescriptionRemoved(t *testing.T) {

	left := `url: https://pb33f.io
description: something`

	right := `url: https://pb33f.io`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc lowbase.ExternalDoc
	var rDoc lowbase.ExternalDoc
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareExternalDocs(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, PropertyRemoved, extChanges.Changes[0].ChangeType)
}

func TestCompareExternalDocs_URLRemoved(t *testing.T) {

	left := `description: something
url: https://pb33f.io`

	right := `description: something`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc lowbase.ExternalDoc
	var rDoc lowbase.ExternalDoc
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare
	extChanges := CompareExternalDocs(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, PropertyRemoved, extChanges.Changes[0].ChangeType)
}
