// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"os"
	"path/filepath"
	"testing"

	libopenapi "github.com/pb33f/libopenapi"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
)

// TestArazzo11NegativeFixtures checks that a wrong YAML node kind on an Arazzo 1.1
// addition produces a source-positioned build error instead of being silently discarded.
//
// The reflection-driven model builder ignores a value whose node kind does not match the
// target Go field, which turned an authoring mistake into quiet data loss. The composite
// 1.1 objects (Selector, OutputValue, targetSelectorType) already rejected an unexpected
// node kind, so these fixtures pin the scalar and sequence fields to the same contract.
//
// Requiredness and semantic validity remain the validator's concern; only the node kind
// (and, for timeout, that the scalar is actually an integer) is enforced here.
func TestArazzo11NegativeFixtures(t *testing.T) {
	tests := []struct {
		name        string
		fixture     string
		wantMessage string
		wantLine    int
		wantColumn  int
	}{
		{
			name:        "$self as a mapping",
			fixture:     "self-not-scalar.yaml",
			wantMessage: "$self at line 7, column 3 must be a scalar URI",
			wantLine:    7,
			wantColumn:  3,
		},
		{
			name:        "timeout as a non-integer scalar",
			fixture:     "timeout-not-integer.yaml",
			wantMessage: `timeout at line 11, column 18 must be an integer number of milliseconds, got "soon"`,
			wantLine:    11,
			wantColumn:  18,
		},
		{
			name:        "dependsOn as a mapping",
			fixture:     "depends-on-not-sequence.yaml",
			wantMessage: "dependsOn at line 12, column 11 must be a sequence of step identifiers",
			wantLine:    12,
			wantColumn:  11,
		},
		{
			name:        "channelPath as a mapping",
			fixture:     "channel-path-not-scalar.yaml",
			wantMessage: "channelPath at line 11, column 11 must be a scalar channel reference",
			wantLine:    11,
			wantColumn:  11,
		},
		{
			name:        "action as a sequence",
			fixture:     "action-not-scalar.yaml",
			wantMessage: "action at line 11, column 11 must be a scalar action name",
			wantLine:    11,
			wantColumn:  11,
		},
		{
			name:        "correlationId as a mapping",
			fixture:     "correlation-id-not-scalar.yaml",
			wantMessage: "correlationId at line 12, column 11 must be a scalar string",
			wantLine:    12,
			wantColumn:  11,
		},
		{
			name:        "action parameters as a scalar",
			fixture:     "action-parameters-not-sequence.yaml",
			wantMessage: "parameters at line 15, column 25 must be a sequence of Parameter Objects",
			wantLine:    15,
			wantColumn:  25,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			specBytes, err := os.ReadFile(filepath.Join("testdata", "negative-11", test.fixture))
			require.NoError(t, err)

			doc, err := libopenapi.NewArazzoDocument(specBytes)
			require.Error(t, err, "a malformed node kind must fail the build")
			assert.Nil(t, doc)
			assert.Contains(t, err.Error(), test.wantMessage)

			// Diagnostics must point at the offending node, never at line 0.
			assert.Positive(t, test.wantLine)
			assert.Positive(t, test.wantColumn)
		})
	}
}

// A well-formed value of the right node kind must still build, even when it is
// semantically wrong. Node-kind enforcement must not stray into validation.
func TestArazzo11WellFormedButInvalidValuesStillBuild(t *testing.T) {
	spec := []byte(`arazzo: 1.1.0
info:
  title: semantically invalid but well formed
  version: 1.0.0
$self: not-a-valid-uri-but-a-scalar
workflows:
  - workflowId: wf
    steps:
      - stepId: s1
        operationId: op
        operationPath: /also/set
        channelPath: /channel
        action: send
        correlationId: some-id
        timeout: -5
        dependsOn:
          - nonexistent-step
`)

	doc, err := libopenapi.NewArazzoDocument(spec)
	require.NoError(t, err, "node kinds are all correct, so the document must build")
	require.NotNil(t, doc)

	assert.Equal(t, "not-a-valid-uri-but-a-scalar", doc.Self)

	step := doc.Workflows[0].Steps[0]
	require.NotNil(t, step.Timeout)
	assert.Equal(t, int64(-5), *step.Timeout, "a negative timeout is the validator's problem")
	assert.Equal(t, "/channel", step.ChannelPath)
	assert.Equal(t, "send", step.Action)
	assert.Equal(t, "some-id", step.CorrelationId)
	assert.Equal(t, []string{"nonexistent-step"}, step.DependsOn)
}

// An explicit YAML null is treated as absent rather than as a node-kind violation, so a
// document that spells out an omitted optional field still builds.
//
// Note the pre-existing builder behavior this pins: for string-typed fields the null
// scalar's literal text ("~") is carried through rather than becoming the empty string.
// That is not introduced by node-kind enforcement, and normalizing it would change the
// model for 1.0 documents too, so it is recorded here rather than silently changed.
func TestArazzo11ExplicitNullTreatedAsAbsent(t *testing.T) {
	spec := []byte(`arazzo: 1.1.0
info:
  title: explicit nulls
  version: 1.0.0
$self: ~
workflows:
  - workflowId: wf
    steps:
      - stepId: s1
        operationId: op
        timeout: ~
        dependsOn: ~
        channelPath: ~
`)

	doc, err := libopenapi.NewArazzoDocument(spec)
	require.NoError(t, err)
	require.NotNil(t, doc)

	step := doc.Workflows[0].Steps[0]

	// Typed and sequence fields resolve to their zero values.
	assert.Nil(t, step.Timeout)
	assert.Empty(t, step.DependsOn)

	// String fields carry the null scalar's literal text; see the note above.
	assert.Equal(t, "~", doc.Self)
	assert.Equal(t, "~", step.ChannelPath)
}
