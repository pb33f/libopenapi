// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package expression

import (
	"fmt"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v4"
)

// Context holds runtime values for expression evaluation.
type Context struct {
	Self            string
	URL             string
	Method          string
	StatusCode      int
	RequestHeaders  map[string]string
	RequestQuery    map[string]string
	RequestPath     map[string]string
	RequestBody     *yaml.Node
	RequestPayload  *yaml.Node
	ResponseHeaders map[string]string
	ResponseQuery   map[string]string
	ResponsePath    map[string]string
	ResponseBody    *yaml.Node
	ResponsePayload *yaml.Node
	MessageHeaders  map[string]string
	MessageQuery    map[string]string
	MessagePath     map[string]string
	MessageBody     *yaml.Node
	MessagePayload  *yaml.Node
	Inputs          map[string]any
	Outputs         map[string]any
	Steps           map[string]*StepContext
	Workflows       map[string]*WorkflowContext
	SourceDescs     map[string]*SourceDescContext
	Components      *ComponentsContext
}

// StepContext holds inputs and outputs for a specific step.
type StepContext struct {
	Inputs  map[string]any
	Outputs map[string]any
}

// WorkflowContext holds inputs and outputs for a specific workflow.
type WorkflowContext struct {
	Inputs  map[string]any
	Outputs map[string]any
}

// SourceDescContext holds resolved source description data.
type SourceDescContext struct {
	URL        string
	Type       string
	Operations map[string]any
	Workflows  map[string]any
	Fields     map[string]any
}

// ComponentsContext holds resolved component data.
type ComponentsContext struct {
	Parameters     map[string]any
	SuccessActions map[string]any
	FailureActions map[string]any
	Inputs         map[string]any
}

