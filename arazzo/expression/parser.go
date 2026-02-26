// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package expression

import (
	"fmt"
	"strings"
)

// tcharTable is a 128-byte lookup table for RFC 7230 token characters.
// tchar = "!" / "#" / "$" / "%" / "&" / "'" / "*" / "+" / "-" / "." /
//
//	"^" / "_" / "`" / "|" / "~" / DIGIT / ALPHA
var tcharTable [128]bool

func init() {
	for c := 'a'; c <= 'z'; c++ {
		tcharTable[c] = true
	}
	for c := 'A'; c <= 'Z'; c++ {
		tcharTable[c] = true
	}
	for c := '0'; c <= '9'; c++ {
		tcharTable[c] = true
	}
	for _, c := range "!#$%&'*+-.^_`|~" {
		tcharTable[c] = true
	}
}

func isTchar(c byte) bool {
	return c < 128 && tcharTable[c]
}

// Parse parses a single Arazzo runtime expression. Returns a value type to avoid heap allocation.
func Parse(input string) (Expression, error) {
	if len(input) == 0 {
		return Expression{}, fmt.Errorf("empty expression")
	}
	if input[0] != '$' {
		return Expression{}, fmt.Errorf("expression must start with '$', got %q", string(input[0]))
	}

	expr := Expression{Raw: input}

	if len(input) < 2 {
		return Expression{}, fmt.Errorf("incomplete expression: %q", input)
	}

	// Fast prefix dispatch on second character
	switch input[1] {
	case 'u': // $url
		if input == "$url" {
			expr.Type = URL
			return expr, nil
		}
		return Expression{}, fmt.Errorf("unknown expression: %q", input)

	case 'm': // $method
		if input == "$method" {
			expr.Type = Method
			return expr, nil
		}
		return Expression{}, fmt.Errorf("unknown expression: %q", input)

	case 's': // $statusCode, $steps., $sourceDescriptions.
		if input == "$statusCode" {
			expr.Type = StatusCode
			return expr, nil
		}
		if strings.HasPrefix(input, "$steps.") {
			return parseNamedExpression(input, "$steps.", Steps)
		}
		if strings.HasPrefix(input, "$sourceDescriptions.") {
			return parseNamedExpression(input, "$sourceDescriptions.", SourceDescriptions)
		}
		return Expression{}, fmt.Errorf("unknown expression: %q", input)

	case 'r': // $request., $response.
		if strings.HasPrefix(input, "$request.") {
			return parseSource(input, "$request.", RequestHeader, RequestQuery, RequestPath, RequestBody)
		}
		if strings.HasPrefix(input, "$response.") {
			return parseSource(input, "$response.", ResponseHeader, ResponseQuery, ResponsePath, ResponseBody)
		}
		return Expression{}, fmt.Errorf("unknown expression: %q", input)

	case 'i': // $inputs.
		if strings.HasPrefix(input, "$inputs.") {
			expr.Type = Inputs
			expr.Name = input[len("$inputs."):]
			if expr.Name == "" {
				return Expression{}, fmt.Errorf("empty name in expression: %q", input)
			}
			return expr, nil
		}
		return Expression{}, fmt.Errorf("unknown expression: %q", input)

	case 'o': // $outputs.
		if strings.HasPrefix(input, "$outputs.") {
			expr.Type = Outputs
			expr.Name = input[len("$outputs."):]
			if expr.Name == "" {
				return Expression{}, fmt.Errorf("empty name in expression: %q", input)
			}
			return expr, nil
		}
		return Expression{}, fmt.Errorf("unknown expression: %q", input)

	case 'w': // $workflows.
		if strings.HasPrefix(input, "$workflows.") {
			return parseNamedExpression(input, "$workflows.", Workflows)
		}
		return Expression{}, fmt.Errorf("unknown expression: %q", input)

	case 'c': // $components.
		if strings.HasPrefix(input, "$components.") {
			return parseComponents(input)
		}
		return Expression{}, fmt.Errorf("unknown expression: %q", input)

	default:
		return Expression{}, fmt.Errorf("unknown expression prefix: %q", input)
	}
}

