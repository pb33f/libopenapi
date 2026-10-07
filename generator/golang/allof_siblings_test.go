// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package golang

import "testing"

func TestAllOfKeepsMembersDeclaredBesideIt(t *testing.T) {
	file := renderSpec(t, `openapi: 3.1.0
info:
  title: allOf siblings
  version: 1.0.0
paths: {}
components:
  schemas:
    Base:
      type: object
      properties:
        kind:
          type: string
    Derived:
      allOf:
        - $ref: '#/components/schemas/Base'
        - type: object
          properties:
            inherited:
              type: string
      required: [extra]
      properties:
        extra:
          type: string
      additionalProperties:
        type: integer
`)
	assertContainsCode(t, string(file.Source),
		"type Derived struct { Base Inherited *string `json:\"inherited,omitempty\"` Extra string `json:\"extra\"` AdditionalProperties map[string]int `json:\"-\"` }",
		`delete(raw, "extra")`,
	)
	assertParsesAndCompiles(t, file.Source)
}
