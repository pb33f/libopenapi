package libopenapi

import (
	stdcontext "context"
	"testing"

	"github.com/pb33f/go-yaml"
	lowbase "github.com/pb33f/libopenapi/datamodel/low/base"
	"github.com/pb33f/libopenapi/utils"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

func TestBuildV3ModelMergedComponentsPreservesSource(t *testing.T) {
	const source = `openapi: 3.1.1
info: {title: Merge example, version: '1'}
paths: {}
x-bounds: &bounds
  type: integer
  minimum: 5 # lower bound
  maximum: 10
x-base: &base
  schemas:
    Bound:
      <<: *bounds
      maximum: 9
x-sources: &sources [*base]
components:
  <<: *sources
`
	document, err := NewDocument([]byte(source))
	require.NoError(t, err)
	root := document.GetSpecInfo().RootNode
	before, err := yaml.Marshal(root)
	require.NoError(t, err)
	_, bounds := utils.FindKeyNodeTop("x-bounds", root.Content[0].Content)
	minimumKey, minimumValue := utils.FindKeyNodeTop("minimum", bounds.Content)
	model, err := document.BuildV3Model()
	require.NoError(t, err)
	require.NotNil(t, model.Model.Components)
	proxy, ok := model.Model.Components.Schemas.Get("Bound")
	require.True(t, ok)
	schema := proxy.Schema()
	require.NotNil(t, schema)
	require.NotNil(t, schema.Minimum)
	require.NotNil(t, schema.Maximum)
	assert.Equal(t, float64(5), *schema.Minimum)
	assert.Equal(t, float64(9), *schema.Maximum)
	assert.Same(t, minimumKey, schema.GoLow().Minimum.KeyNode)
	assert.Same(t, minimumValue, schema.GoLow().Minimum.ValueNode)
	for _, effective := range []*yaml.Node{model.Model.Components.GoLow().GetRootNode(), schema.GoLow().RootNode} {
		require.NotNil(t, effective)
		for i := 0; i < len(effective.Content); i += 2 {
			assert.NotEqual(t, "!!merge", effective.Content[i].Tag, "low-level model roots describe effective fields")
		}
	}
	after, err := yaml.Marshal(root)
	require.NoError(t, err)
	assert.Equal(t, string(before), string(after), "building low models must leave published source untouched")

	rendered, err := document.Render()
	require.NoError(t, err)
	reloaded, err := NewDocument(rendered)
	require.NoError(t, err)
	rebuilt, err := reloaded.BuildV3Model()
	require.NoError(t, err)
	rebuiltProxy, ok := rebuilt.Model.Components.Schemas.Get("Bound")
	require.True(t, ok)
	rebuiltSchema := rebuiltProxy.Schema()
	require.NotNil(t, rebuiltSchema)
	require.NotNil(t, rebuiltSchema.Minimum)
	require.NotNil(t, rebuiltSchema.Maximum)
	assert.Equal(t, *schema.Minimum, *rebuiltSchema.Minimum)
	assert.Equal(t, *schema.Maximum, *rebuiltSchema.Maximum)
}

func TestBuildV3ModelMergeViewsReachNestedConsumers(t *testing.T) {
	const source = `openapi: 3.1.1
x-doc: &doc {x-note: root}
<<: *doc
x-contact: &contact {name: support, x-note: contact}
x-var: &var {default: prod, x-note: variable}
x-flow: &flow {authorizationUrl: 'https://example.test/auth', scopes: {}, x-note: flow}
x-schema: &schema {type: object, x-note: schema}
x-disc: &disc {propertyName: kind}
x-xml: &xml {name: pet}
x-docs: &docs {url: https://example.test/docs}
info:
  title: merge
  version: '1'
  contact: {<<: *contact}
paths: {}
servers:
  - url: 'https://{env}.example.test'
    variables:
      env: {<<: *var}
components:
  securitySchemes:
    Auth:
      type: oauth2
      flows:
        implicit: {<<: *flow}
  schemas:
    Target: {<<: *schema}
    Ref: {$ref: '#/components/schemas/Target'}
    Pet:
      type: object
      discriminator: {<<: *disc}
      xml: {<<: *xml}
      externalDocs: {<<: *docs}
`
	document, err := NewDocument([]byte(source))
	require.NoError(t, err)
	root := document.GetSpecInfo().RootNode
	before, err := yaml.Marshal(root)
	require.NoError(t, err)
	model, err := document.BuildV3Model()
	require.NoError(t, err)
	note, found := model.Model.Extensions.Get("x-note")
	assert.True(t, found, "document extensions use the effective root")
	if found {
		assert.Equal(t, "root", note.Value)
	}
	contact := model.Model.Info.Contact
	variable, found := model.Model.Servers[0].Variables.Get("env")
	require.True(t, found)
	auth, found := model.Model.Components.SecuritySchemes.Get("Auth")
	require.True(t, found)
	flow := auth.Flows.Implicit
	pet, found := model.Model.Components.Schemas.Get("Pet")
	require.True(t, found)
	schema := pet.Schema()
	require.NotNil(t, schema)
	discriminator := schema.Discriminator
	require.NotNil(t, discriminator)
	for _, tc := range []struct {
		name  string
		root  *yaml.Node
		field string
	}{
		{"contact", contact.GoLow().GetRootNode(), "name"},
		{"variable", variable.GoLow().GetRootNode(), "default"},
		{"flow", flow.GoLow().GetRootNode(), "authorizationUrl"},
		{"discriminator", discriminator.GoLow().GetRootNode(), "propertyName"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.NotNil(t, tc.root)
			_, field := utils.FindKeyNodeTop(tc.field, tc.root.Content)
			assert.NotNil(t, field, "model root exposes effective fields")
			for i := 0; i < len(tc.root.Content); i += 2 {
				assert.NotEqual(t, "!!merge", tc.root.Content[i].Tag)
			}
		})
	}

	for _, tc := range []struct {
		name   string
		anchor string
		field  string
		nodes  map[int][]*yaml.Node
	}{
		{"xml", "x-xml", "name", schema.XML.GoLow().GetNodes()},
		{"externalDocs", "x-docs", "url", schema.ExternalDocs.GoLow().GetNodes()},
	} {
		t.Run(tc.name+" nodes", func(t *testing.T) {
			_, anchor := utils.FindKeyNodeTop(tc.anchor, root.Content[0].Content)
			key, _ := utils.FindKeyNodeTop(tc.field, anchor.Content)
			assert.Contains(t, tc.nodes[key.Line], key, "inherited field belongs to model node map")
		})
	}
	assert.Equal(t, 1, contact.Extensions.Len())
	assert.Equal(t, 1, variable.Extensions.Len())
	assert.Equal(t, 1, flow.Extensions.Len())
	for _, tc := range []struct {
		name   string
		render func() ([]byte, error)
		value  string
	}{
		{"contact", contact.Render, "contact"},
		{"variable", variable.Render, "variable"},
		{"flow", flow.Render, "flow"},
	} {
		t.Run(tc.name+" render", func(t *testing.T) {
			rendered, err := tc.render()
			require.NoError(t, err)
			assert.Contains(t, string(rendered), "x-note: "+tc.value)
		})
	}
	_, components := utils.FindKeyNodeTop("components", root.Content[0].Content)
	_, schemas := utils.FindKeyNodeTop("schemas", components.Content)
	_, reference := utils.FindKeyNodeTop("Ref", schemas.Content)
	var resolved lowbase.Schema
	require.NoError(t, resolved.Build(stdcontext.Background(), reference, model.Model.GoLow().Index))
	assert.Equal(t, 1, resolved.Extensions.Len())
	assert.Same(t, reference, resolved.RootNode, "authored reference root is retained")
	_, petNode := utils.FindKeyNodeTop("Pet", schemas.Content)
	_, discriminatorNode := utils.FindKeyNodeTop("discriminator", petNode.Content)
	assert.Same(t, discriminatorNode, schema.GoLow().Discriminator.ValueNode)
	_, info := utils.FindKeyNodeTop("info", root.Content[0].Content)
	_, contactNode := utils.FindKeyNodeTop("contact", info.Content)
	assert.Same(t, contactNode, model.Model.Info.GoLow().Contact.ValueNode)
	_, contactAnchor := utils.FindKeyNodeTop("x-contact", root.Content[0].Content)
	nameKey, nameValue := utils.FindKeyNodeTop("name", contactAnchor.Content)
	assert.Same(t, nameKey, contact.GoLow().Name.KeyNode)
	assert.Same(t, nameValue, contact.GoLow().Name.ValueNode)
	after, err := yaml.Marshal(root)
	require.NoError(t, err)
	assert.Equal(t, string(before), string(after), "all consumers leave source nodes unchanged")
}
