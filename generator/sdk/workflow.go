// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package sdk

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/pb33f/libopenapi/arazzo/expression"
	higharazzo "github.com/pb33f/libopenapi/datamodel/high/arazzo"
	highbase "github.com/pb33f/libopenapi/datamodel/high/base"
	"go.yaml.in/yaml/v4"
)

// PrepareArazzo selects finite Arazzo workflows and binds their operationIds to
// a prepared OpenAPI contract. The current compiler supports one operation step
// with an object payload sourced from workflow inputs and status-code criteria.
func PrepareArazzo(document *higharazzo.Arazzo, contract *Contract, workflowIDs ...string) ([]*Workflow, error) {
	if document == nil {
		return nil, errors.New("sdk: Arazzo document is required")
	}
	if contract == nil {
		return nil, errors.New("sdk: OpenAPI contract is required")
	}
	operations := make(map[string]*Operation, len(contract.Operations))
	for _, operation := range contract.Operations {
		if operation != nil {
			operations[operation.ID] = operation
		}
	}
	selected := make(map[string]struct{}, len(workflowIDs))
	for _, workflowID := range workflowIDs {
		if strings.TrimSpace(workflowID) == "" {
			return nil, errors.New("sdk: selected workflowId cannot be empty")
		}
		selected[workflowID] = struct{}{}
	}
	found := make(map[string]struct{}, len(selected))
	var result []*Workflow
	for _, workflow := range document.Workflows {
		if workflow == nil {
			continue
		}
		if len(selected) > 0 {
			if _, ok := selected[workflow.WorkflowId]; !ok {
				continue
			}
		}
		prepared, err := prepareWorkflow(workflow, operations)
		if err != nil {
			return nil, fmt.Errorf("sdk: prepare Arazzo workflow %q: %w", workflow.WorkflowId, err)
		}
		found[workflow.WorkflowId] = struct{}{}
		result = append(result, prepared)
	}
	for _, workflowID := range workflowIDs {
		if _, ok := found[workflowID]; !ok {
			return nil, fmt.Errorf("sdk: selected workflowId %q was not found", workflowID)
		}
	}
	return result, nil
}

func prepareWorkflow(workflow *higharazzo.Workflow, operations map[string]*Operation) (*Workflow, error) {
	if strings.TrimSpace(workflow.WorkflowId) == "" {
		return nil, errors.New("workflowId is required")
	}
	if workflow.Outputs != nil || len(workflow.DependsOn) > 0 || len(workflow.SuccessActions) > 0 || len(workflow.FailureActions) > 0 || len(workflow.Parameters) > 0 {
		return nil, errors.New("workflow-level dependencies, actions, and parameters are not supported by the finite compiler")
	}
	if len(workflow.Steps) != 1 || workflow.Steps[0] == nil {
		return nil, fmt.Errorf("exactly one operation step is supported, got %d", len(workflow.Steps))
	}
	step := workflow.Steps[0]
	if step.OperationId == "" || step.OperationPath != "" || step.WorkflowId != "" {
		return nil, errors.New("step must reference exactly one operationId")
	}
	if len(step.OnSuccess) > 0 || len(step.OnFailure) > 0 || step.Outputs != nil {
		return nil, errors.New("step actions and outputs are not supported by the finite compiler")
	}
	operation := operations[step.OperationId]
	if operation == nil {
		return nil, fmt.Errorf("operationId %q is not in the prepared OpenAPI contract", step.OperationId)
	}
	prepared := &Workflow{ID: workflow.WorkflowId, Summary: workflow.Summary, Description: workflow.Description, Operation: operation, Inputs: workflow.Inputs}
	parameterBindings, err := prepareParameterBindings(step.Parameters, operation)
	if err != nil {
		return nil, err
	}
	prepared.ParameterBindings = parameterBindings
	if operation.RequestBody != nil && operation.RequestBody.Required && step.RequestBody == nil {
		return nil, errors.New("required OpenAPI request body is not bound by the workflow step")
	}
	if step.RequestBody != nil {
		if operation.RequestBody == nil {
			return nil, errors.New("workflow step declares a request body but the OpenAPI operation does not")
		}
		if len(step.RequestBody.Replacements) > 0 {
			return nil, errors.New("request-body replacements are not supported by the finite compiler")
		}
		contentType := strings.TrimSpace(step.RequestBody.ContentType)
		if contentType == "" {
			return nil, errors.New("workflow request body contentType is required")
		}
		mediaType, ok := operation.RequestBody.Content.Get(contentType)
		if !ok || mediaType == nil {
			return nil, fmt.Errorf("workflow request body contentType %q is not declared by the OpenAPI operation", contentType)
		}
		if mediaType.Schema == nil || mediaType.Schema.Schema() == nil {
			return nil, fmt.Errorf("OpenAPI request body contentType %q has no resolvable schema", contentType)
		}
		bindings, err := preparePayloadBindings(step.RequestBody.Payload)
		if err != nil {
			return nil, err
		}
		if err := validatePayloadBindings(bindings, mediaType.Schema); err != nil {
			return nil, err
		}
		prepared.PayloadBindings = bindings
	}
	for _, criterion := range step.SuccessCriteria {
		if criterion == nil || criterion.Context != "" || criterion.GetEffectiveType() != "simple" {
			return nil, errors.New("only simple status-code success criteria are supported by the finite compiler")
		}
		left, operator, right, found := expression.SplitSimpleCondition(strings.TrimSpace(criterion.Condition))
		expr, parseErr := expression.Parse(left)
		status, statusErr := strconv.Atoi(right)
		if !found || operator != "==" || parseErr != nil || expr.Type != expression.StatusCode || statusErr != nil || status < 100 || status > 599 || right != strconv.Itoa(status) {
			return nil, fmt.Errorf("unsupported success criterion %q", criterion.Condition)
		}
		if len(prepared.SuccessStatuses) > 0 {
			if prepared.SuccessStatuses[0] != right {
				return nil, fmt.Errorf("success criteria %s and %s cannot both match one HTTP response", prepared.SuccessStatuses[0], right)
			}
			continue
		}
		prepared.SuccessStatuses = append(prepared.SuccessStatuses, right)
	}
	if len(prepared.SuccessStatuses) == 0 {
		return nil, errors.New("a status-code success criterion is required")
	}
	return prepared, nil
}

