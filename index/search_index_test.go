// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
    "github.com/stretchr/testify/assert"
    "gopkg.in/yaml.v3"
    "io/ioutil"
    "net/url"
    "testing"
)

func TestSpecIndex_SearchIndexForReference(t *testing.T) {
    petstore, _ := ioutil.ReadFile("../test_specs/petstorev3.json")
    var rootNode yaml.Node
    _ = yaml.Unmarshal(petstore, &rootNode)

    c := CreateOpenAPIIndexConfig()
    idx := NewSpecIndexWithConfig(&rootNode, c)

    ref := idx.SearchIndexForReference("#/components/schemas/Pet")
    assert.NotNil(t, ref)
}

func TestSpecIndex_SearchIndexForReference_ExternalSpecs(t *testing.T) {

    // load up an index with lots of references
    petstore, _ := ioutil.ReadFile("../test_specs/digitalocean.yaml")
    var rootNode yaml.Node
    _ = yaml.Unmarshal(petstore, &rootNode)

    c := CreateOpenAPIIndexConfig()
    c.BaseURL, _ = url.Parse("https://raw.githubusercontent.com/digitalocean/openapi/main/specification")
    idx := NewSpecIndexWithConfig(&rootNode, c)

    ref := idx.SearchIndexForReference("resources/apps/apps_list_instanceSizes.yml")
    assert.NotNil(t, ref)
    assert.Equal(t, "operationId", ref[0].Node.Content[0].Value)

    ref = idx.SearchIndexForReference("examples/ruby/domains_create.yml")
    assert.NotNil(t, ref)
    assert.Equal(t, "lang", ref[0].Node.Content[0].Value)

    ref = idx.SearchIndexForReference("../../shared/responses/server_error.yml")
    assert.NotNil(t, ref)
    assert.Equal(t, "description", ref[0].Node.Content[0].Value)

    ref = idx.SearchIndexForReference("../models/options.yml")
    assert.NotNil(t, ref)
    assert.Equal(t, "kubernetes_options", ref[0].Node.Content[0].Value)

}
