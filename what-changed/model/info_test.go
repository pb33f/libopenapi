// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package model

import (
	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/what-changed/core"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
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
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareInfo(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, core.PropertyAdded, extChanges.Changes[0].ChangeType)
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
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareInfo(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, core.PropertyRemoved, extChanges.Changes[0].ChangeType)
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
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareInfo(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, core.Modified, extChanges.Changes[0].ChangeType)
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
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareInfo(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, core.ObjectRemoved, extChanges.Changes[0].ChangeType)
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
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareInfo(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, core.ObjectAdded, extChanges.Changes[0].ChangeType)
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
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareInfo(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, core.Modified, extChanges.LicenseChanges.Changes[0].ChangeType)
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
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareInfo(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, core.ObjectAdded, extChanges.Changes[0].ChangeType)
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
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareInfo(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, core.ObjectRemoved, extChanges.Changes[0].ChangeType)
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
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareInfo(&lDoc, &rDoc)
	assert.Equal(t, 1, extChanges.TotalChanges())
	assert.Equal(t, core.Modified, extChanges.ContactChanges.Changes[0].ChangeType)
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
	_ = low.BuildModel(&lNode, &lDoc)
	_ = low.BuildModel(&rNode, &rDoc)
	_ = lDoc.Build(lNode.Content[0], nil)
	_ = rDoc.Build(rNode.Content[0], nil)

	// compare.
	extChanges := CompareInfo(&lDoc, &rDoc)
	assert.Nil(t, extChanges)
}
