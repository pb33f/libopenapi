// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/datamodel/low"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"go.yaml.in/yaml/v4"
)

const attributionSpecDir = "../../../test_specs/index_attribution"

func buildAttributionDoc(t *testing.T, file string) *Document {
	t.Helper()

	specPath := filepath.Join(attributionSpecDir, file)
	data, err := os.ReadFile(specPath)
	require.NoError(t, err)

	info, err := datamodel.ExtractSpecInfo(data)
	require.NoError(t, err)

	doc, err := CreateDocumentFromConfig(info, &datamodel.DocumentConfiguration{
		BasePath:                            attributionSpecDir,
		AllowFileReferences:                 true,
		AllowRemoteReferences:               false,
		IgnorePolymorphicCircularReferences: true,
		IgnoreArrayCircularReferences:       true,
	})
	require.NoError(t, err)
	return doc
}

func attributionSchema(t *testing.T, doc *Document, name string) *lowbase.Schema {
	t.Helper()

	found := doc.Components.Value.FindSchema(name)
	require.NotNilf(t, found, "component schema %s not found", name)
	schema := found.Value.Schema()
	require.NotNilf(t, schema, "component schema %s did not build", name)
	return schema
}

// assertOwnedBy is the regression invariant. The index attached to a schema must name the file that
// physically holds the schema's nodes, otherwise line and column numbers cannot be trusted to be
// unique when paired with the index.
func assertOwnedBy(t *testing.T, schema *lowbase.Schema, wantFile string) {
	t.Helper()

	require.NotNil(t, schema.GetIndex(), "schema has no index")
	assert.Same(t, schema.Index, schema.GetIndex(), "Index and index must never diverge")
	assert.Truef(t, strings.HasSuffix(schema.GetIndex().GetSpecAbsolutePath(), wantFile),
		"expected schema to be owned by %s, got %s", wantFile, schema.GetIndex().GetSpecAbsolutePath())
}

// assertNodeOriginAgrees is the property doctor depends on: the index and the rolodex agree on which
// file a node came from, so (index, line, column) is a unique identity across a multi file document.
func assertNodeOriginAgrees(t *testing.T, schema *lowbase.Schema, node *yaml.Node) {
	t.Helper()

	if node == nil {
		return
	}
	rolodex := schema.GetIndex().GetRolodex()
	require.NotNil(t, rolodex)

	origin := rolodex.FindNodeOrigin(node)
	require.NotNil(t, origin, "no origin found for node at %d:%d", node.Line, node.Column)
	assert.Equal(t, origin.AbsoluteLocation, schema.GetIndex().GetSpecAbsolutePath(),
		"node origin and schema index disagree on the owning file")
}

