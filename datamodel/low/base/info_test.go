// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"testing"

	"github.com/pkg-base/libopenapi/datamodel/low"
	"github.com/pkg-base/libopenapi/index"
	"github.com/pkg-base/libopenapi/orderedmap"
	"github.com/pkg-base/yaml"
	"github.com/stretchr/testify/assert"
)

func TestInfo_Build(t *testing.T) {
	yml := `title: pizza
summary: a pizza pie
description: pie
termsOfService: yes indeed.
contact:
  name: buckaroo
  url: https://pb33f.io
  email: buckaroo@pb33f.io
license:
 name: magic
 url: https://pb33f.io/license
x-cli-name: pizza cli`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndex(&idxNode)

	var n Info
	err := low.BuildModel(idxNode.Content[0], &n)
	assert.NoError(t, err)

	err = n.Build(context.Background(), nil, idxNode.Content[0], idx)
	assert.NoError(t, err)

	assert.Equal(t, "pizza", n.Title.Value)
	assert.Equal(t, "a pizza pie", n.Summary.Value)
	assert.Equal(t, "pie", n.Description.Value)
	assert.Equal(t, "yes indeed.", n.TermsOfService.Value)

	con := n.Contact.Value
	assert.NotNil(t, con)
	assert.Equal(t, "buckaroo", con.Name.Value)
	assert.Equal(t, "https://pb33f.io", con.URL.Value)
	assert.Equal(t, "buckaroo@pb33f.io", con.Email.Value)

	lic := n.License.Value
	assert.NotNil(t, lic)
	assert.Equal(t, "magic", lic.Name.Value)
	assert.Equal(t, "https://pb33f.io/license", lic.URL.Value)

	var xCliName string
	_ = n.FindExtension("x-cli-name").Value.Decode(&xCliName)

	assert.Equal(t, "pizza cli", xCliName)
	assert.Equal(t, 1, orderedmap.Len(n.GetExtensions()))
	assert.NotNil(t, n.GetRootNode())
	assert.Nil(t, n.GetKeyNode())
	assert.NotNil(t, n.GetContext())
	assert.NotNil(t, n.GetIndex())
}

func TestContact_Build(t *testing.T) {
	n := &Contact{}
	k := n.Build(context.Background(), nil, nil, nil)
	assert.Nil(t, k)
}

func TestLicense_Build(t *testing.T) {
	n := &License{}
	k := n.Build(context.Background(), nil, nil, nil)
	assert.Nil(t, k)
}

func TestInfo_Hash(t *testing.T) {
	left := `title: princess b33f
summary: a thing
description: a thing
termsOfService: https://pb33f.io
x-princess: b33f
contact:
  name: buckaroo
  url: https://pb33f.io
license:
  name: magic beans
version: 1.2.3
x-b33f: princess`

	right := `title: princess b33f
summary: a thing
description: a thing
termsOfService: https://pb33f.io
x-princess: b33f
contact:
  name: buckaroo
  url: https://pb33f.io
license:
  name: magic beans
version: 1.2.3
x-b33f: princess`

	var lNode, rNode yaml.Node
	_ = yaml.Unmarshal([]byte(left), &lNode)
	_ = yaml.Unmarshal([]byte(right), &rNode)

	// create low level objects
	var lDoc Info
	var rDoc Info
	_ = low.BuildModel(lNode.Content[0], &lDoc)
	_ = low.BuildModel(rNode.Content[0], &rDoc)
	_ = lDoc.Build(context.Background(), nil, lNode.Content[0], nil)
	_ = rDoc.Build(context.Background(), nil, rNode.Content[0], nil)

	assert.Equal(t, lDoc.Hash(), rDoc.Hash())
}
