package bundler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi/datamodel"
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestBundleIssue873PreservesExplicitZeroValues(t *testing.T) {
	spec := []byte(`openapi: 3.0.3
info:
  title: zero constraints
  version: 1.0.0
paths: {}
components:
  schemas:
    Broken:
      type: object
      properties:
        arr:
          type: array
          minItems: 0
        str:
          type: string
          minLength: 0
        obj:
          type: object
          maxProperties: 0
`)

	bundled, err := BundleBytesComposed(spec, &datamodel.DocumentConfiguration{}, nil)
	require.NoError(t, err)

	assert.Contains(t, string(bundled), "minItems: 0")
	assert.Contains(t, string(bundled), "minLength: 0")
	assert.Contains(t, string(bundled), "maxProperties: 0")
}

func TestBundleIssue873PreservesEmptyProperties(t *testing.T) {
	spec := []byte(`openapi: 3.0.3
info:
  title: empty properties
  version: 1.0.0
paths: {}
components:
  schemas:
    EmptyObject:
      type: object
      properties: {}
`)

	bundled, err := BundleBytesComposed(spec, &datamodel.DocumentConfiguration{}, nil)
	require.NoError(t, err)

	props := lookupMappingNode(t, bundled,
		"components", "schemas", "EmptyObject", "properties")
	require.NotNil(t, props)
	assert.Equal(t, yaml.MappingNode, props.Kind)
	assert.Empty(t, props.Content)
}

func TestBundleIssue873RejectsNonStringDiscriminatorMappings(t *testing.T) {
	spec := []byte(`openapi: 3.0.3
info:
  title: invalid discriminator mapping
  version: 1.0.0
paths: {}
components:
  schemas:
    Pet:
      type: object
      required:
        - type
      properties:
        type:
          type: string
      discriminator:
        propertyName: type
        mapping:
          properties:
            type: object
          required:
            - type
          additionalProperties: false
`)

	_, err := BundleBytesComposed(spec, &datamodel.DocumentConfiguration{}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "discriminator.mapping.properties")
	assert.Contains(t, err.Error(), "must be a string")
}

func TestBundleIssue873IgnoresExtensionDiscriminatorLikeData(t *testing.T) {
	spec := []byte(`openapi: 3.0.3
info:
  title: extension discriminator payload
  version: 1.0.0
paths: {}
x-any:
  discriminator:
    mapping:
      enabled: false
components:
  schemas:
    Pet:
      type: object
`)

	bundled, err := BundleBytesComposed(spec, &datamodel.DocumentConfiguration{}, nil)
	require.NoError(t, err)
	assert.Contains(t, string(bundled), "enabled: false")
}

func TestBundleIssue873IgnoresExampleDiscriminatorLikeData(t *testing.T) {
	spec := []byte(`openapi: 3.0.3
info:
  title: example discriminator payload
  version: 1.0.0
paths: {}
components:
  schemas:
    Pet:
      type: object
      example:
        discriminator:
          mapping:
            enabled: false
`)

	bundled, err := BundleBytesComposed(spec, &datamodel.DocumentConfiguration{}, nil)
	require.NoError(t, err)
	assert.Contains(t, string(bundled), "enabled: false")
}

func TestBundleIssue873RejectsInvalidDiscriminatorInMappingOnlyExternalFile(t *testing.T) {
	tmpDir := t.TempDir()

	rootSpec := []byte(`openapi: 3.0.3
info:
  title: discriminator-only external
  version: 1.0.0
paths: {}
components:
  schemas:
    Pet:
      type: object
      discriminator:
        propertyName: kind
        mapping:
          cat: './cat.yaml'
`)
	catSpec := []byte(`type: object
discriminator:
  propertyName: kind
  mapping:
    broken:
      type: object
`)

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "root.yaml"), rootSpec, 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "cat.yaml"), catSpec, 0644))

	cfg := &datamodel.DocumentConfiguration{
		BasePath:            tmpDir,
		SpecFilePath:        "root.yaml",
		AllowFileReferences: true,
	}

	_, err := BundleBytesComposed(rootSpec, cfg, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "discriminator.mapping.broken")
	assert.Contains(t, err.Error(), "must be a string")

	result, err := BundleBytesComposedWithOrigins(rootSpec, cfg, nil)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "discriminator.mapping.broken")
	assert.Contains(t, err.Error(), "must be a string")
}