// Evaluate resolves a parsed Expression against a Context.
func Evaluate(expr Expression, ctx *Context) (any, error) {
	if ctx == nil {
		return nil, fmt.Errorf("nil context")
	}

	switch expr.Type {
	case Self:
		if ctx.Self == "" {
			return nil, fmt.Errorf("no self URI available")
		}
		return ctx.Self, nil
	case URL:
		return ctx.URL, nil
	case Method:
		return ctx.Method, nil
	case StatusCode:
		return ctx.StatusCode, nil

	case RequestHeader:
		if ctx.RequestHeaders == nil {
			return nil, fmt.Errorf("no request headers available")
		}
		v, ok := ctx.RequestHeaders[expr.Property]
		if !ok {
			return nil, fmt.Errorf("request header %q not found", expr.Property)
		}
		return v, nil

	case RequestQuery:
		if ctx.RequestQuery == nil {
			return nil, fmt.Errorf("no request query parameters available")
		}
		v, ok := ctx.RequestQuery[expr.Property]
		if !ok {
			return nil, fmt.Errorf("request query parameter %q not found", expr.Property)
		}
		return v, nil

	case RequestPath:
		if ctx.RequestPath == nil {
			return nil, fmt.Errorf("no request path parameters available")
		}
		v, ok := ctx.RequestPath[expr.Property]
		if !ok {
			return nil, fmt.Errorf("request path parameter %q not found", expr.Property)
		}
		return v, nil

	case RequestBody:
		return resolveStructuredSource(ctx.RequestBody, expr.JSONPointer, "request body")

	case RequestPayload:
		return resolveStructuredSource(ctx.RequestPayload, expr.JSONPointer, "request payload")

	case ResponseHeader:
		if ctx.ResponseHeaders == nil {
			return nil, fmt.Errorf("no response headers available")
		}
		v, ok := ctx.ResponseHeaders[expr.Property]
		if !ok {
			return nil, fmt.Errorf("response header %q not found", expr.Property)
		}
		return v, nil

	case ResponseQuery:
		if ctx.ResponseQuery == nil {
			return nil, fmt.Errorf("no response query parameters available")
		}
		v, ok := ctx.ResponseQuery[expr.Property]
		if !ok {
			return nil, fmt.Errorf("response query parameter %q not found", expr.Property)
		}
		return v, nil

	case ResponsePath:
		if ctx.ResponsePath == nil {
			return nil, fmt.Errorf("no response path parameters available")
		}
		v, ok := ctx.ResponsePath[expr.Property]
		if !ok {
			return nil, fmt.Errorf("response path parameter %q not found", expr.Property)
		}
		return v, nil

	case ResponseBody:
		return resolveStructuredSource(ctx.ResponseBody, expr.JSONPointer, "response body")

	case ResponsePayload:
		return resolveStructuredSource(ctx.ResponsePayload, expr.JSONPointer, "response payload")

	case MessageHeader:
		if ctx.MessageHeaders == nil {
			return nil, fmt.Errorf("no message headers available")
		}
		v, ok := ctx.MessageHeaders[expr.Property]
		if !ok {
			return nil, fmt.Errorf("message header %q not found", expr.Property)
		}
		return v, nil

	case MessageQuery:
		if ctx.MessageQuery == nil {
			return nil, fmt.Errorf("no message query parameters available")
		}
		v, ok := ctx.MessageQuery[expr.Property]
		if !ok {
			return nil, fmt.Errorf("message query parameter %q not found", expr.Property)
		}
		return v, nil

	case MessagePath:
		if ctx.MessagePath == nil {
			return nil, fmt.Errorf("no message path parameters available")
		}
		v, ok := ctx.MessagePath[expr.Property]
		if !ok {
			return nil, fmt.Errorf("message path parameter %q not found", expr.Property)
		}
		return v, nil

	case MessageBody:
		return resolveStructuredSource(ctx.MessageBody, expr.JSONPointer, "message body")

	case MessagePayload:
		return resolveStructuredSource(ctx.MessagePayload, expr.JSONPointer, "message payload")

	case Inputs:
		if ctx.Inputs == nil {
			return nil, fmt.Errorf("no inputs available")
		}
		v, ok, err := resolveNamedValue(ctx.Inputs, expr.Name)
		if err != nil {
			return nil, fmt.Errorf("input %q: %w", expr.Name, err)
		}
		if !ok {
			return nil, fmt.Errorf("input %q not found", expr.Name)
		}
		return resolveValuePointer(v, expr.JSONPointer)

	case Outputs:
		if ctx.Outputs == nil {
			return nil, fmt.Errorf("no outputs available")
		}
		v, ok, err := resolveNamedValue(ctx.Outputs, expr.Name)
		if err != nil {
			return nil, fmt.Errorf("output %q: %w", expr.Name, err)
		}
		if !ok {
			return nil, fmt.Errorf("output %q not found", expr.Name)
		}
		return resolveValuePointer(v, expr.JSONPointer)

	case Steps:
		return resolveSteps(expr, ctx)

	case Workflows:
		return resolveWorkflows(expr, ctx)

	case SourceDescriptions:
		return resolveSourceDescriptions(expr, ctx)

	case Components:
		return resolveComponents(expr, ctx)

	case ComponentParameters:
		return resolveComponentValue(ctx.Components, expr.Name, "parameter")

	case ComponentSuccessActions:
		return resolveComponentValue(ctx.Components, expr.Name, "success action")

	case ComponentFailureActions:
		return resolveComponentValue(ctx.Components, expr.Name, "failure action")

	default:
		return nil, fmt.Errorf("unsupported expression type: %d", expr.Type)
	}
}

// EvaluateString parses and evaluates a runtime expression string in one call.
func EvaluateString(input string, ctx *Context) (any, error) {
	expr, err := Parse(input)
	if err != nil {
		return nil, err
	}
	return Evaluate(expr, ctx)
}

func resolveStructuredSource(node *yaml.Node, pointer, label string) (any, error) {
	if node == nil {
		return nil, fmt.Errorf("no %s available", label)
	}
	if pointer == "" {
		return node, nil
	}
	if pointer[0] != '/' {
		return nil, fmt.Errorf("JSON pointer must start with '/'")
	}
	return resolveJSONPointer(node, pointer)
}

