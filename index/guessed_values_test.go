// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package index

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/internal/specpath"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

const guessedValuePetSchema = `type: object
properties:
  name:
    type: string
`

// indexGuessedValueFiles indexes main.yaml with a rolodex over the files, either loading every
// file up front or opening each when a reference needs it.
func indexGuessedValueFiles(t *testing.T, files map[string]string, upFront bool) (*Rolodex, string) {
	t.Helper()
	tmp := t.TempDir()
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(tmp, name), []byte(content), 0o644))
	}
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(files["main.yaml"]), &root))

	cfg := CreateOpenAPIIndexConfig()
	cfg.BasePath = tmp
	cfg.SpecAbsolutePath = filepath.Join(tmp, "main.yaml")
	fsCfg := &LocalFSConfig{BaseDirectory: tmp, IndexConfig: cfg}
	if upFront {
		fsCfg.DirFS = os.DirFS(tmp)
	}
	localFS, err := NewLocalFSWithConfig(fsCfg)
	require.NoError(t, err)

	rolodex := NewRolodex(cfg)
	rolodex.AddLocalFS(tmp, localFS)
	rolodex.SetRootNode(&root)
	require.NoError(t, rolodex.IndexTheRolodex(context.Background()))
	assert.False(t, rolodex.settlingValues.Load())
	return rolodex, tmp
}

func guessedValueIndex(t *testing.T, rolodex *Rolodex, path string) *SpecIndex {
	t.Helper()
	for _, idx := range rolodex.GetIndexes() {
		if idx.GetSpecAbsolutePath() == path {
			return idx
		}
	}
	require.Failf(t, "no index", "%s is not indexed", path)
	return nil
}

func rawRefFiles(idx *SpecIndex) []string {
	var files []string
	for _, ref := range idx.GetRawReferencesSequenced() {
		file, _, _ := strings.Cut(ref.FullDefinition, "#")
		files = append(files, filepath.Base(file))
	}
	return files
}

func TestRolodex_GuessedValues_CatalogEntrySelectedAsSchema(t *testing.T) {
	for name, upFront := range map[string]bool{"up front": true, "on demand": false} {
		t.Run(name, func(t *testing.T) {
			rolodex, tmp := indexGuessedValueFiles(t, map[string]string{
				"main.yaml": `openapi: 3.1.0
info: {title: Catalog, version: "1"}
paths: {}
components:
  schemas:
    Owner:
      $ref: 'catalog.yaml#/default'
`,
				// The entry named properties makes the catalog look like a schema, and its
				// default like instance data, but the reference selects default as a schema.
				"catalog.yaml": `properties:
  type: string
default:
  type: object
  allOf:
    - $ref: 'base.yaml'
  properties:
    pet:
      $ref: 'pet.yaml'
`,
				"pet.yaml":  guessedValuePetSchema,
				"base.yaml": guessedValuePetSchema,
			}, upFront)

			catalog := guessedValueIndex(t, rolodex, filepath.Join(tmp, "catalog.yaml"))
			assert.ElementsMatch(t, []string{"base.yaml", "pet.yaml"}, rawRefFiles(catalog))
			assert.NotNil(t, catalog.GetMappedReferences()[filepath.Join(tmp, "pet.yaml")])
			assert.NotNil(t, catalog.GetMappedReferences()[filepath.Join(tmp, "base.yaml")])
			assert.NotEmpty(t, catalog.GetPolyAllOfReferences())
			guessedValueIndex(t, rolodex, filepath.Join(tmp, "pet.yaml"))
			guessedValueIndex(t, rolodex, filepath.Join(tmp, "base.yaml"))
			assert.Empty(t, catalog.GetReferenceIndexErrors())
		})
	}
}

func TestRolodex_GuessedValues_ConstOnlyFileSelectedAsSchema(t *testing.T) {
	for name, upFront := range map[string]bool{"up front": true, "on demand": false} {
		t.Run(name, func(t *testing.T) {
			rolodex, tmp := indexGuessedValueFiles(t, map[string]string{
				"main.yaml": `openapi: 3.1.0
info: {title: Const, version: "1"}
paths: {}
components:
  schemas:
    Owner:
      $ref: 'owner.yaml'
`,
				"owner.yaml": "const: {$ref: 'missing.yaml'}\n",
			}, upFront)

			owner := guessedValueIndex(t, rolodex, filepath.Join(tmp, "owner.yaml"))
			assert.Empty(t, owner.GetRawReferencesSequenced(), "the const of a schema is data")
			assert.Empty(t, owner.GetReferenceIndexErrors())
			assert.Empty(t, rolodex.GetCaughtErrors())
		})
	}
}

