package bundler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	highv3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	v3low "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestHandleFileImport_StripsFragment(t *testing.T) {
	pr := &processRef{
		ref:    &index.Reference{FullDefinition: "/tmp/schemas/Cat.yaml#Cat"},
		seqRef: &index.Reference{},
	}

	components := orderedmap.New[string, *highbase.SchemaProxy]()
	location := handleFileImport(pr, v3low.SchemasLabel, "__", components)

	assert.Equal(t, []string{v3low.ComponentsLabel, v3low.SchemasLabel, "Cat"}, location)
	assert.Equal(t, "Cat", pr.name)
	assert.Equal(t, "Cat", pr.ref.Name)
	assert.Equal(t, "Cat", pr.seqRef.Name)
}

func TestComposeReferenceAs_MediaType(t *testing.T) {
	components := &highv3.Components{
		MediaTypes: orderedmap.New[string, *highv3.MediaType](),
	}
	idx := newVersionedIndex(3.2)
	cf := &handleIndexConfig{
		rootIdx: idx,
		compositionConfig: &BundleCompositionConfig{
			Delimiter: "__",
		},
	}
	pr := newProcessRefForTest(t, "media", "schema:\n  type: object")

	handled, err := composeReferenceAs(v3low.MediaTypesLabel, "json", components, pr, idx, cf)
	require.NoError(t, err)
	assert.True(t, handled)
	assert.Equal(t, []string{v3low.ComponentsLabel, v3low.MediaTypesLabel, "json"}, pr.location)
	assert.NotNil(t, components.MediaTypes.GetOrZero("json"))
}

func TestComposeReferenceAs_UnsupportedMediaTypeComponentInlines(t *testing.T) {
	components := &highv3.Components{
		MediaTypes: orderedmap.New[string, *highv3.MediaType](),
	}
	idx := newVersionedIndex(3.1)
	cf := &handleIndexConfig{
		rootIdx: idx,
		compositionConfig: &BundleCompositionConfig{
			Delimiter: "__",
		},
	}
	pr := newProcessRefForTest(t, "media", "schema:\n  type: object")

	handled, err := composeReferenceAs(v3low.MediaTypesLabel, "json", components, pr, idx, cf)
	require.NoError(t, err)
	assert.True(t, handled)
	assert.Nil(t, pr.location)
	assert.Len(t, cf.inlineRequired, 1)
}

func TestComposeReferenceAs_MissingComponentMap(t *testing.T) {
	idx := newVersionedIndex(3.2)
	cf := &handleIndexConfig{
		rootIdx: idx,
		compositionConfig: &BundleCompositionConfig{
			Delimiter: "__",
		},
	}
	pr := newProcessRefForTest(t, "schema", "type: object")

	for _, componentType := range []string{
		v3low.SchemasLabel,
		v3low.ResponsesLabel,
		v3low.ParametersLabel,
		v3low.HeadersLabel,
		v3low.RequestBodiesLabel,
		v3low.ExamplesLabel,
		v3low.LinksLabel,
		v3low.CallbacksLabel,
		v3low.PathItemsLabel,
		v3low.MediaTypesLabel,
		"unknown",
	} {
		t.Run(componentType, func(t *testing.T) {
			handled, err := composeReferenceAs(componentType, "missing", &highv3.Components{}, pr, idx, cf)
			require.NoError(t, err)
			assert.False(t, handled)
		})
	}
}

func TestFileImportLocationForType_MediaType(t *testing.T) {
	components := &highv3.Components{
		MediaTypes: orderedmap.New[string, *highv3.MediaType](),
	}
	cf := &handleIndexConfig{
		rootIdx: newVersionedIndex(3.2),
		compositionConfig: &BundleCompositionConfig{
			Delimiter: "__",
		},
	}
	pr := &processRef{
		ref:    &index.Reference{FullDefinition: "/tmp/application-json.yaml"},
		seqRef: &index.Reference{},
	}

	handled, location := fileImportLocationForType(v3low.MediaTypesLabel, components, pr, cf)
	assert.True(t, handled)
	assert.Equal(t, []string{v3low.ComponentsLabel, v3low.MediaTypesLabel, "application-json"}, location)
}

