// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package bundler

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestPrepareInlineReferences_NoModelState(t *testing.T) {
	ctx := highbase.NewInlineRenderContext()
	require.NoError(t, prepareInlineReferences(nil, ctx, false))
	require.NoError(t, prepareInlineReferences(nil, nil, false))
	targets, err := collectInlineReferenceTargets(nil, false)
	require.NoError(t, err)
	assert.Empty(t, targets)
	require.ErrorContains(t, prepareCollectedInlineReferences(nil, nil, false, nil, errors.New("collect failed")), "collect failed")
}

func TestInlineReferenceTargetHelpers(t *testing.T) {
	var document yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("Thing: {type: object}"), &document))
	idx := index.NewSpecIndexWithConfig(&document, index.CreateOpenAPIIndexConfig())
	idx.SetAbsolutePath(filepath.Join(t.TempDir(), "external.yaml"))
	fullDefinition := idx.GetSpecAbsolutePath() + "#/Thing"
	ref := &index.Reference{FullDefinition: fullDefinition, Node: &document}

	target, err := buildCircularInlineTarget(ref, fullDefinition, idx.GetSpecAbsolutePath(), "#/Thing", map[string]*index.SpecIndex{idx.GetSpecAbsolutePath(): idx})
	require.NoError(t, err)
	assert.Equal(t, document.Content[0], target.node)
	_, err = buildCircularInlineTarget(ref, fullDefinition, idx.GetSpecAbsolutePath(), "#/Thing", nil)
	require.ErrorContains(t, err, "no source index")
	ref.Node = nil
	_, err = buildCircularInlineTarget(ref, fullDefinition, idx.GetSpecAbsolutePath(), "#/Thing", map[string]*index.SpecIndex{idx.GetSpecAbsolutePath(): idx})
	require.ErrorContains(t, err, "no schema node")

	schemaNode := utils.CreateRefNode("#/Thing")
	targets := make(map[string]struct{})
	schemaNodes := map[*yaml.Node]struct{}{schemaNode: {}}
	addInlineSchemaTarget(targets, schemaNodes, nil)
	addInlineSchemaTarget(targets, schemaNodes, &index.ReferenceMapped{})
	addInlineSchemaTarget(targets, schemaNodes, &index.ReferenceMapped{OriginalReference: &index.Reference{Node: utils.CreateRefNode("#/Other")}})
	addInlineSchemaTarget(targets, schemaNodes, &index.ReferenceMapped{OriginalReference: &index.Reference{Node: schemaNode}, FullDefinition: fullDefinition})
	assert.Contains(t, targets, index.CanonicalReferenceIdentity(fullDefinition))
}

func TestPrepareInlineReferences_InitializesSchemaMap(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "node.yaml"), []byte("type: object\nproperties:\n  next:\n    $ref: './node.yaml'\n"), 0o600))
	root := []byte("openapi: 3.1.0\ninfo: {title: t, version: v}\npaths:\n  /node:\n    get:\n      responses:\n        '200':\n          description: ok\n          content:\n            application/json:\n              schema:\n                $ref: './node.yaml'\n")
	config := &datamodel.DocumentConfiguration{BasePath: tmp, AllowFileReferences: true}
	document, err := libopenapi.NewDocumentWithConfiguration(root, config)
	require.NoError(t, err)
	model, err := document.BuildV3Model()
	require.NoError(t, err)
	model.Model.Components = &v3high.Components{}
	require.NoError(t, prepareInlineReferences(&model.Model, highbase.NewInlineRenderContext(), false))
	assert.NotNil(t, model.Model.Components.Schemas)
}

func TestInlineTargetNameAndLookupEdges(t *testing.T) {
	assert.Equal(t, "A/B C", inlineTargetName("schema.yaml", "#/A~1B%20C"))
	assert.Equal(t, "node", inlineTargetName("/schemas/node.yaml", ""))
	assert.Equal(t, "Schema", inlineTargetName(".", ""))
	assert.Empty(t, findInlineTargetComponent(nil, nil))

	schemas := orderedmap.New[string, *highbase.SchemaProxy]()
	schemas.Set("Nil", nil)
	assert.Empty(t, findInlineTargetComponent(schemas, &inlineReferenceTarget{}))
	assert.Equal(t, "raw", inlineAuthoredReference(&index.Reference{RawRef: "raw", Definition: "definition"}))
	assert.Equal(t, "definition", inlineAuthoredReference(&index.Reference{Definition: "definition"}))
}