func TestRolodex_GuessedValues_ChainedConstOnlyFileSelectedAsSchema(t *testing.T) {
	cases := map[string]map[string]string{
		// A file holding only $ref passes on the context it is referenced from.
		"wrapper file": {
			"main.yaml": `openapi: 3.1.0
info: {title: Chain, version: "1"}
paths: {}
components:
  schemas:
    Owner:
      $ref: 'wrapper.yaml'
`,
			"wrapper.yaml": "$ref: 'owner.yaml'\n",
			"owner.yaml":   "const: {$ref: 'missing.yaml'}\n",
		},
		// So does an entry of a catalog, reached through the path of its entry.
		"catalog wrapper": {
			"main.yaml": `openapi: 3.1.0
info: {title: Chain, version: "1"}
paths: {}
components:
  schemas:
    Owner:
      $ref: 'catalog.yaml#/entries/owner'
`,
			"catalog.yaml": `entries:
  owner:
    $ref: 'owner.yaml'
`,
			"owner.yaml": "const: {$ref: 'missing.yaml'}\n",
		},
		// And a chain of entries in one catalog, however many it takes.
		"catalog chain": {
			"main.yaml": `openapi: 3.1.0
info: {title: Chain, version: "1"}
paths: {}
components:
  schemas:
    Owner:
      $ref: 'catalog.yaml#/A'
`,
			"catalog.yaml": "A:\n  $ref: '#/B'\nB:\n  $ref: '#/C'\nC:\n  $ref: 'owner.yaml'\n",
			"owner.yaml":   "const: {$ref: 'missing.yaml'}\n",
		},
		// A file nothing references, loaded with the rest, does not change the context.
		"unreferenced referrer": {
			"main.yaml": `openapi: 3.1.0
info: {title: Chain, version: "1"}
paths: {}
components:
  schemas:
    Owner:
      $ref: 'owner.yaml'
`,
			"unreferenced.yaml": "foo:\n  $ref: 'owner.yaml'\n",
			"owner.yaml":        "const: {$ref: 'missing.yaml'}\n",
		},
	}
	for name, files := range cases {
		for mode, upFront := range map[string]bool{"up front": true, "on demand": false} {
			t.Run(name+"/"+mode, func(t *testing.T) {
				rolodex, tmp := indexGuessedValueFiles(t, files, upFront)

				owner := guessedValueIndex(t, rolodex, filepath.Join(tmp, "owner.yaml"))
				assert.Empty(t, owner.GetRawReferencesSequenced(), "the const of a schema is data")
				assert.Empty(t, owner.GetReferenceIndexErrors())
				assert.Empty(t, rolodex.GetCaughtErrors())
			})
		}
	}
}

func TestRolodex_GuessedValues_ExampleSelectsDefault(t *testing.T) {
	cases := map[string]string{
		"beside a schema": `
              schema:
                $ref: 'owner.yaml'`,
		"alone": "",
	}
	for name, schema := range cases {
		for mode, upFront := range map[string]bool{"up front": true, "on demand": false} {
			t.Run(name+"/"+mode, func(t *testing.T) {
				rolodex, tmp := indexGuessedValueFiles(t, map[string]string{
					"main.yaml": `openapi: 3.1.0
info: {title: Example, version: "1"}
paths:
  /a:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:` + schema + `
              example:
                $ref: 'owner.yaml#/default'
`,
					"owner.yaml": "type: object\ndefault:\n  owner:\n    $ref: 'missing.yaml'\n",
				}, upFront)

				owner := guessedValueIndex(t, rolodex, filepath.Join(tmp, "owner.yaml"))
				assert.Empty(t, owner.GetRawReferencesSequenced(), "an example selecting the default keeps it data")
				assert.Empty(t, owner.GetReferenceIndexErrors())
				assert.Empty(t, rolodex.GetCaughtErrors())
			})
		}
	}
}

