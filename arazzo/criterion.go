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

type criterionJSONPathDialect string

const (
	criterionJSONPathRFC9535 criterionJSONPathDialect = "rfc9535"
	criterionJSONPathLegacy  criterionJSONPathDialect = "draft-goessner-dispatch-jsonpath-00"
)

// criterionCaches holds per-Engine caches for compiled criterion patterns and
// simple conditions. Using plain maps instead of sync.Map because Engine is not
// safe for concurrent use.
type criterionCaches struct {
	regex     map[string]cachedCriterionRegex
	jsonPath  map[string]cachedCriterionJSONPath
	simple    map[string]cachedSimpleCondition
	parseExpr func(string) (expression.Expression, error)
}

func newCriterionCaches() *criterionCaches {
	return &criterionCaches{
		regex:    make(map[string]cachedCriterionRegex),
		jsonPath: make(map[string]cachedCriterionJSONPath),
		simple:   make(map[string]cachedSimpleCondition),
	}
}

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
	// Bare boolean literals skip compilation and the cache lookup. Only the
	// exact words are booleans, matching newOperandNode.
	switch trimmed {
	case "":
		return false, nil
	case "true":
		return true, nil
	case "false":
		return false, nil
	}
	// Compound simple conditions may combine literals, operators, and runtime
	// expressions with &&, ||, and parentheses (Arazzo 1.0.1 Criterion Object).
	return evaluateBooleanExpr(trimmed, exprCtx, caches)
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

	switch op {
	case "true":
		return true, nil
	case "false":
		return false, nil
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
	// A null or missing operand has no ordering, so only equality is meaningful.
	// Without this, nil falls through to the string comparison below as "<nil>",
	// which sorts after digits and makes `$response.body#/count > 0` pass on a 204.
	if left == nil || right == nil {
		bothNil := left == nil && right == nil
		switch op {
		case "==":
			return bothNil, nil
		case "!=":
			return !bothNil, nil
		case ">", "<", ">=", "<=":
			return false, nil
		default:
			return false, fmt.Errorf("unsupported operator %q", op)
		}
	}

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

	dialect := criterionJSONPathRFC9535
	if criterion.ExpressionType != nil && criterion.ExpressionType.Version != "" {
		dialect = criterionJSONPathDialect(criterion.ExpressionType.Version)
	}
	path, err := compileCriterionJSONPath(criterion.Condition, dialect, caches)
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

func compileCriterionJSONPath(
	raw string,
	dialect criterionJSONPathDialect,
	caches *criterionCaches,
) (*jsonpath.JSONPath, error) {
	cacheKey := string(dialect) + "\x00" + raw
	if caches != nil {
		if cached, ok := caches.jsonPath[cacheKey]; ok {
			return cached.path, cached.err
		}
	}
	var path *jsonpath.JSONPath
	var err error
	switch dialect {
	case criterionJSONPathRFC9535:
		path, err = jsonpath.NewPath(
			raw,
			jsonpathconfig.WithStrictRFC9535(),
			jsonpathconfig.WithLazyContextTracking(),
		)
	case criterionJSONPathLegacy:
		err = fmt.Errorf("%w: %s", ErrUnsupportedExpressionDialect, dialect)
	default:
		err = fmt.Errorf("unknown JSONPath dialect %q", dialect)
	}
	if caches != nil {
		caches.jsonPath[cacheKey] = cachedCriterionJSONPath{path: path, err: err}
	}
	return path, err
}

// evaluateExprString evaluates a runtime expression string, using the cached parser when available.
func evaluateExprString(input string, ctx *expression.Context, caches *criterionCaches) (any, error) {
	expr, err := parseRuntimeExpr(input, caches)
	if err != nil {
		return nil, err
	}
	return expression.Evaluate(expr, ctx)
}

// parseRuntimeExpr parses a runtime expression, using the cached parser when available.
func parseRuntimeExpr(input string, caches *criterionCaches) (expression.Expression, error) {
	if caches != nil && caches.parseExpr != nil {
		return caches.parseExpr(input)
	}
	return expression.Parse(input)
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

type boolTokenKind int

const (
	tokEOF boolTokenKind = iota
	tokOperand
	tokAnd
	tokOr
	tokEq
	tokNeq
	tokGt
	tokLt
	tokGte
	tokLte
	tokLParen
	tokRParen
)

type boolToken struct {
	kind boolTokenKind
	raw  string
}

// compiledCondition is a simple condition parsed once into an evaluation tree.
// Literal operands are converted at compile time. Runtime expressions keep their
// raw text and are resolved on every evaluation through evaluateExprString, so
// they still go through the engine's expression cache.
type compiledCondition struct {
	root  *conditionNode
	input string
}

type conditionNodeKind uint8

const (
	condOperand conditionNodeKind = iota
	condGroup
	condOr
	condAnd
	condCompare
)

type conditionNode struct {
	kind  conditionNodeKind
	op    string         // comparison operator, condCompare only
	left  *conditionNode // condOr, condAnd, condCompare; the inner node for condGroup
	right *conditionNode // condOr, condAnd, condCompare
	// Operand fields.
	raw     string // runtime expression text when isExpr is true
	isExpr  bool
	literal any // pre-converted literal when isExpr is false
}

type cachedSimpleCondition struct {
	cond *compiledCondition
	err  error
}

func evaluateBooleanExpr(condition string, exprCtx *expression.Context, caches *criterionCaches) (bool, error) {
	cond, err := compileSimpleCondition(condition, caches)
	if err != nil {
		return false, err
	}
	return cond.truth(cond.root, exprCtx, caches)
}

// compileSimpleCondition returns the compiled tree for condition, caching both
// successful compiles and syntax errors when caches is non-nil.
func compileSimpleCondition(condition string, caches *criterionCaches) (*compiledCondition, error) {
	if caches != nil {
		if cached, ok := caches.simple[condition]; ok {
			return cached.cond, cached.err
		}
	}
	cond, err := parseSimpleCondition(condition)
	if caches != nil {
		caches.simple[condition] = cachedSimpleCondition{cond: cond, err: err}
	}
	return cond, err
}

func parseSimpleCondition(condition string) (*compiledCondition, error) {
	tokens, err := tokenizeSimpleCondition(condition)
	if err != nil {
		return nil, err
	}
	p := &conditionParser{tokens: tokens, input: condition}
	root, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if p.peek().kind != tokEOF {
		return nil, fmt.Errorf("unexpected token %q in simple condition %q", p.peek().raw, condition)
	}
	return &compiledCondition{root: root, input: condition}, nil
}

// truth evaluates n in a boolean position.
func (c *compiledCondition) truth(n *conditionNode, exprCtx *expression.Context, caches *criterionCaches) (bool, error) {
	switch n.kind {
	case condOr, condAnd:
		left, err := c.truth(n.left, exprCtx, caches)
		if err != nil {
			return false, err
		}
		// Short-circuit: a true left side decides ||, a false one decides &&,
		// and the right side is never evaluated, so its errors cannot surface.
		if n.kind == condOr && left {
			return true, nil
		}
		if n.kind == condAnd && !left {
			return false, nil
		}
		return c.truth(n.right, exprCtx, caches)
	case condCompare:
		left, err := c.value(n.left, exprCtx, caches)
		if err != nil {
			return false, err
		}
		right, err := c.value(n.right, exprCtx, caches)
		if err != nil {
			return false, err
		}
		return compareSimpleValues(left, right, n.op)
	case condGroup:
		return c.truth(n.left, exprCtx, caches)
	default:
		v, err := c.value(n, exprCtx, caches)
		if err != nil {
			return false, err
		}
		// A missing runtime value (value maps it to nil) or an explicit null
		// used as a bare operand is falsy, so `$statusCode == 204 || $response.body#/flag`
		// still passes on an empty 204 response.
		if v == nil {
			return false, nil
		}
		b, ok := v.(bool)
		if !ok {
			return false, fmt.Errorf("simple condition %q did not evaluate to a boolean", c.input)
		}
		return b, nil
	}
}

// value evaluates n in a comparison-operand position.
func (c *compiledCondition) value(n *conditionNode, exprCtx *expression.Context, caches *criterionCaches) (any, error) {
	if n.kind != condOperand {
		return c.truth(n, exprCtx, caches)
	}
	if !n.isExpr {
		return n.literal, nil
	}
	expr, err := parseRuntimeExpr(n.raw, caches)
	if err != nil {
		return nil, err
	}
	v, err := expression.Evaluate(expr, exprCtx)
	if err != nil && exprCtx != nil && isOptionalRuntimeValue(expr.Type) {
		return nil, nil
	}
	return v, err
}

type conditionParser struct {
	tokens []boolToken
	pos    int
	input  string
}

func (p *conditionParser) peek() boolToken {
	if p.pos >= len(p.tokens) {
		return boolToken{kind: tokEOF}
	}
	return p.tokens[p.pos]
}

func (p *conditionParser) next() boolToken {
	tok := p.peek()
	if tok.kind != tokEOF {
		p.pos++
	}
	return tok
}

func (p *conditionParser) parseOr() (*conditionNode, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.peek().kind == tokOr {
		p.next()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &conditionNode{kind: condOr, left: left, right: right}
	}
	return left, nil
}

func (p *conditionParser) parseAnd() (*conditionNode, error) {
	left, err := p.parseComparison()
	if err != nil {
		return nil, err
	}
	for p.peek().kind == tokAnd {
		p.next()
		right, err := p.parseComparison()
		if err != nil {
			return nil, err
		}
		left = &conditionNode{kind: condAnd, left: left, right: right}
	}
	return left, nil
}

func (p *conditionParser) parseComparison() (*conditionNode, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	if op, ok := comparisonOp(p.peek().kind); ok {
		p.next()
		right, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		return &conditionNode{kind: condCompare, op: op, left: left, right: right}, nil
	}
	return left, nil
}

func (p *conditionParser) parsePrimary() (*conditionNode, error) {
	tok := p.next()
	switch tok.kind {
	case tokLParen:
		inner, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if p.next().kind != tokRParen {
			return nil, fmt.Errorf("missing closing parenthesis in simple condition %q", p.input)
		}
		// Keep the group node: a parenthesised operand is coerced to a boolean
		// before it is compared, e.g. `($flag) == true`.
		return &conditionNode{kind: condGroup, left: inner}, nil
	case tokOperand:
		return newOperandNode(tok.raw), nil
	default:
		return nil, fmt.Errorf("unexpected token %q in simple condition %q", tok.raw, p.input)
	}
}

func newOperandNode(raw string) *conditionNode {
	op := strings.TrimSpace(raw)
	if strings.HasPrefix(op, "$") {
		return &conditionNode{kind: condOperand, raw: op, isExpr: true}
	}
	// Literals never touch the expression context, so they convert once here.
	literal, _ := evaluateSimpleOperand(op, nil, nil)
	return &conditionNode{kind: condOperand, literal: literal}
}

func comparisonOp(kind boolTokenKind) (string, bool) {
	switch kind {
	case tokEq:
		return "==", true
	case tokNeq:
		return "!=", true
	case tokGte:
		return ">=", true
	case tokLte:
		return "<=", true
	case tokGt:
		return ">", true
	case tokLt:
		return "<", true
	default:
		return "", false
	}
}

func tokenizeSimpleCondition(input string) ([]boolToken, error) {
	var tokens []boolToken
	i := 0
	depth := 0 // open parentheses so far; the parser reports any imbalance
	for i < len(input) {
		for i < len(input) && isASCIISpace(input[i]) {
			i++
		}
		if i >= len(input) {
			break
		}
		rest := input[i:]
		switch {
		case strings.HasPrefix(rest, "&&"):
			tokens = append(tokens, boolToken{kind: tokAnd, raw: "&&"})
			i += 2
		case strings.HasPrefix(rest, "||"):
			tokens = append(tokens, boolToken{kind: tokOr, raw: "||"})
			i += 2
		case strings.HasPrefix(rest, "=="):
			tokens = append(tokens, boolToken{kind: tokEq, raw: "=="})
			i += 2
		case strings.HasPrefix(rest, "!="):
			tokens = append(tokens, boolToken{kind: tokNeq, raw: "!="})
			i += 2
		case strings.HasPrefix(rest, ">="):
			tokens = append(tokens, boolToken{kind: tokGte, raw: ">="})
			i += 2
		case strings.HasPrefix(rest, "<="):
			tokens = append(tokens, boolToken{kind: tokLte, raw: "<="})
			i += 2
		case rest[0] == '>':
			tokens = append(tokens, boolToken{kind: tokGt, raw: ">"})
			i++
		case rest[0] == '<':
			tokens = append(tokens, boolToken{kind: tokLt, raw: "<"})
			i++
		case rest[0] == '(':
			tokens = append(tokens, boolToken{kind: tokLParen, raw: "("})
			depth++
			i++
		case rest[0] == ')':
			tokens = append(tokens, boolToken{kind: tokRParen, raw: ")"})
			depth--
			i++
		default:
			start := i
			if rest[0] == '\'' || rest[0] == '"' {
				q := rest[0]
				i++
				for i < len(input) && input[i] != q {
					i++
				}
				if i >= len(input) {
					return nil, fmt.Errorf("unterminated string in simple condition %q", input)
				}
				i++
			} else if rest[0] == '$' {
				i = runtimeOperandEnd(input, i, depth)
			} else {
				for i < len(input) && !isBoolOpStart(input[i:]) && input[i] != '(' && input[i] != ')' && !isASCIISpace(input[i]) {
					i++
				}
			}
			tokens = append(tokens, boolToken{kind: tokOperand, raw: input[start:i]})
		}
	}
	return tokens, nil
}

// runtimeOperandEnd returns the end of the runtime expression starting at
// input[start]. JSON Pointers, header names and parameter names may contain
// operator characters, so, as with expression.SplitSimpleCondition, a runtime
// expression runs to the next whitespace. Trailing ')' characters the
// expression does not balance itself close open groups instead, so
// `($response.body#/flag)` keeps working while `#/f(x)` stays intact.
func runtimeOperandEnd(input string, start, depth int) int {
	end := start + 1
	opens, closes := 0, 0
	for end < len(input) && !isASCIISpace(input[end]) {
		switch input[end] {
		case '(':
			opens++
		case ')':
			closes++
		}
		end++
	}
	for unmatched := closes - opens; unmatched > 0 && depth > 0 && input[end-1] == ')'; unmatched-- {
		end--
		depth--
	}
	return end
}

// isASCIISpace reports whether c is ASCII whitespace. The tokenizer works on
// bytes, and unicode.IsSpace(rune(c)) would misread UTF-8 continuation bytes
// such as 0x85 (in "Å") and 0xA0 (in "à") as NEL and NBSP.
func isASCIISpace(c byte) bool {
	switch c {
	case ' ', '\t', '\n', '\r', '\v', '\f':
		return true
	}
	return false
}

func isBoolOpStart(s string) bool {
	if s == "" {
		return false
	}
	return strings.HasPrefix(s, "&&") ||
		strings.HasPrefix(s, "||") ||
		strings.HasPrefix(s, "==") ||
		strings.HasPrefix(s, "!=") ||
		strings.HasPrefix(s, ">=") ||
		strings.HasPrefix(s, "<=") ||
		s[0] == '>' ||
		s[0] == '<'
}

// isOptionalRuntimeValue reports whether an expression of type t reads a value
// that legitimately varies per request or response: a header, query or path
// parameter, or the body.
func isOptionalRuntimeValue(t expression.ExpressionType) bool {
	switch t {
	case expression.RequestHeader, expression.RequestQuery, expression.RequestPath, expression.RequestBody,
		expression.ResponseHeader, expression.ResponseQuery, expression.ResponsePath, expression.ResponseBody:
		return true
	default:
		return false
	}
}
