// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

package expression

import (
	"fmt"
	"strings"
	"unicode/utf8"
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

// Parse parses a single Arazzo runtime expression under the Arazzo 1.1 grammar.
// Returns a value type to avoid heap allocation. Use ParseWithVersion for 1.0 documents.
func Parse(input string) (Expression, error) {
	return ParseWithVersion(input, Arazzo11)
}

// ParseWithVersion parses a single Arazzo runtime expression under the grammar for the
// given specification version. See SpecVersion for what differs between them.
func ParseWithVersion(input string, version SpecVersion) (Expression, error) {
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

	case 'm': // $method, $message.
		if input == "$method" {
			expr.Type = Method
			return expr, nil
		}
		if strings.HasPrefix(input, "$message.") {
			return parseSource(input, "$message.", MessageHeader, MessageQuery, MessagePath, MessageBody, MessagePayload)
		}
		return Expression{}, fmt.Errorf("unknown expression: %q", input)

	case 's': // $statusCode, $self, $steps., $sourceDescriptions.
		if input == "$statusCode" {
			expr.Type = StatusCode
			return expr, nil
		}
		if input == "$self" {
			expr.Type = Self
			return expr, nil
		}
		if strings.HasPrefix(input, "$steps.") {
			if version == Arazzo10 {
				return parseArazzo10NamedExpression(input, "$steps.", Steps)
			}
			return parseStepExpression(input)
		}
		if strings.HasPrefix(input, "$sourceDescriptions.") {
			if version == Arazzo10 {
				return parseArazzo10NamedExpression(input, "$sourceDescriptions.", SourceDescriptions)
			}
			return parseSourceDescriptionExpression(input)
		}
		return Expression{}, fmt.Errorf("unknown expression: %q", input)

	case 'r': // $request., $response.
		if strings.HasPrefix(input, "$request.") {
			return parseSource(input, "$request.", RequestHeader, RequestQuery, RequestPath, RequestBody, RequestPayload)
		}
		if strings.HasPrefix(input, "$response.") {
			return parseSource(input, "$response.", ResponseHeader, ResponseQuery, ResponsePath, ResponseBody, ResponsePayload)
		}
		return Expression{}, fmt.Errorf("unknown expression: %q", input)

	case 'i': // $inputs.
		if strings.HasPrefix(input, "$inputs.") {
			return parseValueReference(input, "$inputs.", Inputs)
		}
		return Expression{}, fmt.Errorf("unknown expression: %q", input)

	case 'o': // $outputs.
		if strings.HasPrefix(input, "$outputs.") {
			return parseValueReference(input, "$outputs.", Outputs)
		}
		return Expression{}, fmt.Errorf("unknown expression: %q", input)

	case 'w': // $workflows.
		if strings.HasPrefix(input, "$workflows.") {
			if version == Arazzo10 {
				return parseArazzo10NamedExpression(input, "$workflows.", Workflows)
			}
			return parseWorkflowExpression(input)
		}
		return Expression{}, fmt.Errorf("unknown expression: %q", input)

	case 'c': // $components.
		if strings.HasPrefix(input, "$components.") {
			return parseComponents(input, version)
		}
		return Expression{}, fmt.Errorf("unknown expression: %q", input)

	default:
		return Expression{}, fmt.Errorf("unknown expression prefix: %q", input)
	}
}

// parseArazzo10NamedExpression preserves the general 1.0.1 "$prefix." name
// production. The first segment remains Name and the rest remains Tail so the
// existing evaluator can resolve both whole objects and their nested fields.
func parseArazzo10NamedExpression(input, prefix string, exprType ExpressionType) (Expression, error) {
	rest := input[len(prefix):]
	if rest == "" {
		return Expression{}, fmt.Errorf("empty name in expression: %q", input)
	}
	expr := Expression{Raw: input, Type: exprType}
	if separator := strings.IndexByte(rest, '.'); separator >= 0 {
		if separator == 0 {
			return Expression{}, fmt.Errorf("empty name in expression: %q", input)
		}
		expr.Name = rest[:separator]
		expr.Tail = rest[separator+1:]
		return expr, nil
	}
	expr.Name = rest
	return expr, nil
}

