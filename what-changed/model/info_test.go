// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"context"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestCompareInfo_DescriptionAdded(t *testing.T) {

	left := `title: a nice spec
termsOfService: https://pb33f.io/terms
version: '1.2.3'
contact:
  name: buckaroo
  email: buckaroo@pb33f.io
license:
  name: MIT`

	right := `title: a nice spec
termsOfService: https://pb33f.io/terms
version: '1.2.3'
description: this is a description
contact:
  name: buckaroo
  email: buckaroo@pb33f.io
license:
  name: MIT`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc base.Info
	var rDoc base.Info
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareInfo(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, PropertyAdded, extChanges.Changes[0].ChangeType)
	assert.Equal(t, v3.DescriptionLabel, extChanges.Changes[0].Property)
}

func TestCompareInfo_TitleRemoved(t *testing.T) {

	left := `title: a nice spec
termsOfService: https://pb33f.io/terms
version: '1.2.3'
description: this is a description
contact:
  name: buckaroo
  email: buckaroo@pb33f.io
license:
  name: MIT`

	right := `termsOfService: https://pb33f.io/terms
version: '1.2.3'
description: this is a description
contact:
  name: buckaroo
  email: buckaroo@pb33f.io
license:
  name: MIT`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc base.Info
	var rDoc base.Info
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareInfo(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, PropertyRemoved, extChanges.Changes[0].ChangeType)
	assert.Equal(t, v3.TitleLabel, extChanges.Changes[0].Property)
}

func TestCompareInfo_VersionModified(t *testing.T) {

	left := `title: a nice spec
termsOfService: https://pb33f.io/terms
version: '1.2.3'
contact:
  name: buckaroo
  email: buckaroo@pb33f.io
license:
  name: MIT`

	right := `title: a nice spec
termsOfService: https://pb33f.io/terms
version: '99.99'
contact:
  name: buckaroo
  email: buckaroo@pb33f.io
license:
  name: MIT`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc base.Info
	var rDoc base.Info
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareInfo(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, Modified, extChanges.Changes[0].ChangeType)
	assert.Equal(t, v3.VersionLabel, extChanges.Changes[0].Property)
}

func TestCompareInfo_RemoveLicense(t *testing.T) {

	left := `title: a nice spec
termsOfService: https://pb33f.io/terms
version: '1.2.3'
contact:
  name: buckaroo
  email: buckaroo@pb33f.io
license:
  name: MIT`

	right := `title: a nice spec
termsOfService: https://pb33f.io/terms
version: '1.2.3'
contact:
  name: buckaroo
  email: buckaroo@pb33f.io`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc base.Info
	var rDoc base.Info
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareInfo(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, ObjectRemoved, extChanges.Changes[0].ChangeType)
	assert.Equal(t, v3.LicenseLabel, extChanges.Changes[0].Property)
}

func TestCompareInfo_AddLicense(t *testing.T) {

	left := `title: a nice spec
termsOfService: https://pb33f.io/terms
version: '1.2.3'
contact:
  name: buckaroo
  email: buckaroo@pb33f.io`

	right := `title: a nice spec
termsOfService: https://pb33f.io/terms
version: '1.2.3'
contact:
  name: buckaroo
  email: buckaroo@pb33f.io
license:
  name: MIT`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc base.Info
	var rDoc base.Info
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareInfo(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, ObjectAdded, extChanges.Changes[0].ChangeType)
	assert.Equal(t, v3.LicenseLabel, extChanges.Changes[0].Property)
}

func TestCompareInfo_LicenseChanged(t *testing.T) {

	left := `title: a nice spec
termsOfService: https://pb33f.io/terms
version: '1.2.3'
contact:
  name: buckaroo
  email: buckaroo@pb33f.io
license:
  name: MIT`

	right := `title: a nice spec
termsOfService: https://pb33f.io/terms
version: '1.2.3'
contact:
  name: buckaroo
  email: buckaroo@pb33f.io
license:
  name: Apache`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc base.Info
	var rDoc base.Info
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareInfo(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, Modified, extChanges.LicenseChanges.Changes[0].ChangeType)
	assert.Equal(t, v3.NameLabel, extChanges.LicenseChanges.Changes[0].Property)
}

