// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	low "github.com/pb33f/libopenapi/datamodel/low/arazzo"
	"github.com/pb33f/libopenapi/orderedmap"
	"go.yaml.in/yaml/v4"
)

// Step represents a high-level Arazzo Step Object.
// https://spec.openapis.org/arazzo/v1.1.0#step-object
type Step struct {
	StepId          string                                `json:"stepId,omitempty" yaml:"stepId,omitempty"`
	Description     string                                `json:"description,omitempty" yaml:"description,omitempty"`
	OperationId     string                                `json:"operationId,omitempty" yaml:"operationId,omitempty"`
	OperationPath   string                                `json:"operationPath,omitempty" yaml:"operationPath,omitempty"`
	ChannelPath     string                                `json:"channelPath,omitempty" yaml:"channelPath,omitempty"`
	Action          string                                `json:"action,omitempty" yaml:"action,omitempty"`
	WorkflowId      string                                `json:"workflowId,omitempty" yaml:"workflowId,omitempty"`
	CorrelationId   string                                `json:"correlationId,omitempty" yaml:"correlationId,omitempty"`
	Timeout         *int64                                `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	DependsOn       []string                              `json:"dependsOn,omitempty" yaml:"dependsOn,omitempty"`
	Parameters      []*Parameter                          `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	RequestBody     *RequestBody                          `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
	SuccessCriteria []*Criterion                          `json:"successCriteria,omitempty" yaml:"successCriteria,omitempty"`
	OnSuccess       []*SuccessAction                      `json:"onSuccess,omitempty" yaml:"onSuccess,omitempty"`
	OnFailure       []*FailureAction                      `json:"onFailure,omitempty" yaml:"onFailure,omitempty"`
	Outputs         *orderedmap.Map[string, *OutputValue] `json:"outputs,omitempty" yaml:"outputs,omitempty"`
	Extensions      *orderedmap.Map[string, *yaml.Node]   `json:"-" yaml:"-"`
	low             *low.Step
}

// NewStep creates a new high-level Step instance from a low-level one.
func NewStep(step *low.Step) *Step {
	s := new(Step)
	s.low = step
	if !step.StepId.IsEmpty() {
		s.StepId = step.StepId.Value
	}
	if !step.Description.IsEmpty() {
		s.Description = step.Description.Value
	}
	if !step.OperationId.IsEmpty() {
		s.OperationId = step.OperationId.Value
	}
	if !step.OperationPath.IsEmpty() {
		s.OperationPath = step.OperationPath.Value
	}
	if !step.ChannelPath.IsEmpty() {
		s.ChannelPath = step.ChannelPath.Value
	}
	if !step.Action.IsEmpty() {
		s.Action = step.Action.Value
	}
	if !step.WorkflowId.IsEmpty() {
		s.WorkflowId = step.WorkflowId.Value
	}
	if !step.CorrelationId.IsEmpty() {
		s.CorrelationId = step.CorrelationId.Value
	}
	if !step.Timeout.IsEmpty() {
		timeout := step.Timeout.Value
		s.Timeout = &timeout
	}
	if !step.DependsOn.IsEmpty() {
		s.DependsOn = buildValueSlice(step.DependsOn.Value)
	}
	if !step.Parameters.IsEmpty() {
		s.Parameters = buildSlice(step.Parameters.Value, NewParameter)
	}
	if !step.RequestBody.IsEmpty() {
		s.RequestBody = NewRequestBody(step.RequestBody.Value)
	}
	if !step.SuccessCriteria.IsEmpty() {
		s.SuccessCriteria = buildSlice(step.SuccessCriteria.Value, NewCriterion)
	}
	if !step.OnSuccess.IsEmpty() {
		s.OnSuccess = buildSlice(step.OnSuccess.Value, NewSuccessAction)
	}
	if !step.OnFailure.IsEmpty() {
		s.OnFailure = buildSlice(step.OnFailure.Value, NewFailureAction)
	}
	if !step.Outputs.IsEmpty() {
		s.Outputs = buildOutputValueMap(step.Outputs.Value)
	}
	s.Extensions = high.ExtractExtensions(step.Extensions)
	return s
}

// GoLow returns the low-level Step instance used to create the high-level one.
func (s *Step) GoLow() *low.Step {
	return s.low
}

// GoLowUntyped returns the low-level Step instance with no type.
func (s *Step) GoLowUntyped() any {
	return s.low
}

// Render returns a YAML representation of the Step object as a byte slice.
func (s *Step) Render() ([]byte, error) {
	return yaml.Marshal(s)
}

// MarshalYAML creates a ready to render YAML representation of the Step object.
func (s *Step) MarshalYAML() (any, error) {
	m := orderedmap.New[string, any]()
	if s.StepId != "" {
		m.Set(low.StepIdLabel, s.StepId)
	}
	if s.Description != "" {
		m.Set(low.DescriptionLabel, s.Description)
	}
	if s.OperationId != "" {
		m.Set(low.OperationIdLabel, s.OperationId)
	}
	if s.OperationPath != "" {
		m.Set(low.OperationPathLabel, s.OperationPath)
	}
	if s.ChannelPath != "" {
		m.Set(low.ChannelPathLabel, s.ChannelPath)
	}
	if s.Action != "" {
		m.Set(low.ActionLabel, s.Action)
	}
	if s.WorkflowId != "" {
		m.Set(low.WorkflowIdLabel, s.WorkflowId)
	}
	if s.CorrelationId != "" {
		m.Set(low.CorrelationIdLabel, s.CorrelationId)
	}
	if s.Timeout != nil {
		m.Set(low.TimeoutLabel, *s.Timeout)
	}
	if len(s.DependsOn) > 0 {
		m.Set(low.DependsOnLabel, s.DependsOn)
	}
	if len(s.Parameters) > 0 {
		m.Set(low.ParametersLabel, s.Parameters)
	}
	if s.RequestBody != nil {
		m.Set(low.RequestBodyLabel, s.RequestBody)
	}
	if len(s.SuccessCriteria) > 0 {
		m.Set(low.SuccessCriteriaLabel, s.SuccessCriteria)
	}
	if len(s.OnSuccess) > 0 {
		m.Set(low.OnSuccessLabel, s.OnSuccess)
	}
	if len(s.OnFailure) > 0 {
		m.Set(low.OnFailureLabel, s.OnFailure)
	}
	if s.Outputs != nil && s.Outputs.Len() > 0 {
		m.Set(low.OutputsLabel, s.Outputs)
	}
	marshalExtensions(m, s.Extensions)
	return m, nil
}
