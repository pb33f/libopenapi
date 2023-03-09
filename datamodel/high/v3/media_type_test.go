// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
    "github.com/pb33f/libopenapi/datamodel"
    "github.com/pb33f/libopenapi/datamodel/low/v3"
    "github.com/stretchr/testify/assert"
    "io/ioutil"
    "strings"
    "testing"
)

func TestMediaType_MarshalYAML(t *testing.T) {
    // load the petstore spec
    data, _ := ioutil.ReadFile("../../../test_specs/petstorev3.json")
    info, _ := datamodel.ExtractSpecInfo(data)
    var err []error
    lowDoc, err = v3.CreateDocumentFromConfig(info, &datamodel.DocumentConfiguration{})
    if err != nil {
        panic("broken something")
    }

    // create a new document and extract a media type object from it.
    d := NewDocument(lowDoc)
    mt := d.Paths.PathItems["/pet"].Put.RequestBody.Content["application/json"]

    // render out the media type
    yml, _ := mt.Render()

    // the rendered output should be a ref to the media type.
    op := `schema:
    $ref: '#/components/schemas/Pet'`

    assert.Equal(t, op, strings.TrimSpace(string(yml)))

    // modify the media type to have an example
    mt.Example = "testing a nice mutation"

    op = `schema:
    $ref: '#/components/schemas/Pet'
example: testing a nice mutation`

    yml, _ = mt.Render()

    assert.Equal(t, op, strings.TrimSpace(string(yml)))

}
