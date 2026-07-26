// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"context"
	"hash/maphash"
	"testing"
	"time"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"go.yaml.in/yaml/v4"
)

func mappingNode(t *testing.T, src string) *yaml.Node {
	t.Helper()
	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(src), &node))
	return node.Content[0]
}

// requireNodeKind accepts an absent field, accepts an explicit null as absent, accepts any
// listed kind, and otherwise reports the offending node's position.
func TestRequireNodeKind(t *testing.T) {
	t.Run("absent field is not an error", func(t *testing.T) {
		assert.NoError(t, requireScalar("missing", "a scalar", mappingNode(t, "other: value")))
	})

	t.Run("explicit null is treated as absent", func(t *testing.T) {
		assert.NoError(t, requireScalar("field", "a scalar", mappingNode(t, "field: ~")))
		assert.NoError(t, requireSequence("field", "a sequence", mappingNode(t, "field: null")))
	})

	t.Run("matching kind is accepted", func(t *testing.T) {
		assert.NoError(t, requireScalar("field", "a scalar", mappingNode(t, "field: value")))
		assert.NoError(t, requireSequence("field", "a sequence", mappingNode(t, "field:\n  - one")))
	})

	t.Run("mismatched kind reports position", func(t *testing.T) {
		err := requireScalar("field", "a scalar thing", mappingNode(t, "other: x\nfield:\n  nested: y"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "field at line 3, column 3 must be a scalar thing")
	})

	t.Run("multiple accepted kinds", func(t *testing.T) {
		root := mappingNode(t, "field:\n  nested: y")
		assert.NoError(t, requireNodeKind("field", "a scalar or mapping", root,
			yaml.ScalarNode, yaml.MappingNode))

		err := requireNodeKind("field", "a scalar or sequence", root,
			yaml.ScalarNode, yaml.SequenceNode)
		assert.Error(t, err)
	})
}

// requireInteger additionally rejects a scalar that does not parse as an integer, since a
// wrong-typed scalar passes the node-kind check but is still dropped by the builder.
func TestRequireInteger(t *testing.T) {
	t.Run("absent and null are accepted", func(t *testing.T) {
		assert.NoError(t, requireInteger("timeout", "an integer", mappingNode(t, "other: 1")))
		assert.NoError(t, requireInteger("timeout", "an integer", mappingNode(t, "timeout: ~")))
	})

	t.Run("integers are accepted", func(t *testing.T) {
		assert.NoError(t, requireInteger("timeout", "an integer", mappingNode(t, "timeout: 2500")))
		assert.NoError(t, requireInteger("timeout", "an integer", mappingNode(t, "timeout: -5")))
	})

	t.Run("non-scalar is rejected", func(t *testing.T) {
		err := requireInteger("timeout", "an integer", mappingNode(t, "timeout:\n  - 1"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "timeout at line 2, column 3 must be an integer")
	})

	t.Run("non-integer scalar is rejected with its value", func(t *testing.T) {
		err := requireInteger("timeout", "an integer", mappingNode(t, "timeout: soon"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), `must be an integer, got "soon"`)
	})

	t.Run("float scalar is rejected", func(t *testing.T) {
		err := requireInteger("timeout", "an integer", mappingNode(t, "timeout: 1.5"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), `got "1.5"`)
	})
}

// Each Build method must surface the node-kind failure for its own 1.1 fields.
func TestBuildRejectsMalformed11FieldNodeKinds(t *testing.T) {
	t.Run("arazzo $self", func(t *testing.T) {
		root := mappingNode(t, "arazzo: 1.1.0\n$self:\n  not: scalar")
		doc := new(Arazzo)
		require.NoError(t, low.BuildModel(root, doc))
		err := doc.Build(context.Background(), nil, root, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "$self")
		assert.Contains(t, err.Error(), "must be a scalar URI")
	})

	stepCases := []struct {
		name    string
		src     string
		wantSub string
	}{
		{"channelPath", "stepId: s\nchannelPath:\n  not: scalar", "a scalar channel reference"},
		{"action", "stepId: s\naction:\n  - not-scalar", "a scalar action name"},
		{"correlationId", "stepId: s\ncorrelationId:\n  not: scalar", "a scalar string"},
		{"timeout", "stepId: s\ntimeout: soon", "an integer number of milliseconds"},
		{"dependsOn", "stepId: s\ndependsOn:\n  not: sequence", "a sequence of step identifiers"},
	}
	for _, test := range stepCases {
		t.Run("step "+test.name, func(t *testing.T) {
			root := mappingNode(t, test.src)
			step := new(Step)
			require.NoError(t, low.BuildModel(root, step))
			err := step.Build(context.Background(), nil, root, nil)
			require.Error(t, err)
			assert.Contains(t, err.Error(), test.name)
			assert.Contains(t, err.Error(), test.wantSub)
		})
	}

	t.Run("success action parameters", func(t *testing.T) {
		root := mappingNode(t, "name: n\ntype: goto\nparameters: not-a-sequence")
		action := new(SuccessAction)
		require.NoError(t, low.BuildModel(root, action))
		err := action.Build(context.Background(), nil, root, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "a sequence of Parameter Objects")
	})

	t.Run("failure action parameters", func(t *testing.T) {
		root := mappingNode(t, "name: n\ntype: goto\nparameters: not-a-sequence")
		action := new(FailureAction)
		require.NoError(t, low.BuildModel(root, action))
		err := action.Build(context.Background(), nil, root, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "a sequence of Parameter Objects")
	})
}

func TestBuildRejectsMalformed11CollectionMembers(t *testing.T) {
	t.Run("step dependsOn member must be scalar", func(t *testing.T) {
		root := mappingNode(t, "stepId: s\ndependsOn:\n  - nested: value")
		step := new(Step)
		require.NoError(t, low.BuildModel(root, step))
		err := step.Build(context.Background(), nil, root, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "dependsOn item at line 3, column 5 must be a scalar step identifier")
	})

	t.Run("workflow collections require their authored container kinds", func(t *testing.T) {
		for _, test := range []struct {
			name string
			src  string
			want string
		}{
			{name: "success actions", src: "workflowId: w\nsuccessActions: wrong", want: "successActions"},
			{name: "failure actions", src: "workflowId: w\nfailureActions: wrong", want: "failureActions"},
			{name: "parameters", src: "workflowId: w\nparameters: wrong", want: "parameters"},
			{name: "outputs", src: "workflowId: w\noutputs: []", want: "outputs"},
		} {
			t.Run(test.name, func(t *testing.T) {
				root := mappingNode(t, test.src)
				workflow := new(Workflow)
				require.NoError(t, low.BuildModel(root, workflow))
				err := workflow.Build(context.Background(), nil, root, nil)
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.want)
			})
		}
	})

	t.Run("object array member must be mapping", func(t *testing.T) {
		root := mappingNode(t, "name: done\ntype: end\nparameters:\n  - wrong")
		action := new(SuccessAction)
		require.NoError(t, low.BuildModel(root, action))
		err := action.Build(context.Background(), nil, root, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parameters item")
		assert.Contains(t, err.Error(), "must be a mapping")
	})
}

func TestOutputValueBuildResolvesAliasesWithoutLosingAuthoredNode(t *testing.T) {
	root := mappingNode(t, "expression: &expression $inputs.id\nexpressionAlias: *expression\nselector: &selector\n  context: $response.body\n  selector: $.id\n  type: jsonpath\nselectorAlias: *selector")

	expressionAlias := root.Content[3]
	expression := new(OutputValue)
	require.NoError(t, expression.Build(context.Background(), root.Content[2], expressionAlias, nil))
	assert.True(t, expression.IsExpression())
	assert.Equal(t, "$inputs.id", expression.Expression.Value)
	assert.Same(t, expressionAlias, expression.RootNode)
	assert.Same(t, expressionAlias, expression.Expression.ValueNode)

	selectorAlias := root.Content[7]
	selector := new(OutputValue)
	require.NoError(t, selector.Build(context.Background(), root.Content[6], selectorAlias, nil))
	assert.True(t, selector.IsSelector())
	assert.Equal(t, "$.id", selector.Selector.Value.Selector.Value)
	assert.Same(t, selectorAlias, selector.RootNode)
	assert.Same(t, selectorAlias, selector.Selector.ValueNode)
}

func TestOutputValueBuildRejectsCyclicOrEmptyAliases(t *testing.T) {
	cyclic := &yaml.Node{Kind: yaml.AliasNode, Line: 7, Column: 11}
	cyclic.Alias = cyclic
	value := new(OutputValue)
	err := value.Build(context.Background(), nil, cyclic, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cyclic YAML alias")

	empty := &yaml.Node{Kind: yaml.AliasNode, Line: 8, Column: 12}
	err = value.Build(context.Background(), nil, empty, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty YAML alias")
}

func TestStrictCollectionExtractorsCoverNullAndAliasFailures(t *testing.T) {
	cyclic := func(line int) *yaml.Node {
		node := &yaml.Node{Kind: yaml.AliasNode, Line: line, Column: 3}
		node.Alias = node
		return node
	}
	rootWith := func(label string, value *yaml.Node) *yaml.Node {
		return &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: label}, value,
		}}
	}

	t.Run("resolver accepts nil", func(t *testing.T) {
		resolved, err := resolveAliasNode(nil)
		require.NoError(t, err)
		assert.Nil(t, resolved)
	})

	t.Run("aliases to null are absent consistently", func(t *testing.T) {
		nullNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null"}
		alias := &yaml.Node{Kind: yaml.AliasNode, Alias: nullNode, Line: 2, Column: 3}
		root := rootWith("field", alias)
		array, err := extractArray[Parameter](context.Background(), "field", root, nil)
		require.NoError(t, err)
		assert.Nil(t, array.Value)
		strings, err := extractStringArray("field", root)
		require.NoError(t, err)
		assert.Nil(t, strings.Value)
		outputs, err := extractOutputValuesMap(context.Background(), "field", root, nil)
		require.NoError(t, err)
		assert.Nil(t, outputs.Value)
		assert.NoError(t, requireNodeKind("field", "a mapping", root, yaml.MappingNode))
		assert.NoError(t, requireInteger("field", "an integer", root))
	})

	t.Run("object array null is absent", func(t *testing.T) {
		result, err := extractArray[Parameter](context.Background(), ParametersLabel,
			mappingNode(t, "parameters: null"), nil)
		require.NoError(t, err)
		assert.Nil(t, result.Value)
	})

	t.Run("object array rejects cyclic container alias", func(t *testing.T) {
		_, err := extractArray[Parameter](context.Background(), ParametersLabel,
			rootWith(ParametersLabel, cyclic(2)), nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parameters: cyclic YAML alias")
	})

	t.Run("object array rejects cyclic member alias", func(t *testing.T) {
		sequence := &yaml.Node{Kind: yaml.SequenceNode, Content: []*yaml.Node{cyclic(3)}}
		_, err := extractArray[Parameter](context.Background(), ParametersLabel,
			rootWith(ParametersLabel, sequence), nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parameters item: cyclic YAML alias")
	})

	t.Run("string array null is absent", func(t *testing.T) {
		result, err := extractStringArray(DependsOnLabel, mappingNode(t, "dependsOn: null"))
		require.NoError(t, err)
		assert.Nil(t, result.Value)
	})

	t.Run("string array rejects cyclic container alias", func(t *testing.T) {
		_, err := extractStringArray(DependsOnLabel, rootWith(DependsOnLabel, cyclic(2)))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "dependsOn: cyclic YAML alias")
	})

	t.Run("string array rejects cyclic member alias", func(t *testing.T) {
		sequence := &yaml.Node{Kind: yaml.SequenceNode, Content: []*yaml.Node{cyclic(3)}}
		_, err := extractStringArray(DependsOnLabel, rootWith(DependsOnLabel, sequence))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "dependsOn item: cyclic YAML alias")
	})

	t.Run("output map null is absent", func(t *testing.T) {
		result, err := extractOutputValuesMap(context.Background(), OutputsLabel,
			mappingNode(t, "outputs: null"), nil)
		require.NoError(t, err)
		assert.Nil(t, result.Value)
	})

	t.Run("output map rejects cyclic container alias", func(t *testing.T) {
		_, err := extractOutputValuesMap(context.Background(), OutputsLabel,
			rootWith(OutputsLabel, cyclic(2)), nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "outputs: cyclic YAML alias")
	})

	t.Run("kind and integer guards reject cyclic aliases", func(t *testing.T) {
		root := rootWith("field", cyclic(2))
		assert.Error(t, requireNodeKind("field", "a scalar", root, yaml.ScalarNode))
		assert.Error(t, requireInteger("field", "an integer", root))
	})

	t.Run("workflow propagates malformed dependsOn member", func(t *testing.T) {
		root := mappingNode(t, "workflowId: w\ndependsOn:\n  - nested: value")
		workflow := new(Workflow)
		require.NoError(t, low.BuildModel(root, workflow))
		err := workflow.Build(context.Background(), nil, root, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "dependsOn item")
	})
}

func TestComponentCollectionsRejectMalformedShapesAndResolveAliases(t *testing.T) {
	t.Run("containers must be mappings", func(t *testing.T) {
		for _, field := range []string{InputsLabel, ParametersLabel, SuccessActionsLabel, FailureActionsLabel} {
			root := mappingNode(t, field+": wrong")
			components := new(Components)
			require.NoError(t, low.BuildModel(root, components))
			err := components.Build(context.Background(), nil, root, nil)
			require.Error(t, err)
			assert.Contains(t, err.Error(), field)
			assert.Contains(t, err.Error(), "must be a mapping")
		}
	})

	t.Run("object members must be mappings", func(t *testing.T) {
		root := mappingNode(t, "parameters:\n  bad: wrong")
		components := new(Components)
		require.NoError(t, low.BuildModel(root, components))
		err := components.Build(context.Background(), nil, root, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parameters member")
		assert.Contains(t, err.Error(), "must be a mapping")
	})

	t.Run("mapping container and member aliases are accepted", func(t *testing.T) {
		root := mappingNode(t, "parameter: &parameter\n  name: id\n  in: path\n  value: 1\nparameterMap: &parameterMap\n  id: *parameter\nparameters: *parameterMap")
		components := new(Components)
		require.NoError(t, low.BuildModel(root, components))
		require.NoError(t, components.Build(context.Background(), nil, root, nil))
		require.NotNil(t, components.Parameters.Value)
		pair := components.Parameters.Value.First()
		require.NotNil(t, pair)
		assert.Equal(t, "id", pair.Value().Value.Name.Value)
	})
}

func TestAliasedSelectorUnionMembersBuildAndPreserveAuthoredNodes(t *testing.T) {
	root := mappingNode(t, "scalarType: &scalarType jsonpath\nobjectType: &objectType\n  type: jsonpath\n  version: rfc9535\ncontextValue: &contextValue $response.body\nselectorValue: &selectorValue $.id\ntargetValue: &targetValue /id\nselector:\n  context: *contextValue\n  selector: *selectorValue\n  type: *scalarType\nreplacement:\n  target: *targetValue\n  targetSelectorType: *objectType\n  value: 1")

	selectorNode := root.Content[11]
	selector := new(Selector)
	require.NoError(t, low.BuildModel(selectorNode, selector))
	require.NoError(t, selector.Build(context.Background(), root.Content[10], selectorNode, nil))
	assert.Equal(t, "$response.body", selector.Context.Value)
	assert.Equal(t, "$.id", selector.Selector.Value)
	assert.Equal(t, yaml.AliasNode, selector.Context.ValueNode.Kind)
	assert.Equal(t, yaml.AliasNode, selector.Selector.ValueNode.Kind)
	require.Equal(t, yaml.AliasNode, selector.Type.Value.Kind)
	resolvedType, err := resolveAliasNode(selector.Type.Value)
	require.NoError(t, err)
	assert.Equal(t, "jsonpath", resolvedType.Value)

	replacementNode := root.Content[13]
	replacement := new(PayloadReplacement)
	require.NoError(t, low.BuildModel(replacementNode, replacement))
	require.NoError(t, replacement.Build(context.Background(), root.Content[12], replacementNode, nil))
	assert.Equal(t, "/id", replacement.Target.Value)
	assert.Equal(t, yaml.AliasNode, replacement.Target.ValueNode.Kind)
	require.Equal(t, yaml.AliasNode, replacement.TargetSelectorType.Value.Kind)
	resolvedType, err = resolveAliasNode(replacement.TargetSelectorType.Value)
	require.NoError(t, err)
	assert.Equal(t, yaml.MappingNode, resolvedType.Kind)
}

func TestHashYAMLNodeIncludesContentBelowFormerDepthLimit(t *testing.T) {
	deepNode := func(leaf string) *yaml.Node {
		var node *yaml.Node = &yaml.Node{Kind: yaml.ScalarNode, Value: leaf}
		for range 110 {
			node = &yaml.Node{Kind: yaml.SequenceNode, Content: []*yaml.Node{node}}
		}
		return node
	}
	hash := func(node *yaml.Node) uint64 {
		return low.WithHasher(func(h *maphash.Hash) uint64 {
			hashYAMLNode(h, node)
			return h.Sum64()
		})
	}
	assert.NotEqual(t, hash(deepNode("left")), hash(deepNode("right")))
}

func TestAliasAwareCollectionHelpersCoverMalformedAndNullInputs(t *testing.T) {
	t.Run("scalar extraction", func(t *testing.T) {
		missing, err := extractScalarString("field", mappingNode(t, "other: value"))
		require.NoError(t, err)
		assert.True(t, missing.IsEmpty())

		nullValue, err := extractScalarString("field", mappingNode(t, "field: null"))
		require.NoError(t, err)
		assert.Empty(t, nullValue.Value)

		_, err = extractScalarString("field", mappingNode(t, "field:\n  nested: value"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be a scalar")

		cyclic := &yaml.Node{Kind: yaml.AliasNode, Line: 4, Column: 7}
		cyclic.Alias = cyclic
		root := &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "field"}, cyclic,
		}}
		_, err = extractScalarString("field", root)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cyclic YAML alias")
	})

	t.Run("object map extraction", func(t *testing.T) {
		nullValue, err := extractObjectMap[Parameter](context.Background(), ParametersLabel,
			mappingNode(t, "parameters: null"), nil)
		require.NoError(t, err)
		assert.Nil(t, nullValue.Value)

		cyclic := &yaml.Node{Kind: yaml.AliasNode, Line: 5, Column: 9}
		cyclic.Alias = cyclic
		container := &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: ParametersLabel}, cyclic,
		}}
		_, err = extractObjectMap[Parameter](context.Background(), ParametersLabel, container, nil)
		require.Error(t, err)

		memberMap := &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "bad"}, cyclic,
		}}
		container.Content[1] = memberMap
		_, err = extractObjectMap[Parameter](context.Background(), ParametersLabel, container, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `member "bad"`)
	})

	t.Run("raw map extraction", func(t *testing.T) {
		nullValue, err := extractRawNodeMap(InputsLabel, mappingNode(t, "inputs: null"))
		require.NoError(t, err)
		assert.Nil(t, nullValue.Value)

		cyclic := &yaml.Node{Kind: yaml.AliasNode, Line: 6, Column: 11}
		cyclic.Alias = cyclic
		root := &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: InputsLabel}, cyclic,
		}}
		_, err = extractRawNodeMap(InputsLabel, root)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cyclic YAML alias")
	})
}