func TestFileImportLocationForType_MissingComponentMap(t *testing.T) {
	cf := &handleIndexConfig{
		rootIdx: newVersionedIndex(3.2),
		compositionConfig: &BundleCompositionConfig{
			Delimiter: "__",
		},
	}
	pr := &processRef{
		ref:    &index.Reference{FullDefinition: "/tmp/missing.yaml"},
		seqRef: &index.Reference{},
	}

	for _, componentType := range []string{
		v3low.SchemasLabel,
		v3low.ResponsesLabel,
		v3low.ParametersLabel,
		v3low.HeadersLabel,
		v3low.RequestBodiesLabel,
		v3low.ExamplesLabel,
		v3low.LinksLabel,
		v3low.CallbacksLabel,
		v3low.PathItemsLabel,
		v3low.MediaTypesLabel,
		"unknown",
	} {
		t.Run(componentType, func(t *testing.T) {
			handled, location := fileImportLocationForType(componentType, &highv3.Components{}, pr, cf)
			assert.False(t, handled)
			assert.Nil(t, location)
		})
	}
}

func TestFileImportLocationForType_UnsupportedComponentVersionsInline(t *testing.T) {
	tests := []struct {
		name          string
		version       float32
		componentType string
		components    *highv3.Components
	}{
		{
			name:          "path items before openapi 3.1",
			version:       3.0,
			componentType: v3low.PathItemsLabel,
			components: &highv3.Components{
				PathItems: orderedmap.New[string, *highv3.PathItem](),
			},
		},
		{
			name:          "media types before openapi 3.2",
			version:       3.1,
			componentType: v3low.MediaTypesLabel,
			components: &highv3.Components{
				MediaTypes: orderedmap.New[string, *highv3.MediaType](),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cf := &handleIndexConfig{
				rootIdx: newVersionedIndex(tt.version),
				compositionConfig: &BundleCompositionConfig{
					Delimiter: "__",
				},
			}
			pr := &processRef{
				ref:    &index.Reference{FullDefinition: "/tmp/item.yaml"},
				seqRef: &index.Reference{},
			}

			handled, location := fileImportLocationForType(tt.componentType, tt.components, pr, cf)
			assert.True(t, handled)
			assert.Nil(t, location)
			assert.Nil(t, pr.location)
			assert.Len(t, cf.inlineRequired, 1)
		})
	}
}

func TestRootSupportsMediaTypeComponents(t *testing.T) {
	assert.True(t, rootSupportsMediaTypeComponents(nil))
	assert.False(t, rootSupportsMediaTypeComponents(newVersionedIndex(3.1)))
	assert.True(t, rootSupportsMediaTypeComponents(newVersionedIndex(3.2)))
}

func TestProcessRefMapKeys(t *testing.T) {
	target := &index.Reference{FullDefinition: "/tmp/common.yaml#/Thing"}
	responseSource := &index.Reference{
		SourcePath: []string{"paths", "/pets", "get", "responses", "200"},
	}
	schemaSource := &index.Reference{
		FullDefinition: target.FullDefinition,
		SourcePath:     []string{"paths", "/pets", "get", "responses", "200", "content", "application/json", "schema"},
	}
	unknownSource := &index.Reference{
		SourcePath: []string{"x-private", "thing"},
	}
	explicitComponentTarget := &index.Reference{
		FullDefinition: "/tmp/common.yaml#/components/schemas/Pet",
	}

	assert.Empty(t, processRefMapKey(nil, nil))
	assert.Equal(t, target.FullDefinition, processRefMapKey(target, nil))
	assert.Equal(t, target.FullDefinition+contextualRefKeySeparator+v3low.ResponsesLabel, processRefMapKey(target, responseSource))
	assert.Equal(t, target.FullDefinition+contextualRefKeySeparator+v3low.SchemasLabel, processRefMapKey(&index.Reference{}, schemaSource))
	assert.Equal(t, target.FullDefinition, processRefMapKey(target, unknownSource))
	assert.Equal(t, explicitComponentTarget.FullDefinition, processRefMapKey(explicitComponentTarget, responseSource))

	assert.Empty(t, processRefMapKeyForComponent(nil, v3low.SchemasLabel))
	assert.Empty(t, processRefMapKeyForComponent(&index.Reference{}, v3low.SchemasLabel))
	assert.Equal(t, target.FullDefinition, processRefMapKeyForComponent(target, ""))
	assert.Equal(t, explicitComponentTarget.FullDefinition, processRefMapKeyForComponent(explicitComponentTarget, v3low.ResponsesLabel))
	assert.Equal(t, target.FullDefinition+contextualRefKeySeparator+v3low.SchemasLabel, processRefMapKeyForComponent(target, v3low.SchemasLabel))

	assert.Empty(t, contextualProcessRefKey("", responseSource))
	assert.Equal(t, target.FullDefinition, contextualProcessRefKey(target.FullDefinition, nil))
	assert.Equal(t, explicitComponentTarget.FullDefinition, contextualProcessRefKey(explicitComponentTarget.FullDefinition, responseSource))
	assert.Equal(t, target.FullDefinition+contextualRefKeySeparator+v3low.ResponsesLabel, contextualProcessRefKey(target.FullDefinition, responseSource))
	assert.Equal(t, target.FullDefinition, contextualProcessRefKey(target.FullDefinition, unknownSource))
}

