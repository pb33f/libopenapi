// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

// Package expression implements the Arazzo runtime expression parser, local
// symbol resolver, and evaluator.
// https://spec.openapis.org/arazzo/v1.1.0.html#runtime-expressions
package expression

// SpecVersion selects which Arazzo runtime-expression grammar a parse applies.
//
// Arazzo 1.0.1 uses general name productions for $steps, $workflows,
// $sourceDescriptions and $components. Arazzo 1.1.0 replaces them with more
// structured forms and restricts component types. Callers that know the document
// version should pass it explicitly so legal 1.0 expressions are not rejected by
// the 1.1 grammar.
type SpecVersion uint8

const (
	// Arazzo11 applies the Arazzo 1.1.0 grammar. This is the default for the
	// version-free parser entry points.
	Arazzo11 SpecVersion = iota

	// Arazzo10 applies the Arazzo 1.0.1 grammar, which additionally accepts the
	// general "$components." name production.
	Arazzo10
)

// ExpressionType identifies the kind of runtime expression.
type ExpressionType int

const (
	URL                     ExpressionType = iota // $url
	Method                                        // $method
	StatusCode                                    // $statusCode
	RequestHeader                                 // $request.header.{name}
	RequestQuery                                  // $request.query.{name}
	RequestPath                                   // $request.path.{name}
	RequestBody                                   // $request.body{#/json-pointer}
	ResponseHeader                                // $response.header.{name}
	ResponseQuery                                 // $response.query.{name}
	ResponsePath                                  // $response.path.{name}
	ResponseBody                                  // $response.body{#/json-pointer}
	Inputs                                        // $inputs.{name}
	Outputs                                       // $outputs.{name}
	Steps                                         // $steps.{name}[.tail]
	Workflows                                     // $workflows.{name}[.tail]
	SourceDescriptions                            // $sourceDescriptions.{name}[.tail]
	Components                                    // $components.{name}[.tail]
	ComponentParameters                           // $components.parameters.{name}
	Self                                          // $self
	RequestPayload                                // $request.payload{#/json-pointer}
	ResponsePayload                               // $response.payload{#/json-pointer}
	MessageHeader                                 // $message.header.{name}
	MessageQuery                                  // $message.query.{name}
	MessagePath                                   // $message.path.{name}
	MessageBody                                   // $message.body{#/json-pointer}
	MessagePayload                                // $message.payload{#/json-pointer}
	ComponentSuccessActions                       // $components.successActions.{name}
	ComponentFailureActions                       // $components.failureActions.{name}
)

// Expression represents a parsed Arazzo runtime expression.
type Expression struct {
	Type        ExpressionType // The kind of expression
	Raw         string         // Original input string
	Name        string         // First segment after prefix (header name, step ID, etc.)
	Tail        string         // Everything after name for Steps/Workflows/SourceDescriptions/Components
	Property    string         // Sub-property for request/response sources (header/query/path name)
	JSONPointer string         // For body references: the #/path portion
}

// Token represents a segment in an embedded expression string like "prefix {$expr} suffix".
type Token struct {
	Literal      string     // Non-empty if this is a literal text segment
	Expression   Expression // Valid if IsExpression is true
	IsExpression bool       // True if this token is an expression
}
