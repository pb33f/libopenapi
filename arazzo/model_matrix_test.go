// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"encoding/json"
	"sync"
	"testing"

	libopenapi "github.com/pb33f/libopenapi"
	"github.com/pb33f/testify/assert"
	"github.com/pb33f/testify/require"
	"go.yaml.in/yaml/v4"
)

const arazzo10MatrixYAML = `arazzo: 1.0.1
info:
  title: Compatibility workflow
  version: 1.0.0
sourceDescriptions:
  - name: petstore
    url: https://api.example.com/petstore.yaml
    type: openapi
workflows:
  - workflowId: fetchPet
    steps:
      - stepId: fetch
        operationId: getPet
        parameters:
          - name: petId
            in: path
            value: $inputs.petId
        outputs:
          petId: $response.body#/id
    outputs:
      petId: $steps.fetch.outputs.petId
`

// An Arazzo document is valid JSON as well as YAML. Parsing the JSON form must produce
// the same high-level model as the YAML form; only 1.1 had JSON coverage previously.
func TestArazzo10_JSONInputMatchesYAMLModel(t *testing.T) {
	var intermediate any
	require.NoError(t, yaml.Unmarshal([]byte(arazzo10MatrixYAML), &intermediate))
	jsonBytes, err := json.Marshal(intermediate)
	require.NoError(t, err)

	fromYAML, err := libopenapi.NewArazzoDocument([]byte(arazzo10MatrixYAML))
	require.NoError(t, err)
	fromJSON, err := libopenapi.NewArazzoDocument(jsonBytes)
	require.NoError(t, err)

	assert.Equal(t, fromYAML.Arazzo, fromJSON.Arazzo)
	assert.Equal(t, fromYAML.Info.Title, fromJSON.Info.Title)
	assert.Equal(t, fromYAML.Info.Version, fromJSON.Info.Version)

	require.Len(t, fromJSON.SourceDescriptions, 1)
	assert.Equal(t, fromYAML.SourceDescriptions[0].Name, fromJSON.SourceDescriptions[0].Name)
	assert.Equal(t, fromYAML.SourceDescriptions[0].URL, fromJSON.SourceDescriptions[0].URL)
	assert.Equal(t, fromYAML.SourceDescriptions[0].Type, fromJSON.SourceDescriptions[0].Type)

	require.Len(t, fromJSON.Workflows, 1)
	yamlWorkflow, jsonWorkflow := fromYAML.Workflows[0], fromJSON.Workflows[0]
	assert.Equal(t, yamlWorkflow.WorkflowId, jsonWorkflow.WorkflowId)

	require.Len(t, jsonWorkflow.Steps, 1)
	yamlStep, jsonStep := yamlWorkflow.Steps[0], jsonWorkflow.Steps[0]
	assert.Equal(t, yamlStep.StepId, jsonStep.StepId)
	assert.Equal(t, yamlStep.OperationId, jsonStep.OperationId)

	require.Len(t, jsonStep.Parameters, 1)
	assert.Equal(t, yamlStep.Parameters[0].Name, jsonStep.Parameters[0].Name)
	assert.Equal(t, yamlStep.Parameters[0].In, jsonStep.Parameters[0].In)

	// Output unions must survive the JSON round trip identically.
	yamlOutput, ok := yamlStep.Outputs.Get("petId")
	require.True(t, ok)
	jsonOutput, ok := jsonStep.Outputs.Get("petId")
	require.True(t, ok)
	yamlExpression, ok := yamlOutput.GetExpression()
	require.True(t, ok)
	jsonExpression, ok := jsonOutput.GetExpression()
	require.True(t, ok)
	assert.Equal(t, yamlExpression, jsonExpression)
}

// The low model keeps the original YAML nodes, so authored comments must remain
// reachable. Comments are the main reason the low model exists at all.
func TestArazzo10_CommentsSurviveOnLowNodes(t *testing.T) {
	spec := []byte(`# document level comment
arazzo: 1.0.1
info:
  # info level comment
  title: commented # trailing title comment
  version: 1.0.0
workflows:
  - workflowId: wf
    steps:
      - stepId: s1
        operationId: op
`)

	doc, err := libopenapi.NewArazzoDocument(spec)
	require.NoError(t, err)

	low := doc.GoLow()
	require.NotNil(t, low)

	assert.Equal(t, "# document level comment", low.Arazzo.KeyNode.HeadComment)

	infoRoot := low.Info.Value.GetRootNode()
	require.NotNil(t, infoRoot)

	var titleKey, titleValue *yaml.Node
	for i := 0; i < len(infoRoot.Content)-1; i += 2 {
		if infoRoot.Content[i].Value == "title" {
			titleKey, titleValue = infoRoot.Content[i], infoRoot.Content[i+1]
			break
		}
	}
	require.NotNil(t, titleKey, "title key node should be reachable from the low model")

	assert.Equal(t, "# info level comment", titleKey.HeadComment)
	assert.Equal(t, "# trailing title comment", titleValue.LineComment)
}

// Duplicate mapping keys are not rejected; the first occurrence wins and later ones are
// discarded. This pins the current parser policy so a change to it is a deliberate one.
func TestArazzo10_DuplicateKeysFirstOccurrenceWins(t *testing.T) {
	spec := []byte(`arazzo: 1.0.1
info:
  title: first
  version: 1.0.0
info:
  title: second
  version: 2.0.0
workflows:
  - workflowId: wf
    steps:
      - stepId: s1
        operationId: op
`)

	doc, err := libopenapi.NewArazzoDocument(spec)
	require.NoError(t, err)
	assert.Equal(t, "first", doc.Info.Title)
	assert.Equal(t, "1.0.0", doc.Info.Version)
}

// Building the same bytes repeatedly must produce equivalent, independent documents.
// Arazzo models share package-level hashing infrastructure, so a leak between builds
// would show up as a divergent model or hash on a later construction.
func TestArazzo10_RepeatedConstructionIsIndependent(t *testing.T) {
	first, err := libopenapi.NewArazzoDocument([]byte(arazzo10MatrixYAML))
	require.NoError(t, err)
	firstRender, err := first.Render()
	require.NoError(t, err)
	firstHash := first.GoLow().Hash()

	for i := 0; i < 5; i++ {
		next, buildErr := libopenapi.NewArazzoDocument([]byte(arazzo10MatrixYAML))
		require.NoError(t, buildErr)

		nextRender, renderErr := next.Render()
		require.NoError(t, renderErr)
		assert.Equal(t, string(firstRender), string(nextRender), "render diverged on build %d", i)
		assert.Equal(t, firstHash, next.GoLow().Hash(), "hash diverged on build %d", i)

		// Mutating a later document must not affect the first.
		next.Info.Title = "mutated"
		assert.Equal(t, "Compatibility workflow", first.Info.Title)
	}
}

// Concurrent construction and rendering of independent documents must be race free.
func TestArazzo10_ConcurrentConstructionIsRaceFree(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			doc, err := libopenapi.NewArazzoDocument([]byte(arazzo10MatrixYAML))
			assert.NoError(t, err)
			_, err = doc.Render()
			assert.NoError(t, err)
			_ = doc.GoLow().Hash()
		}()
	}
	wg.Wait()
}
