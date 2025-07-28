// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package what_changed

import (
	"fmt"
	"os"
	"testing"

	"github.com/pb33f/libopenapi/datamodel"
	v2 "github.com/pb33f/libopenapi/datamodel/low/v2"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/stretchr/testify/assert"
)

func TestCompareOpenAPIDocuments(t *testing.T) {
	original, _ := os.ReadFile("../test_specs/burgershop.openapi.yaml")
	modified, _ := os.ReadFile("../test_specs/burgershop.openapi-modified.yaml")
	infoOrig, _ := datamodel.ExtractSpecInfo(original)
	infoMod, _ := datamodel.ExtractSpecInfo(modified)

	origDoc, _ := v3.CreateDocumentFromConfig(infoOrig, datamodel.NewDocumentConfiguration())
	modDoc, _ := v3.CreateDocumentFromConfig(infoMod, datamodel.NewDocumentConfiguration())

	changes := CompareOpenAPIDocuments(origDoc, modDoc)
	assert.Equal(t, 75, changes.TotalChanges())
	assert.Equal(t, 20, changes.TotalBreakingChanges())
}

func TestCompareSwaggerDocuments(t *testing.T) {
	original, _ := os.ReadFile("../test_specs/petstorev2-complete.yaml")
	modified, _ := os.ReadFile("../test_specs/petstorev2-complete-modified.yaml")
	infoOrig, _ := datamodel.ExtractSpecInfo(original)
	infoMod, _ := datamodel.ExtractSpecInfo(modified)

	origDoc, _ := v2.CreateDocumentFromConfig(infoOrig, datamodel.NewDocumentConfiguration())
	modDoc, _ := v2.CreateDocumentFromConfig(infoMod, datamodel.NewDocumentConfiguration())

	changes := CompareSwaggerDocuments(origDoc, modDoc)
	assert.Equal(t, 52, changes.TotalChanges())
	assert.Equal(t, 27, changes.TotalBreakingChanges())
}

// TestCacheCollisionSelfReference reproduces the cache collision bug with self-referencing schemas
// This is the key pattern that triggers the QuickHash cache collision issue
// see: https://github.com/pb33f/libopenapi/pull/441
func TestCacheCollisionSelfReference(t *testing.T) {
	// Original spec - TreeNode schema with basic properties
	original := `{
  "openapi": "3.0.3",
  "info": {"title": "Test API", "version": "1.0.0"},
  "paths": {
    "/tree": {
      "get": {
        "responses": {
          "200": {
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/TreeNode"
                }
              }
            }
          }
        }
      }
    }
  },
  "components": {
    "schemas": {
      "TreeNode": {
        "type": "object",
        "additionalProperties": false,
        "properties": {
          "children": {
            "type": "array",
            "items": {
              "$ref": "#/components/schemas/TreeNode"
            }
          },
          "id": {
            "type": "string"
          },
          "name": {
            "type": "string"
          }
        },
        "required": [
          "id",
          "name"
        ]
      }
    }
  }
}`

	// Modified spec - TreeNode schema with additional properties
	modified := `{
  "openapi": "3.0.3",
  "info": {"title": "Test API", "version": "1.0.0"},
  "paths": {
    "/tree": {
      "get": {
        "responses": {
          "200": {
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/TreeNode"
                }
              }
            }
          }
        }
      }
    }
  },
  "components": {
    "schemas": {
      "TreeNode": {
        "type": "object",
        "additionalProperties": false,
        "properties": {
          "children": {
            "type": "array",
            "items": {
              "$ref": "#/components/schemas/TreeNode"
            }
          },
          "id": {
            "type": "string"
          },
          "name": {
            "type": "string"
          },
          "description": {
            "type": "string"
          },
          "metadata": {
            "type": "object"
          }
        },
        "required": [
          "id",
          "name",
          "description"
        ]
      }
    }
  }
}`

	infoOrig, _ := datamodel.ExtractSpecInfo([]byte(original))
	infoMod, _ := datamodel.ExtractSpecInfo([]byte(modified))

	origDoc, _ := v3.CreateDocumentFromConfig(infoOrig, datamodel.NewDocumentConfiguration())
	modDoc, _ := v3.CreateDocumentFromConfig(infoMod, datamodel.NewDocumentConfiguration())

	changes := CompareOpenAPIDocuments(origDoc, modDoc)

	assert.True(t, changes.TotalChanges() >= 3, "Expected at least 3 changes but got %d", changes.TotalChanges())
}

