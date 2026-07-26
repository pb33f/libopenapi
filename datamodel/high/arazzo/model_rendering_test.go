// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	lowmodel "github.com/pb33f/libopenapi/datamodel/low"
	low "github.com/pb33f/libopenapi/datamodel/low/arazzo"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"go.yaml.in/yaml/v4"
)

const arazzo11HighYAML = `arazzo: 1.1.0
$self: https://example.com/workflows/root.arazzo.yaml
info:
  title: Arazzo 1.1
  version: 1.0.0
sourceDescriptions:
  - name: messages
    url: asyncapi.yaml
    type: asyncapi
workflows:
  - workflowId: publish
    steps:
      - stepId: send
        channelPath: $sourceDescriptions.messages#/channels/orders
        action: send
        correlationId: order-id
        timeout: 2500
        dependsOn: [prepare]
        requestBody:
          replacements:
            - target: $.id
              targetSelectorType:
                type: jsonpath
                version: rfc9535
              value:
                context: $inputs
                selector: $.id
                type: jsonpath
        onSuccess:
          - name: done
            type: end
            parameters:
              - name: result
                value: $outputs.result
        onFailure:
          - name: retry
            type: retry
            retryLimit: 1
            parameters:
              - name: reason
                value: failed
        outputs:
          selected:
            context: $response.body
            selector: $.id
            type:
              type: jsonpath
              version: rfc9535
    outputs:
      status: $steps.send.outputs.selected
`

func TestArazzo11HighModel_RoundTripAndTypedAccess(t *testing.T) {
	document := buildHighArazzo(t, arazzo11HighYAML)
	assert.Equal(t, "https://example.com/workflows/root.arazzo.yaml", document.Self)
	step := document.Workflows[0].Steps[0]
	assert.Equal(t, "$sourceDescriptions.messages#/channels/orders", step.ChannelPath)
	assert.Equal(t, "send", step.Action)
	assert.Equal(t, "order-id", step.CorrelationId)
	require.NotNil(t, step.Timeout)
	assert.EqualValues(t, 2500, *step.Timeout)
	assert.Equal(t, []string{"prepare"}, step.DependsOn)
	require.Len(t, step.OnSuccess[0].Parameters, 1)
	require.Len(t, step.OnFailure[0].Parameters, 1)

	output, ok := step.Outputs.Get("selected")
	require.True(t, ok)
	assert.True(t, output.IsSelector())
	assert.False(t, output.IsExpression())
	selector, ok := output.GetSelector()
	require.True(t, ok)
	assert.Equal(t, "jsonpath", selector.GetEffectiveType())
	assert.Equal(t, "rfc9535", selector.ExpressionType.Version)
	assert.NotNil(t, selector.GoLow())
	assert.NotNil(t, selector.GoLowUntyped())
	assert.NotNil(t, output.GoLow())
	assert.NotNil(t, output.GoLowUntyped())

	workflowOutput, ok := document.Workflows[0].Outputs.Get("status")
	require.True(t, ok)
	expression, ok := workflowOutput.GetExpression()
	require.True(t, ok)
	assert.Equal(t, "$steps.send.outputs.selected", expression)
	assert.False(t, workflowOutput.IsSelector())
	_, ok = workflowOutput.GetSelector()
	assert.False(t, ok)

	replacement := step.RequestBody.Replacements[0]
	assert.Equal(t, "jsonpath", replacement.TargetSelectorExpressionType.Type)
	assert.True(t, replacement.IsSelector())
	assert.False(t, replacement.IsRuntimeExpression())
	replacementSelector, ok := replacement.GetSelector()
	require.True(t, ok)
	assert.Equal(t, "$.id", replacementSelector.Selector)
	assert.Same(t, replacement.Value, replacement.GetValueNode())

	rendered, err := document.Render()
	require.NoError(t, err)
	assert.Contains(t, string(rendered), "$self: https://example.com/workflows/root.arazzo.yaml")
	assert.Contains(t, string(rendered), "channelPath:")
	assert.Contains(t, string(rendered), "targetSelectorType:")

	reloaded := buildHighArazzo(t, string(rendered))
	assert.Equal(t, document.Self, reloaded.Self)
	assert.True(t, reloaded.Workflows[0].Steps[0].Outputs.First().Value().IsSelector())
}