func TestCollectInlineReferenceTargets_CircularMetadataEdges(t *testing.T) {
	var rootNode, externalNode yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("openapi: 3.1.0\ninfo: {title: t, version: v}\npaths: {}"), &rootNode))
	require.NoError(t, yaml.Unmarshal([]byte("Thing: {type: object}"), &externalNode))
	rootIndex := index.NewSpecIndexWithConfig(&rootNode, index.CreateOpenAPIIndexConfig())
	externalIndex := index.NewSpecIndexWithConfig(&externalNode, index.CreateOpenAPIIndexConfig())
	rootIndex.SetAbsolutePath(filepath.Join(t.TempDir(), "root.yaml"))
	externalIndex.SetAbsolutePath(filepath.Join(t.TempDir(), "external.yaml"))
	rolodex := index.NewRolodex(index.CreateOpenAPIIndexConfig())
	rolodex.SetRootIndex(rootIndex)
	rolodex.AddIndex(externalIndex)
	rootIndex.SetRolodex(rolodex)
	externalIndex.SetRolodex(rolodex)

	externalDefinition := externalIndex.GetSpecAbsolutePath() + "#/Thing"
	rootIndex.SetCircularReferences([]*index.CircularReferenceResult{
		nil,
		{},
		{LoopIndex: -2, Journey: []*index.Reference{
			nil,
			{},
			{FullDefinition: "#/NoFile", Node: externalNode.Content[0]},
			{FullDefinition: rootIndex.GetSpecAbsolutePath() + "#/Root", Node: rootNode.Content[0]},
			{FullDefinition: externalDefinition, Node: &externalNode},
			{FullDefinition: externalDefinition, Node: externalNode.Content[0]},
		}},
	})
	targets, err := collectInlineReferenceTargets(rolodex, false)
	require.NoError(t, err)
	assert.Empty(t, targets, "non-schema circular metadata must not be lifted into components.schemas")

	deep := &index.CircularReferenceResult{LoopIndex: 500, LoopPoint: &index.Reference{FullDefinition: externalDefinition}, Journey: []*index.Reference{
		{FullDefinition: rootIndex.GetSpecAbsolutePath() + "#/Root", Node: rootNode.Content[0]},
		{FullDefinition: externalDefinition, Node: externalNode.Content[0]},
	}}
	assert.Equal(t, 1, circularJourneyStart(deep), "depth-limit LoopIndex must fall back to the actual loop point")
	assert.Zero(t, circularJourneyStart(&index.CircularReferenceResult{LoopIndex: -1, Journey: deep.Journey}))
}

func TestPopulateInlineDiscriminatorMappingRewrites_SkipsUnresolvedMapping(t *testing.T) {
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(`openapi: 3.1.0
info: {title: t, version: v}
paths: {}
components:
  schemas:
    Animal:
      discriminator:
        propertyName: kind
        mapping:
          missing: '#/components/schemas/Missing'
`), &root))
	rootIndex := index.NewSpecIndexWithConfig(&root, index.CreateOpenAPIIndexConfig())
	rootIndex.SetAbsolutePath(filepath.Join(t.TempDir(), "root.yaml"))
	rolodex := index.NewRolodex(index.CreateOpenAPIIndexConfig())
	rolodex.SetRootIndex(rootIndex)
	rootIndex.SetRolodex(rolodex)
	require.NoError(t, populateInlineDiscriminatorMappingRewrites(rolodex, nil, highbase.NewInlineRenderContext()))
}

func TestPopulateInlineDiscriminatorMappingRewrites_RejectsUnliftedExternalTarget(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "cat.yaml"), []byte("Cat: {type: object}\n"), 0o600))
	root := []byte(`openapi: 3.1.0
info: {title: t, version: v}
paths: {}
components:
  schemas:
    Animal:
      discriminator:
        propertyName: kind
        mapping:
          cat: './cat.yaml#/Cat'
`)
	doc, err := libopenapi.NewDocumentWithConfiguration(root, &datamodel.DocumentConfiguration{BasePath: tmp, AllowFileReferences: true})
	require.NoError(t, err)
	model, err := doc.BuildV3Model()
	require.NoError(t, err)

	err = populateInlineDiscriminatorMappingRewrites(model.Model.Rolodex, nil, highbase.NewInlineRenderContext())
	require.ErrorContains(t, err, "was not lifted")
	external := model.Model.Rolodex.GetIndexes()[0]
	externalRoot := external.GetRootNode()
	if externalRoot.Kind == yaml.DocumentNode {
		externalRoot = externalRoot.Content[0]
	}
	err = prepareCollectedInlineReferences(&model.Model, highbase.NewInlineRenderContext(), true, []*inlineReferenceTarget{{
		fullDefinition: external.GetSpecAbsolutePath() + "#/Other",
		definition:     "#/Other",
		index:          external,
		node:           externalRoot.Content[1],
		name:           "Other",
	}}, nil)
	require.ErrorContains(t, err, "was not lifted")
}