// parseSource parses $request.{source} or $response.{source} expressions.
func parseSource(
	input, prefix string,
	headerType, queryType, pathType, bodyType, payloadType ExpressionType,
) (Expression, error) {
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
		if err := validateCharSequence(name, true); err != nil {
			return Expression{}, fmt.Errorf("invalid query name in expression %q: %w", input, err)
		}
		expr.Property = name
		return expr, nil
	}

	if strings.HasPrefix(rest, "path.") {
		expr.Type = pathType
		name := rest[len("path."):]
		if err := validateCharSequence(name, true); err != nil {
			return Expression{}, fmt.Errorf("invalid path name in expression %q: %w", input, err)
		}
		expr.Property = name
		return expr, nil
	}

	if parsed, ok, err := parseStructuredSource(expr, rest, "body", bodyType); ok || err != nil {
		return parsed, err
	}
	if parsed, ok, err := parseStructuredSource(expr, rest, "payload", payloadType); ok || err != nil {
		return parsed, err
	}

	return Expression{}, fmt.Errorf("unknown source type in expression: %q", input)
}

func parseStructuredSource(
	expr Expression,
	rest, keyword string,
	exprType ExpressionType,
) (Expression, bool, error) {
	if rest == keyword {
		expr.Type = exprType
		return expr, true, nil
	}
	prefix := keyword + "#"
	if strings.HasPrefix(rest, prefix) {
		pointer := rest[len(prefix):]
		if err := validateJSONPointer(pointer); err != nil {
			return Expression{}, true, fmt.Errorf("invalid JSON Pointer in expression %q: %w", expr.Raw, err)
		}
		expr.Type = exprType
		expr.JSONPointer = pointer
		return expr, true, nil
	}
	return Expression{}, false, nil
}

func parseValueReference(input, prefix string, exprType ExpressionType) (Expression, error) {
	name, pointer, err := parseIdentifierAndPointer(input[len(prefix):], false)
	if err != nil {
		return Expression{}, fmt.Errorf("invalid value reference %q: %w", input, err)
	}
	return Expression{Raw: input, Type: exprType, Name: name, JSONPointer: pointer}, nil
}

func parseStepExpression(input string) (Expression, error) {
	const prefix = "$steps."
	rest := input[len(prefix):]
	separator := strings.Index(rest, ".outputs.")
	if separator <= 0 {
		return Expression{}, fmt.Errorf("invalid step output reference: %q", input)
	}
	stepID := rest[:separator]
	if !isIdentifier(stepID, true) {
		return Expression{}, fmt.Errorf("invalid step identifier %q in expression %q", stepID, input)
	}
	name, pointer, err := parseIdentifierAndPointer(rest[separator+len(".outputs."):], false)
	if err != nil {
		return Expression{}, fmt.Errorf("invalid step output reference %q: %w", input, err)
	}
	tailStart := separator + 1
	tailEnd := tailStart + len("outputs.") + len(name)
	return Expression{
		Raw: input, Type: Steps, Name: stepID,
		Tail: rest[tailStart:tailEnd], JSONPointer: pointer,
	}, nil
}

func parseWorkflowExpression(input string) (Expression, error) {
	const prefix = "$workflows."
	rest := input[len(prefix):]
	fieldPos, field := workflowFieldPosition(rest)
	if fieldPos <= 0 {
		return Expression{}, fmt.Errorf("invalid workflow input/output reference: %q", input)
	}
	workflowID := rest[:fieldPos]
	if !isIdentifier(workflowID, true) {
		return Expression{}, fmt.Errorf("invalid workflow identifier %q in expression %q", workflowID, input)
	}
	name, pointer, err := parseIdentifierAndPointer(rest[fieldPos+len(field)+2:], false)
	if err != nil {
		return Expression{}, fmt.Errorf("invalid workflow %s reference %q: %w", field, input, err)
	}
	tailStart := fieldPos + 1
	tailEnd := tailStart + len(field) + 1 + len(name)
	return Expression{
		Raw: input, Type: Workflows, Name: workflowID,
		Tail: rest[tailStart:tailEnd], JSONPointer: pointer,
	}, nil
}

func workflowFieldPosition(rest string) (int, string) {
	inputs := strings.Index(rest, ".inputs.")
	outputs := strings.Index(rest, ".outputs.")
	switch {
	case inputs >= 0 && (outputs < 0 || inputs < outputs):
		return inputs, "inputs"
	case outputs >= 0:
		return outputs, "outputs"
	default:
		return -1, ""
	}
}