func resolveComponentValue(components *ComponentsContext, name, componentType string) (any, error) {
	if components == nil {
		return nil, fmt.Errorf("no component %ss available", componentType)
	}
	var values map[string]any
	switch componentType {
	case "parameter":
		values = components.Parameters
	case "success action":
		values = components.SuccessActions
	case "failure action":
		values = components.FailureActions
	default:
		return nil, fmt.Errorf("unsupported component type %q", componentType)
	}
	if values == nil {
		return nil, fmt.Errorf("no component %ss available", componentType)
	}
	value, ok := values[name]
	if !ok {
		return nil, fmt.Errorf("component %s %q not found", componentType, name)
	}
	return value, nil
}

func resolveSteps(expr Expression, ctx *Context) (any, error) {
	if ctx.Steps == nil {
		return nil, fmt.Errorf("no steps context available")
	}
	sc, ok := ctx.Steps[expr.Name]
	if !ok {
		return nil, fmt.Errorf("step %q not found", expr.Name)
	}
	if expr.Tail == "" {
		return sc, nil
	}
	return resolveStepTail(expr.Tail, sc, expr.Name, expr.JSONPointer)
}

func splitTail(tail string) (segment, rest string) {
	if dotIdx := strings.IndexByte(tail, '.'); dotIdx == -1 {
		return tail, ""
	} else {
		return tail[:dotIdx], tail[dotIdx+1:]
	}
}

func resolveStepTail(tail string, sc *StepContext, stepName, pointer string) (any, error) {
	segment, rest := splitTail(tail)

	switch segment {
	case "outputs":
		if sc.Outputs == nil {
			return nil, fmt.Errorf("step %q has no outputs", stepName)
		}
		if rest == "" {
			return sc.Outputs, nil
		}
		v, ok, err := resolveNamedValue(sc.Outputs, rest)
		if err != nil {
			return nil, fmt.Errorf("step %q output %q: %w", stepName, rest, err)
		}
		if !ok {
			return nil, fmt.Errorf("step %q output %q not found", stepName, rest)
		}
		return resolveValuePointer(v, pointer)
	case "inputs":
		if sc.Inputs == nil {
			return nil, fmt.Errorf("step %q has no inputs", stepName)
		}
		if rest == "" {
			return sc.Inputs, nil
		}
		v, ok, err := resolveNamedValue(sc.Inputs, rest)
		if err != nil {
			return nil, fmt.Errorf("step %q input %q: %w", stepName, rest, err)
		}
		if !ok {
			return nil, fmt.Errorf("step %q input %q not found", stepName, rest)
		}
		return resolveValuePointer(v, pointer)
	default:
		return nil, fmt.Errorf("unknown step property %q for step %q", segment, stepName)
	}
}

func resolveWorkflows(expr Expression, ctx *Context) (any, error) {
	if ctx.Workflows == nil {
		return nil, fmt.Errorf("no workflows context available")
	}
	wc, ok := ctx.Workflows[expr.Name]
	if !ok {
		return nil, fmt.Errorf("workflow %q not found", expr.Name)
	}
	if expr.Tail == "" {
		return wc, nil
	}

	segment, rest := splitTail(expr.Tail)

	switch segment {
	case "outputs":
		if wc.Outputs == nil {
			return nil, fmt.Errorf("workflow %q has no outputs", expr.Name)
		}
		if rest == "" {
			return wc.Outputs, nil
		}
		v, ok, err := resolveNamedValue(wc.Outputs, rest)
		if err != nil {
			return nil, fmt.Errorf("workflow %q output %q: %w", expr.Name, rest, err)
		}
		if !ok {
			return nil, fmt.Errorf("workflow %q output %q not found", expr.Name, rest)
		}
		return resolveValuePointer(v, expr.JSONPointer)
	case "inputs":
		if wc.Inputs == nil {
			return nil, fmt.Errorf("workflow %q has no inputs", expr.Name)
		}
		if rest == "" {
			return wc.Inputs, nil
		}
		v, ok, err := resolveNamedValue(wc.Inputs, rest)
		if err != nil {
			return nil, fmt.Errorf("workflow %q input %q: %w", expr.Name, rest, err)
		}
		if !ok {
			return nil, fmt.Errorf("workflow %q input %q not found", expr.Name, rest)
		}
		return resolveValuePointer(v, expr.JSONPointer)
	default:
		return nil, fmt.Errorf("unknown workflow property %q for workflow %q", segment, expr.Name)
	}
}