func Benchmark_CompareOpenAPIDocuments(b *testing.B) {
	original, _ := os.ReadFile("../test_specs/burgershop.openapi.yaml")
	modified, _ := os.ReadFile("../test_specs/burgershop.openapi-modified.yaml")

	infoOrig, _ := datamodel.ExtractSpecInfo(original)
	infoMod, _ := datamodel.ExtractSpecInfo(modified)
	origDoc, _ := v3.CreateDocumentFromConfig(infoOrig, datamodel.NewDocumentConfiguration())
	modDoc, _ := v3.CreateDocumentFromConfig(infoMod, datamodel.NewDocumentConfiguration())

	for i := 0; i < b.N; i++ {
		CompareOpenAPIDocuments(origDoc, modDoc)
	}
}

func Benchmark_CompareSwaggerDocuments(b *testing.B) {
	original, _ := os.ReadFile("../test_specs/petstorev2-complete.yaml")
	modified, _ := os.ReadFile("../test_specs/petstorev2-complete-modified.yaml")
	infoOrig, _ := datamodel.ExtractSpecInfo(original)
	infoMod, _ := datamodel.ExtractSpecInfo(modified)

	origDoc, _ := v2.CreateDocumentFromConfig(infoOrig, datamodel.NewDocumentConfiguration())
	modDoc, _ := v2.CreateDocumentFromConfig(infoMod, datamodel.NewDocumentConfiguration())

	for i := 0; i < b.N; i++ {
		CompareSwaggerDocuments(origDoc, modDoc)
	}
}

func Benchmark_CompareOpenAPIDocuments_NoChange(b *testing.B) {
	original, _ := os.ReadFile("../test_specs/burgershop.openapi.yaml")
	modified, _ := os.ReadFile("../test_specs/burgershop.openapi.yaml")

	infoOrig, _ := datamodel.ExtractSpecInfo(original)
	infoMod, _ := datamodel.ExtractSpecInfo(modified)
	origDoc, _ := v3.CreateDocumentFromConfig(infoOrig, datamodel.NewDocumentConfiguration())
	modDoc, _ := v3.CreateDocumentFromConfig(infoMod, datamodel.NewDocumentConfiguration())

	for i := 0; i < b.N; i++ {
		CompareOpenAPIDocuments(origDoc, modDoc)
	}
}

func Benchmark_CompareK8s(b *testing.B) {
	original, _ := os.ReadFile("../test_specs/k8s.json")
	modified, _ := os.ReadFile("../test_specs/k8s.json")

	infoOrig, _ := datamodel.ExtractSpecInfo(original)
	infoMod, _ := datamodel.ExtractSpecInfo(modified)
	origDoc, _ := v2.CreateDocumentFromConfig(infoOrig, datamodel.NewDocumentConfiguration())
	modDoc, _ := v2.CreateDocumentFromConfig(infoMod, datamodel.NewDocumentConfiguration())

	for i := 0; i < b.N; i++ {
		CompareSwaggerDocuments(origDoc, modDoc)
	}
}

func Benchmark_CompareStripe(b *testing.B) {
	original, _ := os.ReadFile("../test_specs/stripe.yaml")
	modified, _ := os.ReadFile("../test_specs/stripe.yaml")

	infoOrig, _ := datamodel.ExtractSpecInfo(original)
	infoMod, _ := datamodel.ExtractSpecInfo(modified)
	origDoc, _ := v3.CreateDocumentFromConfig(infoOrig, datamodel.NewDocumentConfiguration())
	modDoc, _ := v3.CreateDocumentFromConfig(infoMod, datamodel.NewDocumentConfiguration())

	for i := 0; i < b.N; i++ {
		CompareOpenAPIDocuments(origDoc, modDoc)
	}
}

