// Copyright 2022-2025 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.yaml.in/yaml/v4"
)

func TestSchemaIdEntry(t *testing.T) {
	entry := &SchemaIdEntry{
		Id:             "https://example.com/schema.json",
		ResolvedUri:    "https://example.com/schema.json",
		SchemaNode:     &yaml.Node{Kind: yaml.MappingNode},
		ParentId:       "",
		Index:          nil,
		DefinitionPath: "#/components/schemas/Pet",
		Line:           10,
		Column:         5,
	}

	assert.Equal(t, "https://example.com/schema.json", entry.Id)
	assert.Equal(t, "https://example.com/schema.json", entry.ResolvedUri)
	assert.NotNil(t, entry.SchemaNode)
	assert.Equal(t, "", entry.ParentId)
	assert.Nil(t, entry.Index)
	assert.Equal(t, "#/components/schemas/Pet", entry.DefinitionPath)
	assert.Equal(t, 10, entry.Line)
	assert.Equal(t, 5, entry.Column)
}

func TestNewSchemaIdScope(t *testing.T) {
	scope := NewSchemaIdScope("https://example.com/base.json")

	assert.Equal(t, "https://example.com/base.json", scope.BaseUri)
	assert.Empty(t, scope.Chain)
}

func TestSchemaIdScope_PushId(t *testing.T) {
	scope := NewSchemaIdScope("https://example.com/base.json")

	scope.PushId("https://example.com/schema1.json")
	assert.Equal(t, "https://example.com/schema1.json", scope.BaseUri)
	assert.Len(t, scope.Chain, 1)
	assert.Equal(t, "https://example.com/schema1.json", scope.Chain[0])

	scope.PushId("https://example.com/schema2.json")
	assert.Equal(t, "https://example.com/schema2.json", scope.BaseUri)
	assert.Len(t, scope.Chain, 2)
	assert.Equal(t, "https://example.com/schema2.json", scope.Chain[1])
}

func TestSchemaIdScope_PopId(t *testing.T) {
	scope := NewSchemaIdScope("https://example.com/base.json")
	scope.PushId("https://example.com/schema1.json")
	scope.PushId("https://example.com/schema2.json")

	scope.PopId()
	assert.Equal(t, "https://example.com/schema1.json", scope.BaseUri)
	assert.Len(t, scope.Chain, 1)

	scope.PopId()
	assert.Empty(t, scope.Chain)

	// Pop on empty chain should not panic
	scope.PopId()
	assert.Empty(t, scope.Chain)
}

func TestSchemaIdScope_Copy(t *testing.T) {
	scope := NewSchemaIdScope("https://example.com/base.json")
	scope.PushId("https://example.com/schema1.json")

	copied := scope.Copy()

	assert.Equal(t, scope.BaseUri, copied.BaseUri)
	assert.Equal(t, scope.Chain, copied.Chain)

	// Modifying original should not affect copy
	scope.PushId("https://example.com/schema2.json")
	assert.NotEqual(t, scope.BaseUri, copied.BaseUri)
	assert.Len(t, copied.Chain, 1)
	assert.Len(t, scope.Chain, 2)
}

