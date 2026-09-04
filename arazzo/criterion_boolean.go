// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package arazzo

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/pb33f/libopenapi/arazzo/expression"
)

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

func evaluateBooleanExpr(condition string, exprCtx *expression.Context, caches *criterionCaches) (bool, error) {
	tokens, err := tokenizeSimpleCondition(condition)
	if err != nil {
		return false, err
	}
	p := &boolParser{
		tokens:  tokens,
		exprCtx: exprCtx,
		caches:  caches,
		input:   condition,
	}
	value, err := p.parseOr()
	if err != nil {
		return false, err
	}
	if p.peek().kind != tokEOF {
		return false, fmt.Errorf("unexpected token %q in simple condition %q", p.peek().raw, condition)
	}
	return value, nil
}

type boolParser struct {
	tokens  []boolToken
	pos     int
	exprCtx *expression.Context
	caches  *criterionCaches
	input   string
}

func (p *boolParser) peek() boolToken {
	if p.pos >= len(p.tokens) {
		return boolToken{kind: tokEOF}
	}
	return p.tokens[p.pos]
}

func (p *boolParser) next() boolToken {
	tok := p.peek()
	if tok.kind != tokEOF {
		p.pos++
	}
	return tok
}

func (p *boolParser) parseOr() (bool, error) {
	left, err := p.parseAnd()
	if err != nil {
		if !isOptionalRuntimeValueError(err) {
			return false, err
		}
		left = false
	}
	for p.peek().kind == tokOr {
		p.next()
		right, err := p.parseAnd()
		if err != nil {
			if !isOptionalRuntimeValueError(err) {
				return false, err
			}
			right = false
		}
		left = left || right
	}
	return left, nil
}

func (p *boolParser) parseAnd() (bool, error) {
	left, err := p.parseComparison()
	if err != nil {
		if !isOptionalRuntimeValueError(err) {
			return false, err
		}
		left = false
	}
	for p.peek().kind == tokAnd {
		p.next()
		right, err := p.parseComparison()
		if err != nil {
			if !isOptionalRuntimeValueError(err) {
				return false, err
			}
			right = false
		}
		left = left && right
	}
	return left, nil
}

func (p *boolParser) parseComparison() (bool, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return false, err
	}
	if op, ok := comparisonOp(p.peek().kind); ok {
		p.next()
		right, err := p.parsePrimary()
		if err != nil {
			return false, err
		}
		return compareSimpleValues(left, right, op)
	}
	b, ok := left.(bool)
	if !ok {
		return false, fmt.Errorf("simple condition %q did not evaluate to a boolean", p.input)
	}
	return b, nil
}

func (p *boolParser) parsePrimary() (any, error) {
	tok := p.next()
	switch tok.kind {
	case tokLParen:
		value, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if p.next().kind != tokRParen {
			return nil, fmt.Errorf("missing closing parenthesis in simple condition %q", p.input)
		}
		return value, nil
	case tokOperand:
		value, err := evaluateSimpleOperand(tok.raw, p.exprCtx, p.caches)
		if err != nil && isOptionalRuntimeValueError(err) {
			return nil, nil
		}
		return value, err
	default:
		return nil, fmt.Errorf("unexpected token %q in simple condition %q", tok.raw, p.input)
	}
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
	for i < len(input) {
		for i < len(input) && unicode.IsSpace(rune(input[i])) {
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
			i++
		case rest[0] == ')':
			tokens = append(tokens, boolToken{kind: tokRParen, raw: ")"})
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
				i++
				for i < len(input) && !isBoolOpStart(input[i:]) && input[i] != '(' && input[i] != ')' && !unicode.IsSpace(rune(input[i])) {
					i++
				}
			} else {
				for i < len(input) && !isBoolOpStart(input[i:]) && input[i] != '(' && input[i] != ')' && !unicode.IsSpace(rune(input[i])) {
					i++
				}
			}
			tokens = append(tokens, boolToken{kind: tokOperand, raw: input[start:i]})
		}
	}
	return tokens, nil
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

func isOptionalRuntimeValueError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "not found") ||
		strings.Contains(msg, "no response body available") ||
		strings.Contains(msg, "no request body available") ||
		strings.Contains(msg, "no response headers available")
}