func TestProcessedRefFor(t *testing.T) {
	fullDefinition := "/tmp/common.yaml#/Thing"
	responseSource := &index.Reference{
		SourcePath: []string{"paths", "/pets", "get", "responses", "200"},
	}
	contextualKey := fullDefinition + contextualRefKeySeparator + v3low.ResponsesLabel

	assert.Nil(t, processedRefFor(nil, fullDefinition, responseSource))

	processedNodes := orderedmap.New[string, *processRef]()
	fallbackRef := &processRef{name: "fallback"}
	contextualRef := &processRef{name: "contextual"}

	processedNodes.Set(fullDefinition, fallbackRef)
	assert.Same(t, fallbackRef, processedRefFor(processedNodes, fullDefinition, responseSource))

	processedNodes.Set(contextualKey, contextualRef)
	assert.Same(t, contextualRef, processedRefFor(processedNodes, fullDefinition, responseSource))
	assert.Nil(t, processedRefFor(processedNodes, "/tmp/missing.yaml#/Thing", nil))
}

func TestInlineProcessRef(t *testing.T) {
	assert.Nil(t, inlineProcessRef(nil))

	seqNode := testYAMLContentNode(t, "$ref: old.yaml\n")
	assert.Nil(t, inlineProcessRef(&processRef{
		ref:    &index.Reference{FullDefinition: "/tmp/missing.yaml"},
		seqRef: &index.Reference{Node: seqNode},
	}))

	replacement := testYAMLContentNode(t, "description: inlined\n")
	directNode := testYAMLContentNode(t, "$ref: direct.yaml\n")
	directRef := &processRef{
		ref: &index.Reference{
			FullDefinition: "/tmp/direct.yaml",
			Node:           replacement,
		},
		seqRef: &index.Reference{Node: directNode},
	}
	assert.Same(t, replacement, inlineProcessRef(directRef))
	assert.Equal(t, replacement.Content, directNode.Content)

	missingPointerNode := testYAMLContentNode(t, "$ref: pointer.yaml\n")
	assert.Nil(t, inlineProcessRef(&processRef{
		idx:        newVersionedIndex(3.1),
		refPointer: "missing.yaml#/components/schemas/Missing",
		ref:        &index.Reference{FullDefinition: "/tmp/pointer.yaml", Node: replacement},
		seqRef:     &index.Reference{Node: missingPointerNode},
	}))

	tmpDir := t.TempDir()
	rootPath := filepath.Join(tmpDir, "root.yaml")
	rootSource := []byte(`openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths: {}
components:
  schemas:
    Pet:
      type: object
`)
	require.NoError(t, os.WriteFile(rootPath, rootSource, 0644))

	var root yaml.Node
	require.NoError(t, yaml.Unmarshal(rootSource, &root))

	cfg := index.CreateOpenAPIIndexConfig()
	cfg.BasePath = tmpDir
	cfg.SpecAbsolutePath = rootPath
	idx := index.NewSpecIndexWithConfig(&root, cfg)

	pointerNode := testYAMLContentNode(t, "$ref: pointer.yaml\n")
	pointerRef := &processRef{
		idx:        idx,
		refPointer: rootPath + "#/components/schemas/Pet",
		ref:        &index.Reference{FullDefinition: "/tmp/pointer.yaml", Node: replacement},
		seqRef:     &index.Reference{Node: pointerNode},
	}
	inlined := inlineProcessRef(pointerRef)
	require.NotNil(t, inlined)
	assert.Equal(t, inlined.Content, pointerNode.Content)
}