func TestSchemaIndexAttribution_ExternalRefAndChildren(t *testing.T) {
	doc := buildAttributionDoc(t, "root.yaml")
	direct := attributionSchema(t, doc, "Direct")

	// the schema reached through the external reference is owned by the file it points at.
	assertOwnedBy(t, direct, "level1.yaml")
	assertNodeOriginAgrees(t, direct, direct.Type.ValueNode)

	// every inline property below the reference inherits the same owning file. this is the
	// primary regression: before the fix these reported the referring file, so two identical
	// external files collided on (index, line, column).
	t.Run("properties", func(t *testing.T) {
		require.NotNil(t, direct.Properties.Value)
		count := 0
		for pair := direct.Properties.Value.First(); pair != nil; pair = pair.Next() {
			property := pair.Value().Value.Schema()
			require.NotNil(t, property)
			count++
			// 'deep' references a third file and is covered by the chain test below.
			if pair.Key().Value == "deep" {
				continue
			}
			assertOwnedBy(t, property, "level1.yaml")
			assertNodeOriginAgrees(t, property, property.Type.ValueNode)
		}
		assert.Equal(t, 3, count)
	})

	// one case per child building call site in schema_build.go, all of which receive the index
	// that Build resolved.
	singles := map[string]*lowbase.Schema{
		"not":                   safeSchema(direct.Not.Value),
		"if":                    safeSchema(direct.If.Value),
		"then":                  safeSchema(direct.Then.Value),
		"else":                  safeSchema(direct.Else.Value),
		"items":                 safeDynamicSchema(direct.Items.Value),
		"contains":              safeSchema(direct.Contains.Value),
		"propertyNames":         safeSchema(direct.PropertyNames.Value),
		"unevaluatedItems":      safeSchema(direct.UnevaluatedItems.Value),
		"unevaluatedProperties": safeDynamicSchema(direct.UnevaluatedProperties.Value),
		"additionalProperties":  safeDynamicSchema(direct.AdditionalProperties.Value),
		"contentSchema":         safeSchema(direct.ContentSchema.Value),
	}
	for label, schema := range singles {
		t.Run(label, func(t *testing.T) {
			require.NotNilf(t, schema, "%s did not build", label)
			assertOwnedBy(t, schema, "level1.yaml")
		})
	}

	lists := map[string][]*lowbase.Schema{
		"allOf":       collectSchemas(direct.AllOf.Value),
		"anyOf":       collectSchemas(direct.AnyOf.Value),
		"oneOf":       collectSchemas(direct.OneOf.Value),
		"prefixItems": collectSchemas(direct.PrefixItems.Value),
	}
	for label, schemas := range lists {
		t.Run(label, func(t *testing.T) {
			require.NotEmptyf(t, schemas, "%s did not build", label)
			for _, schema := range schemas {
				assertOwnedBy(t, schema, "level1.yaml")
			}
		})
	}

	maps := map[string]*lowbase.Schema{
		"patternProperties": firstMapSchema(direct.PatternProperties.Value),
		"dependentSchemas":  firstMapSchema(direct.DependentSchemas.Value),
		"$defs":             firstMapSchema(direct.Defs.Value),
	}
	for label, schema := range maps {
		t.Run(label, func(t *testing.T) {
			require.NotNilf(t, schema, "%s did not build", label)
			assertOwnedBy(t, schema, "level1.yaml")
		})
	}

	t.Run("externalDocs and xml still build", func(t *testing.T) {
		require.NotNil(t, direct.ExternalDocs.Value)
		assert.Equal(t, "https://pb33f.io", direct.ExternalDocs.Value.URL.Value)
		require.NotNil(t, direct.XML.Value)
		assert.Equal(t, "l1", direct.XML.Value.Name.Value)
	})
}

// attributionDeepSchema walks root.yaml -> level1.yaml -> level2.yaml. The hop into level2 is a
// property reference, which resolves through schema_build_helpers rather than through Build, so it
// was already attributed correctly before this fix.
func attributionDeepSchema(t *testing.T, doc *Document) *lowbase.Schema {
	t.Helper()

	direct := attributionSchema(t, doc, "Direct")
	require.NotNil(t, direct.Properties.Value)

	for pair := direct.Properties.Value.First(); pair != nil; pair = pair.Next() {
		if pair.Key().Value != "deep" {
			continue
		}
		deep := pair.Value().Value.Schema()
		require.NotNil(t, deep)
		return deep
	}
	require.FailNow(t, "deep property not found on L1")
	return nil
}

// TestSchemaIndexAttribution_ThreeFileChain is a lock-in rather than a regression test. Property
// level references were always attributed correctly, and this pins that the swap in Build did not
// disturb them while carrying attribution across a third file.
func TestSchemaIndexAttribution_ThreeFileChain(t *testing.T) {
	doc := buildAttributionDoc(t, "root.yaml")
	deep := attributionDeepSchema(t, doc)

	// the schema lands on the file that actually holds it, two files away from the root.
	assertOwnedBy(t, deep, "level2.yaml")

	require.NotNil(t, deep.Properties.Value)
	require.Equal(t, 1, deep.Properties.Value.Len())
	for pair := deep.Properties.Value.First(); pair != nil; pair = pair.Next() {
		property := pair.Value().Value.Schema()
		require.NotNil(t, property)
		assertOwnedBy(t, property, "level2.yaml")
	}
}

