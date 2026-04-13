// Copyright 2023 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package v3

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unsafe"

	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/datamodel/low"
	v3 "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestComponents_MarshalYAML(t *testing.T) {
	comp := &Components{
		Responses: orderedmap.ToOrderedMap(map[string]*Response{
			"200": {
				Description: "OK",
			},
		}),
		Parameters: orderedmap.ToOrderedMap(map[string]*Parameter{
			"id": {
				Name: "id",
				In:   "path",
			},
		}),
		RequestBodies: orderedmap.ToOrderedMap(map[string]*RequestBody{
			"body": {
				Content: orderedmap.ToOrderedMap(map[string]*MediaType{
					"application/json": {
						Example: utils.CreateStringNode("why?"),
					},
				}),
			},
		}),
		PathItems: orderedmap.ToOrderedMap(map[string]*PathItem{
			"/ding/dong/{bing}/{bong}/go": {
				Get: &Operation{
					Description: "get",
				},
			},
		}),
	}

	dat, _ := comp.Render()

	var idxNode yaml.Node
	_ = yaml.Unmarshal(dat, &idxNode)
	idx := index.NewSpecIndexWithConfig(&idxNode, index.CreateOpenAPIIndexConfig())

	var n v3.Components
	_ = low.BuildModel(idxNode.Content[0], &n)
	_ = n.Build(context.Background(), idxNode.Content[0], idx)

	r := NewComponents(&n)

	desired := `responses:
    "200":
        description: OK
parameters:
    id:
        name: id
        in: path
requestBodies:
    body:
        content:
            application/json:
                example: why?
pathItems:
    /ding/dong/{bing}/{bong}/go:
        get:
            description: get`

	dat, _ = r.Render()
	assert.Equal(t, desired, strings.TrimSpace(string(dat)))
	assert.NotNil(t, r.GoLowUntyped())
}

func TestComponents_RenderInline(t *testing.T) {
	comp := &Components{
		Responses: orderedmap.ToOrderedMap(map[string]*Response{
			"200": {
				Description: "OK",
			},
		}),
		Parameters: orderedmap.ToOrderedMap(map[string]*Parameter{
			"id": {
				Name: "id",
				In:   "path",
			},
		}),
	}

	rendered, err := comp.RenderInline()
	assert.NoError(t, err)

	assert.Contains(t, string(rendered), "responses:")
	assert.Contains(t, string(rendered), "description: OK")
	assert.Contains(t, string(rendered), "parameters:")
	assert.Contains(t, string(rendered), "name: id")
}

func TestComponents_MarshalYAMLInline(t *testing.T) {
	comp := &Components{
		Responses: orderedmap.ToOrderedMap(map[string]*Response{
			"404": {
				Description: "Not Found",
			},
		}),
	}

	node, err := comp.MarshalYAMLInline()
	assert.NoError(t, err)
	assert.NotNil(t, node)

	// Verify it can be marshaled to YAML
	rendered, err := yaml.Marshal(node)
	assert.NoError(t, err)
	assert.Contains(t, string(rendered), "responses:")
	assert.Contains(t, string(rendered), "description: Not Found")
}