func TestRolodex_GuessedValues_UnselectedFilesKeepTheGuess(t *testing.T) {
	rolodex, tmp := indexGuessedValueFiles(t, map[string]string{
		"main.yaml": `openapi: 3.1.0
info: {title: Unselected, version: "1"}
paths: {}
`,
		"schema.yaml": `type: object
default:
  $ref: 'pet.yaml'
`,
		"catalog.yaml": `default:
  type: object
  properties:
    pet:
      $ref: 'pet.yaml'
`,
		"pet.yaml": guessedValuePetSchema,
	}, true)

	assert.Empty(t, guessedValueIndex(t, rolodex, filepath.Join(tmp, "schema.yaml")).GetRawReferencesSequenced(),
		"the default of a schema is data")
	catalog := guessedValueIndex(t, rolodex, filepath.Join(tmp, "catalog.yaml"))
	assert.Equal(t, []string{"pet.yaml"}, rawRefFiles(catalog), "a mapping with no schema keyword is not a schema")
	assert.NotNil(t, catalog.GetMappedReferences()[filepath.Join(tmp, "pet.yaml")])
}

func TestRolodex_GuessedValues_NestedValuesSettle(t *testing.T) {
	rolodex, tmp := indexGuessedValueFiles(t, map[string]string{
		"main.yaml": `openapi: 3.1.0
info: {title: Nested, version: "1"}
paths: {}
components:
  schemas:
    Owner:
      $ref: 'catalog.yaml#/default/properties/inner'
`,
		// Selecting inner places default/properties/inner at a schema, which makes the outer
		// default structure, and the inner const data.
		"catalog.yaml": `properties:
  type: string
default:
  properties:
    inner:
      const:
        $ref: 'missing.yaml'
    pet:
      $ref: 'pet.yaml'
`,
		"pet.yaml": guessedValuePetSchema,
	}, false)

	catalog := guessedValueIndex(t, rolodex, filepath.Join(tmp, "catalog.yaml"))
	assert.Equal(t, []string{"pet.yaml"}, rawRefFiles(catalog))
	assert.Empty(t, catalog.GetReferenceIndexErrors())
}

func TestRolodex_GuessedValues_FailedLookupSelectsNothing(t *testing.T) {
	rolodex, tmp := indexGuessedValueFiles(t, map[string]string{
		"main.yaml": `openapi: 3.1.0
info: {title: Failed, version: "1"}
paths: {}
`,
		// The entries default is structure and refers to a file that does not exist, while the
		// default of the string schema stays data and keeps the settle going for another pass.
		"catalog.yaml": `variants:
  oneOf:
    - $ref: 'pet.yaml'
entries:
  default:
    $ref: 'missing.yaml'
name:
  type: string
  default:
    $ref: 'gone.yaml'
`,
		"pet.yaml": guessedValuePetSchema,
	}, true)

	catalog := guessedValueIndex(t, rolodex, filepath.Join(tmp, "catalog.yaml"))
	assert.Equal(t, []string{"pet.yaml", "missing.yaml"}, rawRefFiles(catalog))
	assert.Nil(t, catalog.GetMappedReferences()[filepath.Join(tmp, "missing.yaml")])
	assert.NotEmpty(t, catalog.GetReferenceIndexErrors())
}

func TestRolodex_GuessedValues_MergedValueIsDataOnlyWhereverItIs(t *testing.T) {
	rolodex, tmp := indexGuessedValueFiles(t, map[string]string{
		"main.yaml": `openapi: 3.1.0
info: {title: Merged, version: "1"}
paths: {}
`,
		// The default is structure under one, and data under two, which merges it beside type.
		"catalog.yaml": `one: &one
  default:
    $ref: 'pet.yaml'
two:
  <<: *one
  type: object
`,
		"pet.yaml": guessedValuePetSchema,
	}, true)

	catalog := guessedValueIndex(t, rolodex, filepath.Join(tmp, "catalog.yaml"))
	require.Equal(t, []string{"pet.yaml"}, rawRefFiles(catalog))
	require.Len(t, catalog.guessedValues, 2)
	shared := catalog.guessedValues[0].node
	require.Same(t, shared, catalog.guessedValues[1].node, "both mappings hold the same default")
	literal, settled := rolodex.settledValue(shared)
	assert.True(t, settled)
	assert.False(t, literal, "the index holds the reference in it")
}

