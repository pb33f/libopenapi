// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"context"
	"errors"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/low"
	"github.com/pb33f/libopenapi/index"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

type gapBadArrayModel struct {
	Bad chan int
}

func (g *gapBadArrayModel) Build(context.Context, *yaml.Node, *yaml.Node, *index.SpecIndex) error {
	return nil
}

type gapBuildErrorModel struct{}

func (g *gapBuildErrorModel) Build(context.Context, *yaml.Node, *yaml.Node, *index.SpecIndex) error {
	return errors.New("build boom")
}

func parseYAMLNode(t *testing.T, yml string) (*yaml.Node, *yaml.Node) {
	t.Helper()
	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yml), &node))
	require.NotEmpty(t, node.Content)
	return &node, node.Content[0]
}

func mapRootNode(t *testing.T, yml string) *yaml.Node {
	_, root := parseYAMLNode(t, yml)
	return root
}

func TestGap_ExtractArray_BuildModelError(t *testing.T) {
	root := mapRootNode(t, `items:
  - bad: value`)

	_, err := extractArray[gapBadArrayModel](context.Background(), "items", root, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported type")
}

func TestGap_AssignNodeReference(t *testing.T) {
	called := false
	ref := low.NodeReference[string]{Value: "ok"}
	err := assignNodeReference(ref, nil, func(v low.NodeReference[string]) {
		called = true
		assert.Equal(t, "ok", v.Value)
	})
	require.NoError(t, err)
	assert.True(t, called)

	err = assignNodeReference(ref, errors.New("boom"), func(low.NodeReference[string]) {
		t.Fatal("assign should not be called on error")
	})
	require.Error(t, err)
}

func TestGap_ExtractArray_BuildError(t *testing.T) {
	root := mapRootNode(t, `items:
  - any: value`)

	_, err := extractArray[gapBuildErrorModel](context.Background(), "items", root, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "build boom")
}

func TestGap_ExtractObjectMap_BuildModelError(t *testing.T) {
	root := mapRootNode(t, `things:
  x:
    bad: value`)

	_, err := extractObjectMap[gapBadArrayModel](context.Background(), "things", root, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported type")
}

func TestGap_ExtractObjectMap_BuildError(t *testing.T) {
	root := mapRootNode(t, `things:
  x:
    any: value`)

	_, err := extractObjectMap[gapBuildErrorModel](context.Background(), "things", root, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "build boom")
}

func TestGap_ArazzoBuild_InfoRefError(t *testing.T) {
	docNode, root := parseYAMLNode(t, `arazzo: 1.0.1
info:
  $ref: '#/missing'
sourceDescriptions:
  - name: api
    url: https://example.com
workflows:
  - workflowId: wf
    steps:
      - stepId: s1
        operationId: op1`)

var a Arazzo
	require.NoError(t, low.BuildModel(root, &a))
	err := a.Build(context.Background(), nil, root, index.NewSpecIndex(docNode))
	require.Error(t, err)
}

func TestGap_ArazzoBuild_WorkflowsError(t *testing.T) {
	docNode, root := parseYAMLNode(t, `arazzo: 1.0.1
info:
  title: t
  version: v
sourceDescriptions:
  - name: api
    url: https://example.com
workflows:
  - workflowId: wf
    steps:
      - stepId: s1
        operationId: op1
        requestBody:
          $ref: '#/missing'`)

var a Arazzo
	require.NoError(t, low.BuildModel(root, &a))
	err := a.Build(context.Background(), nil, root, index.NewSpecIndex(docNode))
	require.Error(t, err)
}

func TestGap_ArazzoBuild_ComponentsError(t *testing.T) {
	root := mapRootNode(t, `arazzo: 1.0.1
info:
  title: t
  version: v
sourceDescriptions:
  - name: api
    url: https://example.com
workflows:
  - workflowId: wf
    steps:
      - stepId: s1
        operationId: op1
components:
  failureActions:
    bad:
      name: bad
      type: retry
      retryAfter: nope`)

	var a Arazzo
	require.NoError(t, low.BuildModel(root, &a))
	err := a.Build(context.Background(), nil, root, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid retryAfter")
}

func TestGap_WorkflowBuild_AllErrorBranches(t *testing.T) {
	cases := []struct {
		name string
		yml  string
	}{
		{
			name: "steps",
			yml: `workflowId: wf
steps:
  - stepId: s1
    operationId: op1
    requestBody:
      $ref: '#/missing'`,
		},
		{
			name: "failureActions",
			yml: `workflowId: wf
steps:
  - stepId: s1
    operationId: op1
failureActions:
  - name: bad
    type: retry
    retryAfter: nope`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			docNode, root := parseYAMLNode(t, tc.yml)
			var wf Workflow
			require.NoError(t, low.BuildModel(root, &wf))
			require.Error(t, wf.Build(context.Background(), nil, root, index.NewSpecIndex(docNode)))
		})
	}
}

func TestGap_StepBuild_AllErrorBranches(t *testing.T) {
	cases := []struct {
		name string
		yml  string
	}{
		{
			name: "requestBody",
			yml: `stepId: s1
operationId: op1
requestBody:
  $ref: '#/missing'`,
		},
		{
			name: "onFailure",
			yml: `stepId: s1
operationId: op1
onFailure:
  - name: f1
    type: retry
    retryAfter: nope`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			docNode, root := parseYAMLNode(t, tc.yml)
			var s Step
			require.NoError(t, low.BuildModel(root, &s))
			require.Error(t, s.Build(context.Background(), nil, root, index.NewSpecIndex(docNode)))
		})
	}
}

func TestGap_FailureActionBuild_RetryLimitParseError(t *testing.T) {
	root := mapRootNode(t, `name: bad
type: retry
retryAfter: 1
retryLimit: nope`)

	var fa FailureAction
	require.NoError(t, low.BuildModel(root, &fa))
	err := fa.Build(context.Background(), nil, root, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid retryLimit")
}

func TestGap_AssignmentClosures_SuccessPaths(t *testing.T) {
	t.Run("Arazzo sourceDescriptions assignment", func(t *testing.T) {
		_, root := parseYAMLNode(t, `arazzo: 1.0.1
info:
  title: t
  version: v
sourceDescriptions:
  - name: src
    url: https://example.com
workflows:
  - workflowId: wf
    steps:
      - stepId: s1
        operationId: op1`)
		var a Arazzo
		require.NoError(t, low.BuildModel(root, &a))
		require.NoError(t, a.Build(context.Background(), nil, root, nil))
		assert.False(t, a.SourceDescriptions.IsEmpty())
	})

	t.Run("Components params and successActions assignment", func(t *testing.T) {
		_, root := parseYAMLNode(t, `parameters:
  p1:
    name: p1
    in: query
    value: v1
successActions:
  s1:
    name: done
    type: end`)
		var c Components
		require.NoError(t, low.BuildModel(root, &c))
		require.NoError(t, c.Build(context.Background(), nil, root, nil))
		assert.False(t, c.Parameters.IsEmpty())
		assert.False(t, c.SuccessActions.IsEmpty())
	})

	t.Run("FailureAction criteria assignment", func(t *testing.T) {
		_, root := parseYAMLNode(t, `name: f
type: end
criteria:
  - condition: true`)
		var fa FailureAction
		require.NoError(t, low.BuildModel(root, &fa))
		require.NoError(t, fa.Build(context.Background(), nil, root, nil))
		assert.False(t, fa.Criteria.IsEmpty())
	})

	t.Run("RequestBody replacements assignment", func(t *testing.T) {
		_, root := parseYAMLNode(t, `contentType: application/json
replacements:
  - target: /a
    value: b`)
		var rb RequestBody
		require.NoError(t, low.BuildModel(root, &rb))
		require.NoError(t, rb.Build(context.Background(), nil, root, nil))
		assert.False(t, rb.Replacements.IsEmpty())
	})

	t.Run("Step params criteria onSuccess assignment", func(t *testing.T) {
		_, root := parseYAMLNode(t, `stepId: s1
operationId: op1
parameters:
  - name: p1
    in: query
    value: v1
successCriteria:
  - condition: true
onSuccess:
  - name: done
    type: end`)
		var s Step
		require.NoError(t, low.BuildModel(root, &s))
		require.NoError(t, s.Build(context.Background(), nil, root, nil))
		assert.False(t, s.Parameters.IsEmpty())
		assert.False(t, s.SuccessCriteria.IsEmpty())
		assert.False(t, s.OnSuccess.IsEmpty())
	})

	t.Run("SuccessAction criteria assignment", func(t *testing.T) {
		_, root := parseYAMLNode(t, `name: s
type: end
criteria:
  - condition: true`)
		var sa SuccessAction
		require.NoError(t, low.BuildModel(root, &sa))
		require.NoError(t, sa.Build(context.Background(), nil, root, nil))
		assert.False(t, sa.Criteria.IsEmpty())
	})

	t.Run("Workflow successActions and params assignment", func(t *testing.T) {
		_, root := parseYAMLNode(t, `workflowId: wf
steps:
  - stepId: s1
    operationId: op1
successActions:
  - name: done
    type: end
parameters:
  - name: p1
    in: query
    value: v1`)
		var wf Workflow
		require.NoError(t, low.BuildModel(root, &wf))
		require.NoError(t, wf.Build(context.Background(), nil, root, nil))
		assert.False(t, wf.SuccessActions.IsEmpty())
		assert.False(t, wf.Parameters.IsEmpty())
	})
}

func TestGap_InjectableExtractorErrorBranches(t *testing.T) {
	t.Run("Arazzo sourceDescriptions extractor error", func(t *testing.T) {
		orig := extractArazzoSourceDescriptions
		extractArazzoSourceDescriptions = func(context.Context, string, *yaml.Node, *index.SpecIndex) (low.NodeReference[[]low.ValueReference[*SourceDescription]], error) {
			return low.NodeReference[[]low.ValueReference[*SourceDescription]]{}, errors.New("boom")
		}
		defer func() { extractArazzoSourceDescriptions = orig }()

		_, root := parseYAMLNode(t, `arazzo: 1.0.1`)
		var a Arazzo
		require.NoError(t, low.BuildModel(root, &a))
		require.Error(t, a.Build(context.Background(), nil, root, nil))
	})

	t.Run("Components extractors error", func(t *testing.T) {
		origParams := extractComponentsParametersMap
		origSuccess := extractComponentsSuccessActionsMap
		extractComponentsParametersMap = func(context.Context, string, *yaml.Node, *index.SpecIndex) (low.NodeReference[*orderedmap.Map[low.KeyReference[string], low.ValueReference[*Parameter]]], error) {
			return low.NodeReference[*orderedmap.Map[low.KeyReference[string], low.ValueReference[*Parameter]]]{}, errors.New("boom")
		}
		defer func() { extractComponentsParametersMap = origParams }()
		_, root := parseYAMLNode(t, `parameters: {}`)
		var c Components
		require.NoError(t, low.BuildModel(root, &c))
		require.Error(t, c.Build(context.Background(), nil, root, nil))

		extractComponentsParametersMap = origParams
		extractComponentsSuccessActionsMap = func(context.Context, string, *yaml.Node, *index.SpecIndex) (low.NodeReference[*orderedmap.Map[low.KeyReference[string], low.ValueReference[*SuccessAction]]], error) {
			return low.NodeReference[*orderedmap.Map[low.KeyReference[string], low.ValueReference[*SuccessAction]]]{}, errors.New("boom")
		}
		defer func() { extractComponentsSuccessActionsMap = origSuccess }()
		require.Error(t, c.Build(context.Background(), nil, root, nil))
	})

	t.Run("RequestBody replacements extractor error", func(t *testing.T) {
		orig := extractRequestBodyReplacements
		extractRequestBodyReplacements = func(context.Context, string, *yaml.Node, *index.SpecIndex) (low.NodeReference[[]low.ValueReference[*PayloadReplacement]], error) {
			return low.NodeReference[[]low.ValueReference[*PayloadReplacement]]{}, errors.New("boom")
		}
		defer func() { extractRequestBodyReplacements = orig }()
		_, root := parseYAMLNode(t, `contentType: application/json`)
		var rb RequestBody
		require.NoError(t, low.BuildModel(root, &rb))
		require.Error(t, rb.Build(context.Background(), nil, root, nil))
	})

	t.Run("Step extractors error", func(t *testing.T) {
		origParams := extractStepParameters
		origCriteria := extractStepSuccessCriteria
		origOnSuccess := extractStepOnSuccess
		defer func() {
			extractStepParameters = origParams
			extractStepSuccessCriteria = origCriteria
			extractStepOnSuccess = origOnSuccess
		}()

		extractStepParameters = func(context.Context, string, *yaml.Node, *index.SpecIndex) (low.NodeReference[[]low.ValueReference[*Parameter]], error) {
			return low.NodeReference[[]low.ValueReference[*Parameter]]{}, errors.New("boom")
		}
		_, root := parseYAMLNode(t, `stepId: s1`)
		var s Step
		require.NoError(t, low.BuildModel(root, &s))
		require.Error(t, s.Build(context.Background(), nil, root, nil))

		extractStepParameters = origParams
		extractStepSuccessCriteria = func(context.Context, string, *yaml.Node, *index.SpecIndex) (low.NodeReference[[]low.ValueReference[*Criterion]], error) {
			return low.NodeReference[[]low.ValueReference[*Criterion]]{}, errors.New("boom")
		}
		require.Error(t, s.Build(context.Background(), nil, root, nil))

		extractStepSuccessCriteria = origCriteria
		extractStepOnSuccess = func(context.Context, string, *yaml.Node, *index.SpecIndex) (low.NodeReference[[]low.ValueReference[*SuccessAction]], error) {
			return low.NodeReference[[]low.ValueReference[*SuccessAction]]{}, errors.New("boom")
		}
		require.Error(t, s.Build(context.Background(), nil, root, nil))
	})

	t.Run("Action/workflow extractors error", func(t *testing.T) {
		origSuccessActionCriteria := extractSuccessActionCriteria
		origFailureActionCriteria := extractFailureActionCriteria
		origWfSuccess := extractWorkflowSuccessActions
		origWfParams := extractWorkflowParameters
		defer func() {
			extractSuccessActionCriteria = origSuccessActionCriteria
			extractFailureActionCriteria = origFailureActionCriteria
			extractWorkflowSuccessActions = origWfSuccess
			extractWorkflowParameters = origWfParams
		}()

		extractSuccessActionCriteria = func(context.Context, string, *yaml.Node, *index.SpecIndex) (low.NodeReference[[]low.ValueReference[*Criterion]], error) {
			return low.NodeReference[[]low.ValueReference[*Criterion]]{}, errors.New("boom")
		}
		_, rootSA := parseYAMLNode(t, `name: s`)
		var sa SuccessAction
		require.NoError(t, low.BuildModel(rootSA, &sa))
		require.Error(t, sa.Build(context.Background(), nil, rootSA, nil))

		extractSuccessActionCriteria = origSuccessActionCriteria
		extractFailureActionCriteria = func(context.Context, string, *yaml.Node, *index.SpecIndex) (low.NodeReference[[]low.ValueReference[*Criterion]], error) {
			return low.NodeReference[[]low.ValueReference[*Criterion]]{}, errors.New("boom")
		}
		_, rootFA := parseYAMLNode(t, `name: f`)
		var fa FailureAction
		require.NoError(t, low.BuildModel(rootFA, &fa))
		require.Error(t, fa.Build(context.Background(), nil, rootFA, nil))

		extractFailureActionCriteria = origFailureActionCriteria
		extractWorkflowSuccessActions = func(context.Context, string, *yaml.Node, *index.SpecIndex) (low.NodeReference[[]low.ValueReference[*SuccessAction]], error) {
			return low.NodeReference[[]low.ValueReference[*SuccessAction]]{}, errors.New("boom")
		}
		_, rootWf := parseYAMLNode(t, `workflowId: wf`)
		var wf Workflow
		require.NoError(t, low.BuildModel(rootWf, &wf))
		require.Error(t, wf.Build(context.Background(), nil, rootWf, nil))

		extractWorkflowSuccessActions = origWfSuccess
		extractWorkflowParameters = func(context.Context, string, *yaml.Node, *index.SpecIndex) (low.NodeReference[[]low.ValueReference[*Parameter]], error) {
			return low.NodeReference[[]low.ValueReference[*Parameter]]{}, errors.New("boom")
		}
		require.Error(t, wf.Build(context.Background(), nil, rootWf, nil))
	})
}