func ExampleCompareOpenAPIDocuments() {
	// Read in a 'left' (original) OpenAPI specification
	original, _ := os.ReadFile("../test_specs/burgershop.openapi.yaml")

	// Read in a 'right' (modified) OpenAPI specification
	modified, _ := os.ReadFile("../test_specs/burgershop.openapi-modified.yaml")

	// Extract SpecInfo from bytes
	infoOriginal, _ := datamodel.ExtractSpecInfo(original)
	infoModified, _ := datamodel.ExtractSpecInfo(modified)

	// Build OpenAPI Documents from SpecInfo
	origDocument, _ := v3.CreateDocumentFromConfig(infoOriginal, datamodel.NewDocumentConfiguration())
	modDocDocument, _ := v3.CreateDocumentFromConfig(infoModified, datamodel.NewDocumentConfiguration())

	// Compare OpenAPI Documents and extract to *DocumentChanges
	changes := CompareOpenAPIDocuments(origDocument, modDocDocument)

	// Extract SchemaChanges from components changes.
	schemaChanges := changes.ComponentsChanges.SchemaChanges

	// Print out some interesting stats.
	fmt.Printf("There are %d changes, of which %d are breaking. %v schemas have changes.",
		changes.TotalChanges(), changes.TotalBreakingChanges(), len(schemaChanges))
	// Output: There are 75 changes, of which 20 are breaking. 6 schemas have changes.
}

func TestCheckExplodedFileCheck(t *testing.T) {
	original, _ := os.ReadFile("../test_specs/a.yaml")
	modified, _ := os.ReadFile("../test_specs/a-alt.yaml")
	infoOrig, _ := datamodel.ExtractSpecInfo(original)
	infoMod, _ := datamodel.ExtractSpecInfo(modified)

	// set the basepath:
	config := datamodel.NewDocumentConfiguration()
	config.BasePath = "../test_specs"
	config.AllowFileReferences = true

	origDoc, _ := v3.CreateDocumentFromConfig(infoOrig, config)
	modDoc, _ := v3.CreateDocumentFromConfig(infoMod, config)

	changes := CompareOpenAPIDocuments(origDoc, modDoc)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Equal(t, 0, changes.TotalBreakingChanges())

	allChanges := changes.GetAllChanges()
	assert.Len(t, allChanges, 1)
	assert.Equal(t, "b.yaml#/components/schemas/SchemaB", allChanges[0].OriginalObject)
	assert.Equal(t, "b-alt.yaml#/components/schemas/SchemaB", allChanges[0].NewObject)

}

func TestCheckExplodedFileCheck_IdenticalRefNames(t *testing.T) {
	original, _ := os.ReadFile("../test_specs/ref_test/orig/a.yaml")
	modified, _ := os.ReadFile("../test_specs/ref_test/mod/a.yaml")
	infoOrig, _ := datamodel.ExtractSpecInfo(original)
	infoMod, _ := datamodel.ExtractSpecInfo(modified)

	origDoc, _ := v3.CreateDocumentFromConfig(infoOrig, &datamodel.DocumentConfiguration{
		BasePath:            "../test_specs/ref_test/orig",
		AllowFileReferences: true,
	})
	modDoc, _ := v3.CreateDocumentFromConfig(infoMod, &datamodel.DocumentConfiguration{
		BasePath:            "../test_specs/ref_test/mod",
		AllowFileReferences: true,
	})

	changes := CompareOpenAPIDocuments(origDoc, modDoc)
	assert.Equal(t, 1, changes.TotalChanges())
	assert.Equal(t, 0, changes.TotalBreakingChanges())

	allChanges := changes.GetAllChanges()
	assert.Len(t, allChanges, 1)

}
