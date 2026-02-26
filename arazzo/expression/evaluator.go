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
	URL             string
	Method          string
	StatusCode      int
	RequestHeaders  map[string]string
	RequestQuery    map[string]string
	RequestPath     map[string]string
	RequestBody     *yaml.Node
	ResponseHeaders map[string]string
	ResponseBody    *yaml.Node
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
	URL string
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
		if ctx.RequestBody == nil {
			return nil, fmt.Errorf("no request body available")
		}
		if expr.JSONPointer == "" {
			return ctx.RequestBody, nil
		}
		return resolveJSONPointer(ctx.RequestBody, expr.JSONPointer)

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
		return nil, fmt.Errorf("response query parameters are not supported")

	case ResponsePath:
		return nil, fmt.Errorf("response path parameters are not supported")

	case ResponseBody:
		if ctx.ResponseBody == nil {
			return nil, fmt.Errorf("no response body available")
		}
		if expr.JSONPointer == "" {
			return ctx.ResponseBody, nil
		}
		return resolveJSONPointer(ctx.ResponseBody, expr.JSONPointer)

	case Inputs:
		if ctx.Inputs == nil {
			return nil, fmt.Errorf("no inputs available")
		}
		v, ok := ctx.Inputs[expr.Name]
		if !ok {
			return nil, fmt.Errorf("input %q not found", expr.Name)
		}
		return v, nil

	case Outputs:
		if ctx.Outputs == nil {
			return nil, fmt.Errorf("no outputs available")
		}
		v, ok := ctx.Outputs[expr.Name]
		if !ok {
			return nil, fmt.Errorf("output %q not found", expr.Name)
		}
		return v, nil

	case Steps:
		return resolveSteps(expr, ctx)

	case Workflows:
		return resolveWorkflows(expr, ctx)

	case SourceDescriptions:
		return resolveSourceDescriptions(expr, ctx)

	case Components:
		return resolveComponents(expr, ctx)

	case ComponentParameters:
		if ctx.Components == nil || ctx.Components.Parameters == nil {
			return nil, fmt.Errorf("no component parameters available")
		}
		v, ok := ctx.Components.Parameters[expr.Name]
		if !ok {
			return nil, fmt.Errorf("component parameter %q not found", expr.Name)
		}
		return v, nil

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
	return resolveStepTail(expr.Tail, sc, expr.Name)
}

func splitTail(tail string) (segment, rest string) {
	if dotIdx := strings.IndexByte(tail, '.'); dotIdx == -1 {
		return tail, ""
	} else {
		return tail[:dotIdx], tail[dotIdx+1:]
	}
}

func resolveStepTail(tail string, sc *StepContext, stepName string) (any, error) {
	segment, rest := splitTail(tail)

	switch segment {
	case "outputs":
		if sc.Outputs == nil {
			return nil, fmt.Errorf("step %q has no outputs", stepName)
		}
		if rest == "" {
			return sc.Outputs, nil
		}
		v, ok := sc.Outputs[rest]
		if !ok {
			return nil, fmt.Errorf("step %q output %q not found", stepName, rest)
		}
		return v, nil
	case "inputs":
		if sc.Inputs == nil {
			return nil, fmt.Errorf("step %q has no inputs", stepName)
		}
		if rest == "" {
			return sc.Inputs, nil
		}
		v, ok := sc.Inputs[rest]
		if !ok {
			return nil, fmt.Errorf("step %q input %q not found", stepName, rest)
		}
		return v, nil
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
		v, ok := wc.Outputs[rest]
		if !ok {
			return nil, fmt.Errorf("workflow %q output %q not found", expr.Name, rest)
		}
		return v, nil
	case "inputs":
		if wc.Inputs == nil {
			return nil, fmt.Errorf("workflow %q has no inputs", expr.Name)
		}
		if rest == "" {
			return wc.Inputs, nil
		}
		v, ok := wc.Inputs[rest]
		if !ok {
			return nil, fmt.Errorf("workflow %q input %q not found", expr.Name, rest)
		}
		return v, nil
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
	if expr.Tail == "url" {
		return sd.URL, nil
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
	if pointer == "" || pointer == "/" {
		return node, nil
	}

	// Unwrap document nodes
	current := node
	if current.Kind == yaml.DocumentNode && len(current.Content) > 0 {
		current = current.Content[0]
	}

	pos := 0
	if pointer[0] == '/' {
		pos = 1
	}

	for pos < len(pointer) {
		// Find next segment boundary
		nextSlash := strings.IndexByte(pointer[pos:], '/')
		var segment string
		if nextSlash == -1 {
			segment = pointer[pos:]
			pos = len(pointer)
		} else {
			segment = pointer[pos : pos+nextSlash]
			pos = pos + nextSlash + 1
		}

		// Unescape JSON Pointer: ~1 -> /, ~0 -> ~
		segment = UnescapeJSONPointer(segment)

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