func TestValidateSchemaId(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid absolute URI",
			id:      "https://example.com/schema.json",
			wantErr: false,
		},
		{
			name:    "valid relative URI",
			id:      "schema.json",
			wantErr: false,
		},
		{
			name:    "valid relative path",
			id:      "./schemas/pet.json",
			wantErr: false,
		},
		{
			name:    "empty $id",
			id:      "",
			wantErr: true,
			errMsg:  "$id cannot be empty",
		},
		{
			name:    "$id with fragment",
			id:      "https://example.com/schema.json#/definitions",
			wantErr: true,
			errMsg:  "$id must not contain fragment identifier '#'",
		},
		{
			name:    "$id with just fragment",
			id:      "#anchor",
			wantErr: true,
			errMsg:  "$id must not contain fragment identifier '#'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSchemaId(tt.id)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestResolveSchemaId(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		baseUri  string
		expected string
		wantErr  bool
	}{
		{
			name:     "absolute $id ignores base",
			id:       "https://example.com/schema.json",
			baseUri:  "https://other.com/base.json",
			expected: "https://example.com/schema.json",
			wantErr:  false,
		},
		{
			name:     "relative $id resolved against base",
			id:       "schema.json",
			baseUri:  "https://example.com/schemas/",
			expected: "https://example.com/schemas/schema.json",
			wantErr:  false,
		},
		{
			name:     "relative path with directory",
			id:       "../common/types.json",
			baseUri:  "https://example.com/schemas/pets/",
			expected: "https://example.com/schemas/common/types.json",
			wantErr:  false,
		},
		{
			name:     "relative $id without base",
			id:       "schema.json",
			baseUri:  "",
			expected: "schema.json",
			wantErr:  false,
		},
		{
			name:     "empty $id",
			id:       "",
			baseUri:  "https://example.com/",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ResolveSchemaId(tt.id, tt.baseUri)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestResolveRefAgainstSchemaId(t *testing.T) {
	tests := []struct {
		name     string
		ref      string
		scope    *SchemaIdScope
		expected string
		wantErr  bool
	}{
		{
			name:     "absolute ref returns as-is",
			ref:      "https://example.com/schema.json",
			scope:    NewSchemaIdScope("https://other.com/base.json"),
			expected: "https://example.com/schema.json",
			wantErr:  false,
		},
		{
			name:     "relative ref resolved against scope base",
			ref:      "pet.json",
			scope:    NewSchemaIdScope("https://example.com/schemas/"),
			expected: "https://example.com/schemas/pet.json",
			wantErr:  false,
		},
		{
			name:     "relative ref with fragment",
			ref:      "common.json#/definitions/Error",
			scope:    NewSchemaIdScope("https://example.com/schemas/"),
			expected: "https://example.com/schemas/common.json#/definitions/Error",
			wantErr:  false,
		},
		{
			name:     "relative ref without scope",
			ref:      "pet.json",
			scope:    nil,
			expected: "pet.json",
			wantErr:  false,
		},
		{
			name:     "relative ref with empty base",
			ref:      "pet.json",
			scope:    NewSchemaIdScope(""),
			expected: "pet.json",
			wantErr:  false,
		},
		{
			name:     "empty ref",
			ref:      "",
			scope:    NewSchemaIdScope("https://example.com/"),
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ResolveRefAgainstSchemaId(tt.ref, tt.scope)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestSplitRefFragment(t *testing.T) {
	tests := []struct {
		name         string
		ref          string
		wantBase     string
		wantFragment string
	}{
		{
			name:         "ref with fragment",
			ref:          "https://example.com/schema.json#/definitions/Pet",
			wantBase:     "https://example.com/schema.json",
			wantFragment: "#/definitions/Pet",
		},
		{
			name:         "ref without fragment",
			ref:          "https://example.com/schema.json",
			wantBase:     "https://example.com/schema.json",
			wantFragment: "",
		},
		{
			name:         "relative ref with fragment",
			ref:          "pet.json#/properties/name",
			wantBase:     "pet.json",
			wantFragment: "#/properties/name",
		},
		{
			name:         "just fragment",
			ref:          "#/definitions/Pet",
			wantBase:     "",
			wantFragment: "#/definitions/Pet",
		},
		{
			name:         "empty ref",
			ref:          "",
			wantBase:     "",
			wantFragment: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base, fragment := SplitRefFragment(tt.ref)
			assert.Equal(t, tt.wantBase, base)
			assert.Equal(t, tt.wantFragment, fragment)
		})
	}
}

func TestResolveSchemaId_InvalidURIs(t *testing.T) {
	// Test invalid base URI
	_, err := ResolveSchemaId("schema.json", "://invalid-base")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid base URI")

	// Test invalid $id URI (control characters)
	_, err = ResolveSchemaId("schema\x00.json", "https://example.com/")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid $id URI")
}

func TestResolveRefAgainstSchemaId_InvalidBaseInScope(t *testing.T) {
	// Test invalid base URI in scope
	scope := &SchemaIdScope{
		BaseUri: "://invalid-base",
		Chain:   []string{},
	}
	_, err := ResolveRefAgainstSchemaId("schema.json", scope)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid base URI in scope")
}

func TestSchemaIdScope_NestedScopes(t *testing.T) {
	// Test a realistic nested $id scenario
	// Document base: https://example.com/openapi.yaml
	// Schema 1: $id: "https://example.com/schemas/pet.json"
	// Schema 2 (nested): $id: "definitions/category.json" (relative)

	scope := NewSchemaIdScope("https://example.com/openapi.yaml")

	// First $id is absolute
	scope.PushId("https://example.com/schemas/pet.json")
	assert.Equal(t, "https://example.com/schemas/pet.json", scope.BaseUri)

	// Resolve a relative $id against current base
	resolved, err := ResolveSchemaId("definitions/category.json", scope.BaseUri)
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com/schemas/definitions/category.json", resolved)

	// Push the nested $id
	scope.PushId(resolved)
	assert.Equal(t, "https://example.com/schemas/definitions/category.json", scope.BaseUri)
	assert.Len(t, scope.Chain, 2)

	// Resolve a relative $ref from this nested scope
	refResolved, err := ResolveRefAgainstSchemaId("../common/types.json", scope)
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com/schemas/common/types.json", refResolved)
}

// SpecIndex registry tests

func TestSpecIndex_RegisterSchemaId(t *testing.T) {
	index := &SpecIndex{}

	entry := &SchemaIdEntry{
		Id:             "https://example.com/schema.json",
		ResolvedUri:    "https://example.com/schema.json",
		SchemaNode:     &yaml.Node{Kind: yaml.MappingNode},
		Index:          index,
		DefinitionPath: "#/components/schemas/Pet",
		Line:           10,
		Column:         5,
	}

	err := index.RegisterSchemaId(entry)
	assert.NoError(t, err)

	// Verify registration
	found := index.GetSchemaById("https://example.com/schema.json")
	assert.NotNil(t, found)
	assert.Equal(t, entry.Id, found.Id)
}

func TestSpecIndex_RegisterSchemaId_Nil(t *testing.T) {
	index := &SpecIndex{}
	err := index.RegisterSchemaId(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot register nil")
}

func TestSpecIndex_RegisterSchemaId_Invalid(t *testing.T) {
	index := &SpecIndex{}
	entry := &SchemaIdEntry{
		Id:   "https://example.com/schema.json#fragment",
		Line: 10,
	}

	err := index.RegisterSchemaId(entry)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fragment")
}

func TestSpecIndex_RegisterSchemaId_Duplicate(t *testing.T) {
	index := &SpecIndex{}

	entry1 := &SchemaIdEntry{
		Id:          "https://example.com/schema.json",
		ResolvedUri: "https://example.com/schema.json",
		Index:       index,
		Line:        10,
	}

	entry2 := &SchemaIdEntry{
		Id:          "https://example.com/schema.json",
		ResolvedUri: "https://example.com/schema.json",
		Index:       index,
		Line:        20,
	}

	err := index.RegisterSchemaId(entry1)
	assert.NoError(t, err)

	// Second registration should not error (first-wins)
	err = index.RegisterSchemaId(entry2)
	assert.NoError(t, err)

	// Verify first entry is kept
	found := index.GetSchemaById("https://example.com/schema.json")
	assert.Equal(t, 10, found.Line)
}

func TestSpecIndex_RegisterSchemaId_UsesIdWhenResolvedUriEmpty(t *testing.T) {
	index := &SpecIndex{}

	entry := &SchemaIdEntry{
		Id:          "schema.json",
		ResolvedUri: "", // Empty resolved URI
		Index:       index,
		Line:        10,
	}

	err := index.RegisterSchemaId(entry)
	assert.NoError(t, err)

	// Should be registered under Id, not ResolvedUri
	found := index.GetSchemaById("schema.json")
	assert.NotNil(t, found)
}

func TestSpecIndex_GetSchemaById_Empty(t *testing.T) {
	index := &SpecIndex{}

	found := index.GetSchemaById("https://example.com/not-found.json")
	assert.Nil(t, found)
}

func TestSpecIndex_GetAllSchemaIds(t *testing.T) {
	index := &SpecIndex{}

	entry1 := &SchemaIdEntry{
		Id:          "https://example.com/a.json",
		ResolvedUri: "https://example.com/a.json",
		Index:       index,
	}
	entry2 := &SchemaIdEntry{
		Id:          "https://example.com/b.json",
		ResolvedUri: "https://example.com/b.json",
		Index:       index,
	}

	_ = index.RegisterSchemaId(entry1)
	_ = index.RegisterSchemaId(entry2)

	all := index.GetAllSchemaIds()
	assert.Len(t, all, 2)
	assert.NotNil(t, all["https://example.com/a.json"])
	assert.NotNil(t, all["https://example.com/b.json"])
}

func TestSpecIndex_GetAllSchemaIds_Empty(t *testing.T) {
	index := &SpecIndex{}
	all := index.GetAllSchemaIds()
	assert.NotNil(t, all)
	assert.Empty(t, all)
}

// Rolodex registry tests

func TestRolodex_RegisterGlobalSchemaId(t *testing.T) {
	config := CreateClosedAPIIndexConfig()
	rolodex := NewRolodex(config)
	index := &SpecIndex{}

	entry := &SchemaIdEntry{
		Id:          "https://example.com/schema.json",
		ResolvedUri: "https://example.com/schema.json",
		Index:       index,
		Line:        10,
	}

	err := rolodex.RegisterGlobalSchemaId(entry)
	assert.NoError(t, err)

	// Verify registration
	found := rolodex.LookupSchemaById("https://example.com/schema.json")
	assert.NotNil(t, found)
	assert.Equal(t, entry.Id, found.Id)
}

func TestRolodex_RegisterGlobalSchemaId_NilRolodex(t *testing.T) {
	var rolodex *Rolodex
	entry := &SchemaIdEntry{Id: "test"}
	err := rolodex.RegisterGlobalSchemaId(entry)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil Rolodex")
}

func TestRolodex_RegisterGlobalSchemaId_NilEntry(t *testing.T) {
	config := CreateClosedAPIIndexConfig()
	rolodex := NewRolodex(config)
	err := rolodex.RegisterGlobalSchemaId(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil SchemaIdEntry")
}

func TestRolodex_RegisterGlobalSchemaId_Invalid(t *testing.T) {
	config := CreateClosedAPIIndexConfig()
	rolodex := NewRolodex(config)
	entry := &SchemaIdEntry{
		Id:   "https://example.com/schema.json#fragment",
		Line: 10,
	}

	err := rolodex.RegisterGlobalSchemaId(entry)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fragment")
}

func TestRolodex_RegisterGlobalSchemaId_Duplicate(t *testing.T) {
	config := CreateClosedAPIIndexConfig()
	rolodex := NewRolodex(config)
	index := &SpecIndex{}

	entry1 := &SchemaIdEntry{
		Id:          "https://example.com/schema.json",
		ResolvedUri: "https://example.com/schema.json",
		Index:       index,
		Line:        10,
	}

	entry2 := &SchemaIdEntry{
		Id:          "https://example.com/schema.json",
		ResolvedUri: "https://example.com/schema.json",
		Index:       index,
		Line:        20,
	}

	err := rolodex.RegisterGlobalSchemaId(entry1)
	assert.NoError(t, err)

	// Second registration should not error (first-wins)
	err = rolodex.RegisterGlobalSchemaId(entry2)
	assert.NoError(t, err)

	// Verify first entry is kept
	found := rolodex.LookupSchemaById("https://example.com/schema.json")
	assert.Equal(t, 10, found.Line)
}

func TestRolodex_LookupSchemaById_NilRolodex(t *testing.T) {
	var rolodex *Rolodex
	found := rolodex.LookupSchemaById("test")
	assert.Nil(t, found)
}

func TestRolodex_LookupSchemaById_Empty(t *testing.T) {
	config := CreateClosedAPIIndexConfig()
	rolodex := NewRolodex(config)
	found := rolodex.LookupSchemaById("https://example.com/not-found.json")
	assert.Nil(t, found)
}

func TestRolodex_GetAllGlobalSchemaIds(t *testing.T) {
	config := CreateClosedAPIIndexConfig()
	rolodex := NewRolodex(config)
	index := &SpecIndex{}

	entry1 := &SchemaIdEntry{
		Id:          "https://example.com/a.json",
		ResolvedUri: "https://example.com/a.json",
		Index:       index,
	}
	entry2 := &SchemaIdEntry{
		Id:          "https://example.com/b.json",
		ResolvedUri: "https://example.com/b.json",
		Index:       index,
	}

	_ = rolodex.RegisterGlobalSchemaId(entry1)
	_ = rolodex.RegisterGlobalSchemaId(entry2)

	all := rolodex.GetAllGlobalSchemaIds()
	assert.Len(t, all, 2)
	assert.NotNil(t, all["https://example.com/a.json"])
	assert.NotNil(t, all["https://example.com/b.json"])
}

func TestRolodex_GetAllGlobalSchemaIds_NilRolodex(t *testing.T) {
	var rolodex *Rolodex
	all := rolodex.GetAllGlobalSchemaIds()
	assert.NotNil(t, all)
	assert.Empty(t, all)
}

func TestRolodex_GetAllGlobalSchemaIds_Empty(t *testing.T) {
	config := CreateClosedAPIIndexConfig()
	rolodex := NewRolodex(config)
	all := rolodex.GetAllGlobalSchemaIds()
	assert.NotNil(t, all)
	assert.Empty(t, all)
}

func TestRolodex_RegisterIdsFromIndex(t *testing.T) {
	config := CreateClosedAPIIndexConfig()
	rolodex := NewRolodex(config)
	index := &SpecIndex{}

	// Register entries in the index
	entry1 := &SchemaIdEntry{
		Id:          "https://example.com/a.json",
		ResolvedUri: "https://example.com/a.json",
		Index:       index,
	}
	entry2 := &SchemaIdEntry{
		Id:          "https://example.com/b.json",
		ResolvedUri: "https://example.com/b.json",
		Index:       index,
	}

	_ = index.RegisterSchemaId(entry1)
	_ = index.RegisterSchemaId(entry2)

	// Aggregate to rolodex
	rolodex.RegisterIdsFromIndex(index)

	// Verify both are in global registry
	all := rolodex.GetAllGlobalSchemaIds()
	assert.Len(t, all, 2)
}

func TestRolodex_RegisterIdsFromIndex_NilRolodex(t *testing.T) {
	var rolodex *Rolodex
	index := &SpecIndex{}
	// Should not panic
	rolodex.RegisterIdsFromIndex(index)
}

func TestRolodex_RegisterIdsFromIndex_NilIndex(t *testing.T) {
	config := CreateClosedAPIIndexConfig()
	rolodex := NewRolodex(config)
	// Should not panic
	rolodex.RegisterIdsFromIndex(nil)
}

// Integration test: verify $id extraction during indexing

func TestSchemaId_ExtractionDuringIndexing(t *testing.T) {
	// OpenAPI 3.1 spec with $id declarations
	spec := `openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Pet:
      $id: "https://example.com/schemas/pet.json"
      type: object
      properties:
        name:
          type: string
    Category:
      $id: "https://example.com/schemas/category.json"
      type: object
      properties:
        id:
          type: integer
`

	var rootNode yaml.Node
	err := yaml.Unmarshal([]byte(spec), &rootNode)
	assert.NoError(t, err)

	config := CreateClosedAPIIndexConfig()
	config.SpecAbsolutePath = "https://example.com/openapi.yaml"
	rolodex := NewRolodex(config)

	index := NewSpecIndexWithConfig(&rootNode, config)
	assert.NotNil(t, index)

	// Add index to rolodex (this triggers RegisterIdsFromIndex)
	rolodex.AddIndex(index)

	// Verify $id entries were registered in the index
	allIds := index.GetAllSchemaIds()
	assert.Len(t, allIds, 2)
	assert.NotNil(t, allIds["https://example.com/schemas/pet.json"])
	assert.NotNil(t, allIds["https://example.com/schemas/category.json"])

	// Verify $id entries were aggregated to rolodex
	globalIds := rolodex.GetAllGlobalSchemaIds()
	assert.Len(t, globalIds, 2)
	assert.NotNil(t, globalIds["https://example.com/schemas/pet.json"])
	assert.NotNil(t, globalIds["https://example.com/schemas/category.json"])

	// Verify lookup works
	petEntry := rolodex.LookupSchemaById("https://example.com/schemas/pet.json")
	assert.NotNil(t, petEntry)
	assert.Equal(t, "https://example.com/schemas/pet.json", petEntry.Id)
	assert.Contains(t, petEntry.DefinitionPath, "Pet")
}

func TestSchemaId_ExtractionWithInvalidId(t *testing.T) {
	// OpenAPI 3.1 spec with invalid $id (contains fragment)
	spec := `openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Pet:
      $id: "https://example.com/schemas/pet.json#invalid"
      type: object
`

	var rootNode yaml.Node
	err := yaml.Unmarshal([]byte(spec), &rootNode)
	assert.NoError(t, err)

	config := CreateClosedAPIIndexConfig()
	index := NewSpecIndexWithConfig(&rootNode, config)
	assert.NotNil(t, index)

	// Invalid $id should not be registered
	allIds := index.GetAllSchemaIds()
	assert.Len(t, allIds, 0)
}

func TestSchemaId_ExtractionWithRelativeId(t *testing.T) {
	// OpenAPI 3.1 spec with relative $id
	spec := `openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Pet:
      $id: "schemas/pet.json"
      type: object
`

	var rootNode yaml.Node
	err := yaml.Unmarshal([]byte(spec), &rootNode)
	assert.NoError(t, err)

	config := CreateClosedAPIIndexConfig()
	config.SpecAbsolutePath = "https://example.com/api/openapi.yaml"
	index := NewSpecIndexWithConfig(&rootNode, config)
	assert.NotNil(t, index)

	// Relative $id should be resolved against document base
	allIds := index.GetAllSchemaIds()
	assert.Len(t, allIds, 1)

	// Should be resolved to absolute URI
	resolved := allIds["https://example.com/api/schemas/pet.json"]
	assert.NotNil(t, resolved)
	assert.Equal(t, "schemas/pet.json", resolved.Id)
	assert.Equal(t, "https://example.com/api/schemas/pet.json", resolved.ResolvedUri)
}

// Resolution tests

func TestResolveRefViaSchemaId(t *testing.T) {
	spec := `openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Pet:
      $id: "https://example.com/schemas/pet.json"
      type: object
      properties:
        name:
          type: string
`

	var rootNode yaml.Node
	err := yaml.Unmarshal([]byte(spec), &rootNode)
	assert.NoError(t, err)

	config := CreateClosedAPIIndexConfig()
	index := NewSpecIndexWithConfig(&rootNode, config)
	assert.NotNil(t, index)

	// Test resolution by $id
	resolved := index.ResolveRefViaSchemaId("https://example.com/schemas/pet.json")
	assert.NotNil(t, resolved)
	assert.Equal(t, "https://example.com/schemas/pet.json", resolved.FullDefinition)
}

func TestResolveRefViaSchemaId_NotFound(t *testing.T) {
	spec := `openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
`

	var rootNode yaml.Node
	err := yaml.Unmarshal([]byte(spec), &rootNode)
	assert.NoError(t, err)

	config := CreateClosedAPIIndexConfig()
	index := NewSpecIndexWithConfig(&rootNode, config)
	assert.NotNil(t, index)

	// Should return nil for unknown $id
	resolved := index.ResolveRefViaSchemaId("https://example.com/not-found.json")
	assert.Nil(t, resolved)
}

func TestResolveRefViaSchemaId_EmptyRef(t *testing.T) {
	index := &SpecIndex{}
	resolved := index.ResolveRefViaSchemaId("")
	assert.Nil(t, resolved)
}

func TestResolveRefViaSchemaId_LocalFragment(t *testing.T) {
	index := &SpecIndex{}
	// Local fragments (starting with #) should not be resolved via $id
	resolved := index.ResolveRefViaSchemaId("#/components/schemas/Pet")
	assert.Nil(t, resolved)
}

func TestResolveRefViaSchemaId_WithFragment(t *testing.T) {
	spec := `openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Pet:
      $id: "https://example.com/schemas/pet.json"
      type: object
      properties:
        name:
          type: string
`

	var rootNode yaml.Node
	err := yaml.Unmarshal([]byte(spec), &rootNode)
	assert.NoError(t, err)

	config := CreateClosedAPIIndexConfig()
	index := NewSpecIndexWithConfig(&rootNode, config)
	assert.NotNil(t, index)

	// Test resolution with fragment
	resolved := index.ResolveRefViaSchemaId("https://example.com/schemas/pet.json#/properties/name")
	assert.NotNil(t, resolved)
	// The resolved node should be the "name" property schema
	if resolved.Node != nil {
		// Check it's the right node (type: string)
		for i := 0; i < len(resolved.Node.Content)-1; i += 2 {
			if resolved.Node.Content[i].Value == "type" {
				assert.Equal(t, "string", resolved.Node.Content[i+1].Value)
				break
			}
		}
	}
}

// Fragment navigation tests

func TestNavigateToFragment(t *testing.T) {
	yamlContent := `type: object
properties:
  name:
    type: string
  age:
    type: integer
items:
  - first
  - second
`
	var node yaml.Node
	err := yaml.Unmarshal([]byte(yamlContent), &node)
	assert.NoError(t, err)

	root := node.Content[0] // Get the mapping node

	tests := []struct {
		name     string
		fragment string
		wantNil  bool
		checkVal string // If not empty, check this value in the result
	}{
		{
			name:     "empty fragment returns root",
			fragment: "",
			wantNil:  true, // Empty returns nil, not root
		},
		{
			name:     "hash only returns root",
			fragment: "#",
			wantNil:  false,
		},
		{
			name:     "single slash returns root",
			fragment: "#/",
			wantNil:  false,
		},
		{
			name:     "navigate to type",
			fragment: "#/type",
			wantNil:  false,
			checkVal: "object",
		},
		{
			name:     "navigate to properties/name",
			fragment: "#/properties/name",
			wantNil:  false,
		},
		{
			name:     "navigate to properties/name/type",
			fragment: "#/properties/name/type",
			wantNil:  false,
			checkVal: "string",
		},
		{
			name:     "navigate to items/0",
			fragment: "#/items/0",
			wantNil:  false,
			checkVal: "first",
		},
		{
			name:     "navigate to items/1",
			fragment: "#/items/1",
			wantNil:  false,
			checkVal: "second",
		},
		{
			name:    "navigate to non-existent path",
			fragment: "#/nonexistent",
			wantNil:  true,
		},
		{
			name:    "navigate to invalid array index",
			fragment: "#/items/99",
			wantNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := navigateToFragment(root, tt.fragment)
			if tt.wantNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				if tt.checkVal != "" {
					assert.Equal(t, tt.checkVal, result.Value)
				}
			}
		})
	}
}

func TestNavigateToFragment_NilRoot(t *testing.T) {
	result := navigateToFragment(nil, "#/test")
	assert.Nil(t, result)
}

func TestNavigateToFragment_EscapedCharacters(t *testing.T) {
	// Test JSON pointer escape sequences (~0 = ~, ~1 = /)
	yamlContent := `properties:
  "key/with/slashes":
    type: string
  "key~with~tildes":
    type: integer
`
	var node yaml.Node
	err := yaml.Unmarshal([]byte(yamlContent), &node)
	assert.NoError(t, err)

	root := node.Content[0]

	// Test ~1 escaping for slashes
	result := navigateToFragment(root, "#/properties/key~1with~1slashes")
	assert.NotNil(t, result)

	// Test ~0 escaping for tildes
	result = navigateToFragment(root, "#/properties/key~0with~0tildes")
	assert.NotNil(t, result)
}

// Circular reference protection tests

func TestGetResolvingIds_Empty(t *testing.T) {
	ctx := context.Background()
	ids := GetResolvingIds(ctx)
	assert.Nil(t, ids)
}

func TestAddResolvingId(t *testing.T) {
	ctx := context.Background()

	ctx = AddResolvingId(ctx, "https://example.com/a.json")
	ids := GetResolvingIds(ctx)
	assert.NotNil(t, ids)
	assert.True(t, ids["https://example.com/a.json"])

	ctx = AddResolvingId(ctx, "https://example.com/b.json")
	ids = GetResolvingIds(ctx)
	assert.Len(t, ids, 2)
	assert.True(t, ids["https://example.com/a.json"])
	assert.True(t, ids["https://example.com/b.json"])
}

func TestIsIdBeingResolved(t *testing.T) {
	ctx := context.Background()

	// Not being resolved initially
	assert.False(t, IsIdBeingResolved(ctx, "https://example.com/a.json"))

	// Add to resolving set
	ctx = AddResolvingId(ctx, "https://example.com/a.json")
	assert.True(t, IsIdBeingResolved(ctx, "https://example.com/a.json"))
	assert.False(t, IsIdBeingResolved(ctx, "https://example.com/b.json"))
}

func TestIsIdBeingResolved_EmptyContext(t *testing.T) {
	ctx := context.Background()
	assert.False(t, IsIdBeingResolved(ctx, "anything"))
}

// Test nested $id resolution - critical for JSON Schema 2020-12 compliance
func TestSchemaId_NestedIdResolution(t *testing.T) {
	// This tests the critical fix: nested $id should resolve against parent $id, not document base
	spec := `openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Parent:
      $id: "https://example.com/schemas/base.json"
      type: object
      properties:
        child:
          $id: "subschema.json"
          type: string
`

	var rootNode yaml.Node
	err := yaml.Unmarshal([]byte(spec), &rootNode)
	assert.NoError(t, err)

	config := CreateClosedAPIIndexConfig()
	config.SpecAbsolutePath = "https://example.com/openapi.yaml"
	index := NewSpecIndexWithConfig(&rootNode, config)
	assert.NotNil(t, index)

	allIds := index.GetAllSchemaIds()
	assert.Len(t, allIds, 2, "Should have 2 $id entries")

	// Parent $id should resolve to its declared value
	parentEntry := allIds["https://example.com/schemas/base.json"]
	assert.NotNil(t, parentEntry, "Parent $id should be registered")
	assert.Equal(t, "https://example.com/schemas/base.json", parentEntry.ResolvedUri)

	// Nested $id should resolve against parent, NOT document base
	// subschema.json relative to https://example.com/schemas/base.json = https://example.com/schemas/subschema.json
	nestedEntry := allIds["https://example.com/schemas/subschema.json"]
	assert.NotNil(t, nestedEntry, "Nested $id should be registered with correct resolution")
	assert.Equal(t, "subschema.json", nestedEntry.Id)
	assert.Equal(t, "https://example.com/schemas/subschema.json", nestedEntry.ResolvedUri)
	assert.Equal(t, "https://example.com/schemas/base.json", nestedEntry.ParentId, "Should track parent $id")
}

func TestGetSchemaIdScope_Empty(t *testing.T) {
	ctx := context.Background()
	scope := GetSchemaIdScope(ctx)
	assert.Nil(t, scope)
}

func TestWithSchemaIdScope(t *testing.T) {
	ctx := context.Background()
	scope := NewSchemaIdScope("https://example.com/base.json")
	ctx = WithSchemaIdScope(ctx, scope)

	retrieved := GetSchemaIdScope(ctx)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "https://example.com/base.json", retrieved.BaseUri)
}

func TestSchemaId_DeeplyNestedIdResolution(t *testing.T) {
	// Test 3-level deep nesting
	spec := `openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Root:
      $id: "https://example.com/root.json"
      type: object
      properties:
        level1:
          $id: "level1/"
          type: object
          properties:
            level2:
              $id: "level2.json"
              type: string
`

	var rootNode yaml.Node
	err := yaml.Unmarshal([]byte(spec), &rootNode)
	assert.NoError(t, err)

	config := CreateClosedAPIIndexConfig()
	config.SpecAbsolutePath = "https://example.com/openapi.yaml"
	index := NewSpecIndexWithConfig(&rootNode, config)
	assert.NotNil(t, index)

	allIds := index.GetAllSchemaIds()
	assert.Len(t, allIds, 3, "Should have 3 $id entries")

	// Root: https://example.com/root.json
	assert.NotNil(t, allIds["https://example.com/root.json"])

	// Level 1: level1/ relative to https://example.com/root.json = https://example.com/level1/
	assert.NotNil(t, allIds["https://example.com/level1/"])

	// Level 2: level2.json relative to https://example.com/level1/ = https://example.com/level1/level2.json
	level2Entry := allIds["https://example.com/level1/level2.json"]
	assert.NotNil(t, level2Entry, "Level 2 should resolve against level 1, not root")
	assert.Equal(t, "https://example.com/level1/level2.json", level2Entry.ResolvedUri)
}

// Test $id lookup through rolodex global registry
func TestResolveRefViaSchemaId_WithRolodex(t *testing.T) {
	spec := `openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Pet:
      $id: "https://example.com/schemas/pet.json"
      type: object
      properties:
        name:
          type: string
`

	var rootNode yaml.Node
	err := yaml.Unmarshal([]byte(spec), &rootNode)
	assert.NoError(t, err)

	config := CreateClosedAPIIndexConfig()
	rolodex := NewRolodex(config)
	config.SpecAbsolutePath = "https://example.com/openapi.yaml"

	index := NewSpecIndexWithConfig(&rootNode, config)
	assert.NotNil(t, index)

	// Add index to rolodex (triggers RegisterIdsFromIndex)
	rolodex.AddIndex(index)

	// Create a second index that references the first via $id
	spec2 := `openapi: "3.1.0"
info:
  title: Test API 2
  version: 1.0.0
`
	var rootNode2 yaml.Node
	err = yaml.Unmarshal([]byte(spec2), &rootNode2)
	assert.NoError(t, err)

	config2 := CreateClosedAPIIndexConfig()
	config2.SpecAbsolutePath = "https://example.com/api2.yaml"
	index2 := NewSpecIndexWithConfig(&rootNode2, config2)

	// Set rolodex on the second index
	index2.rolodex = rolodex

	// ResolveRefViaSchemaId should find the schema via rolodex global registry
	resolved := index2.ResolveRefViaSchemaId("https://example.com/schemas/pet.json")
	assert.NotNil(t, resolved)
	assert.Equal(t, "https://example.com/schemas/pet.json", resolved.FullDefinition)
}

// Test that findSchemaIdInNode returns empty for non-mapping nodes
func TestFindSchemaIdInNode_NonMapping(t *testing.T) {
	// Sequence node
	seqNode := &yaml.Node{Kind: yaml.SequenceNode}
	assert.Equal(t, "", findSchemaIdInNode(seqNode))

	// Nil node
	assert.Equal(t, "", findSchemaIdInNode(nil))
}

// Test that findSchemaIdInNode returns empty when $id is not a string
func TestFindSchemaIdInNode_NonStringId(t *testing.T) {
	yml := `$id: 123`
	var node yaml.Node
	_ = yaml.Unmarshal([]byte(yml), &node)

	assert.Equal(t, "", findSchemaIdInNode(node.Content[0]))
}

// Test error path when $id contains fragment
func TestSchemaId_ExtractionWithFragmentError(t *testing.T) {
	spec := `openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Pet:
      $id: "https://example.com/schemas/pet.json#invalid"
      type: object
`

	var rootNode yaml.Node
	err := yaml.Unmarshal([]byte(spec), &rootNode)
	assert.NoError(t, err)

	config := CreateClosedAPIIndexConfig()
	index := NewSpecIndexWithConfig(&rootNode, config)
	assert.NotNil(t, index)

	// Invalid $id should not be registered
	allIds := index.GetAllSchemaIds()
	assert.Len(t, allIds, 0)

	// Should have an error recorded
	errors := index.GetReferenceIndexErrors()
	assert.True(t, len(errors) > 0, "Should have recorded an error for invalid $id")

	found := false
	for _, e := range errors {
		if e != nil && strings.Contains(e.Error(), "invalid $id") {
			found = true
			break
		}
	}
	assert.True(t, found, "Should find invalid $id error")
}

// Test fragment navigation with DocumentNode wrapper
func TestNavigateToFragment_DocumentNode(t *testing.T) {
	yamlContent := `type: object
properties:
  name:
    type: string
`
	var node yaml.Node
	err := yaml.Unmarshal([]byte(yamlContent), &node)
	assert.NoError(t, err)

	// node is a DocumentNode wrapping the actual content
	assert.Equal(t, yaml.DocumentNode, node.Kind)

	// Navigate should handle DocumentNode
	result := navigateToFragment(&node, "#/type")
	assert.NotNil(t, result)
	assert.Equal(t, "object", result.Value)

	result = navigateToFragment(&node, "#/properties/name/type")
	assert.NotNil(t, result)
	assert.Equal(t, "string", result.Value)
}

// Test fragment navigation with invalid array index format
func TestNavigateToFragment_InvalidArrayIndex(t *testing.T) {
	yamlContent := `items:
  - first
  - second
`
	var node yaml.Node
	err := yaml.Unmarshal([]byte(yamlContent), &node)
	assert.NoError(t, err)

	root := node.Content[0]

	// Non-numeric index
	result := navigateToFragment(root, "#/items/abc")
	assert.Nil(t, result)

	// Negative-like index (actually invalid format)
	result = navigateToFragment(root, "#/items/-1")
	assert.Nil(t, result)
}

// Test ResolveRefViaSchemaId caches results
func TestResolveRefViaSchemaId_Caching(t *testing.T) {
	spec := `openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Pet:
      $id: "https://example.com/schemas/pet.json"
      type: object
`

	var rootNode yaml.Node
	err := yaml.Unmarshal([]byte(spec), &rootNode)
	assert.NoError(t, err)

	config := CreateClosedAPIIndexConfig()
	index := NewSpecIndexWithConfig(&rootNode, config)
	assert.NotNil(t, index)

	// First resolution
	resolved1 := index.ResolveRefViaSchemaId("https://example.com/schemas/pet.json")
	assert.NotNil(t, resolved1)

	// Second resolution should use cache
	resolved2 := index.ResolveRefViaSchemaId("https://example.com/schemas/pet.json")
	assert.NotNil(t, resolved2)

	// Results should be equivalent
	assert.Equal(t, resolved1.FullDefinition, resolved2.FullDefinition)
}

// Test $id extraction uses document base when no scope exists
func TestSchemaId_ExtractionWithDocumentBase(t *testing.T) {
	spec := `openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Pet:
      $id: "schemas/pet.json"
      type: object
`

	var rootNode yaml.Node
	err := yaml.Unmarshal([]byte(spec), &rootNode)
	assert.NoError(t, err)

	config := CreateClosedAPIIndexConfig()
	config.SpecAbsolutePath = "https://example.com/api/openapi.yaml"
	index := NewSpecIndexWithConfig(&rootNode, config)
	assert.NotNil(t, index)

	allIds := index.GetAllSchemaIds()
	assert.Len(t, allIds, 1)

	// Should be resolved relative to document base
	entry := allIds["https://example.com/api/schemas/pet.json"]
	assert.NotNil(t, entry)
	assert.Equal(t, "schemas/pet.json", entry.Id)
	assert.Equal(t, "https://example.com/api/schemas/pet.json", entry.ResolvedUri)
}

// Test that SearchIndexForReferenceByReferenceWithContext uses $id lookup
func TestSearchIndexForReferenceByReferenceWithContext_ViaSchemaId(t *testing.T) {
	spec := `openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Pet:
      $id: "https://example.com/schemas/pet.json"
      type: object
      properties:
        name:
          type: string
    Owner:
      type: object
      properties:
        pet:
          $ref: "https://example.com/schemas/pet.json"
`

	var rootNode yaml.Node
	err := yaml.Unmarshal([]byte(spec), &rootNode)
	assert.NoError(t, err)

	config := CreateClosedAPIIndexConfig()
	index := NewSpecIndexWithConfig(&rootNode, config)
	assert.NotNil(t, index)

	// Search for the reference using the $id
	ref, foundIdx, ctx := index.SearchIndexForReferenceWithContext(context.Background(), "https://example.com/schemas/pet.json")

	assert.NotNil(t, ref, "Should find reference via $id")
	assert.NotNil(t, foundIdx)
	assert.NotNil(t, ctx)
	assert.Equal(t, "https://example.com/schemas/pet.json", ref.FullDefinition)
}

// Test SchemaIdEntry GetKey with empty ResolvedUri falls back to Id
func TestSchemaIdEntry_GetKey_FallbackToId(t *testing.T) {
	entry := &SchemaIdEntry{
		Id:          "schema.json",
		ResolvedUri: "", // Empty
	}

	assert.Equal(t, "schema.json", entry.GetKey())
}

// Test copySchemaIdRegistry with nil registry
func TestCopySchemaIdRegistry_Nil(t *testing.T) {
	result := copySchemaIdRegistry(nil)
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

// Test copySchemaIdRegistry creates independent copy
func TestCopySchemaIdRegistry_IndependentCopy(t *testing.T) {
	original := make(map[string]*SchemaIdEntry)
	original["key1"] = &SchemaIdEntry{Id: "a.json"}

	copied := copySchemaIdRegistry(original)

	// Should be equal initially
	assert.Len(t, copied, 1)
	assert.NotNil(t, copied["key1"])

	// Modify original
	original["key2"] = &SchemaIdEntry{Id: "b.json"}

	// Copy should not be affected
	assert.Len(t, copied, 1)
	_, exists := copied["key2"]
	assert.False(t, exists)
}

// Test $id at document root level (definitionPath = "#")
func TestSchemaId_RootLevelId(t *testing.T) {
	// A schema with $id at the root level (not nested under components/schemas)
	spec := `$id: "https://example.com/root-schema.json"
type: object
properties:
  name:
    type: string
`

	var rootNode yaml.Node
	err := yaml.Unmarshal([]byte(spec), &rootNode)
	assert.NoError(t, err)

	config := CreateClosedAPIIndexConfig()
	config.SpecAbsolutePath = "https://example.com/openapi.yaml"
	index := NewSpecIndexWithConfig(&rootNode, config)
	assert.NotNil(t, index)

	allIds := index.GetAllSchemaIds()
	assert.Len(t, allIds, 1)

	entry := allIds["https://example.com/root-schema.json"]
	assert.NotNil(t, entry)
	assert.Equal(t, "https://example.com/root-schema.json", entry.Id)
	// Root level should have definitionPath of "#"
	assert.Equal(t, "#", entry.DefinitionPath)
}

// Test malformed $id URL that causes ResolveSchemaId to fail
// This tests the fallback path where resolvedNodeId == "" or resolveErr != nil
func TestSchemaId_MalformedUrlFallback(t *testing.T) {
	// A $id with a malformed URL that url.Parse will reject
	// "://missing-scheme" should cause a parse error
	spec := `openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Pet:
      $id: "://missing-scheme"
      type: object
`

	var rootNode yaml.Node
	err := yaml.Unmarshal([]byte(spec), &rootNode)
	assert.NoError(t, err)

	config := CreateClosedAPIIndexConfig()
	config.SpecAbsolutePath = "https://example.com/openapi.yaml"
	index := NewSpecIndexWithConfig(&rootNode, config)
	assert.NotNil(t, index)

	// The $id should still be registered using the original value as fallback
	allIds := index.GetAllSchemaIds()
	assert.Len(t, allIds, 1)

	// Should be registered with original value since resolution failed
	entry := allIds["://missing-scheme"]
	assert.NotNil(t, entry, "Malformed $id should still be registered with original value")
	assert.Equal(t, "://missing-scheme", entry.Id)
	assert.Equal(t, "://missing-scheme", entry.ResolvedUri) // Falls back to original
}

// Test malformed $id URL in nested context (tests scope update fallback)
func TestSchemaId_MalformedUrlInNestedContext(t *testing.T) {
	spec := `openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
components:
  schemas:
    Parent:
      $id: "https://example.com/parent.json"
      type: object
      properties:
        child:
          $id: "://bad-child-url"
          type: string
`

	var rootNode yaml.Node
	err := yaml.Unmarshal([]byte(spec), &rootNode)
	assert.NoError(t, err)

	config := CreateClosedAPIIndexConfig()
	config.SpecAbsolutePath = "https://example.com/openapi.yaml"
	index := NewSpecIndexWithConfig(&rootNode, config)
	assert.NotNil(t, index)

	allIds := index.GetAllSchemaIds()
	assert.Len(t, allIds, 2)

	// Parent should resolve normally
	parentEntry := allIds["https://example.com/parent.json"]
	assert.NotNil(t, parentEntry)

	// Child has malformed URL, should fall back to original value
	childEntry := allIds["://bad-child-url"]
	assert.NotNil(t, childEntry, "Malformed nested $id should still be registered")
	assert.Equal(t, "://bad-child-url", childEntry.Id)
}