func TestSelectorAndPayloadReplacementSurfaceAliasExtractionErrors(t *testing.T) {
	for _, test := range []struct {
		name  string
		field string
		build func(*yaml.Node) error
	}{
		{
			name: "selector context", field: ContextLabel,
			build: func(root *yaml.Node) error {
				return new(Selector).Build(context.Background(), nil, root, nil)
			},
		},
		{
			name: "selector selector", field: SelectorLabel,
			build: func(root *yaml.Node) error {
				return new(Selector).Build(context.Background(), nil, root, nil)
			},
		},
		{
			name: "selector type", field: TypeLabel,
			build: func(root *yaml.Node) error {
				return new(Selector).Build(context.Background(), nil, root, nil)
			},
		},
		{
			name: "replacement target", field: TargetLabel,
			build: func(root *yaml.Node) error {
				return new(PayloadReplacement).Build(context.Background(), nil, root, nil)
			},
		},
		{
			name: "replacement selector type", field: TargetSelectorTypeLabel,
			build: func(root *yaml.Node) error {
				return new(PayloadReplacement).Build(context.Background(), nil, root, nil)
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cyclic := &yaml.Node{Kind: yaml.AliasNode, Line: 8, Column: 13}
			cyclic.Alias = cyclic
			root := &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: test.field}, cyclic,
			}}
			err := test.build(root)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "cyclic YAML alias")
		})
	}
}