func TestRolodex_GuessedValues_SchemaIdRegisteredGlobally(t *testing.T) {
	for name, upFront := range map[string]bool{"up front": true, "on demand": false} {
		t.Run(name, func(t *testing.T) {
			rolodex, _ := indexGuessedValueFiles(t, map[string]string{
				"main.yaml": `openapi: 3.1.0
info: {title: Ids, version: "1"}
paths:
  /x:
    get:
      responses:
        default:
          $ref: 'responses.yaml#/default'
`,
				// default is a response here, so the $id in its schema is one to register.
				"responses.yaml": `default:
  description: error
  content:
    application/json:
      schema:
        $id: 'https://example.com/schemas/err'
        type: object
`,
			}, upFront)

			entry := rolodex.LookupSchemaById("https://example.com/schemas/err")
			require.NotNil(t, entry)
			assert.Equal(t, "object", entry.SchemaNode.Content[3].Value)
		})
	}
}

func TestValueContext_Literal(t *testing.T) {
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(`entry:
  default:
    inner: {type: string}
`), &root))
	entry := root.Content[0].Content[1]
	value := entry.Content[1]
	inner := value.Content[1]
	v := &guessedValue{path: []string{"entry", "default"}, node: value, holder: entry, view: entry, literal: true}
	literal := func(selections ...specpath.Selection) bool {
		return newValueContext(specpath.NewTree(&root), selections).literal(v)
	}

	var none *valueContext
	assert.True(t, none.literal(v), "with no context the guess stands")
	assert.True(t, literal(), "with no selection the guess stands")
	assert.True(t, literal(specpath.Selection{Node: &yaml.Node{Kind: yaml.MappingNode}}), "a selection elsewhere does not cover the value")
	assert.False(t, literal(specpath.Selection{Node: inner}), "a selection inside the value makes it structure")
	assert.False(t, literal(specpath.Selection{Node: value}), "a selection of the value makes it structure")
	assert.True(t, literal(specpath.Selection{Node: value, Path: []string{"components", "schemas", "S", "example"}}),
		"an example selecting the value says nothing of its structure")
	assert.False(t, literal(specpath.Selection{Node: entry, Path: []string{"components", "parameters", "P"}}))
	assert.True(t, literal(specpath.Selection{Node: entry, Path: []string{"components", "schemas", "S"}}))
	assert.True(t, literal(specpath.Selection{Node: root.Content[0], Path: []string{"components", "schemas"}}),
		"the keys between a selection and the holder extend the path")
	assert.False(t, literal(
		specpath.Selection{Node: entry, Path: []string{"components", "schemas", "S"}},
		specpath.Selection{Node: entry, Path: []string{"components", "parameters", "P"}},
	), "the value is data only when every selection says so")
}

func BenchmarkRolodex_GuessedValues_Catalog(b *testing.B) {
	var catalog, main strings.Builder
	main.WriteString("openapi: 3.1.0\ninfo: {title: Bench, version: \"1\"}\npaths: {}\ncomponents:\n  schemas:\n")
	for i := range 2000 {
		fmt.Fprintf(&catalog, "E%d:\n  type: string\n  enum: [a, b]\n  default: a\n  x-meta: {default: {k: v}}\n", i)
		fmt.Fprintf(&main, "    S%d:\n      $ref: 'catalog.yaml#/E%d'\n", i, i)
	}
	tmp := b.TempDir()
	require.NoError(b, os.WriteFile(filepath.Join(tmp, "catalog.yaml"), []byte(catalog.String()), 0o644))
	b.ResetTimer()
	for b.Loop() {
		var root yaml.Node
		require.NoError(b, yaml.Unmarshal([]byte(main.String()), &root))
		cfg := CreateOpenAPIIndexConfig()
		cfg.BasePath = tmp
		cfg.SpecAbsolutePath = filepath.Join(tmp, "main.yaml")
		localFS, err := NewLocalFSWithConfig(&LocalFSConfig{BaseDirectory: tmp, IndexConfig: cfg})
		require.NoError(b, err)
		rolodex := NewRolodex(cfg)
		rolodex.AddLocalFS(tmp, localFS)
		rolodex.SetRootNode(&root)
		require.NoError(b, rolodex.IndexTheRolodex(context.Background()))
	}
}