func TestInlineRequiredRefsCopiesMatchingOccurrences(t *testing.T) {
	tmpDir := t.TempDir()
	rootPath := filepath.Join(tmpDir, "root.yaml")
	rootSource := []byte(`openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths:
  /as-response:
    get:
      responses:
        '200':
          $ref: 'target.yaml#/Thing'
  /as-schema:
    get:
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                $ref: 'target.yaml#/Thing'
  /other:
    get:
      responses:
        '200':
          $ref: 'other.yaml#/Thing'
x-extension:
  $ref: 'target.yaml#/Thing'
`)
	require.NoError(t, os.WriteFile(rootPath, rootSource, 0644))

	var root yaml.Node
	require.NoError(t, yaml.Unmarshal(rootSource, &root))

	cfg := index.CreateOpenAPIIndexConfig()
	cfg.BasePath = tmpDir
	cfg.SpecAbsolutePath = rootPath
	idx := index.NewSpecIndexWithConfig(&root, cfg)

	responseRef := findRawRefByComponentType(t, idx, v3low.ResponsesLabel, "target.yaml#/Thing")
	schemaRef := findRawRefByComponentType(t, idx, v3low.SchemasLabel, "target.yaml#/Thing")
	require.Equal(t, responseRef.FullDefinition, schemaRef.FullDefinition)

	rolodex := index.NewRolodex(cfg)
	rolodex.SetRootIndex(idx)
	rolodex.AddIndex(idx)
	assert.Empty(t, inlineRequiredRefs(nil, rolodex))
	assert.Empty(t, sequencedRefsByFullDefinition(nil))

	refsByDefinition := sequencedRefsByFullDefinition(rolodex)
	assert.Len(t, refsByDefinition[responseRef.FullDefinition], 2)

	replacement := testYAMLContentNode(t, "description: inlined\n")
	inlineMatchingRefs(nil, nil, nil)
	inlineMatchingRefs(&processRef{
		ref: &index.Reference{FullDefinition: responseRef.FullDefinition},
	}, replacement, nil)
	inlineMatchingRefs(&processRef{
		ref:    &index.Reference{FullDefinition: responseRef.FullDefinition},
		seqRef: responseRef,
	}, replacement, sequencedRefsByFullDefinition(index.NewRolodex(cfg)))

	inlinedPaths := inlineRequiredRefs([]*processRef{
		nil,
		{
			ref: &index.Reference{
				FullDefinition: responseRef.FullDefinition,
				Node:           replacement,
			},
			seqRef: responseRef,
			mapKey: contextualProcessRefKey(responseRef.FullDefinition, responseRef),
		},
	}, rolodex)

	assert.Same(t, replacement, inlinedPaths[responseRef.FullDefinition])
	assert.Equal(t, replacement.Content, responseRef.Node.Content)
	assert.NotEqual(t, replacement.Content, schemaRef.Node.Content)
}

func newVersionedIndex(version float32) *index.SpecIndex {
	var root yaml.Node
	_ = yaml.Unmarshal([]byte(`openapi: 3.1.0
info:
  title: Test
  version: 1.0.0
paths: {}`), &root)

	cfg := index.CreateClosedAPIIndexConfig()
	cfg.SpecInfo = &datamodel.SpecInfo{VersionNumeric: version}
	idx := index.NewSpecIndexWithConfig(&root, cfg)
	idx.GetConfig().SpecInfo.VersionNumeric = version
	return idx
}

func testYAMLContentNode(t *testing.T, source string) *yaml.Node {
	t.Helper()

	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(source), &root))
	return unwrapDocumentNode(&root)
}