// A quoted integer is a well-formed scalar that parses as an integer, but the model
// builder gates on the YAML tag and drops it. requireInteger must gate on the same
// predicate so the value is reported rather than silently lost.
func TestRequireInteger_RejectsQuotedIntegerTheBuilderWouldDrop(t *testing.T) {
	err := requireInteger("timeout", "an integer", mappingNode(t, `timeout: "5000"`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), `must be an integer, got "5000"`)
}

// Booleans and other non-integer tags are rejected for the same reason.
func TestRequireInteger_RejectsNonIntegerTaggedScalars(t *testing.T) {
	for _, src := range []string{"timeout: true", `timeout: "abc"`, "timeout: 1.5"} {
		err := requireInteger("timeout", "an integer", mappingNode(t, src))
		assert.Error(t, err, src)
	}
}

// Hex, octal and underscore-separated literals are tagged !!int by the YAML parser, so
// they pass the tag gate, but the builder parses base 10 and would coerce them to zero.
// Reporting them is better than silently storing the wrong number.
func TestRequireInteger_RejectsIntTaggedValuesTheBuilderWouldCoerceToZero(t *testing.T) {
	for _, src := range []string{"timeout: 0x1F", "timeout: 0o17", "timeout: 1_000"} {
		err := requireInteger("timeout", "an integer", mappingNode(t, src))
		require.Error(t, err, src)
		assert.Contains(t, err.Error(), "must be an integer, got")
	}
}