func resolveSourceDescriptions(expr Expression, ctx *Context) (any, error) {
	if ctx.SourceDescs == nil {
		return nil, fmt.Errorf("no source descriptions context available")
	}
	sd, ok := ctx.SourceDescs[expr.Name]
	if !ok {
		return nil, fmt.Errorf("source description %q not found", expr.Name)
	}
	if expr.Tail == "" {
		return sd, nil
	}
	if value, found := sd.Operations[expr.Tail]; found {
		return value, nil
	}
	if value, found := sd.Workflows[expr.Tail]; found {
		return value, nil
	}
	if value, found := sd.Fields[expr.Tail]; found {
		return value, nil
	}
	switch expr.Tail {
	case "url":
		return sd.URL, nil
	case "type":
		return sd.Type, nil
	}
	return nil, fmt.Errorf("unknown source description property %q for %q", expr.Tail, expr.Name)
}

func resolveComponents(expr Expression, ctx *Context) (any, error) {
	if ctx.Components == nil {
		return nil, fmt.Errorf("no components context available")
	}
	if expr.Tail == "" {
		return nil, fmt.Errorf("incomplete components expression for %q", expr.Name)
	}

	segment, rest := splitTail(expr.Tail)

	var v any
	var ok bool
	switch expr.Name {
	case "parameters":
		if ctx.Components.Parameters == nil {
			return nil, fmt.Errorf("no component parameters available")
		}
		v, ok = ctx.Components.Parameters[segment]
		if !ok {
			return nil, fmt.Errorf("component parameter %q not found", segment)
		}
	case "successActions":
		if ctx.Components.SuccessActions == nil {
			return nil, fmt.Errorf("no component success actions available")
		}
		v, ok = ctx.Components.SuccessActions[segment]
		if !ok {
			return nil, fmt.Errorf("component success action %q not found", segment)
		}
	case "failureActions":
		if ctx.Components.FailureActions == nil {
			return nil, fmt.Errorf("no component failure actions available")
		}
		v, ok = ctx.Components.FailureActions[segment]
		if !ok {
			return nil, fmt.Errorf("component failure action %q not found", segment)
		}
	case "inputs":
		if ctx.Components.Inputs == nil {
			return nil, fmt.Errorf("no component inputs available")
		}
		v, ok = ctx.Components.Inputs[segment]
		if !ok {
			return nil, fmt.Errorf("component input %q not found", segment)
		}
	default:
		return nil, fmt.Errorf("unknown component type %q", expr.Name)
	}

	if rest == "" {
		return v, nil
	}
	return resolveDeepValue(v, rest, expr.Name, segment)
}