func TestCriterionBuildsAliasedTypeVariants(t *testing.T) {
	for _, test := range []struct {
		name           string
		source         string
		wantType       string
		wantObjectType string
		wantVersion    string
	}{
		{
			name:     "scalar",
			source:   "typeValue: &typeValue jsonpath\ncriterion:\n  context: $response.body\n  condition: $.items\n  type: *typeValue",
			wantType: "jsonpath",
		},
		{
			name:           "expression type object",
			source:         "typeValue: &typeValue\n  type: jsonpath\n  version: rfc9535\ncriterion:\n  context: $response.body\n  condition: $.items\n  type: *typeValue",
			wantObjectType: "jsonpath",
			wantVersion:    "rfc9535",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var document yaml.Node
			require.NoError(t, yaml.Unmarshal([]byte(test.source), &document))
			root := document.Content[0]
			criterionNode := root.Content[3]
			lowCriterion := new(low.Criterion)
			require.NoError(t, lowmodel.BuildModel(criterionNode, lowCriterion))
			require.NoError(t, lowCriterion.Build(context.Background(), root.Content[2], criterionNode, nil))

			criterion := NewCriterion(lowCriterion)
			assert.Equal(t, test.wantType, criterion.Type)
			if test.wantObjectType == "" {
				assert.Nil(t, criterion.ExpressionType)
				return
			}
			require.NotNil(t, criterion.ExpressionType)
			assert.Equal(t, test.wantObjectType, criterion.ExpressionType.Type)
			assert.Equal(t, test.wantVersion, criterion.ExpressionType.Version)
		})
	}
}

func TestArazzo11RootRenderingPreservesAuthoredSelfOrder(t *testing.T) {
	document := buildHighArazzo(t, `info:
  title: ordered
  version: 1.0.0
x-before: true
arazzo: 1.1.0
$self: https://example.com/root.yaml
sourceDescriptions:
  - name: source
    url: openapi.yaml
workflows:
  - workflowId: workflow
    steps: []
`)
	rendered, err := document.Render()
	require.NoError(t, err)
	text := string(rendered)
	infoOffset := strings.Index(text, "info:")
	extensionOffset := strings.Index(text, "x-before:")
	arazzoOffset := strings.Index(text, "arazzo:")
	selfOffset := strings.Index(text, "$self:")
	sourceOffset := strings.Index(text, "sourceDescriptions:")
	assert.True(t,
		infoOffset < extensionOffset &&
			extensionOffset < arazzoOffset &&
			arazzoOffset < selfOffset &&
			selfOffset < sourceOffset,
		text,
	)

	document.Self = ""
	rendered, err = document.Render()
	require.NoError(t, err)
	assert.NotContains(t, string(rendered), "$self:")
}

func TestSelector_ScalarAndObjectRendering(t *testing.T) {
	scalar := &Selector{Context: "$inputs", Selector: "$.id", Type: "jsonpath"}
	assert.Equal(t, "jsonpath", scalar.GetEffectiveType())
	rendered, err := scalar.Render()
	require.NoError(t, err)
	assert.Contains(t, string(rendered), "type: jsonpath")
	jsonBytes, err := json.Marshal(scalar)
	require.NoError(t, err)
	assert.JSONEq(t, `{"context":"$inputs","selector":"$.id","type":"jsonpath"}`, string(jsonBytes))

	object := &Selector{
		Context:  "$response.body",
		Selector: "/id",
		ExpressionType: &ExpressionType{
			Type:    "jsonpointer",
			Version: "rfc6901",
		},
	}
	assert.Equal(t, "jsonpointer", object.GetEffectiveType())
	jsonBytes, err = json.Marshal(object)
	require.NoError(t, err)
	assert.JSONEq(t, `{"context":"$response.body","selector":"/id","type":{"type":"jsonpointer","version":"rfc6901"}}`, string(jsonBytes))

	invalid := &Selector{Type: "jsonpath", ExpressionType: &ExpressionType{Type: "xpath"}}
	_, err = invalid.MarshalYAML()
	require.Error(t, err)
	_, err = invalid.MarshalJSON()
	require.Error(t, err)
}

