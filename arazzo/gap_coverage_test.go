// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"
	"unsafe"

	"github.com/pb33f/libopenapi/arazzo/expression"
	high "github.com/pb33f/libopenapi/datamodel/high/arazzo"
	lowmodel "github.com/pb33f/libopenapi/datamodel/low"
	lowarazzo "github.com/pb33f/libopenapi/datamodel/low/arazzo"
	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func gapState() *executionState {
	return &executionState{
		workflowResults:  make(map[string]*WorkflowResult),
		workflowContexts: make(map[string]*expression.WorkflowContext),
		activeWorkflows:  make(map[string]struct{}),
	}
}

func gapMapNode(entries map[string]string) *yaml.Node {
	content := make([]*yaml.Node, 0, len(entries)*2)
	for k, v := range entries {
		content = append(content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: k},
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: v},
		)
	}
	return &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: content}
}

type gapBadMarshaler struct{}

func (gapBadMarshaler) MarshalYAML() (any, error) {
	return nil, errors.New("marshal boom")
}

func TestGap_ProcessActionTypeResult_Branches(t *testing.T) {
	t.Run("goto missing workflow returns run error", func(t *testing.T) {
		e := NewEngine(&high.Arazzo{}, nil, nil)
		_, err := e.processActionTypeResult(context.Background(), &actionTypeRequest{
			actionType: "goto",
			workflowId: "missing",
		}, &expression.Context{}, gapState(), map[string]int{})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrUnresolvedWorkflowRef)
	})

	t.Run("goto workflow success sets endWorkflow", func(t *testing.T) {
		doc := &high.Arazzo{
			Workflows: []*high.Workflow{{WorkflowId: "sub"}},
		}
		e := NewEngine(doc, &mockExec{resp: &ExecutionResponse{StatusCode: 200}}, nil)
		res, err := e.processActionTypeResult(context.Background(), &actionTypeRequest{
			actionType: "goto",
			workflowId: "sub",
		}, &expression.Context{}, gapState(), map[string]int{})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.True(t, res.endWorkflow)
	})

	t.Run("goto workflow failed surfaces workflow error", func(t *testing.T) {
		doc := &high.Arazzo{
			Workflows: []*high.Workflow{{
				WorkflowId: "sub",
				Steps: []*high.Step{{
					StepId:      "s1",
					OperationId: "op1",
				}},
			}},
		}
		e := NewEngine(doc, nil, nil)
		_, err := e.processActionTypeResult(context.Background(), &actionTypeRequest{
			actionType: "goto",
			workflowId: "sub",
		}, &expression.Context{}, gapState(), map[string]int{})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrExecutorNotConfigured)
	})

	t.Run("goto unknown step id returns action error", func(t *testing.T) {
		e := NewEngine(&high.Arazzo{}, nil, nil)
		_, err := e.processActionTypeResult(context.Background(), &actionTypeRequest{
			actionType: "goto",
			stepId:     "missing",
		}, &expression.Context{}, gapState(), map[string]int{"s1": 0})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrStepIdNotInWorkflow)
	})

	t.Run("retry default limit and already exhausted", func(t *testing.T) {
		e := NewEngine(&high.Arazzo{}, nil, nil)
		res, err := e.processActionTypeResult(context.Background(), &actionTypeRequest{
			actionType:     "retry",
			retryLimit:     0,
			currentRetries: 1,
		}, &expression.Context{}, gapState(), map[string]int{})
		require.NoError(t, err)
		assert.False(t, res.retryCurrent)
	})

	t.Run("retry with delay", func(t *testing.T) {
		e := NewEngine(&high.Arazzo{}, nil, nil)
		res, err := e.processActionTypeResult(context.Background(), &actionTypeRequest{
			actionType:     "retry",
			retryLimit:     2,
			currentRetries: 0,
			retryAfterSec:  0.25,
		}, &expression.Context{}, gapState(), map[string]int{})
		require.NoError(t, err)
		assert.True(t, res.retryCurrent)
		assert.Greater(t, res.retryAfter, time.Duration(0))
	})
}