func TestBundleIssue873IgnoresDefaultPayloadInMappingOnlyExternalSchema(t *testing.T) {
	tmpDir := t.TempDir()

	rootSpec := []byte(`openapi: 3.0.3
info:
  title: discriminator-only external default
  version: 1.0.0
paths: {}
components:
  schemas:
    Pet:
      type: object
      discriminator:
        propertyName: kind
        mapping:
          cat: './cat.yaml'
`)
	catSpec := []byte(`type: object
default:
  schema:
    discriminator:
      propertyName: kind
      mapping:
        bad:
          type: object
`)

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "root.yaml"), rootSpec, 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "cat.yaml"), catSpec, 0644))

	cfg := &datamodel.DocumentConfiguration{
		BasePath:            tmpDir,
		SpecFilePath:        "root.yaml",
		AllowFileReferences: true,
	}

	bundled, err := BundleBytesComposed(rootSpec, cfg, nil)
	require.NoError(t, err)
	assert.Contains(t, string(bundled), "default:")

	result, err := BundleBytesComposedWithOrigins(rootSpec, cfg, nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Contains(t, string(result.Bytes), "default:")
}

func TestBundleIssue873PreservesFloatLexemesAndIndentation(t *testing.T) {
	spec := []byte(`openapi: 3.0.3
info:
  title: float formatting
  version: 1.0.0
paths: {}
components:
  schemas:
    Number:
      type: number
      minimum: 0.0
      maximum: 100.0
`)

	bundled, err := BundleBytesComposed(spec, &datamodel.DocumentConfiguration{}, nil)
	require.NoError(t, err)

	output := string(bundled)
	assert.Contains(t, output, "minimum: 0.0")
	assert.Contains(t, output, "maximum: 100.0")
	assert.Contains(t, output, "\n  title: float formatting")
	assert.NotContains(t, output, "\n    title: float formatting")
}

func TestRenderBundledModelUsesSourceIndentationAndFallback(t *testing.T) {
	model := &v3.Document{
		Version: "3.1.0",
		Info: &highbase.Info{
			Title:   "render helper",
			Version: "1.0.0",
		},
	}

	rendered, err := renderBundledModel(model, nil)
	require.NoError(t, err)
	assert.Contains(t, string(rendered), "    title: render helper")

	jsonIndex := newSpecIndexForRenderHelper(t, datamodel.JSONFileType, 2)
	rendered, err = renderBundledModel(model, jsonIndex)
	require.NoError(t, err)
	assert.Contains(t, string(rendered), "    title: render helper")

	yamlIndex := newSpecIndexForRenderHelper(t, datamodel.YAMLFileType, 2)
	rendered, err = renderBundledModel(model, yamlIndex)
	require.NoError(t, err)
	assert.Contains(t, string(rendered), "  title: render helper")
	assert.NotContains(t, string(rendered), "    title: render helper")
}

