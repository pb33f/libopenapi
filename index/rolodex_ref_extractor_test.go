// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestExtractRefs_Local(t *testing.T) {

	test := `openapi: 3.0
paths:
  /burgers:
    post:
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Nine'
components:
  schemas:
    One:
      description: "test one"
      properties:
        things:
          "$ref": "#/components/schemas/Two"
      required:
        - things
    Two:
      description: "test two"
      properties:
        testThing:
          "$ref": "#/components/schemas/One"
        anyOf:
          - "$ref": "#/components/schemas/Four"
      required:
        - testThing
        - anyOf
    Three:
      description: "test three"
      properties:
        tester:
          "$ref": "#/components/schemas/Four"
        bester:
          "$ref": "#/components/schemas/Seven"
        yester:
          "$ref": "#/components/schemas/Seven"
      required:
        - tester
        - bester
        - yester
    Four:
      description: "test four"
      properties:
        lemons:
          "$ref": "#/components/schemas/Nine"
      required:
        - lemons
    Five:
      properties:
        rice:
          "$ref": "#/components/schemas/Six"
      required:
        - rice
    Six:
      properties:
        mints:
          "$ref": "#/components/schemas/Nine"
      required:
        - mints
    Seven:
      properties:
        wow:
          "$ref": "#/components/schemas/Three"
      required:
        - wow
    Nine:
      description: done.
    Ten:
      properties:
        yeah:
          "$ref": "#/components/schemas/Ten"
      required:
        - yeah`

	results := ExtractRefs(test)

	assert.Len(t, results, 12)

}

func TestExtractRefs_File(t *testing.T) {

	test := `openapi: 3.0
paths:
  /burgers:
    post:
      requestBody:
        content:
          application/json:
            schema:
              $ref: 'pizza.yaml#/components/schemas/Nine'
components:
  schemas:
    One:
      description: "test one"
      properties:
        things:
          "$ref": "../../fish.yaml#/components/schemas/Two"
      required:
        - things
    Two:
      description: "test two"
      properties:
        testThing:
          "$ref": "../../../lost/no.yaml#/components/schemas/One"
        anyOf:
          - "$ref": "why.yaml#/components/schemas/Four"
      required:
        - testThing
        - anyOf
    Three:
      description: "test three"
      properties:
        tester:
          "$ref": "no_more.yaml"
        bester:
          "$ref": 'why.yaml'
        yester:
          "$ref": "../../yes.yaml"
      required:
        - tester
        - bester
        - yester`

	results := ExtractRefs(test)

	assert.Len(t, results, 7)

}

func TestExtractRefType(t *testing.T) {
	assert.Equal(t, Local, ExtractRefType("#/components/schemas/One"))
	assert.Equal(t, File, ExtractRefType("pizza.yaml#/components/schemas/One"))
	assert.Equal(t, File, ExtractRefType("/pizza.yaml#/components/schemas/One"))
	assert.Equal(t, File, ExtractRefType("/something/pizza.yaml#/components/schemas/One"))
	assert.Equal(t, File, ExtractRefType("./pizza.yaml#/components/schemas/One"))
	assert.Equal(t, File, ExtractRefType("../pizza.yaml#/components/schemas/One"))
	assert.Equal(t, File, ExtractRefType("../../../pizza.yaml#/components/schemas/One"))
	assert.Equal(t, HTTP, ExtractRefType("http://yeah.com/pizza.yaml#/components/schemas/One"))
	assert.Equal(t, HTTP, ExtractRefType("https://yeah.com/pizza.yaml#/components/schemas/One"))
}

func TestExtractedRef_GetFile(t *testing.T) {

	a := &ExtractedRef{Location: "#/components/schemas/One", Type: Local}
	assert.Equal(t, "#/components/schemas/One", a.GetFile())

	a = &ExtractedRef{Location: "pizza.yaml#/components/schemas/One", Type: File}
	assert.Equal(t, "pizza.yaml", a.GetFile())

	a = &ExtractedRef{Location: "https://api.pb33f.io/openapi.yaml#/components/schemas/One", Type: File}
	assert.Equal(t, "https://api.pb33f.io/openapi.yaml", a.GetFile())

}

func TestExtractedRef_GetReference(t *testing.T) {

	a := &ExtractedRef{Location: "#/components/schemas/One", Type: Local}
	assert.Equal(t, "#/components/schemas/One", a.GetReference())

	a = &ExtractedRef{Location: "pizza.yaml#/components/schemas/One", Type: File}
	assert.Equal(t, "#/components/schemas/One", a.GetReference())

	a = &ExtractedRef{Location: "https://api.pb33f.io/openapi.yaml#/components/schemas/One", Type: File}
	assert.Equal(t, "#/components/schemas/One", a.GetReference())

}