func parseSourceDescriptionExpression(input string) (Expression, error) {
	const prefix = "$sourceDescriptions."
	rest := input[len(prefix):]
	separator := strings.IndexByte(rest, '.')
	if separator <= 0 || separator == len(rest)-1 {
		return Expression{}, fmt.Errorf("invalid source description reference: %q", input)
	}
	name := rest[:separator]
	if !isIdentifier(name, true) {
		return Expression{}, fmt.Errorf("invalid source description name %q in expression %q", name, input)
	}
	reference := rest[separator+1:]
	if err := validateCharSequence(reference, false); err != nil {
		return Expression{}, fmt.Errorf("invalid source description reference %q: %w", input, err)
	}
	return Expression{Raw: input, Type: SourceDescriptions, Name: name, Tail: reference}, nil
}

// parseComponents parses $components references.
//
// Under Arazzo 1.1 only parameters, successActions and failureActions are addressable.
// Under Arazzo 1.0 those three keep their specific expression types, and any other
// component type falls back to the general "$components." name production, yielding a
// Components expression whose Name is the component type and whose Tail is the remainder.
func parseComponents(input string, version SpecVersion) (Expression, error) {
	rest := input[len("$components."):]
	separator := strings.IndexByte(rest, '.')

	// 1.0.1 permits a single-segment reference: "$components." name does not require a
	// second dot. 1.1.0 always needs component-type "." component-name.
	if separator < 0 {
		if version == Arazzo10 && isIdentifier(rest, true) {
			return Expression{Raw: input, Type: Components, Name: rest}, nil
		}
		return Expression{}, fmt.Errorf("invalid component reference: %q", input)
	}
	if separator == 0 || separator == len(rest)-1 {
		return Expression{}, fmt.Errorf("invalid component reference: %q", input)
	}
	componentType := rest[:separator]
	name := rest[separator+1:]

	expr := Expression{Raw: input, Name: name}
	switch componentType {
	case "parameters":
		expr.Type = ComponentParameters
	case "successActions":
		expr.Type = ComponentSuccessActions
	case "failureActions":
		expr.Type = ComponentFailureActions
	default:
		if version == Arazzo10 {
			// 1.0.1: "$components." name, where name = *( CHAR ) and may contain dots.
			// The evaluator keys off the component type, so carry it in Name and the
			// remaining path in Tail.
			if !isIdentifier(componentType, true) {
				return Expression{}, fmt.Errorf("invalid component type %q in expression %q", componentType, input)
			}
			if !isIdentifier(name, false) {
				return Expression{}, fmt.Errorf("invalid component name %q in expression %q", name, input)
			}
			return Expression{Raw: input, Type: Components, Name: componentType, Tail: name}, nil
		}
		return Expression{}, fmt.Errorf("unknown component type %q in expression %q", componentType, input)
	}

	if !isIdentifier(name, false) {
		return Expression{}, fmt.Errorf("invalid component name %q in expression %q", name, input)
	}
	return expr, nil
}

func parseIdentifierAndPointer(rest string, strict bool) (string, string, error) {
	name := rest
	pointer := ""
	if hash := strings.IndexByte(rest, '#'); hash >= 0 {
		name = rest[:hash]
		pointer = rest[hash+1:]
		if err := validateJSONPointer(pointer); err != nil {
			return "", "", err
		}
	}
	if !isIdentifier(name, strict) {
		return "", "", fmt.Errorf("invalid identifier %q", name)
	}
	return name, pointer, nil
}

func isIdentifier(value string, strict bool) bool {
	if value == "" {
		return false
	}
	for i := 0; i < len(value); i++ {
		c := value[i]
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' ||
			c >= '0' && c <= '9' || c == '-' || c == '_' ||
			(!strict && c == '.') {
			continue
		}
		return false
	}
	return true
}

func validateJSONPointer(pointer string) error {
	if pointer == "" {
		return nil
	}
	if pointer[0] != '/' {
		return fmt.Errorf("must be empty or start with '/'")
	}
	if !utf8.ValidString(pointer) {
		return fmt.Errorf("contains invalid UTF-8")
	}
	for i := 0; i < len(pointer); i++ {
		switch pointer[i] {
		case '{', '}':
			return fmt.Errorf("contains unescaped brace at byte %d", i)
		case '~':
			if i+1 >= len(pointer) || pointer[i+1] != '0' && pointer[i+1] != '1' {
				return fmt.Errorf("contains invalid escape at byte %d", i)
			}
			i++
		}
	}
	return nil
}