func TestOutputValue_AllVariantsAndInvalidStates(t *testing.T) {
	expression := NewExpressionOutputValue("$statusCode")
	value, ok := expression.GetExpression()
	require.True(t, ok)
	assert.Equal(t, "$statusCode", value)
	rendered, err := expression.Render()
	require.NoError(t, err)
	assert.Equal(t, "$statusCode\n", string(rendered))
	jsonBytes, err := expression.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, `"$statusCode"`, string(jsonBytes))

	emptyExpression := NewExpressionOutputValue("")
	assert.True(t, emptyExpression.IsExpression())
	rendered, err = emptyExpression.Render()
	require.NoError(t, err)
	assert.Equal(t, `""`+"\n", string(rendered))

	selector := NewSelectorOutputValue(&Selector{Context: "$response.body", Selector: "$.id", Type: "jsonpath"})
	require.True(t, selector.IsSelector())
	_, ok = selector.GetExpression()
	assert.False(t, ok)
	rendered, err = selector.Render()
	require.NoError(t, err)
	assert.Contains(t, string(rendered), "selector: $.id")
	jsonBytes, err = selector.MarshalJSON()
	require.NoError(t, err)
	assert.JSONEq(t, `{"context":"$response.body","selector":"$.id","type":"jsonpath"}`, string(jsonBytes))

	for _, invalid := range []*OutputValue{
		nil,
		{},
		NewSelectorOutputValue(nil),
	} {
		_, err = invalid.MarshalYAML()
		require.Error(t, err)
		_, err = invalid.MarshalJSON()
		require.Error(t, err)
	}

	expression.SetSelector(&Selector{Context: "$response.body", Selector: "$.id", Type: "jsonpath"})
	assert.True(t, expression.IsSelector())
	_, ok = expression.GetExpression()
	assert.False(t, ok)

	expression.SetExpression("")
	assert.True(t, expression.IsExpression())
	value, ok = expression.GetExpression()
	assert.True(t, ok)
	assert.Empty(t, value)

	expression.SetSelector(nil)
	assert.False(t, expression.IsExpression())
	assert.False(t, expression.IsSelector())
}

func TestPayloadReplacement_TypedInspectionAndTypeUnion(t *testing.T) {
	var nilReplacement *PayloadReplacement
	assert.False(t, nilReplacement.IsRuntimeExpression())
	assert.False(t, nilReplacement.IsSelector())
	assert.Nil(t, nilReplacement.GetValueNode())
	_, ok := nilReplacement.GetSelector()
	assert.False(t, ok)

	runtimeNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "{$inputs.id}"}
	replacement := &PayloadReplacement{
		Target:             "/id",
		TargetSelectorType: "jsonpointer",
		Value:              runtimeNode,
	}
	assert.True(t, replacement.IsRuntimeExpression())
	rendered, err := replacement.Render()
	require.NoError(t, err)
	assert.Contains(t, string(rendered), "targetSelectorType: jsonpointer")

	literal := &PayloadReplacement{Value: &yaml.Node{Kind: yaml.ScalarNode, Value: "literal"}}
	assert.False(t, literal.IsRuntimeExpression())

	invalid := &PayloadReplacement{
		TargetSelectorType:           "jsonpath",
		TargetSelectorExpressionType: &ExpressionType{Type: "jsonpath", Version: "rfc9535"},
	}
	_, err = invalid.MarshalYAML()
	require.Error(t, err)

	scalarTypeNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "jsonpath"}
	lowReplacement := &low.PayloadReplacement{
		TargetSelectorType: lowmodel.NodeReference[*yaml.Node]{
			Value:     scalarTypeNode,
			ValueNode: scalarTypeNode,
		},
	}
	highReplacement := NewPayloadReplacement(lowReplacement)
	assert.Equal(t, "jsonpath", highReplacement.TargetSelectorType)

	var selectorValue yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("context: $inputs\nselector: $.id\ntype: jsonpath"), &selectorValue))
	highReplacement.Value = selectorValue.Content[0]
	assert.True(t, highReplacement.IsSelector())
	highReplacement.Value = &yaml.Node{Kind: yaml.ScalarNode, Value: "mutated"}
	assert.False(t, highReplacement.IsSelector())
	_, ok = highReplacement.GetSelector()
	assert.False(t, ok)
}

func TestArazzo11BuildHelpers_Boundaries(t *testing.T) {
	assert.Nil(t, buildOutputValueMap(nil))
	assert.Nil(t, selectorFromNode(nil))
	assert.Nil(t, selectorFromNode(&yaml.Node{Kind: yaml.ScalarNode, Value: "no"}))

	var missing yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("context: $inputs"), &missing))
	assert.Nil(t, selectorFromNode(missing.Content[0]))

	var malformed yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("context: $inputs\nselector: $.id\ntype: [jsonpath]"), &malformed))
	assert.Nil(t, selectorFromNode(malformed.Content[0]))
	require.NoError(t, yaml.Unmarshal([]byte("context: [bad]\nselector: $.id\ntype: jsonpath"), &malformed))
	assert.Nil(t, selectorFromNode(malformed.Content[0]))
	require.NoError(t, yaml.Unmarshal([]byte("context: $inputs\nselector: [bad]\ntype: jsonpath"), &malformed))
	assert.Nil(t, selectorFromNode(malformed.Content[0]))

	empty := orderedmap.New[string, *OutputValue]()
	assert.NotNil(t, empty)

	assert.Nil(t, selectorsFromNode(nil))
	var document yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("- plain\n- context: $inputs\n  selector: $.id\n  type: jsonpath"), &document))
	selectors := selectorsFromNode(&document)
	require.Len(t, selectors, 1)
	assert.Equal(t, "$.id", selectors[0].Selector)
}

