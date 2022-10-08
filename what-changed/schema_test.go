// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package what_changed

import (
	"github.com/pb33f/libopenapi/datamodel"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/stretchr/testify/assert"
	"testing"
)

// These tests require full documents to be tested properly. schemas are perhaps the most complex
// of all the things in OpenAPI, to ensure correctness, we must test the whole document structure.
func TestCompareSchemas(t *testing.T) {

	// to test this correctly, we need a simulated document with inline schemas for recursive
	// checking, as well as a couple of references, so we can avoid that disaster.
	// in our model, components/definitions will be checked independently for changes
	// and references will be checked only for value changes (points to a different reference)
	//	left := `openapi: 3.1.0
	//paths:
	//  /chicken/nuggets:
	//    get:
	//      responses:
	//        "200":
	//          content:
	//            application/json:
	//              schema:
	//                $ref: '#/components/schemas/OK'
	//  /chicken/soup:
	//    get:
	//      responses:
	//        "200":
	//          content:
	//            application/json:
	//              schema:
	//                title: an OK message
	//                allOf:
	//                  - type: int
	//                properties:
	//                  propA:
	//                    title: a proxy property
	//                    type: string
	//components:
	//  schemas:
	//    OK:
	//      title: an OK message
	//      allOf:
	//        - type: string
	//      properties:
	//        propA:
	//          title: a proxy property
	//          type: string`
	//
	//	right := `openapi: 3.1.0
	//paths:
	//  /chicken/nuggets:
	//    get:
	//      responses:
	//        "200":
	//          content:
	//            application/json:
	//              schema:
	//                $ref: '#/components/schemas/OK'
	//  /chicken/soup:
	//    get:
	//      responses:
	//        "200":
	//          content:
	//            application/json:
	//              schema:
	//                title: an OK message that is different
	//                allOf:
	//                  - type: int
	//                    description: oh my stars
	//                  - $ref: '#/components/schemas/NoWay'
	//                properties:
	//                  propA:
	//                    title: a proxy property
	//                    type: string
	//components:
	//  schemas:
	//    NoWay:
	//      type: string
	//    OK:
	//      title: an OK message that has now changed.
	//      allOf:
	//        - type: string
	//      properties:
	//        propA:
	//          title: a proxy property
	//          type: string`

	left := `openapi: 3.1.0
paths:
  /chicken/nuggets:
    get:
      responses:
        "200":
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/OK'
  /chicken/soup:
    get:
      responses:
        "200":
          content:
            application/json:
              schema:
                title: an OK message
                allOf:
                  - type: int
                properties:
                  propA:
                    title: a proxy property
                    type: string
components:
  schemas:
    OK:
      title: an OK message
      allOf:
        - type: string
      properties:
        propA:
          title: a proxy property
          type: string`

	right := `openapi: 3.1.0
paths:
  /chicken/nuggets:
    get:
      responses:
        "200":
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/OK'
  /chicken/soup:
    get:
      responses:
        "200":
          content:
            application/json:
              schema:
                title: an OK message that is different
                allOf:
                  - type: int
                    description: oh my stars
                  - $ref: '#/components/schemas/NoWay'
                properties:
                  propA:
                    title: a proxy property
                    type: string
components:
  schemas:
    NoWay:
      type: string
    OK:
      title: an OK message that has now changed.
      allOf:
        - type: string
      properties:
        propA:
          title: a proxy property
          type: string`

	leftInfo, _ := datamodel.ExtractSpecInfo([]byte(left))
	rightInfo, _ := datamodel.ExtractSpecInfo([]byte(right))

	leftDoc, _ := v3.CreateDocument(leftInfo)
	rightDoc, _ := v3.CreateDocument(rightInfo)

	// extract left reference schema and non reference schema.
	lSchemaProxy := leftDoc.Paths.Value.FindPath("/chicken/soup").Value.Get.
		Value.Responses.Value.FindResponseByCode("200").Value.
		FindContent("application/json").Value.Schema

	// extract right reference schema and non reference schema.
	rSchemaProxy := rightDoc.Paths.Value.FindPath("/chicken/soup").Value.Get.
		Value.Responses.Value.FindResponseByCode("200").Value.
		FindContent("application/json").Value.Schema

	changes := CompareSchemas(lSchemaProxy.Value, rSchemaProxy.Value)
	assert.NotNil(t, changes)

}