func TestGap_ProcessActionSelectionAndResolution(t *testing.T) {
	t.Run("processSuccessActions selection error", func(t *testing.T) {
		e := NewEngine(&high.Arazzo{}, nil, nil)
		step := &high.Step{
			OnSuccess: []*high.SuccessAction{{
				Name: "bad",
				Type: "end",
				Criteria: []*high.Criterion{{
					Condition: "$notAValidExpression",
				}},
			}},
		}
		_, err := e.processSuccessActions(context.Background(), step, &high.Workflow{}, &expression.Context{}, gapState(), map[string]int{})
		require.Error(t, err)
	})

	t.Run("processFailureActions selection error", func(t *testing.T) {
		e := NewEngine(&high.Arazzo{}, nil, nil)
		step := &high.Step{
			OnFailure: []*high.FailureAction{{
				Name: "bad",
				Type: "end",
				Criteria: []*high.Criterion{{
					Condition: "$notAValidExpression",
				}},
			}},
		}
		_, err := e.processFailureActions(context.Background(), step, &high.Workflow{}, &expression.Context{}, gapState(), map[string]int{}, 0)
		require.Error(t, err)
	})

	t.Run("processFailureActions reads retry fields", func(t *testing.T) {
		retryAfter := 0.1
		retryLimit := int64(3)
		e := NewEngine(&high.Arazzo{}, nil, nil)
		step := &high.Step{
			OnFailure: []*high.FailureAction{{
				Name:       "retry",
				Type:       "retry",
				RetryAfter: &retryAfter,
				RetryLimit: &retryLimit,
			}},
		}
		res, err := e.processFailureActions(context.Background(), step, &high.Workflow{}, &expression.Context{}, gapState(), map[string]int{}, 0)
		require.NoError(t, err)
		assert.True(t, res.retryCurrent)
	})

	t.Run("findMatchingAction resolve and eval errors", func(t *testing.T) {
		_, err := findMatchingAction([]int{1},
			func(int) (int, error) { return 0, errors.New("resolve") },
			func(int) []*high.Criterion { return nil },
			func([]*high.Criterion, *expression.Context) (bool, error) { return true, nil },
			&expression.Context{},
		)
		require.Error(t, err)

		_, err = findMatchingAction([]int{1},
			func(v int) (int, error) { return v, nil },
			func(int) []*high.Criterion { return nil },
			func([]*high.Criterion, *expression.Context) (bool, error) { return false, errors.New("eval") },
			&expression.Context{},
		)
		require.Error(t, err)
	})

	t.Run("resolve success and failure reusable action", func(t *testing.T) {
		saMap := orderedmap.New[string, *high.SuccessAction]()
		saMap.Set("ok", &high.SuccessAction{Name: "ok", Type: "end"})
		faMap := orderedmap.New[string, *high.FailureAction]()
		faMap.Set("bad", &high.FailureAction{Name: "bad", Type: "end"})

		e := NewEngine(&high.Arazzo{
			Components: &high.Components{
				SuccessActions: saMap,
				FailureActions: faMap,
			},
		}, nil, nil)

		a, err := e.resolveSuccessAction(&high.SuccessAction{Reference: "$components.successActions.ok"})
		require.NoError(t, err)
		assert.Equal(t, "ok", a.Name)

		b, err := e.resolveFailureAction(&high.FailureAction{Reference: "$components.failureActions.bad"})
		require.NoError(t, err)
		assert.Equal(t, "bad", b.Name)
	})

	t.Run("resolve reusable action without components", func(t *testing.T) {
		e := NewEngine(&high.Arazzo{}, nil, nil)
		_, err := e.resolveSuccessAction(&high.SuccessAction{Reference: "$components.successActions.missing"})
		require.Error(t, err)
		_, err = e.resolveFailureAction(&high.FailureAction{Reference: "$components.failureActions.missing"})
		require.Error(t, err)
	})

	t.Run("resolve nil actions", func(t *testing.T) {
		e := NewEngine(&high.Arazzo{}, nil, nil)
		a, err := e.resolveSuccessAction(nil)
		require.NoError(t, err)
		assert.Nil(t, a)
		b, err := e.resolveFailureAction(nil)
		require.NoError(t, err)
		assert.Nil(t, b)
	})

	t.Run("lookupComponent validation branches", func(t *testing.T) {
		_, err := lookupComponent("bad.ref", "$components.successActions.", orderedmap.New[string, *high.SuccessAction]())
		require.Error(t, err)

		_, err = lookupComponent("$components.successActions.ok", "$components.successActions.", (*orderedmap.Map[string, *high.SuccessAction])(nil))
		require.Error(t, err)

		_, err = lookupComponent("$components.successActions.missing", "$components.successActions.", orderedmap.New[string, *high.SuccessAction]())
		require.Error(t, err)
	})
}