// TestSchemaIndexAttribution_ExclusiveMinimumSurvivesSwap guards the silent corruption that a naive
// index swap introduces. level2.yaml has no openapi key, so its own SpecInfo reports version 0. If
// the version were read from the swapped index, exclusiveMinimum would be parsed with 3.0 boolean
// semantics and the numeric 3 would be lost, and no other test in this repo would notice.
func TestSchemaIndexAttribution_ExclusiveMinimumSurvivesSwap(t *testing.T) {
	doc := buildAttributionDoc(t, "root.yaml")
	deep := attributionDeepSchema(t, doc)

	assertOwnedBy(t, deep, "level2.yaml")

	require.NotNil(t, deep.GetIndex().GetConfig())
	require.NotNil(t, deep.GetIndex().GetConfig().SpecInfo)
	assert.Zero(t, deep.GetIndex().GetConfig().SpecInfo.VersionNumeric,
		"fixture must keep level2.yaml versionless for this test to mean anything")

	require.NotNil(t, deep.ExclusiveMinimum.Value)
	assert.Equal(t, 1, deep.ExclusiveMinimum.Value.N, "expected 3.1 numeric form")
	assert.Equal(t, float64(3), deep.ExclusiveMinimum.Value.B)
	assert.False(t, deep.ExclusiveMinimum.Value.A)
}

func TestSchemaIndexAttribution_ExclusiveMinimumBooleanUnder30(t *testing.T) {
	doc := buildAttributionDoc(t, "root_30.yaml")
	ext := attributionSchema(t, doc, "Ext")

	assertOwnedBy(t, ext, "frag30.yaml")

	require.NotNil(t, ext.ExclusiveMinimum.Value)
	assert.Equal(t, 0, ext.ExclusiveMinimum.Value.N, "expected 3.0 boolean form")
	assert.True(t, ext.ExclusiveMinimum.Value.A)
}

// TestSchemaIndexAttribution_LocalAndInlineUnchanged locks in the blast radius. Schemas that do not
// travel through an external reference must be attributed exactly as they were before.
func TestSchemaIndexAttribution_LocalAndInlineUnchanged(t *testing.T) {
	doc := buildAttributionDoc(t, "root.yaml")

	local := attributionSchema(t, doc, "Local")
	assertOwnedBy(t, local, "root.yaml")

	plain := attributionSchema(t, doc, "Plain")
	assertOwnedBy(t, plain, "root.yaml")

	require.NotNil(t, plain.Properties.Value)
	require.Equal(t, 1, plain.Properties.Value.Len())
	for pair := plain.Properties.Value.First(); pair != nil; pair = pair.Next() {
		property := pair.Value().Value.Schema()
		require.NotNil(t, property)
		assertOwnedBy(t, property, "root.yaml")
	}
}

