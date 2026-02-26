// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/pb33f/jsonpath/pkg/jsonpath"
	jsonpathconfig "github.com/pb33f/jsonpath/pkg/jsonpath/config"
	"github.com/pb33f/libopenapi/arazzo/expression"
	high "github.com/pb33f/libopenapi/datamodel/high/arazzo"
)

type cachedCriterionRegex struct {
	regex *regexp.Regexp
	err   error
}

type cachedCriterionJSONPath struct {
	path *jsonpath.JSONPath
	err  error
}

// criterionCaches holds per-Engine caches for compiled criterion patterns.
// Using plain maps instead of sync.Map because Engine is not safe for concurrent use.
type criterionCaches struct {
	regex     map[string]cachedCriterionRegex
	jsonPath  map[string]cachedCriterionJSONPath
	parseExpr func(string) (expression.Expression, error)
}

func newCriterionCaches() *criterionCaches {
	return &criterionCaches{
		regex:    make(map[string]cachedCriterionRegex),
		jsonPath: make(map[string]cachedCriterionJSONPath),
	}
}

// simpleConditionOperators is kept at package level to avoid allocation per call.
var simpleConditionOperators = []string{"==", "!=", ">=", "<=", ">", "<"}

// ClearCriterionCaches is a no-op retained for backward compatibility.
// Criterion caches are now scoped per-Engine instance and cleared via Engine.ClearCaches().
//
// Deprecated: Use Engine.ClearCaches() instead.
func ClearCriterionCaches() {}

// EvaluateCriterion evaluates a single criterion against an expression context.
// This standalone function does not use caching. For cached evaluation, use an Engine.
func EvaluateCriterion(criterion *high.Criterion, exprCtx *expression.Context) (bool, error) {
	return evaluateCriterionImpl(criterion, exprCtx, nil)
}

// evaluateCriterionImpl is the shared implementation that optionally uses caches.
func evaluateCriterionImpl(criterion *high.Criterion, exprCtx *expression.Context, caches *criterionCaches) (bool, error) {
	effectiveType := criterion.GetEffectiveType()

	switch effectiveType {
	case "simple":
		return evaluateSimpleCriterion(criterion, exprCtx, caches)
	case "regex":
		return evaluateRegexCriterion(criterion, exprCtx, caches)
	case "jsonpath":
		return evaluateJSONPathCriterion(criterion, exprCtx, caches)
	case "xpath":
		return false, fmt.Errorf("xpath criterion evaluation is not yet supported")
	default:
		return false, fmt.Errorf("unknown criterion type: %q", effectiveType)
	}
}

func evaluateSimpleCriterion(criterion *high.Criterion, exprCtx *expression.Context, caches *criterionCaches) (bool, error) {
	condition := criterion.Condition

	if criterion.Context != "" {
		val, err := evaluateExprString(criterion.Context, exprCtx, caches)
		if err != nil {
			return false, fmt.Errorf("failed to evaluate context expression: %w", err)
		}
		return evaluateSimpleCondition(condition, val)
	}

	return evaluateSimpleConditionString(condition, exprCtx, caches)
}

func evaluateSimpleCondition(condition string, value any) (bool, error) {
	valStr := sprintValue(value)
	return valStr == condition, nil
}

func evaluateSimpleConditionString(condition string, exprCtx *expression.Context, caches *criterionCaches) (bool, error) {
	trimmed := strings.TrimSpace(condition)
	if trimmed == "" {
		return false, nil
	}

	if b, err := strconv.ParseBool(trimmed); err == nil {
		return b, nil
	}

	leftRaw, op, rightRaw, found := splitSimpleCondition(trimmed)
	if found {
		left, err := evaluateSimpleOperand(leftRaw, exprCtx, caches)
		if err != nil {
			return false, err
		}
		right, err := evaluateSimpleOperand(rightRaw, exprCtx, caches)
		if err != nil {
			return false, err
		}
		return compareSimpleValues(left, right, op)
	}

	val, err := evaluateSimpleOperand(trimmed, exprCtx, caches)
	if err != nil {
		return false, err
	}
	b, ok := val.(bool)
	if !ok {
		return false, fmt.Errorf("simple condition %q did not evaluate to a boolean", condition)
	}
	return b, nil
}

func splitSimpleCondition(input string) (left, op, right string, found bool) {
	// Find where the left operand ends. If input starts with "$", skip past
	// the expression boundary (first unescaped space) so that operators
	// inside JSON pointer paths like "/data/>=threshold" are not matched.
	searchStart := 0
	if strings.HasPrefix(input, "$") {
		if spaceIdx := strings.IndexByte(input, ' '); spaceIdx >= 0 {
			searchStart = spaceIdx
		} else {
			return "", "", "", false
		}
	}
	for _, candidate := range simpleConditionOperators {
		if idx := strings.Index(input[searchStart:], candidate); idx >= 0 {
			idx += searchStart
			left = strings.TrimSpace(input[:idx])
			right = strings.TrimSpace(input[idx+len(candidate):])
			if left == "" || right == "" {
				return "", "", "", false
			}
			return left, candidate, right, true
		}
	}
	return "", "", "", false
}