func validateCharSequence(value string, allowEmpty bool) error {
	if value == "" {
		if allowEmpty {
			return nil
		}
		return fmt.Errorf("must not be empty")
	}
	if !utf8.ValidString(value) {
		return fmt.Errorf("contains invalid UTF-8")
	}
	for i := 0; i < len(value); i++ {
		switch value[i] {
		case '{', '}':
			return fmt.Errorf("contains unescaped brace at byte %d", i)
		case '"':
			return fmt.Errorf("contains unescaped quotation mark at byte %d", i)
		case '\\':
			if i+1 >= len(value) {
				return fmt.Errorf("contains incomplete escape at byte %d", i)
			}
			next := value[i+1]
			if strings.ContainsRune(`"\/bfnrt`, rune(next)) {
				i++
				continue
			}
			if next != 'u' || i+5 >= len(value) || !isHex4(value[i+2:i+6]) {
				return fmt.Errorf("contains invalid escape at byte %d", i)
			}
			i += 5
		default:
			if value[i] < 0x20 {
				return fmt.Errorf("contains unescaped control character at byte %d", i)
			}
		}
	}
	return nil
}

func isHex4(value string) bool {
	if len(value) != 4 {
		return false
	}
	for i := 0; i < 4; i++ {
		c := value[i]
		if c >= '0' && c <= '9' || c >= 'A' && c <= 'F' || c >= 'a' && c <= 'f' {
			continue
		}
		return false
	}
	return true
}

// ParseEmbedded parses a string that may contain embedded runtime expressions in {$...}
// blocks, under the Arazzo 1.1 grammar. Returns alternating literal and expression tokens.
func ParseEmbedded(input string) ([]Token, error) {
	return ParseEmbeddedWithVersion(input, Arazzo11)
}

// ParseEmbeddedWithVersion parses embedded runtime expressions under the grammar for the
// given specification version.
func ParseEmbeddedWithVersion(input string, version SpecVersion) ([]Token, error) {
	if len(input) == 0 {
		return nil, nil
	}
	if !utf8.ValidString(input) {
		return nil, fmt.Errorf("expression string contains invalid UTF-8")
	}

	var tokens []Token
	pos := 0

	for pos < len(input) {
		openIdx := strings.IndexByte(input[pos:], '{')
		if openIdx == -1 {
			if closeIdx := strings.IndexByte(input[pos:], '}'); closeIdx >= 0 {
				return nil, fmt.Errorf("unexpected closing brace at position %d", pos+closeIdx)
			}
			tokens = append(tokens, Token{Literal: input[pos:]})
			break
		}
		openIdx += pos

		if closeIdx := strings.IndexByte(input[pos:openIdx], '}'); closeIdx >= 0 {
			return nil, fmt.Errorf("unexpected closing brace at position %d", pos+closeIdx)
		}
		if openIdx+1 >= len(input) || input[openIdx+1] != '$' {
			return nil, fmt.Errorf("literal opening brace at position %d", openIdx)
		}
		if openIdx > pos {
			tokens = append(tokens, Token{Literal: input[pos:openIdx]})
		}
		exprStart := openIdx + 1
		closeIdx := strings.IndexByte(input[exprStart:], '}')
		if closeIdx == -1 {
			return nil, fmt.Errorf("unclosed expression brace at position %d", openIdx)
		}
		exprStr := input[exprStart : exprStart+closeIdx]
		expr, err := ParseWithVersion(exprStr, version)
		if err != nil {
			return nil, fmt.Errorf("invalid embedded expression at position %d: %w", openIdx, err)
		}

		tokens = append(tokens, Token{Expression: expr, IsExpression: true})
		pos = exprStart + closeIdx + 1
	}

	return tokens, nil
}

// Validate checks whether a string is a valid runtime expression under the Arazzo 1.1
// grammar, without allocating a full AST.
func Validate(input string) error {
	return ValidateWithVersion(input, Arazzo11)
}

// ValidateWithVersion checks whether a string is a valid runtime expression under the
// grammar for the given specification version.
func ValidateWithVersion(input string, version SpecVersion) error {
	_, err := ParseWithVersion(input, version)
	return err
}