func TestComponents_Render_PreservesInvalidComponentMapRefsAndWarns(t *testing.T) {
	tmpDir := t.TempDir()

	spec := `openapi: 3.0.3
info:
  title: Test API
  version: 1.0.0
components:
  parameters:
    $ref: "./params.yaml"
    LocalParam:
      name: local
      in: query
      schema:
        type: string
  schemas:
    $ref: "./schemas.yaml"
    LocalSchema:
      type: object
      properties:
        local:
          type: string
paths: {}
`

	params := `RemoteParam:
  name: remote
  in: query
  schema:
    type: string
`

	schemas := `RemoteSchema:
  type: object
  properties:
    id:
      type: string
`

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "params.yaml"), []byte(params), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "schemas.yaml"), []byte(schemas), 0o644))

	var logBuf bytes.Buffer
	info, err := datamodel.ExtractSpecInfo([]byte(spec))
	require.NoError(t, err)

	cfg := &datamodel.DocumentConfiguration{
		BasePath:            tmpDir,
		AllowFileReferences: true,
		Logger: slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{
			Level: slog.LevelWarn,
		})),
	}

	lowDoc, err := v3.CreateDocumentFromConfig(info, cfg)
	require.NoError(t, err)

	doc := NewDocument(lowDoc)
	rendered, err := doc.Components.Render()
	require.NoError(t, err)

	renderedStr := string(rendered)
	assert.Contains(t, renderedStr, "$ref: \"./params.yaml\"")
	assert.Contains(t, renderedStr, "$ref: \"./schemas.yaml\"")
	assert.NotContains(t, renderedStr, "$ref: {}")

	logOutput := logBuf.String()
	assert.Contains(t, logOutput, "preserving invalid component map $ref entry during render")
	assert.Contains(t, logOutput, "\"section\":\"parameters\"")
	assert.Contains(t, logOutput, "\"section\":\"schemas\"")
}

func TestComponents_warnPreservedComponentMapRefs_Guards(t *testing.T) {
	var nilComp *Components
	nilComp.warnPreservedComponentMapRefs()

	comp := &Components{}
	comp.warnPreservedComponentMapRefs()

	lowComp := &v3.Components{}
	comp = &Components{low: lowComp}
	comp.warnPreservedComponentMapRefs()

	setLowComponentsIndex(lowComp, &index.SpecIndex{})
	comp.warnPreservedComponentMapRefs()
}

func TestWarnComponentRefEntries_OnlyWarnsForScalarRefEntries(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))

	warnComponentRefEntries[*Parameter](logger, "parameters", nil)

	nonScalarEntries := orderedmap.New[low.KeyReference[string], low.ValueReference[*Parameter]]()
	nonScalarEntries.Set(
		low.KeyReference[string]{Value: "LocalParam", KeyNode: utils.CreateStringNode("LocalParam")},
		low.ValueReference[*Parameter]{
			ValueNode: utils.CreateStringNode("ignored"),
		},
	)
	nonScalarEntries.Set(
		low.KeyReference[string]{Value: "$ref", KeyNode: utils.CreateStringNode("$ref")},
		low.ValueReference[*Parameter]{
			ValueNode: utils.CreateEmptyMapNode(),
		},
	)

	warnComponentRefEntries(logger, "parameters", nonScalarEntries)
	assert.Empty(t, logBuf.String())

	scalarEntries := orderedmap.New[low.KeyReference[string], low.ValueReference[*Parameter]]()
	scalarEntries.Set(
		low.KeyReference[string]{Value: "$ref", KeyNode: utils.CreateStringNode("$ref")},
		low.ValueReference[*Parameter]{
			ValueNode: utils.CreateStringNode("./params.yaml"),
		},
	)

	warnComponentRefEntries(logger, "parameters", scalarEntries)

	logOutput := logBuf.String()
	assert.Contains(t, logOutput, "preserving invalid component map $ref entry during render")
	assert.Contains(t, logOutput, "\"section\":\"parameters\"")
	assert.Contains(t, logOutput, "\"ref\":\"./params.yaml\"")
}