func TestValidateDiscriminatorMappingsHelperPaths(t *testing.T) {
	assert.NoError(t, validateDiscriminatorMappings(nil))
	assert.NoError(t, validateDiscriminatorMappingsFromIndex(nil))
	assert.NoError(t, validateDiscriminatorMappingsFromNode(nil))

	root := parseYAMLNode(t, []byte(`openapi: 3.1.0
info:
  title: root
  version: 1.0.0
paths: {}
`))
	invalid := parseYAMLNode(t, []byte(`components:
  schemas:
    Pet:
      discriminator:
        propertyName: type
        mapping:
          required:
            - type
`))
	extensionPayload := parseYAMLNode(t, []byte(`openapi: 3.1.0
info:
  title: extension payload
  version: 1.0.0
paths: {}
x-any:
  discriminator:
    mapping:
      enabled: false
`))
	examplePayload := parseYAMLNode(t, []byte(`components:
  schemas:
    Pet:
      example:
        discriminator:
          mapping:
            enabled: false
`))

	assert.ErrorContains(t, validateDiscriminatorMappingsFromNode(invalid), "discriminator.mapping.required")
	assert.NoError(t, validateDiscriminatorMappingsFromNode(extensionPayload))
	assert.NoError(t, validateDiscriminatorMappingsFromNode(examplePayload))

	cfg := index.CreateOpenAPIIndexConfig()
	rolodex := index.NewRolodex(cfg)
	rolodex.SetRootIndex(index.NewSpecIndexWithConfig(root, cfg))
	rolodex.AddIndex(index.NewSpecIndexWithConfig(invalid, cfg))

	assert.ErrorContains(t, validateDiscriminatorMappings(rolodex), "discriminator.mapping.required")
}

