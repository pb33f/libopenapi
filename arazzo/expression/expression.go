// Copyright 2022-2026 Princess Beef Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

// Package expression implements the Arazzo runtime expression parser and evaluator.
// https://spec.openapis.org/arazzo/v1.0.1#runtime-expressions
package expression

// ExpressionType identifies the kind of runtime expression.
type ExpressionType int

const (
	URL                ExpressionType = iota // $url
	Method                                   // $method
	StatusCode                               // $statusCode
	RequestHeader                            // $request.header.{name}
	RequestQuery                             // $request.query.{name}
	RequestPath                              // $request.path.{name}
	RequestBody                              // $request.body{#/json-pointer}
	ResponseHeader                           // $response.header.{name}
	ResponseQuery                            // $response.query.{name}
	ResponsePath                             // $response.path.{name}
	ResponseBody                             // $response.body{#/json-pointer}
	Inputs                                   // $inputs.{name}
	Outputs                                  // $outputs.{name}
	Steps                                    // $steps.{name}[.tail]
	Workflows                                // $workflows.{name}[.tail]
	SourceDescriptions                       // $sourceDescriptions.{name}[.tail]
	Components                               // $components.{name}[.tail]
	ComponentParameters                      // $components.parameters.{name}
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
	Literal    string     // Non-empty if this is a literal text segment
	Expression Expression // Valid if IsExpression is true
	IsExpression bool     // True if this token is an expression
}