// parseSource parses $request.{source} or $response.{source} expressions.
func parseSource(input, prefix string, headerType, queryType, pathType, bodyType ExpressionType) (Expression, error) {
	expr := Expression{Raw: input}
	rest := input[len(prefix):]
	if len(rest) == 0 {
		return Expression{}, fmt.Errorf("incomplete source expression: %q", input)
	}

	if strings.HasPrefix(rest, "header.") {
		expr.Type = headerType
		name := rest[len("header."):]
		if name == "" {
			return Expression{}, fmt.Errorf("empty header name in expression: %q", input)
		}
		// Validate tchar for header names
		for i := 0; i < len(name); i++ {
			if !isTchar(name[i]) {
				return Expression{}, fmt.Errorf("invalid character %q at position %d in header name: %q", name[i], len(prefix)+len("header.")+i, input)
			}
		}
		expr.Property = name
		return expr, nil
	}

	if strings.HasPrefix(rest, "query.") {
		expr.Type = queryType
		name := rest[len("query."):]
		if name == "" {
			return Expression{}, fmt.Errorf("empty query name in expression: %q", input)
		}
		expr.Property = name
		return expr, nil
	}

	if strings.HasPrefix(rest, "path.") {
		expr.Type = pathType
		name := rest[len("path."):]
		if name == "" {
			return Expression{}, fmt.Errorf("empty path name in expression: %q", input)
		}
		expr.Property = name
		return expr, nil
	}

	if rest == "body" || strings.HasPrefix(rest, "body#") {
		expr.Type = bodyType
		if strings.HasPrefix(rest, "body#") {
			expr.JSONPointer = rest[len("body#"):]
		}
		return expr, nil
	}

	return Expression{}, fmt.Errorf("unknown source type in expression: %q", input)
}

// parseNamedExpression parses expressions like $steps.{name}[.tail], $workflows.{name}[.tail], etc.
func parseNamedExpression(input, prefix string, exprType ExpressionType) (Expression, error) {
	expr := Expression{Raw: input, Type: exprType}
	rest := input[len(prefix):]
	if rest == "" {
		return Expression{}, fmt.Errorf("empty name in expression: %q", input)
	}

	// Find the first dot to split name from tail
	dotIdx := strings.IndexByte(rest, '.')
	if dotIdx == -1 {
		expr.Name = rest
	} else {
		if dotIdx == 0 {
			return Expression{}, fmt.Errorf("empty name in expression: %q", input)
		}
		expr.Name = rest[:dotIdx]
		expr.Tail = rest[dotIdx+1:]
	}
	return expr, nil
}

// parseComponents parses $components.{name} and $components.parameters.{name} expressions.
func parseComponents(input string) (Expression, error) {
	expr := Expression{Raw: input}
	rest := input[len("$components."):]
	if rest == "" {
		return Expression{}, fmt.Errorf("empty name in expression: %q", input)
	}

	// Special case: $components.parameters.{name}
	if strings.HasPrefix(rest, "parameters.") {
		name := rest[len("parameters."):]
		if name == "" {
			return Expression{}, fmt.Errorf("empty parameter name in expression: %q", input)
		}
		expr.Type = ComponentParameters
		expr.Name = name
		return expr, nil
	}

	// General: $components.{name}[.tail]
	expr.Type = Components
	dotIdx := strings.IndexByte(rest, '.')
	if dotIdx == -1 {
		expr.Name = rest
	} else {
		expr.Name = rest[:dotIdx]
		expr.Tail = rest[dotIdx+1:]
	}
	return expr, nil
}

// ParseEmbedded parses a string that may contain embedded runtime expressions in {$...} blocks.
// Returns alternating literal and expression tokens.
func ParseEmbedded(input string) ([]Token, error) {
	if len(input) == 0 {
		return nil, nil
	}

	var tokens []Token
	pos := 0

	for pos < len(input) {
		// Find the next embedded expression start.
		openIdx := strings.Index(input[pos:], "{$")
		if openIdx == -1 {
			// No more expressions, rest is literal
			tokens = append(tokens, Token{Literal: input[pos:]})
			break
		}

		// Add literal before the brace
		if openIdx > 0 {
			tokens = append(tokens, Token{Literal: input[pos : pos+openIdx]})
		}

		exprStart := pos + openIdx + 1

		// Find closing brace
		closeIdx := strings.IndexByte(input[exprStart:], '}')
		if closeIdx == -1 {
			return nil, fmt.Errorf("unclosed expression brace at position %d", pos+openIdx)
		}

		// Extract and parse the expression (without the surrounding braces).
		exprStr := input[exprStart : exprStart+closeIdx]
		expr, err := Parse(exprStr)
		if err != nil {
			return nil, fmt.Errorf("invalid embedded expression at position %d: %w", pos+openIdx, err)
		}

		tokens = append(tokens, Token{Expression: expr, IsExpression: true})
		pos = exprStart + closeIdx + 1
	}

	return tokens, nil
}

// Validate checks whether a string is a valid runtime expression without allocating a full AST.
func Validate(input string) error {
	_, err := Parse(input)
	return err
}