func TestGap_EvaluateActionCriteria_Branches(t *testing.T) {
	e := NewEngine(&high.Arazzo{}, nil, nil)
	ok, err := e.evaluateActionCriteria(nil, &expression.Context{})
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = e.evaluateActionCriteria([]*high.Criterion{{Condition: "false"}}, &expression.Context{})
	require.NoError(t, err)
	assert.False(t, ok)

	_, err = e.evaluateActionCriteria([]*high.Criterion{{Condition: "$badExpr"}}, &expression.Context{})
	require.Error(t, err)

	ok, err = e.evaluateActionCriteria([]*high.Criterion{{Condition: "true"}}, &expression.Context{})
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestGap_CriterionCachesAndHelpers(t *testing.T) {
	ClearCriterionCaches()

	caches := newCriterionCaches()
	_, _ = compileCriterionRegex(`^a+$`, caches)
	_, _ = compileCriterionRegex(`^a+$`, caches)
	_, _ = compileCriterionJSONPath(`$.a`, caches)
	_, _ = compileCriterionJSONPath(`$.a`, caches)

	caches.parseExpr = func(string) (expression.Expression, error) {
		return expression.Expression{}, errors.New("parse failed")
	}
	_, err := evaluateExprString("$statusCode", &expression.Context{StatusCode: 200}, caches)
	require.Error(t, err)

	assert.Equal(t, "7", sprintValue(int64(7)))
	assert.Equal(t, "1.5", sprintValue(float64(1.5)))
	assert.Equal(t, "true", sprintValue(true))
	assert.Equal(t, "{x}", sprintValue(struct{ A string }{A: "x"}))

	ok, err := evaluateJSONPathCriterion(&high.Criterion{
		Condition: "$.id",
		Context:   "$inputs.empty",
	}, &expression.Context{Inputs: map[string]any{"empty": nil}}, nil)
	require.NoError(t, err)
	assert.False(t, ok)

	_, err = evaluateJSONPathCriterion(&high.Criterion{
		Condition: "$.id",
		Context:   "$inputs.bad",
	}, &expression.Context{Inputs: map[string]any{"bad": gapBadMarshaler{}}}, nil)
	require.Error(t, err)

	_, err = evaluateJSONPathCriterion(&high.Criterion{
		Condition: "$[",
		Context:   "$statusCode",
	}, &expression.Context{StatusCode: 200}, nil)
	require.Error(t, err)
}

func TestGap_EngineInitAndClearCaches(t *testing.T) {
	doc := &high.Arazzo{
		SourceDescriptions: []*high.SourceDescription{
			nil,
			{Name: "s1", URL: "https://example.com"},
		},
		Workflows: []*high.Workflow{
			nil,
			{WorkflowId: "wf1"},
		},
	}
	e := NewEngine(doc, nil, []*ResolvedSource{{Name: "s1", URL: "https://example.com"}})
	require.NotNil(t, e.defaultSource)
	assert.Len(t, e.sourceOrder, 1)
	assert.NotNil(t, e.workflows["wf1"])

	e.exprCache["x"] = expression.Expression{Raw: "$url"}
	e.ClearCaches()
	assert.Empty(t, e.exprCache)
}

func TestGap_RunWorkflow_ActionErrorBranches(t *testing.T) {
	t.Run("success action evaluation error", func(t *testing.T) {
		doc := &high.Arazzo{
			Workflows: []*high.Workflow{{
				WorkflowId: "wf",
				Steps: []*high.Step{{
					StepId:      "s1",
					OperationId: "op1",
					OnSuccess: []*high.SuccessAction{{
						Name: "bad",
						Type: "end",
						Criteria: []*high.Criterion{{
							Condition: "$badExpr",
						}},
					}},
				}},
			}},
		}
		e := NewEngine(doc, &mockExec{resp: &ExecutionResponse{StatusCode: 200}}, nil)
		res, err := e.RunWorkflow(context.Background(), "wf", nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.False(t, res.Success)
		require.Error(t, res.Error)
	})

	t.Run("failure action evaluation error", func(t *testing.T) {
		doc := &high.Arazzo{
			Workflows: []*high.Workflow{{
				WorkflowId: "wf",
				Steps: []*high.Step{{
					StepId:      "s1",
					OperationId: "op1",
					SuccessCriteria: []*high.Criterion{{
						Condition: "$statusCode == 201",
					}},
					OnFailure: []*high.FailureAction{{
						Name: "bad",
						Type: "end",
						Criteria: []*high.Criterion{{
							Condition: "$badExpr",
						}},
					}},
				}},
			}},
		}
		e := NewEngine(doc, &mockExec{resp: &ExecutionResponse{StatusCode: 200}}, nil)
		res, err := e.RunWorkflow(context.Background(), "wf", nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.False(t, res.Success)
		require.Error(t, res.Error)
	})

	t.Run("failure action retry with canceled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		exec := &mockCallbackExec{
			fn: func(_ context.Context, _ *ExecutionRequest) (*ExecutionResponse, error) {
				cancel()
				return &ExecutionResponse{StatusCode: 200}, nil
			},
		}
		delay := 0.5
		limit := int64(1)
		doc := &high.Arazzo{
			Workflows: []*high.Workflow{{
				WorkflowId: "wf",
				Steps: []*high.Step{{
					StepId:      "s1",
					OperationId: "op1",
					SuccessCriteria: []*high.Criterion{{
						Condition: "$statusCode == 201",
					}},
					OnFailure: []*high.FailureAction{{
						Name:       "retry",
						Type:       "retry",
						RetryAfter: &delay,
						RetryLimit: &limit,
					}},
				}},
			}},
		}
		e := NewEngine(doc, exec, nil)
		res, err := e.RunWorkflow(ctx, "wf", nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.False(t, res.Success)
		require.Error(t, res.Error)
		assert.ErrorIs(t, res.Error, context.Canceled)
	})

	t.Run("failure action end branch", func(t *testing.T) {
		doc := &high.Arazzo{
			Workflows: []*high.Workflow{{
				WorkflowId: "wf",
				Steps: []*high.Step{{
					StepId:      "s1",
					OperationId: "op1",
					SuccessCriteria: []*high.Criterion{{
						Condition: "$statusCode == 201",
					}},
					OnFailure: []*high.FailureAction{{
						Name: "end",
						Type: "end",
					}},
				}},
			}},
		}
		e := NewEngine(doc, &mockExec{resp: &ExecutionResponse{StatusCode: 200}}, nil)
		res, err := e.RunWorkflow(context.Background(), "wf", nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.False(t, res.Success)
		require.Error(t, res.Error)
	})

	t.Run("step transition guard", func(t *testing.T) {
		doc := &high.Arazzo{
			Workflows: []*high.Workflow{{
				WorkflowId: "wf",
				Steps: []*high.Step{{
					StepId:      "s1",
					OperationId: "op1",
					OnSuccess: []*high.SuccessAction{{
						Name:   "loop",
						Type:   "goto",
						StepId: "s1",
					}},
				}},
			}},
		}
		e := NewEngine(doc, &mockExec{resp: &ExecutionResponse{StatusCode: 200}}, nil)
		res, err := e.RunWorkflow(context.Background(), "wf", nil)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.False(t, res.Success)
		require.Error(t, res.Error)
		assert.Contains(t, res.Error.Error(), "max step transitions")
	})
}

func TestGap_RunAll_ExecutionErrorBranch(t *testing.T) {
	doc := &high.Arazzo{
		Workflows: []*high.Workflow{{
			WorkflowId: "wf1",
			Steps: []*high.Step{{
				StepId:      "s1",
				OperationId: "op1",
			}},
		}},
	}
	e := NewEngine(doc, &mockExec{resp: &ExecutionResponse{StatusCode: 200}}, nil)
	delete(e.workflows, "wf1")

	result, err := e.RunAll(context.Background(), nil)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success)
	require.Len(t, result.Workflows, 1)
	assert.ErrorIs(t, result.Workflows[0].Error, ErrUnresolvedWorkflowRef)
}

func TestGap_TopologicalSort_WithNilWorkflowEntries(t *testing.T) {
	e := NewEngine(&high.Arazzo{
		Workflows: []*high.Workflow{
			nil,
			{WorkflowId: "a"},
			nil,
		},
	}, nil, nil)
	order, err := e.topologicalSort()
	require.NoError(t, err)
	assert.Equal(t, []string{"a"}, order)
}

func TestGap_ErrorTypes(t *testing.T) {
	base := errors.New("boom")
	e1 := &StepFailureError{StepId: "s1", Cause: base}
	assert.Contains(t, e1.Error(), "boom")
	assert.ErrorIs(t, e1.Unwrap(), base)

	e2 := &StepFailureError{StepId: "s2", CriterionIndex: 1, Message: "failed"}
	assert.Contains(t, e2.Error(), "successCriteria[1]")

	e3 := &StepFailureError{StepId: "s3", CriterionIndex: -1}
	assert.Contains(t, e3.Error(), "s3")

	assert.Equal(t, "workflow \"wf\" failed", workflowFailureError("wf", &WorkflowResult{}).Error())
	assert.Equal(t, base, workflowFailureError("wf", &WorkflowResult{Error: base}))

	assert.Nil(t, workflowExecutionFailureResult("wf", nil, nil))
	require.NotNil(t, workflowExecutionFailureResult("wf", map[string]any{"a": 1}, errors.New("x")))

	assert.ErrorIs(t, stepFailureOrDefault("s4", base), base)
	assert.Contains(t, stepFailureOrDefault("s5", nil).Error(), "s5")
}

func TestGap_OperationResolver_DefaultDoc(t *testing.T) {
	docA := &v3high.Document{}
	docB := &v3high.Document{}
	r := &operationResolver{
		sourceDocs:  map[string]*v3high.Document{"a": docA},
		sourceOrder: []string{"a"},
		searchDocs:  []*v3high.Document{docB},
	}
	assert.Same(t, docA, r.defaultDoc())

	r = &operationResolver{
		sourceDocs: map[string]*v3high.Document{},
		searchDocs: []*v3high.Document{docB},
	}
	assert.Same(t, docB, r.defaultDoc())

	r = &operationResolver{}
	assert.Nil(t, r.defaultDoc())
}

type gapErrReader struct{}

func (gapErrReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }

type gapRoundTripper struct{}

func (gapRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(gapErrReader{}),
		Header:     make(http.Header),
	}, nil
}

func TestGap_FetchHTTPSourceBytes_ReadError(t *testing.T) {
	_, err := fetchHTTPSourceBytes("http://example.com", &ResolveConfig{
		Timeout:     time.Second,
		MaxBodySize: 1024,
		HTTPClient:  &http.Client{Transport: gapRoundTripper{}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read failed")
}

func TestGap_ResolveFilePath_LstatPermissionError(t *testing.T) {
	root := t.TempDir()
	private := filepath.Join(root, "no-access")
	require.NoError(t, os.Mkdir(private, 0o700))
	require.NoError(t, os.Chmod(private, 0o000))
	defer func() { _ = os.Chmod(private, 0o700) }()

	_, err := resolveFilePath(filepath.Join("no-access", "x.yaml"), []string{root})
	require.Error(t, err)
}

func TestGap_ResolvePathHelpers(t *testing.T) {
	_, err := resolveFilePath("/tmp/x.yaml", []string{"\x00bad"})
	require.Error(t, err)

	assert.False(t, isPathWithinRoots("/tmp/x", []string{"relative-root"}))
	assert.Empty(t, canonicalizeRoots([]string{"\x00bad"}))
}

func TestGap_ExecuteStepAndHelpers(t *testing.T) {
	t.Run("workflow step parameter resolution error", func(t *testing.T) {
		e := NewEngine(&high.Arazzo{}, nil, nil)
		step := &high.Step{
			StepId:      "s1",
			WorkflowId:  "wf2",
			Parameters:  []*high.Parameter{nil},
			OperationId: "",
		}
		res := e.executeStep(context.Background(), step, &high.Workflow{}, &expression.Context{
			Inputs:    map[string]any{},
			Outputs:   map[string]any{},
			Steps:     map[string]*expression.StepContext{},
			Workflows: map[string]*expression.WorkflowContext{},
		}, gapState())
		assert.False(t, res.Success)
		require.Error(t, res.Error)
	})

	t.Run("workflow step parameter value eval error", func(t *testing.T) {
		e := NewEngine(&high.Arazzo{}, nil, nil)
		step := &high.Step{
			StepId:     "s1",
			WorkflowId: "wf2",
			Parameters: []*high.Parameter{
				{Name: "p", In: "query", Value: makeValueNode("$badExpr")},
			},
		}
		res := e.executeStep(context.Background(), step, &high.Workflow{}, &expression.Context{
			Inputs:    map[string]any{},
			Outputs:   map[string]any{},
			Steps:     map[string]*expression.StepContext{},
			Workflows: map[string]*expression.WorkflowContext{},
		}, gapState())
		assert.False(t, res.Success)
		require.Error(t, res.Error)
	})

	t.Run("operation step response body conversion error", func(t *testing.T) {
		e := NewEngine(&high.Arazzo{}, &mockExec{resp: &ExecutionResponse{
			StatusCode: 200,
			Body:       gapBadMarshaler{},
		}}, nil)
		res := e.executeStep(context.Background(), &high.Step{
			StepId:      "s1",
			OperationId: "op1",
		}, &high.Workflow{}, &expression.Context{
			Inputs:    map[string]any{},
			Outputs:   map[string]any{},
			Steps:     map[string]*expression.StepContext{},
			Workflows: map[string]*expression.WorkflowContext{},
		}, gapState())
		assert.False(t, res.Success)
		require.Error(t, res.Error)
	})

	t.Run("evaluateStepSuccessCriteria error branch", func(t *testing.T) {
		e := NewEngine(&high.Arazzo{}, nil, nil)
		err := e.evaluateStepSuccessCriteria(&high.Step{
			StepId: "s1",
			SuccessCriteria: []*high.Criterion{{
				Condition: "$badExpr",
			}},
		}, &expression.Context{})
		require.Error(t, err)
	})

	t.Run("buildExecutionRequest replacement errors", func(t *testing.T) {
		e := NewEngine(&high.Arazzo{}, nil, nil)
		exprCtx := &expression.Context{
			Inputs:  map[string]any{},
			Outputs: map[string]any{},
			Steps:   map[string]*expression.StepContext{},
		}
		_, err := e.buildExecutionRequest(&high.Step{
			StepId:      "s1",
			OperationId: "op1",
			RequestBody: &high.RequestBody{
				Payload: gapMapNode(map[string]string{"a": "b"}),
				Replacements: []*high.PayloadReplacement{
					{Target: "/x", Value: makeValueNode("$badExpr")},
				},
			},
		}, exprCtx)
		require.Error(t, err)

		_, err = e.buildExecutionRequest(&high.Step{
			StepId:      "s1",
			OperationId: "op1",
			RequestBody: &high.RequestBody{
				Payload: gapMapNode(map[string]string{"a": "b"}),
				Replacements: []*high.PayloadReplacement{
					{Target: "bad-pointer", Value: makeValueNode("x")},
				},
			},
		}, &expression.Context{Inputs: map[string]any{}, Outputs: map[string]any{}, Steps: map[string]*expression.StepContext{}})
		require.Error(t, err)
	})

	t.Run("buildExecutionRequest request body conversion error", func(t *testing.T) {
		e := NewEngine(&high.Arazzo{}, nil, nil)
		_, err := e.buildExecutionRequest(&high.Step{
			StepId:      "s1",
			OperationId: "op1",
			RequestBody: &high.RequestBody{
				Payload: makeValueNode("$inputs.fn"),
			},
		}, &expression.Context{Inputs: map[string]any{"fn": gapBadMarshaler{}}})
		require.Error(t, err)
	})

	t.Run("resolveStepSource deterministic fallback", func(t *testing.T) {
		doc := &high.Arazzo{
			SourceDescriptions: []*high.SourceDescription{
				{Name: "s1", URL: "u1"},
				{Name: "s2", URL: "u2"},
			},
		}
		e := NewEngine(doc, nil, []*ResolvedSource{
			{Name: "s1", URL: "u1"},
			{Name: "s2", URL: "u2"},
		})
		src := e.resolveStepSource(&high.Step{OperationPath: "{$sourceDescriptions.unknown}/pets"})
		require.NotNil(t, src)
		assert.Equal(t, "s1", src.Name)
	})

	t.Run("resolve expression short-circuit helpers", func(t *testing.T) {
		e := NewEngine(&high.Arazzo{}, nil, nil)
		v, err := e.resolveExpressionValues([]any{"a", 1}, &expression.Context{})
		require.NoError(t, err)
		assert.Equal(t, []any{"a", 1}, v)

		v, err = e.resolveExpressionValues(map[any]any{"a": 1}, &expression.Context{})
		require.NoError(t, err)
		assert.Equal(t, map[string]any{"a": 1}, v)
	})

	t.Run("applyPayloadReplacements skip and failure branches", func(t *testing.T) {
		e := NewEngine(&high.Arazzo{}, nil, nil)
		root := map[string]any{"a": "b"}
		_, err := e.applyPayloadReplacements(root, []*high.PayloadReplacement{
			nil,
			{Target: "", Value: makeValueNode("x")},
		}, &expression.Context{}, "s1")
		require.NoError(t, err)

		_, err = e.applyPayloadReplacements(root, []*high.PayloadReplacement{
			{Target: "/x", Value: makeValueNode("$badExpr")},
		}, &expression.Context{}, "s1")
		require.Error(t, err)

		_, err = e.applyPayloadReplacements(root, []*high.PayloadReplacement{
			{Target: "bad", Value: makeValueNode("x")},
		}, &expression.Context{}, "s1")
		require.Error(t, err)
	})
}

func TestGap_JSONPointerAndResolutionHelpers(t *testing.T) {
	root := map[string]any{"a": "b"}
	require.Error(t, setJSONPointerValue(root, "/a/b/c", "x"))
	require.Error(t, setJSONPointerValue(root, "/a/b", "x"))

	assert.False(t, sliceNeedsResolution([]any{"x", 1, true}))
	assert.False(t, mapAnyNeedsResolution(map[any]any{"x": 1}))
}

func TestGap_ResolveStepSource_DefaultAndNilBranches(t *testing.T) {
	oneSourceEngine := NewEngine(&high.Arazzo{
		SourceDescriptions: []*high.SourceDescription{{Name: "s1", URL: "u1"}},
	}, nil, []*ResolvedSource{{Name: "s1", URL: "u1"}})
	src := oneSourceEngine.resolveStepSource(&high.Step{OperationPath: "/anything"})
	require.NotNil(t, src)
	assert.Equal(t, "s1", src.Name)

	noOrderEngine := NewEngine(nil, nil, []*ResolvedSource{
		{Name: "a", URL: "ua"},
		{Name: "b", URL: "ub"},
	})
	assert.Nil(t, noOrderEngine.resolveStepSource(&high.Step{OperationPath: "{$sourceDescriptions.none}/x"}))
}

func TestGap_SleepWithContext_Branches(t *testing.T) {
	require.NoError(t, sleepWithContext(context.Background(), 0))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.ErrorIs(t, sleepWithContext(ctx, time.Millisecond), context.Canceled)
	require.NoError(t, sleepWithContext(context.Background(), time.Millisecond))
}

func TestGap_DirectYAMLNodeBranches(t *testing.T) {
	n := yaml.Node{Kind: yaml.ScalarNode, Value: "x"}
	out, err := directYAMLNode(n)
	require.NoError(t, err)
	require.NotNil(t, out)

	_, err = directYAMLNode(map[string]any{"a": gapBadMarshaler{}})
	require.Error(t, err)

	_, err = directYAMLNode(map[any]any{"a": gapBadMarshaler{}})
	require.Error(t, err)

	_, err = directYAMLNode([]any{gapBadMarshaler{}})
	require.Error(t, err)

	out, err = directYAMLNode([]string{"a", "b"})
	require.NoError(t, err)
	assert.Equal(t, yaml.SequenceNode, out.Kind)

	out, err = directYAMLNode(false)
	require.NoError(t, err)
	assert.Equal(t, "false", out.Value)

	out, err = directYAMLNode(uint64(7))
	require.NoError(t, err)
	assert.Equal(t, "!!int", out.Tag)

	out, err = directYAMLNode(float32(1.25))
	require.NoError(t, err)
	assert.Equal(t, "!!float", out.Tag)

	out, err = directYAMLNode(nil)
	require.NoError(t, err)
	assert.Nil(t, out)

	_, err = directYAMLNode(gapBadMarshaler{})
	require.Error(t, err)

	out, err = directYAMLNode(map[any]any{"a": "b"})
	require.NoError(t, err)
	assert.Equal(t, yaml.MappingNode, out.Kind)

	type okStruct struct {
		Name string
	}
	out, err = directYAMLNode(okStruct{Name: "ok"})
	require.NoError(t, err)
	assert.NotNil(t, out)
}

func TestGap_ValidationHelperBranches(t *testing.T) {
	line, col := lowNodePos(nil)
	assert.Equal(t, 0, line)
	assert.Equal(t, 0, col)
	line, col = lowNodePos(&yaml.Node{Line: 3, Column: 4})
	assert.Equal(t, 3, line)
	assert.Equal(t, 4, col)

	var info *lowarazzo.Info
	line, col = rootPos(info, (*lowarazzo.Info).GetRootNode)
	assert.Equal(t, 0, line)
	assert.Equal(t, 0, col)

	info = &lowarazzo.Info{RootNode: &yaml.Node{Line: 10, Column: 11}}
	line, col = rootPos(info, (*lowarazzo.Info).GetRootNode)
	assert.Equal(t, 10, line)
	assert.Equal(t, 11, col)
}

func TestGap_ValidationOperationLookupHelpers(t *testing.T) {
	// Build high-level doc from low model so checkVersion has low node metadata.
	yml := `arazzo: 2.0.0
info:
  title: t
  version: v
sourceDescriptions:
  - name: src
    url: https://example.com/openapi.yaml
workflows:
  - workflowId: wf
    steps:
      - stepId: s1
        operationId: op1`
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(yml), &root))
	var lowDoc lowarazzo.Arazzo
	require.NoError(t, lowmodel.BuildModel(root.Content[0], &lowDoc))
	require.NoError(t, lowDoc.Build(context.Background(), nil, root.Content[0], nil))
	doc := high.NewArazzo(&lowDoc)

	v := &validator{doc: doc, result: &ValidationResult{}}
	v.checkVersion()
	require.NotEmpty(t, v.result.Errors)
	assert.Greater(t, v.result.Errors[0].Line, 0)

	// buildOperationLookupContext branches: nil docs, duplicates, no openapi sources.
	docNoOpenAPI := validMinimalDoc()
	docNoOpenAPI.SourceDescriptions = []*high.SourceDescription{{Name: "a", URL: " ", Type: "arazzo"}, nil}
	openDoc := &v3high.Document{}
	docNoOpenAPI.AddOpenAPISourceDocument(nil, openDoc, openDoc)
	v2 := &validator{doc: docNoOpenAPI, result: &ValidationResult{}}
	v2.buildOperationLookupContext()
	assert.True(t, v2.result.HasWarnings())

	// Fallback mapping branch when identities are empty/non-matching.
	docMap := validMinimalDoc()
	docMap.SourceDescriptions = []*high.SourceDescription{
		{Name: "s1", URL: "https://example.com/a.yaml", Type: "openapi"},
		{Name: "s2", URL: "https://example.com/b.yaml", Type: "openapi"},
	}
	docMap.AddOpenAPISourceDocument(&v3high.Document{}, &v3high.Document{})
	v3 := &validator{doc: docMap, result: &ValidationResult{}}
	v3.buildOperationLookupContext()
	require.NotNil(t, v3.opLookup)
	assert.Contains(t, v3.opLookup.sourceDocs, "s1")
	assert.Contains(t, v3.opLookup.sourceDocs, "s2")

	// validateStepOperationLookup branches.
	v4 := &validator{doc: validMinimalDoc(), result: &ValidationResult{}, opLookup: &operationResolver{}}
	v4.validateStepOperationLookup(nil, "x", 1, 1)
	v4.validateStepOperationLookup(&high.Step{OperationId: "missing"}, "x", 1, 1)
	assert.True(t, v4.result.HasWarnings())

	v5 := &validator{
		doc:    validMinimalDoc(),
		result: &ValidationResult{},
		opLookup: &operationResolver{
			sourceDocs: map[string]*v3high.Document{},
			searchDocs: nil,
		},
	}
	v5.validateStepOperationLookup(&high.Step{OperationPath: "not-a-pointer"}, "x", 1, 1)
	assert.True(t, v5.result.HasWarnings())

	// Ensure checkable=false branch with a fallback document present.
	v6 := &validator{
		doc:    validMinimalDoc(),
		result: &ValidationResult{},
		opLookup: &operationResolver{
			searchDocs: []*v3high.Document{{}},
		},
	}
	v6.validateStepOperationLookup(&high.Step{OperationPath: "not-a-pointer"}, "x", 1, 1)
	assert.True(t, v6.result.HasWarnings())

	// buildOperationLookupContext with only nil attached docs to hit uniqueDocs empty branch.
	docNilAttached := validMinimalDoc()
	setOpenAPISourceDocsUnsafe(docNilAttached, []*v3high.Document{nil})
	v7 := &validator{doc: docNilAttached, result: &ValidationResult{}}
	v7.buildOperationLookupContext()

	// buildOperationLookupContext where source URL normalizes to empty string.
	docEmptyURL := validMinimalDoc()
	docEmptyURL.SourceDescriptions = []*high.SourceDescription{{Name: "s1", URL: " ", Type: "openapi"}}
	docEmptyURL.AddOpenAPISourceDocument(&v3high.Document{})
	v8 := &validator{doc: docEmptyURL, result: &ValidationResult{}}
	v8.buildOperationLookupContext()
}

func TestGap_ValidationStandaloneHelpers(t *testing.T) {
	assert.Equal(t, "", openAPIDocumentIdentity(nil))
	assert.Equal(t, "", openAPIDocumentIdentity(&v3high.Document{}))

	assert.Equal(t, "", normalizeLookupLocation(""))
	assert.NotEmpty(t, normalizeLookupLocation(" . "))
	assert.NotEmpty(t, normalizeLookupLocation("relative/path.yaml"))
	assert.Equal(t, "https://example.com", normalizeLookupLocation("https://example.com"))

	assert.False(t, operationIDExistsInDoc(nil, "x"))
	docNilPaths := &v3high.Document{Paths: &v3high.Paths{PathItems: orderedmap.New[string, *v3high.PathItem]()}}
	docNilPaths.Paths.PathItems.Set("/x", nil)
	assert.False(t, operationIDExistsInDoc(docNilPaths, "x"))

	docNoOps := &v3high.Document{Paths: &v3high.Paths{PathItems: orderedmap.New[string, *v3high.PathItem]()}}
	docNoOps.Paths.PathItems.Set("/x", &v3high.PathItem{})
	assert.False(t, operationIDExistsInDoc(docNoOps, "x"))

	exists, checkable := operationPathExistsInDoc(nil, "not-a-pointer")
	assert.False(t, exists)
	assert.False(t, checkable)
	exists, checkable = operationPathExistsInDoc(nil, "#/paths/~1pets/get")
	assert.False(t, exists)
	assert.True(t, checkable)
	exists, checkable = operationPathExistsInDoc(docNilPaths, "#/paths/~1missing/get")
	assert.False(t, exists)
	assert.True(t, checkable)
	exists, checkable = operationPathExistsInDoc(docNoOps, "#/paths/~1x/get")
	assert.False(t, exists)
	assert.True(t, checkable)

	_, _, ok := parseOperationPathPointer("not-a-pointer")
	assert.False(t, ok)
	_, _, ok = parseOperationPathPointer("#/paths/")
	assert.False(t, ok)
	_, _, ok = parseOperationPathPointer("#/paths//get")
	assert.False(t, ok)
	_, _, ok = parseOperationPathPointer("#/paths/~1pets/get extra")
	assert.True(t, ok)

	_, found := extractSourceNameFromOperationPath("no source expression")
	assert.False(t, found)
}

func TestGap_PathAbsErrorBranches(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows locks the CWD directory, preventing os.Remove while chdir'd into it")
	}
	orig, err := os.Getwd()
	require.NoError(t, err)

	tmp := t.TempDir()
	inner := filepath.Join(tmp, "inner")
	require.NoError(t, os.Mkdir(inner, 0o755))
	require.NoError(t, os.Chdir(inner))
	require.NoError(t, os.Remove(inner))
	defer func() {
		_ = os.Chdir(orig)
	}()

	_, _ = resolveFilePath("/tmp/x.yaml", []string{"relative-root"})
	_ = canonicalizeRoots([]string{"relative-root"})
	_ = normalizeLookupLocation(".")
}

func TestGap_ResolveFilepathAbsHook(t *testing.T) {
	orig := resolveFilepathAbs
	resolveFilepathAbs = func(string) (string, error) {
		return "", errors.New("abs failed")
	}
	defer func() { resolveFilepathAbs = orig }()

	_, _ = resolveFilePath("/tmp/x.yaml", []string{"relative-root"})
	_ = canonicalizeRoots([]string{"relative-root"})
	assert.Equal(t, "", normalizeLookupLocation("."))
}

func setOpenAPISourceDocsUnsafe(doc *high.Arazzo, docs []*v3high.Document) {
	v := reflect.ValueOf(doc).Elem().FieldByName("openAPISourceDocs")
	ptr := reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem()
	ptr.Set(reflect.ValueOf(docs))
}