func TestInlineCircularReferences_CollisionsAndIdempotency(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "external.yaml"), []byte(`Thing:
  type: object
  properties:
    next:
      $ref: '#/Thing'
`), 0o600))
	root := []byte(`openapi: 3.1.0
info: {title: collision, version: 1.0.0}
paths:
  /thing:
    get:
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                $ref: './external.yaml#/Thing'
components:
  schemas:
    Thing:
      type: string
`)
	config := &datamodel.DocumentConfiguration{BasePath: tmp, AllowFileReferences: true}
	doc, err := libopenapi.NewDocumentWithConfiguration(root, config)
	require.NoError(t, err)
	model, err := doc.BuildV3Model()
	require.NoError(t, err)

	first, err := BundleDocument(&model.Model)
	require.NoError(t, err)
	second, err := BundleDocument(&model.Model)
	require.NoError(t, err)
	assert.Equal(t, first, second)
	assert.Contains(t, string(first), "$ref: '#/components/schemas/Thing__external'")
	assert.Equal(t, 1, strings.Count(string(first), "        Thing__external:"))
}

func TestCollectInlineReferenceTargets_NormalizesJourneyPaths(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "external.yaml"), []byte("Thing:\n  type: object\n  properties:\n    next:\n      $ref: '#/Thing'\n"), 0o600))
	root := []byte("openapi: 3.1.0\ninfo: {title: normalized, version: v}\npaths: {}\ncomponents:\n  schemas:\n    Thing:\n      $ref: './external.yaml#/Thing'\n")
	config := &datamodel.DocumentConfiguration{BasePath: tmp, AllowFileReferences: true}
	doc, err := libopenapi.NewDocumentWithConfiguration(root, config)
	require.NoError(t, err)
	model, err := doc.BuildV3Model()
	require.NoError(t, err)

	allCircular := append([]*index.CircularReferenceResult{}, model.Model.Rolodex.GetRootIndex().GetCircularReferences()...)
	for _, idx := range model.Model.Rolodex.GetIndexes() {
		allCircular = append(allCircular, idx.GetCircularReferences()...)
	}
	require.NotEmpty(t, allCircular)
	var externalJourneyRef *index.Reference
	for _, result := range allCircular {
		for _, ref := range result.Journey {
			if ref == nil {
				continue
			}
			_, fragment := index.SplitRefFragment(ref.FullDefinition)
			if strings.Contains(ref.FullDefinition, "external.yaml") {
				ref.FullDefinition = filepath.ToSlash(tmp) + "/nested/../external.yaml" + fragment
				externalJourneyRef = ref
			}
		}
	}

	targets, err := collectInlineReferenceTargets(model.Model.Rolodex, false)
	require.NoError(t, err)
	require.Len(t, targets, 1)
	assert.Equal(t, index.CanonicalReferenceIdentity(filepath.Join(tmp, "external.yaml")+"#/Thing"), targets[0].fullDefinition)

	require.NotNil(t, externalJourneyRef)
	originalNode := externalJourneyRef.Node
	externalJourneyRef.Node = nil
	_, err = collectInlineReferenceTargets(model.Model.Rolodex, false)
	require.ErrorContains(t, err, "no schema node")
	externalJourneyRef.Node = originalNode
	model.Model.Rolodex.GetIndexes()[0].SetAbsolutePath(filepath.Join(tmp, "renamed.yaml"))
	_, err = collectInlineReferenceTargets(model.Model.Rolodex, false)
	require.ErrorContains(t, err, "no source index")
}

func TestBundleBytes_SkipCircularReferenceCheckFailsClosed(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "node.yaml"), []byte("type: object\nproperties:\n  next:\n    $ref: './node.yaml'\n"), 0o600))
	root := []byte("openapi: 3.1.0\ninfo: {title: skipped, version: v}\npaths: {}\ncomponents:\n  schemas:\n    Node:\n      $ref: './node.yaml'\n")
	config := &datamodel.DocumentConfiguration{BasePath: tmp, AllowFileReferences: true, SkipCircularReferenceCheck: true}

	bundled, err := BundleBytes(root, config)
	assert.Nil(t, bundled)
	require.ErrorContains(t, err, "circular reference")
}

func TestBundleBytes_UnresolvedExternalDiscriminatorMappingFails(t *testing.T) {
	root := []byte(`openapi: 3.1.0
info: {title: discriminator, version: v}
paths: {}
components:
  schemas:
    Animal:
      discriminator:
        propertyName: kind
        mapping:
          missing: './missing.yaml#/Missing'
`)
	config := &datamodel.DocumentConfiguration{BasePath: t.TempDir(), AllowFileReferences: true}

	bundled, err := BundleBytesWithConfig(root, config, &BundleInlineConfig{ResolveDiscriminatorExternalRefs: true})
	assert.Nil(t, bundled)
	require.ErrorContains(t, err, "unable to resolve external discriminator mapping")
}