func findRawRefByComponentType(t *testing.T, idx *index.SpecIndex, componentType, refSuffix string) *index.Reference {
	t.Helper()

	for _, ref := range idx.GetRawReferencesSequenced() {
		if ref == nil || !strings.HasSuffix(ref.FullDefinition, refSuffix) {
			continue
		}
		if inferred, ok := inferComponentTypeFromSourcePath(ref.SourcePath); ok && inferred == componentType {
			return ref
		}
	}
	t.Fatalf("expected raw %s ref ending in %q", componentType, refSuffix)
	return nil
}

func newProcessRefForTest(t *testing.T, name, source string) *processRef {
	t.Helper()

	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(source), &root))
	node := unwrapDocumentNode(&root)
	return &processRef{
		ref: &index.Reference{
			Name:           name,
			FullDefinition: "/tmp/" + name + ".yaml",
			Node:           node,
		},
		seqRef: &index.Reference{},
	}
}

func TestWalkAndRewriteRefs_NilNode(t *testing.T) {
	require.NotPanics(t, func() {
		walkAndRewriteRefs(nil, nil, nil, nil, false)
	})
}

func TestRemapIndexSkipsMappedExtensionRefs(t *testing.T) {
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(`openapi: 3.1.0`), &root))
	idx := index.NewSpecIndexWithConfig(&root, index.CreateOpenAPIIndexConfig())

	refNode := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "$ref"},
			{Kind: yaml.ScalarNode, Value: "./sample.md"},
		},
	}
	idx.SetMappedReferences(map[string]*index.Reference{
		"./sample.md": {
			FullDefinition: "./sample.md",
			Node:           refNode,
			IsExtensionRef: true,
		},
	})

	remapIndex(idx, orderedmap.New[string, *processRef]())
	assert.Equal(t, "./sample.md", refNode.Content[1].Value)
}

func TestResolveRefToComposed_PreservesExternalRefs(t *testing.T) {
	assert.Equal(t, "https://example.com/schema.json", resolveRefToComposed("https://example.com/schema.json", nil, nil, nil))
	assert.Equal(t, "urn:example:thing", resolveRefToComposed("urn:example:thing", nil, nil, nil))
}

func TestResolveRefToComposed_RootFallbackFromExternalSource(t *testing.T) {
	_, rolodex := buildResolveRefContext(t)

	indexes := rolodex.GetIndexes()
	require.NotEmpty(t, indexes)

	got := resolveRefToComposed("#/components/schemas/RootThing", indexes[0], orderedmap.New[string, *processRef](), rolodex)
	assert.Equal(t, "#/components/schemas/RootThing", got)
}

func TestResolveRefToComposed_UnindexedLocalRef_NoProcessedNode(t *testing.T) {
	rootIdx, rolodex := buildResolveRefContext(t)
	refValue := "#/components/schemas/Missing"

	got := resolveRefToComposed(refValue, rootIdx, orderedmap.New[string, *processRef](), rolodex)
	assert.Equal(t, refValue, got)
}

func TestResolveRefToComposed_UnindexedLocalRef_UsesProcessedNode(t *testing.T) {
	rootIdx, rolodex := buildResolveRefContext(t)
	refValue := "#/components/schemas/Missing"

	absKey := rootIdx.GetSpecAbsolutePath() + refValue
	require.NotEmpty(t, rootIdx.GetSpecAbsolutePath())

	processed := orderedmap.New[string, *processRef]()
	processed.Set(absKey, &processRef{name: "Renamed"})

	got := resolveRefToComposed(refValue, rootIdx, processed, rolodex)
	assert.Equal(t, "#/components/schemas/Renamed", got)
}

func TestResolveRefToComposed_UnresolvedRef_ReturnsOriginal(t *testing.T) {
	rootIdx, rolodex := buildResolveRefContext(t)
	refValue := "./missing.yaml"

	got := resolveRefToComposed(refValue, rootIdx, orderedmap.New[string, *processRef](), rolodex)
	assert.Equal(t, refValue, got)
}