func TestPreserveComponentRefEntries_CreatesAndUpdatesSectionNodes(t *testing.T) {
	rendered := utils.CreateEmptyMapNode()

	preserveComponentRefEntries[*Parameter](rendered, "parameters", nil)
	assert.Empty(t, rendered.Content)

	entries := orderedmap.New[low.KeyReference[string], low.ValueReference[*Parameter]]()
	entries.Set(
		low.KeyReference[string]{Value: "$ref", KeyNode: utils.CreateStringNode("$ref")},
		low.ValueReference[*Parameter]{
			ValueNode: utils.CreateStringNode("./params.yaml"),
		},
	)
	entries.Set(
		low.KeyReference[string]{Value: "LocalParam", KeyNode: utils.CreateStringNode("LocalParam")},
		low.ValueReference[*Parameter]{
			ValueNode: utils.CreateStringNode("ignored"),
		},
	)

	preserveComponentRefEntries(rendered, "parameters", entries)

	sectionNode := findMapValueNode(rendered, "parameters")
	require.NotNil(t, sectionNode)
	require.Len(t, sectionNode.Content, 2)
	assert.Equal(t, "$ref", sectionNode.Content[0].Value)
	assert.Equal(t, "./params.yaml", sectionNode.Content[1].Value)

	upsertMapNodeEntry(sectionNode, utils.CreateStringNode("$ref"), utils.CreateEmptyMapNode())
	preserveComponentRefEntries(rendered, "parameters", entries)
	require.Len(t, sectionNode.Content, 2)
	assert.Equal(t, "./params.yaml", sectionNode.Content[1].Value)
}

func TestPreserveComponentRefEntries_IgnoresInvalidNodes(t *testing.T) {
	rendered := utils.CreateEmptyMapNode()
	entries := orderedmap.New[low.KeyReference[string], low.ValueReference[*Parameter]]()
	entries.Set(
		low.KeyReference[string]{Value: "$ref"},
		low.ValueReference[*Parameter]{
			ValueNode: utils.CreateStringNode("./missing-key-node.yaml"),
		},
	)
	entries.Set(
		low.KeyReference[string]{Value: "$ref", KeyNode: utils.CreateStringNode("$ref")},
		low.ValueReference[*Parameter]{
			ValueNode: utils.CreateEmptyMapNode(),
		},
	)

	preserveComponentRefEntries(rendered, "parameters", entries)
	assert.Nil(t, findMapValueNode(rendered, "parameters"))
}

func TestFindMapValueNodeAndUpsertMapNodeEntry(t *testing.T) {
	assert.Nil(t, findMapValueNode(nil, "parameters"))
	assert.Nil(t, findMapValueNode(utils.CreateEmptySequenceNode(), "parameters"))

	rendered := utils.CreateEmptyMapNode()
	assert.Nil(t, findMapValueNode(rendered, "parameters"))

	upsertMapNodeEntry(nil, utils.CreateStringNode("$ref"), utils.CreateStringNode("./ignored.yaml"))
	upsertMapNodeEntry(rendered, nil, utils.CreateStringNode("./ignored.yaml"))
	upsertMapNodeEntry(rendered, utils.CreateStringNode("$ref"), nil)

	upsertMapNodeEntry(rendered, utils.CreateStringNode("$ref"), utils.CreateStringNode("./params.yaml"))
	require.Len(t, rendered.Content, 2)
	found := findMapValueNode(rendered, "$ref")
	require.NotNil(t, found)
	assert.Equal(t, "./params.yaml", found.Value)

	upsertMapNodeEntry(rendered, utils.CreateStringNode("$ref"), utils.CreateStringNode("./updated.yaml"))
	require.Len(t, rendered.Content, 2)
	assert.Equal(t, "./updated.yaml", rendered.Content[1].Value)
}

func TestCloneYAMLNode_ClonesRecursively(t *testing.T) {
	assert.Nil(t, cloneYAMLNode(nil))

	original := utils.CreateEmptyMapNode()
	original.Content = append(
		original.Content,
		utils.CreateStringNode("child"),
		utils.CreateStringNode("value"),
	)

	cloned := cloneYAMLNode(original)
	require.NotNil(t, cloned)
	require.NotSame(t, original, cloned)
	require.Len(t, cloned.Content, 2)
	assert.NotSame(t, original.Content[0], cloned.Content[0])
	assert.Equal(t, "child", cloned.Content[0].Value)
	assert.Equal(t, "value", cloned.Content[1].Value)
}

func setLowComponentsIndex(comp *v3.Components, idx *index.SpecIndex) {
	field := reflect.ValueOf(comp).Elem().FieldByName("index")
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Set(reflect.ValueOf(idx))
}