func TestInlineCircularReferences_SameFragmentInDifferentFiles(t *testing.T) {
	tmp := t.TempDir()
	for _, dir := range []string{"a", "b"} {
		require.NoError(t, os.MkdirAll(filepath.Join(tmp, dir), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(tmp, dir, "node.yaml"), []byte(`Thing:
  type: object
  properties:
    next:
      $ref: '#/Thing'
`), 0o600))
	}
	root := []byte(`openapi: 3.1.0
info: {title: scoped refs, version: 1.0.0}
paths: {}
components:
  schemas:
    A:
      $ref: './a/node.yaml#/Thing'
    B:
      $ref: './b/node.yaml#/Thing'
`)
	config := &datamodel.DocumentConfiguration{BasePath: tmp, AllowFileReferences: true}
	bundled, err := BundleBytes(root, config)
	require.NoError(t, err)
	output := string(bundled)
	assert.NotContains(t, output, "node.yaml")
	assert.Contains(t, output, "$ref: '#/components/schemas/Thing'")
	assert.Contains(t, output, "$ref: '#/components/schemas/Thing__node'")
	assert.Equal(t, 1, strings.Count(output, "        Thing:"))
	assert.Equal(t, 1, strings.Count(output, "        Thing__node:"))
}

func TestInlineDiscriminatorPreparationDoesNotMutateIndexedNodes(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "cat.yaml"), []byte(`components:
  schemas:
    Cat:
      type: object
      properties:
        next:
          $ref: '#/components/schemas/Cat'
`), 0o600))
	root := []byte(`openapi: 3.1.0
info: {title: discriminator, version: 1.0.0}
paths: {}
components:
  schemas:
    Animal:
      type: object
      discriminator:
        propertyName: kind
        mapping:
          cat: './cat.yaml#/components/schemas/Cat'
      oneOf:
        - $ref: './cat.yaml#/components/schemas/Cat'
`)
	config := &datamodel.DocumentConfiguration{BasePath: tmp, AllowFileReferences: true}
	doc, err := libopenapi.NewDocumentWithConfiguration(root, config)
	require.NoError(t, err)
	model, err := doc.BuildV3Model()
	require.NoError(t, err)
	mapping := collectDiscriminatorMappingNodesWithContext(model.Model.Rolodex)[0].node
	originalMapping := mapping.Value
	unionRef := model.Model.Components.Schemas.GetOrZero("Animal").Schema().OneOf[0].GetReferenceNode()
	originalUnion := unionRef.Content[1].Value

	bundled, err := BundleDocumentWithConfig(&model.Model, &BundleInlineConfig{ResolveDiscriminatorExternalRefs: true})
	require.NoError(t, err)
	assert.Contains(t, string(bundled), "cat: '#/components/schemas/Cat'")
	assert.Equal(t, originalMapping, mapping.Value)
	assert.Equal(t, originalUnion, unionRef.Content[1].Value)
}

func TestBundleInlineLocalRefsPolicyIsPerRender(t *testing.T) {
	spec := []byte(`openapi: 3.1.0
info: {title: concurrent, version: 1.0.0}
paths:
  /pet:
    get:
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Pet'
components:
  schemas:
    Pet:
      type: object
`)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		for _, inline := range []bool{false, true} {
			inline := inline
			wg.Add(1)
			go func() {
				defer wg.Done()
				out, err := BundleBytesWithConfig(spec, datamodel.NewDocumentConfiguration(), &BundleInlineConfig{InlineLocalRefs: &inline})
				require.NoError(t, err)
				if inline {
					assert.NotContains(t, string(out), "$ref: '#/components/schemas/Pet'")
				} else {
					assert.Contains(t, string(out), "$ref: '#/components/schemas/Pet'")
				}
			}()
		}
	}
	wg.Wait()
}

func BenchmarkBundleBytes_InlineReferencePreparation(b *testing.B) {
	for _, recursive := range []bool{false, true} {
		name := "acyclic"
		external := "type: object\nproperties:\n  value: {type: string}\n"
		if recursive {
			name = "external-circular"
			external = "type: object\nproperties:\n  next:\n    $ref: './node.yaml'\n"
		}
		b.Run(name, func(b *testing.B) {
			tmp := b.TempDir()
			require.NoError(b, os.WriteFile(filepath.Join(tmp, "node.yaml"), []byte(external), 0o600))
			root := []byte("openapi: 3.1.0\ninfo: {title: benchmark, version: v}\npaths: {}\ncomponents:\n  schemas:\n    Node:\n      $ref: './node.yaml'\n")
			config := &datamodel.DocumentConfiguration{BasePath: tmp, AllowFileReferences: true}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := BundleBytes(root, config); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
