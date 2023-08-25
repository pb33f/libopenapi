// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package base

import (
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
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

	err = n.Build(nil,idxNode.Content[0], idx)
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

	cliName := n.FindExtension("x-cli-name")
	assert.NotNil(t, cliName)
	assert.Equal(t, "pizza cli", cliName.Value)
	assert.Len(t, n.GetExtensions(), 1)
}

func TestContact_Build(t *testing.T) {
	n := &Contact{}
	k := n.Build(nil, nil, nil)
	assert.Nil(t, k)
}

func TestLicense_Build(t *testing.T) {
	n := &License{}
	k := n.Build(nil, nil, nil)
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
	_ = lDoc.Build(nil, lNode.Content[0], nil)
	_ = rDoc.Build(nil, rNode.Content[0], nil)

	assert.Equal(t, lDoc.Hash(), rDoc.Hash())
}