// TestSchemaIndexAttribution_ProxyAndRootNodeUnchanged locks in the two things the fix deliberately
// leaves alone, because consumers key caches and circular detection on them.
func TestSchemaIndexAttribution_ProxyAndRootNodeUnchanged(t *testing.T) {
	doc := buildAttributionDoc(t, "root.yaml")

	found := doc.Components.Value.FindSchema("Direct")
	require.NotNil(t, found)
	proxy := found.Value

	// the proxy owns the $ref node, which lives in the referring file.
	require.NotNil(t, proxy.GetIndex())
	assert.True(t, strings.HasSuffix(proxy.GetIndex().GetSpecAbsolutePath(), "root.yaml"),
		"SchemaProxy.GetIndex must keep naming the referring file, got %s",
		proxy.GetIndex().GetSpecAbsolutePath())

	schema := proxy.Schema()
	require.NotNil(t, schema)

	// RootNode stays the authored $ref node, it does not follow the index.
	require.NotNil(t, schema.RootNode)
	isRef := false
	for i := 0; i < len(schema.RootNode.Content)-1; i += 2 {
		if schema.RootNode.Content[i].Value == "$ref" {
			isRef = true
		}
	}
	assert.True(t, isRef, "RootNode must remain the authored $ref node")

	// the built schema is where the content's owning file is available.
	require.NotNil(t, schema.GetIndex())
	assert.True(t, strings.HasSuffix(schema.GetIndex().GetSpecAbsolutePath(), "level1.yaml"),
		"the built schema must name the referenced file, got %s",
		schema.GetIndex().GetSpecAbsolutePath())
}

func TestSchemaIndexAttribution_ExternalPathItem(t *testing.T) {
	doc := buildAttributionDoc(t, "root.yaml")

	var external *PathItem
	for pair := doc.Paths.Value.PathItems.First(); pair != nil; pair = pair.Next() {
		if pair.Key().Value == "/external" {
			external = pair.Value().Value
		}
	}
	require.NotNil(t, external, "external path item not found")

	require.NotNil(t, external.GetIndex())
	assert.True(t, strings.HasSuffix(external.GetIndex().GetSpecAbsolutePath(), "level1.yaml"),
		"external path item must be owned by level1.yaml, got %s",
		external.GetIndex().GetSpecAbsolutePath())
}

func TestSchemaIndexAttribution_CircularExternalRefs(t *testing.T) {
	doc := buildAttributionDoc(t, "circ_root.yaml")
	start := attributionSchema(t, doc, "Start")

	// a loop that crosses files still builds, and the schema is attributed to the file holding it.
	assertOwnedBy(t, start, "circ_a.yaml")
	require.NotNil(t, start.Properties.Value)
	require.Equal(t, 1, start.Properties.Value.Len())

	// the loop is registered on the rolodex, never on the external file this schema is now
	// attributed to. a consumer reading circular references off the schema's own index finds
	// nothing, which is why the guards have to combine the rolodex sets with the root index rather
	// than trusting whatever index the schema arrived with.
	rolodex := start.GetIndex().GetRolodex()
	require.NotNil(t, rolodex)
	require.NotSame(t, rolodex.GetRootIndex(), start.GetIndex(),
		"fixture must attribute the schema away from the root for this to prove anything")
	require.Empty(t, start.GetIndex().GetCircularReferences(),
		"the owning file's index is not where loops live")

	discovered := len(rolodex.GetRootIndex().GetCircularReferences()) +
		len(rolodex.GetSafeCircularReferences()) +
		len(rolodex.GetIgnoredCircularReferences())
	require.Positive(t, discovered, "the cross-file loop must be discoverable through the rolodex")
}

// TestResolveDocumentVersion covers the helper directly, including the paths that no fixture can
// reach through a real document build.
func TestResolveDocumentVersion(t *testing.T) {
	t.Run("nil index", func(t *testing.T) {
		var idx *index.SpecIndex
		version, ok := idx.ResolveDocumentVersion()
		assert.Zero(t, version)
		assert.False(t, ok)
	})

	t.Run("no spec info anywhere", func(t *testing.T) {
		idx := index.NewSpecIndexWithConfig(&yaml.Node{Kind: yaml.MappingNode},
			index.CreateOpenAPIIndexConfig())
		version, ok := idx.ResolveDocumentVersion()
		assert.Zero(t, version)
		assert.False(t, ok)
	})

	t.Run("own spec info when no rolodex root", func(t *testing.T) {
		config := index.CreateOpenAPIIndexConfig()
		config.SpecInfo = &datamodel.SpecInfo{VersionNumeric: 3.1}
		idx := index.NewSpecIndexWithConfig(&yaml.Node{Kind: yaml.MappingNode}, config)
		version, ok := idx.ResolveDocumentVersion()
		assert.Equal(t, float32(3.1), version)
		assert.True(t, ok)
	})

	t.Run("rolodex root wins over versionless external file", func(t *testing.T) {
		doc := buildAttributionDoc(t, "root.yaml")
		deep := attributionDeepSchema(t, doc)

		// the external file itself reports nothing useful.
		require.NotNil(t, deep.GetIndex().GetConfig())
		require.NotNil(t, deep.GetIndex().GetConfig().SpecInfo)
		assert.Zero(t, deep.GetIndex().GetConfig().SpecInfo.VersionNumeric)

		// but the document version resolves through the rolodex root.
		version, ok := deep.GetIndex().ResolveDocumentVersion()
		assert.True(t, ok)
		assert.Equal(t, float32(3.1), version)
	})
}

