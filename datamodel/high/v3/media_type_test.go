// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/pkg-base/libopenapi/datamodel"
	"github.com/pkg-base/libopenapi/datamodel/low"
	v3 "github.com/pkg-base/libopenapi/datamodel/low/v3"
	"github.com/pkg-base/libopenapi/index"
	"github.com/pkg-base/libopenapi/orderedmap"
	"github.com/pkg-base/libopenapi/utils"
	"github.com/pkg-base/yaml"
	"github.com/stretchr/testify/assert"
)

func TestMediaType_MarshalYAMLInline(t *testing.T) {
	// load the petstore spec
	data, _ := os.ReadFile("../../../test_specs/petstorev3.json")
	info, _ := datamodel.ExtractSpecInfo(data)
	var err error
	lowDoc, err = v3.CreateDocumentFromConfig(info, &datamodel.DocumentConfiguration{})
	if err != nil {
		panic("broken something")
	}

	// create a new document and extract a media type object from it.
	d := NewDocument(lowDoc)
	mt := d.Paths.PathItems.GetOrZero("/pet").Put.RequestBody.Content.GetOrZero("application/json")

	// render out the media type
	yml, _ := mt.Render()

	// the rendered output should be a ref to the media type.
	op := `schema: {"$ref": "#/components/schemas/Pet"}`

	assert.Equal(t, op, strings.TrimSpace(string(yml)))

	// modify the media type to have an example
	mt.Example = utils.CreateStringNode("testing a nice mutation")

	op = `schema:
    required:
        - "name"
        - "photoUrls"
    type: "object"
    properties:
        "id":
            type: "integer"
            format: "int64"
            example: 10
        "name":
            type: "string"
            example: "doggie"
        "category":
            type: "object"
            properties:
                "id":
                    type: "integer"
                    format: "int64"
                    example: 1
                "name":
                    type: "string"
                    example: "Dogs"
            xml:
                name: "category"
        "photoUrls":
            type: "array"
            xml:
                wrapped: true
            items:
                type: "string"
                xml:
                    name: "photoUrl"
        "tags":
            type: "array"
            xml:
                wrapped: true
            items:
                type: "object"
                properties:
                    "id":
                        type: "integer"
                        format: "int64"
                    "name":
                        type: "string"
                xml:
                    name: "tag"
        "status":
            type: "string"
            description: "pet status in the store"
            enum:
                - "available"
                - "pending"
                - "sold"
    xml:
        name: "pet"
example: testing a nice mutation`

	yml, _ = mt.RenderInline()

	assert.Equal(t, op, strings.TrimSpace(string(yml)))
}

func TestMediaType_MarshalYAML(t *testing.T) {
	// load the petstore spec
	data, _ := os.ReadFile("../../../test_specs/petstorev3.json")
	info, _ := datamodel.ExtractSpecInfo(data)
	var err error
	lowDoc, err = v3.CreateDocumentFromConfig(info, &datamodel.DocumentConfiguration{})
	if err != nil {
		panic("broken something")
	}

	// create a new document and extract a media type object from it.
	d := NewDocument(lowDoc)
	mt := d.Paths.PathItems.GetOrZero("/pet").Put.RequestBody.Content.GetOrZero("application/json")

	// render out the media type
	yml, _ := mt.Render()

	// the rendered output should be a ref to the media type.
	op := `schema: {"$ref": "#/components/schemas/Pet"}`

	assert.Equal(t, op, strings.TrimSpace(string(yml)))

	// modify the media type to have an example
	mt.Example = utils.CreateStringNode("testing a nice mutation")

	op = `schema: {"$ref": "#/components/schemas/Pet"}
example: testing a nice mutation`

	yml, _ = mt.Render()

	assert.Equal(t, op, strings.TrimSpace(string(yml)))
}

func TestMediaType_Examples(t *testing.T) {
	yml := `examples:
    pbjBurger:
        summary: A horrible, nutty, sticky mess.
        value:
            name: Peanut And Jelly
            numPatties: 3
    cakeBurger:
        summary: A sickly, sweet, atrocity
        value:
            name: Chocolate Cake Burger
            numPatties: 5`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateOpenAPIIndexConfig())

	var n v3.MediaType
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

	r := NewMediaType(&n)

	assert.Equal(t, 2, orderedmap.Len(r.Examples))

	rend, _ := r.Render()
	assert.Len(t, rend, 290)
}

func TestMediaType_Examples_NotFromSchema(t *testing.T) {
	yml := `schema:
  type: string
  examples:
    - example 1
    - example 2
    - example 3`

	var idxNode yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &idxNode)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateOpenAPIIndexConfig())

	var n v3.MediaType
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(context.Background(), nil, idxNode.Content[0], idx)

	r := NewMediaType(&n)

	assert.Equal(t, 0, orderedmap.Len(r.Examples))
}