func TestValidateDiscriminatorMappingsCoverageBranches(t *testing.T) {
	validSchema := parseYAMLNode(t, []byte(`type: object
discriminator:
  propertyName: type
  mapping:
    dog: '#/components/schemas/Dog'
`)).Content[0]

	assert.NoError(t, validateDiscriminatorMappingsFromOpenAPIObject(nil, nil, nil, nil))
	assert.False(t, isOpenAPIDocumentRoot(&yaml.Node{Kind: yaml.ScalarNode}))
	assert.False(t, isOpenAPIDocumentRoot(&yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			nil,
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "ignored"},
		},
	}))
	assert.False(t, isDiscriminatorValidationSchemaCandidate(nil))
	assert.False(t, isDiscriminatorValidationSchemaCandidate(&yaml.Node{Kind: yaml.ScalarNode}))
	assert.False(t, isDiscriminatorValidationSchemaCandidate(&yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			nil,
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "ignored"},
		},
	}))

	seenOpenAPI := make(map[*yaml.Node]struct{})
	openAPIRootWithSchemaCandidate := parseYAMLNode(t, []byte(`openapi: 3.1.0
info:
  title: root discriminator
  version: 1.0.0
paths: {}
discriminator:
  propertyName: type
  mapping:
    bad:
      type: object
`)).Content[0]
	assert.ErrorContains(t,
		validateDiscriminatorMappingsFromRootNode(openAPIRootWithSchemaCandidate, seenOpenAPI),
		"discriminator.mapping.bad",
	)

	seenObject := make(map[*yaml.Node]struct{})
	assert.NoError(t, validateDiscriminatorMappingsFromOpenAPIObject(openAPIRootWithSchemaCandidate, nil, seenObject, nil))
	assert.NoError(t, validateDiscriminatorMappingsFromOpenAPIObject(openAPIRootWithSchemaCandidate, nil, seenObject, nil))

	assert.NoError(t, validateDiscriminatorMappingsFromOpenAPIObject(&yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			nil,
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "ignored"},
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "ignored"},
			nil,
		},
	}, nil, make(map[*yaml.Node]struct{}), nil))

	invalidSchemaObject := parseYAMLNode(t, []byte(`schema:
  discriminator:
    propertyName: type
    mapping:
      bad:
        type: object
`)).Content[0]
	assert.ErrorContains(t,
		validateDiscriminatorMappingsFromOpenAPIObject(invalidSchemaObject, make(map[*yaml.Node]struct{}), make(map[*yaml.Node]struct{}), nil),
		"discriminator.mapping.bad",
	)

	invalidComponentsSchemas := parseYAMLNode(t, []byte(`components:
  schemas:
    Pet:
      discriminator:
        propertyName: type
        mapping:
          bad:
            type: object
`)).Content[0]
	assert.ErrorContains(t,
		validateDiscriminatorMappingsFromOpenAPIObject(invalidComponentsSchemas, make(map[*yaml.Node]struct{}), make(map[*yaml.Node]struct{}), nil),
		"discriminator.mapping.bad",
	)

	invalidDefinitions := parseYAMLNode(t, []byte(`definitions:
  Pet:
    discriminator:
      propertyName: type
      mapping:
        bad:
          type: object
`)).Content[0]
	assert.ErrorContains(t,
		validateDiscriminatorMappingsFromOpenAPIObject(invalidDefinitions, make(map[*yaml.Node]struct{}), make(map[*yaml.Node]struct{}), nil),
		"discriminator.mapping.bad",
	)
	validDefinitions := parseYAMLNode(t, []byte(`definitions:
  Pet:
    discriminator:
      propertyName: type
      mapping:
        dog: '#/definitions/Dog'
`)).Content[0]
	assert.NoError(t,
		validateDiscriminatorMappingsFromOpenAPIObject(validDefinitions, make(map[*yaml.Node]struct{}), make(map[*yaml.Node]struct{}), nil),
	)

	invalidNestedObject := parseYAMLNode(t, []byte(`nested:
  schema:
    discriminator:
      propertyName: type
      mapping:
        bad:
          type: object
`)).Content[0]
	assert.ErrorContains(t,
		validateDiscriminatorMappingsFromOpenAPIObject(invalidNestedObject, make(map[*yaml.Node]struct{}), make(map[*yaml.Node]struct{}), nil),
		"discriminator.mapping.bad",
	)

	assert.NoError(t, validateDiscriminatorMappingsFromOpenAPIObject(&yaml.Node{
		Kind: yaml.SequenceNode,
		Content: []*yaml.Node{
			parseYAMLNode(t, []byte("x-any:\n  discriminator:\n    mapping:\n      bad:\n        type: object\n")).Content[0],
		},
	}, make(map[*yaml.Node]struct{}), make(map[*yaml.Node]struct{}), nil))
	assert.ErrorContains(t, validateDiscriminatorMappingsFromOpenAPIObject(&yaml.Node{
		Kind:    yaml.SequenceNode,
		Content: []*yaml.Node{invalidSchemaObject},
	}, make(map[*yaml.Node]struct{}), make(map[*yaml.Node]struct{}), nil), "discriminator.mapping.bad")

	assert.NoError(t, validateDiscriminatorMappingsFromSchemaNode(nil, make(map[*yaml.Node]struct{})))
	assert.NoError(t, validateDiscriminatorMappingsFromSchemaNode(&yaml.Node{Kind: yaml.ScalarNode}, make(map[*yaml.Node]struct{})))
	seenSchema := make(map[*yaml.Node]struct{})
	assert.NoError(t, validateDiscriminatorMappingsFromSchemaNode(validSchema, seenSchema))
	assert.NoError(t, validateDiscriminatorMappingsFromSchemaNode(validSchema, seenSchema))

	assert.NoError(t, validateDiscriminatorMappingsFromSchemaNode(&yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			nil,
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "ignored"},
			{Kind: yaml.ScalarNode, Tag: "!!str", Value: "ignored"},
			nil,
		},
	}, make(map[*yaml.Node]struct{})))

	assert.ErrorContains(t, validateDiscriminatorMappingsFromSchemaNode(parseYAMLNode(t, []byte(`items:
  discriminator:
    propertyName: type
    mapping:
      bad:
        type: object
`)).Content[0], make(map[*yaml.Node]struct{})), "discriminator.mapping.bad")

	assert.ErrorContains(t, validateDiscriminatorMappingsFromSchemaNode(parseYAMLNode(t, []byte(`properties:
  pet:
    discriminator:
      propertyName: type
      mapping:
        bad:
          type: object
`)).Content[0], make(map[*yaml.Node]struct{})), "discriminator.mapping.bad")

	assert.ErrorContains(t, validateDiscriminatorMappingsFromSchemaNode(parseYAMLNode(t, []byte(`oneOf:
  - discriminator:
      propertyName: type
      mapping:
        bad:
          type: object
`)).Content[0], make(map[*yaml.Node]struct{})), "discriminator.mapping.bad")

	assert.NoError(t, validateDiscriminatorMappingsFromSchemaMap(nil, make(map[*yaml.Node]struct{})))
	assert.NoError(t, validateDiscriminatorMappingsFromSchemaMap(&yaml.Node{Kind: yaml.ScalarNode}, make(map[*yaml.Node]struct{})))
	assert.NoError(t, validateDiscriminatorMappingsFromSchemaArray(nil, make(map[*yaml.Node]struct{})))
	assert.NoError(t, validateDiscriminatorMappingsFromSchemaArray(&yaml.Node{Kind: yaml.ScalarNode}, make(map[*yaml.Node]struct{})))
}