// resolveJSONPointer navigates a yaml.Node tree using a JSON Pointer (RFC 6901).
// The pointer should start with "/" (the leading "#" has already been stripped).
func resolveJSONPointer(node *yaml.Node, pointer string) (any, error) {
	if pointer == "" {
		return node, nil
	}

	// Unwrap document nodes
	current := node
	if current.Kind == yaml.DocumentNode && len(current.Content) > 0 {
		current = current.Content[0]
	}

	for _, rawSegment := range strings.Split(pointer[1:], "/") {
		segment := UnescapeJSONPointer(rawSegment)

		switch current.Kind {
		case yaml.MappingNode:
			found := false
			for i := 0; i < len(current.Content)-1; i += 2 {
				if current.Content[i].Value == segment {
					current = current.Content[i+1]
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("JSON pointer segment %q not found", segment)
			}

		case yaml.SequenceNode:
			idx, err := strconv.Atoi(segment)
			if err != nil {
				return nil, fmt.Errorf("invalid array index %q in JSON pointer", segment)
			}
			if idx < 0 || idx >= len(current.Content) {
				return nil, fmt.Errorf("array index %d out of bounds (length %d)", idx, len(current.Content))
			}
			current = current.Content[idx]

		default:
			return nil, fmt.Errorf("cannot traverse into scalar node with pointer segment %q", segment)
		}
	}

	return yamlNodeToValue(current), nil
}

func resolveValuePointer(value any, pointer string) (any, error) {
	if pointer == "" {
		return value, nil
	}
	if pointer[0] != '/' {
		return nil, fmt.Errorf("JSON pointer must start with '/'")
	}
	if node, ok := value.(*yaml.Node); ok {
		return resolveJSONPointer(node, pointer)
	}
	current := value
	for _, rawSegment := range strings.Split(pointer[1:], "/") {
		segment := UnescapeJSONPointer(rawSegment)
		switch typed := current.(type) {
		case map[string]any:
			next, ok := typed[segment]
			if !ok {
				return nil, fmt.Errorf("JSON pointer segment %q not found", segment)
			}
			current = next
		case []any:
			index, err := strconv.Atoi(segment)
			if err != nil {
				return nil, fmt.Errorf("invalid array index %q in JSON pointer", segment)
			}
			if index < 0 || index >= len(typed) {
				return nil, fmt.Errorf("array index %d out of bounds (length %d)", index, len(typed))
			}
			current = typed[index]
		default:
			return nil, fmt.Errorf("cannot traverse into %T with pointer segment %q", current, segment)
		}
	}
	return current, nil
}

// resolveNamedValue preserves exact dotted names, then treats the suffix after
// the longest declared name as member traversal. This supports both legal
// dotted Arazzo names and expressions such as "$inputs.order.id".
func resolveNamedValue(values map[string]any, name string) (any, bool, error) {
	if value, found := values[name]; found {
		return value, true, nil
	}
	for separator := strings.LastIndexByte(name, '.'); separator > 0; separator = strings.LastIndexByte(name[:separator], '.') {
		if value, found := values[name[:separator]]; found {
			resolved, err := resolveDottedValue(value, name[separator+1:])
			return resolved, true, err
		}
	}
	return nil, false, nil
}

func resolveDottedValue(value any, path string) (any, error) {
	current := value
	for _, segment := range strings.Split(path, ".") {
		if node, ok := current.(*yaml.Node); ok {
			resolved, err := resolveJSONPointer(node, "/"+escapeJSONPointer(segment))
			if err != nil {
				return nil, err
			}
			current = resolved
			continue
		}
		values, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("cannot traverse into %T with member %q", current, segment)
		}
		next, found := values[segment]
		if !found {
			return nil, fmt.Errorf("member %q not found", segment)
		}
		current = next
	}
	return current, nil
}

func escapeJSONPointer(segment string) string {
	segment = strings.ReplaceAll(segment, "~", "~0")
	return strings.ReplaceAll(segment, "/", "~1")
}

// UnescapeJSONPointer applies RFC 6901 unescaping: ~1 -> /, ~0 -> ~
func UnescapeJSONPointer(s string) string {
	if !strings.Contains(s, "~") {
		return s
	}
	s = strings.ReplaceAll(s, "~1", "/")
	s = strings.ReplaceAll(s, "~0", "~")
	return s
}

// yamlNodeToValue converts a yaml.Node to a Go native value.
func yamlNodeToValue(node *yaml.Node) any {
	if node == nil {
		return nil
	}
	switch node.Kind {
	case yaml.ScalarNode:
		switch node.Tag {
		case "!!int":
			if v, err := strconv.ParseInt(node.Value, 10, 64); err == nil {
				return v
			}
		case "!!float":
			if v, err := strconv.ParseFloat(node.Value, 64); err == nil {
				return v
			}
		case "!!bool":
			if v, err := strconv.ParseBool(node.Value); err == nil {
				return v
			}
		case "!!null":
			return nil
		}
		return node.Value
	case yaml.MappingNode:
		return node
	case yaml.SequenceNode:
		return node
	default:
		return node
	}
}

// resolveDeepValue traverses into a resolved component value using a dot-separated path.
func resolveDeepValue(v any, path, componentType, componentName string) (any, error) {
	segments := strings.Split(path, ".")
	current := v
	for _, seg := range segments {
		switch typed := current.(type) {
		case map[string]any:
			next, ok := typed[seg]
			if !ok {
				return nil, fmt.Errorf("property %q not found on component %s %q", seg, componentType, componentName)
			}
			current = next
		default:
			return nil, fmt.Errorf("cannot traverse into %T with property %q on component %s %q", current, seg, componentType, componentName)
		}
	}
	return current, nil
}