// A leading plus is a valid base-10 integer and must be accepted.
func TestRequireInteger_AcceptsExplicitlySignedInteger(t *testing.T) {
	assert.NoError(t, requireInteger("timeout", "an integer", mappingNode(t, "timeout: +5")))
}

// The hash must encode node kind. Without it a mapping and a sequence holding the same
// scalars are indistinguishable, so structurally different values compare as equal.
func TestHashYAMLNode_DistinguishesMappingFromSequence(t *testing.T) {
	build := func(src string) uint64 {
		root := mappingNode(t, src)
		s := new(Selector)
		require.NoError(t, low.BuildModel(root, s))
		require.NoError(t, s.Build(context.Background(), nil, root, nil))
		return s.Hash()
	}

	mapForm := build("context: $response.body\nselector: $.id\nx-e:\n  a: b")
	seqForm := build("context: $response.body\nselector: $.id\nx-e:\n  - a\n  - b")
	assert.NotEqual(t, mapForm, seqForm,
		"a mapping and a sequence of the same scalars must not hash alike")

	nestedMap := build("context: $c\nselector: $.id\nx-e:\n  a:\n    b: c")
	flatSeq := build("context: $c\nselector: $.id\nx-e:\n  - a\n  - b\n  - c")
	assert.NotEqual(t, nestedMap, flatSeq)
}

// A YAML alias may point back into its own ancestor, which parses without error. Hashing
// must terminate rather than recurse until the stack gives out.
func TestHashYAMLNode_TerminatesOnSelfReferentialAlias(t *testing.T) {
	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("context: $c\nselector: $.id\nx-e: &loop [*loop]"), &node))

	s := new(Selector)
	require.NoError(t, low.BuildModel(node.Content[0], s))
	require.NoError(t, s.Build(context.Background(), nil, node.Content[0], nil))

	done := make(chan uint64, 1)
	go func() { done <- s.Hash() }()
	select {
	case h := <-done:
		assert.NotZero(t, h)
	case <-time.After(5 * time.Second):
		t.Fatal("hashing a self-referential alias did not terminate")
	}
}