func TestResolveRefToComposed_FindsInOtherIndexes(t *testing.T) {
	rootIdx, rolodex, idxA, idxB := buildMultiIndexResolveContext(t)
	require.NotNil(t, rootIdx)
	require.NotNil(t, idxA)
	require.NotNil(t, idxB)

	refValue := "./B.yaml#/components/schemas/BThing"
	if r, _ := idxB.SearchIndexForReference(refValue); r == nil {
		t.Fatalf("expected idxB to resolve %s", refValue)
	}
	if r, _ := rootIdx.SearchIndexForReference(refValue); r != nil {
		t.Fatalf("expected root index to NOT resolve %s", refValue)
	}

	got := resolveRefToComposed(refValue, rootIdx, orderedmap.New[string, *processRef](), rolodex)
	assert.Equal(t, refValue, got)
}

func TestRenameRef_FallbackKeepsLastSegment(t *testing.T) {
	def := "/tmp/spec.yaml#/components/schemas/Thing"
	got := renameRef(nil, def, orderedmap.New[string, *processRef]())
	assert.Equal(t, "#/components/schemas/Thing", got)
}

func buildResolveRefContext(t *testing.T) (*index.SpecIndex, *index.Rolodex) {
	tmpDir := t.TempDir()

	rootSpec := `openapi: 3.1.0
info:
  title: Ref Context
  version: 1.0.0
paths:
  /ext:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: './schemas/External.yaml'
components:
  schemas:
    RootThing:
      type: object`

	extSchema := `type: object
properties:
  id:
    type: string`

	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "root.yaml"), []byte(rootSpec), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "schemas", "External.yaml"), []byte(extSchema), 0644))

	specBytes, err := os.ReadFile(filepath.Join(tmpDir, "root.yaml"))
	require.NoError(t, err)

	cfg := datamodel.NewDocumentConfiguration()
	cfg.BasePath = tmpDir
	cfg.SpecFilePath = "root.yaml"
	cfg.AllowFileReferences = true

	doc, err := libopenapi.NewDocumentWithConfiguration(specBytes, cfg)
	require.NoError(t, err)

	v3Doc, err := doc.BuildV3Model()
	require.NoError(t, err)
	require.NotNil(t, v3Doc)

	if v3Doc.Index == nil || v3Doc.Index.GetRolodex() == nil {
		t.Fatalf("expected index and rolodex to be initialized")
	}

	return v3Doc.Index, v3Doc.Index.GetRolodex()
}

func buildMultiIndexResolveContext(t *testing.T) (*index.SpecIndex, *index.Rolodex, *index.SpecIndex, *index.SpecIndex) {
	tmpDir := t.TempDir()

	rootSpec := `openapi: 3.1.0
info:
  title: Multi Index
  version: 1.0.0
paths:
  /a:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: './schemas/A.yaml#/components/schemas/AThing'
  /b:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: './schemas/B.yaml#/components/schemas/BThing'`

	aSchema := `openapi: 3.1.0
info:
  title: A
  version: 1.0.0
components:
  schemas:
    AThing:
      type: object`

	bSchema := `openapi: 3.1.0
info:
  title: B
  version: 1.0.0
components:
  schemas:
    BThing:
      type: object`

	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "root.yaml"), []byte(rootSpec), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "schemas", "A.yaml"), []byte(aSchema), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "schemas", "B.yaml"), []byte(bSchema), 0644))

	specBytes, err := os.ReadFile(filepath.Join(tmpDir, "root.yaml"))
	require.NoError(t, err)

	cfg := datamodel.NewDocumentConfiguration()
	cfg.BasePath = tmpDir
	cfg.SpecFilePath = "root.yaml"
	cfg.AllowFileReferences = true

	doc, err := libopenapi.NewDocumentWithConfiguration(specBytes, cfg)
	require.NoError(t, err)

	v3Doc, err := doc.BuildV3Model()
	require.NoError(t, err)
	require.NotNil(t, v3Doc)

	rolodex := v3Doc.Index.GetRolodex()
	var idxA, idxB *index.SpecIndex
	for _, idx := range rolodex.GetIndexes() {
		switch filepath.Base(idx.GetSpecAbsolutePath()) {
		case "A.yaml":
			idxA = idx
		case "B.yaml":
			idxB = idx
		}
	}

	return v3Doc.Index, rolodex, idxA, idxB
}