func evaluateSimpleOperand(operand string, exprCtx *expression.Context, caches *criterionCaches) (any, error) {
	op := strings.TrimSpace(operand)
	if op == "" {
		return "", nil
	}

	if strings.HasPrefix(op, "$") {
		return evaluateExprString(op, exprCtx, caches)
	}

	if (strings.HasPrefix(op, "\"") && strings.HasSuffix(op, "\"")) ||
		(strings.HasPrefix(op, "'") && strings.HasSuffix(op, "'")) {
		return op[1 : len(op)-1], nil
	}

	if b, err := strconv.ParseBool(op); err == nil {
		return b, nil
	}
	if i, err := strconv.ParseInt(op, 10, 64); err == nil {
		return i, nil
	}
	if f, err := strconv.ParseFloat(op, 64); err == nil {
		return f, nil
	}

	return op, nil
}

func compareSimpleValues(left, right any, op string) (bool, error) {
	if ln, lok := numericValue(left); lok {
		if rn, rok := numericValue(right); rok {
			switch op {
			case "==":
				return ln == rn, nil
			case "!=":
				return ln != rn, nil
			case ">":
				return ln > rn, nil
			case "<":
				return ln < rn, nil
			case ">=":
				return ln >= rn, nil
			case "<=":
				return ln <= rn, nil
			}
		}
	}

	ls := sprintValue(left)
	rs := sprintValue(right)
	switch op {
	case "==":
		return ls == rs, nil
	case "!=":
		return ls != rs, nil
	case ">":
		return ls > rs, nil
	case "<":
		return ls < rs, nil
	case ">=":
		return ls >= rs, nil
	case "<=":
		return ls <= rs, nil
	default:
		return false, fmt.Errorf("unsupported operator %q", op)
	}
}

func numericValue(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int8:
		return float64(n), true
	case int16:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint8:
		return float64(n), true
	case uint16:
		return float64(n), true
	case uint32:
		return float64(n), true
	case uint64:
		return float64(n), true
	case float32:
		return float64(n), true
	case float64:
		return n, true
	default:
		return 0, false
	}
}

func evaluateRegexCriterion(criterion *high.Criterion, exprCtx *expression.Context, caches *criterionCaches) (bool, error) {
	if criterion.Context == "" {
		return false, fmt.Errorf("regex criterion requires a context expression")
	}

	val, err := evaluateExprString(criterion.Context, exprCtx, caches)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate context expression: %w", err)
	}

	re, err := compileCriterionRegex(criterion.Condition, caches)
	if err != nil {
		return false, fmt.Errorf("invalid regex pattern %q: %w", criterion.Condition, err)
	}

	valStr := sprintValue(val)
	return re.MatchString(valStr), nil
}

func evaluateJSONPathCriterion(criterion *high.Criterion, exprCtx *expression.Context, caches *criterionCaches) (bool, error) {
	if criterion.Context == "" {
		return false, fmt.Errorf("jsonpath criterion requires a context expression")
	}

	target, err := evaluateExprString(criterion.Context, exprCtx, caches)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate context expression: %w", err)
	}

	path, err := compileCriterionJSONPath(criterion.Condition, caches)
	if err != nil {
		return false, fmt.Errorf("invalid jsonpath %q: %w", criterion.Condition, err)
	}
	node, err := toYAMLNode(target)
	if err != nil {
		return false, fmt.Errorf("failed to prepare context for jsonpath evaluation: %w", err)
	}
	if node == nil {
		return false, nil
	}

	matches := path.Query(node)
	return len(matches) > 0, nil
}

func compileCriterionRegex(raw string, caches *criterionCaches) (*regexp.Regexp, error) {
	if caches != nil {
		if cached, ok := caches.regex[raw]; ok {
			return cached.regex, cached.err
		}
	}
	re, err := regexp.Compile(raw)
	if caches != nil {
		caches.regex[raw] = cachedCriterionRegex{regex: re, err: err}
	}
	return re, err
}

func compileCriterionJSONPath(raw string, caches *criterionCaches) (*jsonpath.JSONPath, error) {
	if caches != nil {
		if cached, ok := caches.jsonPath[raw]; ok {
			return cached.path, cached.err
		}
	}
	path, err := jsonpath.NewPath(raw, jsonpathconfig.WithPropertyNameExtension(), jsonpathconfig.WithLazyContextTracking())
	if caches != nil {
		caches.jsonPath[raw] = cachedCriterionJSONPath{path: path, err: err}
	}
	return path, err
}

// evaluateExprString evaluates a runtime expression string, using the cached parser when available.
func evaluateExprString(input string, ctx *expression.Context, caches *criterionCaches) (any, error) {
	if caches != nil && caches.parseExpr != nil {
		expr, err := caches.parseExpr(input)
		if err != nil {
			return nil, err
		}
		return expression.Evaluate(expr, ctx)
	}
	return expression.EvaluateString(input, ctx)
}

// sprintValue converts a value to its string representation using type-specific fast paths
// to avoid the overhead of fmt.Sprintf for common types.
func sprintValue(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(t)
	default:
		return fmt.Sprintf("%v", v)
	}
}