func TestCompareInfo_AddContact(t *testing.T) {

	left := `title: a nice spec
termsOfService: https://pb33f.io/terms
version: '1.2.3'
license:
  name: MIT`

	right := `title: a nice spec
termsOfService: https://pb33f.io/terms
version: '1.2.3'
contact:
  name: buckaroo
  email: buckaroo@pb33f.io
license:
  name: MIT`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc base.Info
	var rDoc base.Info
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareInfo(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, ObjectAdded, extChanges.Changes[0].ChangeType)
	assert.Equal(t, v3.ContactLabel, extChanges.Changes[0].Property)
}

func TestCompareInfo_RemoveContact(t *testing.T) {

	left := `title: a nice spec
termsOfService: https://pb33f.io/terms
version: '1.2.3'
contact:
  name: buckaroo
  email: buckaroo@pb33f.io
license:
  name: MIT`

	right := `title: a nice spec
termsOfService: https://pb33f.io/terms
version: '1.2.3'
license:
  name: MIT`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc base.Info
	var rDoc base.Info
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareInfo(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, ObjectRemoved, extChanges.Changes[0].ChangeType)
	assert.Equal(t, v3.ContactLabel, extChanges.Changes[0].Property)
}

func TestCompareInfo_ContactModified(t *testing.T) {

	left := `title: a nice spec
termsOfService: https://pb33f.io/terms
version: '1.2.3'
contact:
  name: buckaroo
  email: buckaroo@pb33f.io
license:
  name: MIT`

	right := `title: a nice spec
termsOfService: https://pb33f.io/terms
version: '1.2.3'
contact:
  name: the buckaroo
  email: buckaroo@pb33f.io
license:
  name: MIT`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc base.Info
	var rDoc base.Info
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareInfo(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, Modified, extChanges.ContactChanges.Changes[0].ChangeType)
	assert.Equal(t, v3.NameLabel, extChanges.ContactChanges.Changes[0].Property)
	assert.Equal(t, 0, extChanges.TotalBreakingChanges())
}

func TestCompareInfo_Equal(t *testing.T) {

	left := `title: a nice spec
termsOfService: https://pb33f.io/terms
version: '1.2.3'
contact:
  name: buckaroo
  email: buckaroo@pb33f.io
license:
  name: MIT
x-extension: extension`

	right := `title: a nice spec
termsOfService: https://pb33f.io/terms
version: '1.2.3'
contact:
  name: buckaroo
  email: buckaroo@pb33f.io
license:
  name: MIT
x-extension: extension`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc base.Info
	var rDoc base.Info
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareInfo(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}

func TestCompareInfo_ExtensionAdded(t *testing.T) {

	left := `title: a nice spec
version: '1.2.3'
`

	right := `title: a nice spec
version: '1.2.3'
x-extension: new extension
`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc base.Info
	var rDoc base.Info
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareInfo(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, ObjectAdded, extChanges.ExtensionChanges.Changes[0].ChangeType)
	assert.Equal(t, "x-extension", extChanges.ExtensionChanges.Changes[0].Property)
}

func TestCompareInfo_ExtensionRemoved(t *testing.T) {

	left := `title: a nice spec
version: '1.2.3'
x-extension: extension
`

	right := `title: a nice spec
version: '1.2.3'
`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc base.Info
	var rDoc base.Info
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareInfo(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, ObjectRemoved, extChanges.ExtensionChanges.Changes[0].ChangeType)
	assert.Equal(t, "x-extension", extChanges.ExtensionChanges.Changes[0].Property)
}

func TestCompareInfo_ExtensionModified(t *testing.T) {

	left := `title: a nice spec
version: '1.2.3'
x-extension: original extension
`

	right := `title: a nice spec
version: '1.2.3'
x-extension: new extension
`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc base.Info
	var rDoc base.Info
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	// compare.
	extChanges := CompareInfo(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Len(t, extChanges.GetAllChanges(), 1)
	assert.Equal(t, Modified, extChanges.ExtensionChanges.Changes[0].ChangeType)
	assert.Equal(t, "x-extension", extChanges.ExtensionChanges.Changes[0].Property)
}