func prepareParameterBindings(parameters []*higharazzo.Parameter, operation *Operation) ([]*ParameterBinding, error) {
	operationParameters := make(map[string]*Parameter, len(operation.Parameters))
	for _, parameter := range operation.Parameters {
		if parameter != nil {
			operationParameters[parameter.In+"\x00"+parameter.Name] = parameter
		}
	}
	bindings := make([]*ParameterBinding, 0, len(parameters))
	bound := make(map[string]struct{}, len(parameters))
	for _, parameter := range parameters {
		if parameter == nil || parameter.Reference != "" || strings.TrimSpace(parameter.Name) == "" || strings.TrimSpace(parameter.In) == "" {
			return nil, errors.New("step parameters must be direct named parameter bindings")
		}
		key := parameter.In + "\x00" + parameter.Name
		if _, exists := operationParameters[key]; !exists {
			return nil, fmt.Errorf("step parameter %s %q is not declared by the OpenAPI operation", parameter.In, parameter.Name)
		}
		if _, exists := bound[key]; exists {
			return nil, fmt.Errorf("step parameter %s %q is bound more than once", parameter.In, parameter.Name)
		}
		if parameter.Value == nil || parameter.Value.Kind != yaml.ScalarNode {
			return nil, fmt.Errorf("step parameter %s %q must bind directly to a workflow input", parameter.In, parameter.Name)
		}
		expr, err := expression.Parse(parameter.Value.Value)
		if err != nil || expr.Type != expression.Inputs || strings.Contains(expr.Name, ".") {
			return nil, fmt.Errorf("unsupported step parameter expression %q", parameter.Value.Value)
		}
		bound[key] = struct{}{}
		bindings = append(bindings, &ParameterBinding{Name: parameter.Name, In: parameter.In, Input: expr.Name})
	}
	for key, parameter := range operationParameters {
		if parameter.Required {
			if _, exists := bound[key]; !exists {
				return nil, fmt.Errorf("required OpenAPI parameter %s %q is not bound by the workflow step", parameter.In, parameter.Name)
			}
		}
	}
	return bindings, nil
}

func validatePayloadBindings(bindings []*PayloadBinding, proxy *highbase.SchemaProxy) error {
	properties := make(map[string]struct{})
	required := make(map[string]struct{})
	if err := collectPayloadShape(proxy, properties, required, make(map[*highbase.SchemaProxy]struct{})); err != nil {
		return err
	}
	bound := make(map[string]struct{}, len(bindings))
	for _, binding := range bindings {
		if binding == nil {
			continue
		}
		if _, ok := properties[binding.Property]; !ok {
			return fmt.Errorf("workflow payload property %q is not declared by the OpenAPI request schema", binding.Property)
		}
		bound[binding.Property] = struct{}{}
	}
	requiredNames := make([]string, 0, len(required))
	for property := range required {
		requiredNames = append(requiredNames, property)
	}
	sort.Strings(requiredNames)
	for _, property := range requiredNames {
		if _, ok := bound[property]; !ok {
			return fmt.Errorf("workflow payload does not bind required OpenAPI property %q", property)
		}
	}
	return nil
}

func collectPayloadShape(proxy *highbase.SchemaProxy, properties, required map[string]struct{}, visited map[*highbase.SchemaProxy]struct{}) error {
	if proxy == nil {
		return errors.New("OpenAPI request body schema is required")
	}
	if _, seen := visited[proxy]; seen {
		return nil
	}
	visited[proxy] = struct{}{}
	schema := proxy.Schema()
	if schema == nil {
		return errors.New("OpenAPI request body schema could not be resolved")
	}
	if len(schema.OneOf) > 0 || len(schema.AnyOf) > 0 {
		return errors.New("oneOf and anyOf request bodies are not supported by the finite compiler")
	}
	if schema.Properties != nil {
		for name := range schema.Properties.FromOldest() {
			properties[name] = struct{}{}
		}
	}
	for _, name := range schema.Required {
		required[name] = struct{}{}
	}
	for _, child := range schema.AllOf {
		if err := collectPayloadShape(child, properties, required, visited); err != nil {
			return err
		}
	}
	return nil
}

func preparePayloadBindings(payload *yaml.Node) ([]*PayloadBinding, error) {
	if payload == nil {
		return nil, nil
	}
	if payload.Kind != yaml.MappingNode || len(payload.Content)%2 != 0 {
		return nil, errors.New("request payload must be an object")
	}
	bindings := make([]*PayloadBinding, 0, len(payload.Content)/2)
	for index := 0; index < len(payload.Content); index += 2 {
		property := payload.Content[index]
		value := payload.Content[index+1]
		if property.Kind != yaml.ScalarNode || value.Kind != yaml.ScalarNode {
			return nil, errors.New("request payload properties must map directly to workflow inputs")
		}
		expr, err := expression.Parse(value.Value)
		if err != nil || expr.Type != expression.Inputs {
			return nil, fmt.Errorf("unsupported payload expression %q", value.Value)
		}
		bindings = append(bindings, &PayloadBinding{Property: property.Value, Input: expr.Name})
	}
	return bindings, nil
}