// TestExclusiveWithoutDocumentVersion covers the branch taken when no SpecInfo is reachable, which
// a caller hand-building an index will hit. An absent keyword must stay absent rather than become a
// bound of zero the author never wrote.
func TestExclusiveWithoutDocumentVersion(t *testing.T) {
	cases := []struct {
		name      string
		value     string
		present   bool
		numeric   bool
		boolValue bool
		number    float64
	}{
		{name: "integer reads as a 3.1 bound", value: "3", present: true, numeric: true, number: 3},
		{name: "float reads as a 3.1 bound", value: "3.5", present: true, numeric: true, number: 3.5},
		{name: "boolean reads as the 3.0 form", value: "true", present: true, boolValue: true},
		{name: "string is not a bound", value: "'abc'"},
		{name: "null is not a bound", value: "null"},
		{name: "sequence is not a bound", value: "[1, 2]"},
	}

	// both keywords share one implementation, so both are driven to keep the wiring honest.
	keywords := []struct {
		name  string
		read  func(*lowbase.Schema) *lowbase.SchemaDynamicValue[bool, float64]
		build func(string) string
	}{
		{
			name:  "exclusiveMinimum",
			read:  func(s *lowbase.Schema) *lowbase.SchemaDynamicValue[bool, float64] { return s.ExclusiveMinimum.Value },
			build: func(v string) string { return "type: number\nexclusiveMinimum: " + v },
		},
		{
			name:  "exclusiveMaximum",
			read:  func(s *lowbase.Schema) *lowbase.SchemaDynamicValue[bool, float64] { return s.ExclusiveMaximum.Value },
			build: func(v string) string { return "type: number\nexclusiveMaximum: " + v },
		},
	}

	for _, keyword := range keywords {
		for _, testCase := range cases {
			t.Run(keyword.name+"/"+testCase.name, func(t *testing.T) {
				var root yaml.Node
				require.NoError(t, yaml.Unmarshal([]byte(keyword.build(testCase.value)), &root))

				schema := new(lowbase.Schema)
				// a nil index means no SpecInfo is reachable, so the version is unknown.
				require.NoError(t, schema.Build(t.Context(), root.Content[0], nil))

				built := keyword.read(schema)
				if !testCase.present {
					require.Nil(t, built, "an unreadable value must not become a bound")
					return
				}

				require.NotNil(t, built)
				if testCase.numeric {
					require.Equal(t, 1, built.N)
					require.Equal(t, testCase.number, built.B)
					return
				}
				require.Equal(t, 0, built.N)
				require.Equal(t, testCase.boolValue, built.A)
			})
		}
	}
}

