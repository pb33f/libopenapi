// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"github.com/stretchr/testify/assert"
	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
	"gopkg.in/yaml.v3"
	"path/filepath"
	"strings"
	"testing"
)

func TestRolodex_FindNodeOrigin(t *testing.T) {

	baseDir := "rolodex_test_data"

	cf := CreateOpenAPIIndexConfig()
	cf.BasePath = baseDir
	cf.AvoidCircularReferenceCheck = true

	fileFS, err := NewLocalFSWithConfig(&LocalFSConfig{
		BaseDirectory: baseDir,
		IndexConfig:   cf,
	})
	if err != nil {
		t.Fatal(err)
	}

	rolo := NewRolodex(cf)
	rolo.AddLocalFS(baseDir, fileFS)

	// open doc2
	f, rerr := rolo.Open("doc2.yaml")
	assert.Nil(t, rerr)
	assert.NotNil(t, f)

	node, _ := f.GetContentAsYAMLNode()

	rolo.SetRootNode(node)

	err = rolo.IndexTheRolodex()
	rolo.Resolve()

	assert.Len(t, rolo.indexes, 4)

	// extract something that can only exist after resolution
	path := "$.paths./nested/files3.get.responses.200.content.application/json.schema.properties.message.properties.utilMessage.properties.message.description"
	yp, _ := yamlpath.NewPath(path)
	results, _ := yp.Find(node)

	assert.NotNil(t, results)
	assert.Len(t, results, 1)
	assert.Equal(t, "I am pointless dir2 utility, I am multiple levels deep.", results[0].Value)

	// now for the truth, where did this come from?
	origin := rolo.FindNodeOrigin(results[0])

	assert.NotNil(t, origin)
	sep := string(filepath.Separator)
	assert.True(t, strings.HasSuffix(origin.AbsoluteLocation, "index"+sep+
		"rolodex_test_data"+sep+"dir2"+sep+"utils"+sep+"utils.yaml"))

	// should be identical to the original node
	assert.Equal(t, results[0], origin.Node)

	// look for something that cannot exist
	origin = rolo.FindNodeOrigin(nil)
	assert.Nil(t, origin)

	// modify the node and try again
	m := *results[0]
	m.Value = "I am a new message"
	origin = rolo.FindNodeOrigin(&m)
	assert.Nil(t, origin)

	// extract the doc root
	origin = rolo.FindNodeOrigin(node)
	assert.Nil(t, origin)
}

func TestRolodex_FindNodeOrigin_ModifyLookup(t *testing.T) {

	baseDir := "rolodex_test_data"

	cf := CreateOpenAPIIndexConfig()
	cf.BasePath = baseDir
	cf.AvoidCircularReferenceCheck = true

	fileFS, err := NewLocalFSWithConfig(&LocalFSConfig{
		BaseDirectory: baseDir,
		IndexConfig:   cf,
	})
	if err != nil {
		t.Fatal(err)
	}

	rolo := NewRolodex(cf)
	rolo.AddLocalFS(baseDir, fileFS)

	// open doc2
	f, rerr := rolo.Open("doc2.yaml")
	assert.Nil(t, rerr)
	assert.NotNil(t, f)

	node, _ := f.GetContentAsYAMLNode()

	rolo.SetRootNode(node)

	err = rolo.IndexTheRolodex()
	rolo.Resolve()

	assert.Len(t, rolo.indexes, 4)

	path := "$.paths./nested/files3.get.responses.200.content.application/json.schema"
	yp, _ := yamlpath.NewPath(path)
	results, _ := yp.Find(node)

	// copy, modify, and try again
	o := *results[0]
	o.Content = []*yaml.Node{
		{Value: "beer"}, {Value: "wine"}, {Value: "cake"}, {Value: "burgers"}, {Value: "herbs"}, {Value: "spices"},
	}
	origin := rolo.FindNodeOrigin(&o)
	assert.Nil(t, origin)
}
