package bundler

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/datamodel"
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	v3low "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestWalkAndRewriteRefs_NilNode(t *testing.T) {
	require.NotPanics(t, func() {
		walkAndRewriteRefs(nil, nil, nil, nil, false)
	})
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