func TestSelectorHelpersResolveAliasesAndTerminateCycles(t *testing.T) {
	var document yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(`selector: &selector
  context: $inputs
  selector: $.id
  type: jsonpath
alias: *selector
`), &document))
	root := document.Content[0]
	alias := root.Content[3]
	selector := selectorFromNode(alias)
	require.NotNil(t, selector)
	assert.Equal(t, "$.id", selector.Selector)

	cyclic := &yaml.Node{Kind: yaml.MappingNode}
	aliasCycle := &yaml.Node{Kind: yaml.AliasNode, Alias: cyclic}
	cyclic.Content = []*yaml.Node{
		{Kind: yaml.ScalarNode, Value: "selector"}, alias,
		{Kind: yaml.ScalarNode, Value: "cycle"}, aliasCycle,
	}
	selectors := selectorsFromNode(cyclic)
	require.Len(t, selectors, 1)
	assert.Equal(t, "$.id", selectors[0].Selector)

	selfAlias := &yaml.Node{Kind: yaml.AliasNode}
	selfAlias.Alias = selfAlias
	assert.Nil(t, selectorFromNode(selfAlias))
	assert.Nil(t, selectorsFromNode(selfAlias))
	assert.Nil(t, selectorFromNode(&yaml.Node{Kind: yaml.AliasNode}))
}

func TestRequestBodyAndReplacement_DiscoverNestedSelectors(t *testing.T) {
	var payload yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(`items:
  - literal
  - selected:
      context: $inputs
      selector: $.id
      type: jsonpath
second:
  context: $response.body
  selector: /id
  type: jsonpointer
`), &payload))
	requestBody := &RequestBody{Payload: payload.Content[0]}
	selectors := requestBody.GetSelectors()
	require.Len(t, selectors, 2)
	assert.Equal(t, "$.id", selectors[0].Selector)
	assert.Equal(t, "/id", selectors[1].Selector)
	var nilRequestBody *RequestBody
	assert.Nil(t, nilRequestBody.GetSelectors())

	replacement := &PayloadReplacement{Value: payload.Content[0]}
	require.Len(t, replacement.GetSelectors(), 2)
	var nilReplacement *PayloadReplacement
	assert.Nil(t, nilReplacement.GetSelectors())
}

func TestParameter_TypedValueInspection(t *testing.T) {
	var nilParameter *Parameter
	assert.False(t, nilParameter.IsRuntimeExpression())
	assert.False(t, nilParameter.IsSelector())
	assert.Nil(t, nilParameter.GetValueNode())
	_, ok := nilParameter.GetSelector()
	assert.False(t, ok)

	runtime := &Parameter{Value: &yaml.Node{Kind: yaml.ScalarNode, Value: "$inputs.id"}}
	assert.True(t, runtime.IsRuntimeExpression())
	assert.Same(t, runtime.Value, runtime.GetValueNode())

	literal := &Parameter{Value: &yaml.Node{Kind: yaml.ScalarNode, Value: "literal"}}
	assert.False(t, literal.IsRuntimeExpression())

	var selectorNode yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("context: $inputs\nselector: $.id\ntype: jsonpath"), &selectorNode))
	lowParameter := &low.Parameter{
		Value: lowmodel.NodeReference[*yaml.Node]{
			Value:     selectorNode.Content[0],
			ValueNode: selectorNode.Content[0],
		},
	}
	parameter := NewParameter(lowParameter)
	assert.True(t, parameter.IsSelector())
	selector, ok := parameter.GetSelector()
	require.True(t, ok)
	assert.Equal(t, "$.id", selector.Selector)
	parameter.Value = &yaml.Node{Kind: yaml.ScalarNode, Value: "mutated"}
	assert.False(t, parameter.IsSelector())
	_, ok = parameter.GetSelector()
	assert.False(t, ok)
}