func TestComposeWithOriginsReturnsDiscriminatorValidationError(t *testing.T) {
	invalidRoot := parseYAMLNode(t, []byte(`components:
  schemas:
    Pet:
      discriminator:
        propertyName: type
        mapping:
          bad:
            type: object
`))

	cfg := index.CreateOpenAPIIndexConfig()
	rolodex := index.NewRolodex(cfg)
	rolodex.SetRootIndex(index.NewSpecIndexWithConfig(invalidRoot, cfg))

	result, err := composeWithOrigins(&v3.Document{Rolodex: rolodex}, nil)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "discriminator.mapping.bad")
}

func TestDiscriminatorMappingCollectorsIgnoreMalformedMappingNode(t *testing.T) {
	spec := []byte(`discriminator:
  propertyName: type
  mapping: nope
`)
	discriminator := lookupMappingNode(t, spec, "discriminator")

	assert.NotPanics(t, func() {
		walkDiscriminatorMapping(nil, discriminator, map[string]struct{}{})
	})

	var mappings []*discriminatorMappingWithContext
	collectDiscriminatorMappingNodesFromIndexWithContext(nil, parseYAMLNode(t, spec), &mappings)
	assert.Empty(t, mappings)

}

func lookupMappingNode(t *testing.T, data []byte, path ...string) *yaml.Node {
	t.Helper()

	var root yaml.Node
	require.NoError(t, yaml.Unmarshal(data, &root))
	require.NotEmpty(t, root.Content)

	node := root.Content[0]
	for _, segment := range path {
		require.Equal(t, yaml.MappingNode, node.Kind, "path %s is not a mapping", strings.Join(path, "."))
		var next *yaml.Node
		for i := 0; i < len(node.Content); i += 2 {
			if node.Content[i].Value == segment {
				next = node.Content[i+1]
				break
			}
		}
		if next == nil {
			return nil
		}
		node = next
	}
	return node
}

func parseYAMLNode(t *testing.T, data []byte) *yaml.Node {
	t.Helper()

	var root yaml.Node
	require.NoError(t, yaml.Unmarshal(data, &root))
	return &root
}

func newSpecIndexForRenderHelper(t *testing.T, fileType string, indent int) *index.SpecIndex {
	t.Helper()

	root := parseYAMLNode(t, []byte("openapi: 3.1.0\n"))
	cfg := index.CreateOpenAPIIndexConfig()
	cfg.SpecInfo = &datamodel.SpecInfo{
		SpecFileType:        fileType,
		OriginalIndentation: indent,
	}
	return index.NewSpecIndexWithConfig(root, cfg)
}

func assertYAMLEquivalent(t *testing.T, expected, actual []byte) {
	t.Helper()

	var expectedNode any
	var actualNode any
	require.NoError(t, yaml.Unmarshal(expected, &expectedNode))
	require.NoError(t, yaml.Unmarshal(actual, &actualNode))
	assert.Equal(t, expectedNode, actualNode)
}