// TestExclusiveWithNilSpecInfo covers an index that exists but carries no SpecInfo. Reading the
// version straight off such an index panics, and the reference swap made that reachable by handing
// schema building an external file's index.
func TestExclusiveWithNilSpecInfo(t *testing.T) {
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("type: number\nexclusiveMinimum: 3"), &root))

	config := index.CreateOpenAPIIndexConfig()
	require.Nil(t, config.SpecInfo, "fixture assumes a config with no SpecInfo")
	idx := index.NewSpecIndexWithConfig(root.Content[0], config)

	schema := new(lowbase.Schema)
	require.NotPanics(t, func() {
		require.NoError(t, schema.Build(t.Context(), root.Content[0], idx))
	})

	require.NotNil(t, schema.ExclusiveMinimum.Value)
	require.Equal(t, 1, schema.ExclusiveMinimum.Value.N)
	require.Equal(t, float64(3), schema.ExclusiveMinimum.Value.B)
}

func TestExclusiveWithZeroSpecVersionInfersNumericShape(t *testing.T) {
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("type: number\nexclusiveMinimum: 3"), &root))

	config := index.CreateOpenAPIIndexConfig()
	config.SpecInfo = &datamodel.SpecInfo{VersionNumeric: 0}
	idx := index.NewSpecIndexWithConfig(root.Content[0], config)

	schema := new(lowbase.Schema)
	require.NoError(t, schema.Build(t.Context(), root.Content[0], idx))
	require.NotNil(t, schema.ExclusiveMinimum.Value)
	assert.Equal(t, 1, schema.ExclusiveMinimum.Value.N)
	assert.Equal(t, float64(3), schema.ExclusiveMinimum.Value.B)
}

func safeSchema(proxy *lowbase.SchemaProxy) *lowbase.Schema {
	if proxy == nil {
		return nil
	}
	return proxy.Schema()
}

func safeDynamicSchema[T any](dynamic *lowbase.SchemaDynamicValue[*lowbase.SchemaProxy, T]) *lowbase.Schema {
	if dynamic == nil {
		return nil
	}
	return safeSchema(dynamic.A)
}

func collectSchemas(refs []low.ValueReference[*lowbase.SchemaProxy]) []*lowbase.Schema {
	schemas := make([]*lowbase.Schema, 0, len(refs))
	for _, ref := range refs {
		if schema := safeSchema(ref.Value); schema != nil {
			schemas = append(schemas, schema)
		}
	}
	return schemas
}

func firstMapSchema(
	entries *orderedmap.Map[low.KeyReference[string], low.ValueReference[*lowbase.SchemaProxy]],
) *lowbase.Schema {
	if entries == nil {
		return nil
	}
	if pair := entries.First(); pair != nil {
		return safeSchema(pair.Value().Value)
	}
	return nil
}

// A referenced file that declares its own version describes how its own keywords are written,
// whatever version referenced it. exclusiveMinimum is the keyword that notices: 3.0 reads it as
// a boolean modifier, 3.1 reads it as the bound itself.
func TestResolveDocumentVersionPrefersFilesOwnVersion(t *testing.T) {
	doc := buildAttributionDoc(t, "root_30_ext31.yaml")
	schema := attributionSchema(t, doc, "Ext")

	require.NotNil(t, schema.ExclusiveMinimum.Value)
	assert.Equal(t, 1, schema.ExclusiveMinimum.Value.N,
		"a 3.1 file's numeric bound must be read as a number, not coerced through the 3.0 boolean form")
	assert.Equal(t, float64(5), schema.ExclusiveMinimum.Value.B)
}

// A bare fragment declares no version of its own, so it inherits the document being built.
// This is the case the rolodex-root fallback exists for and must keep working.
func TestResolveDocumentVersionFragmentInheritsRoot(t *testing.T) {
	doc := buildAttributionDoc(t, "root_30.yaml")
	schema := attributionSchema(t, doc, "Ext")

	require.NotNil(t, schema.ExclusiveMinimum.Value)
	assert.Equal(t, 0, schema.ExclusiveMinimum.Value.N,
		"a versionless fragment under a 3.0 root keeps the 3.0 boolean form")
	assert.True(t, schema.ExclusiveMinimum.Value.A)
}
