// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

// Package typescript generates TypeScript type declarations from OpenAPI
// schemas.
//
// The package is library-only and emits types, never runtime code: the output
// is a type-only module of interfaces and type aliases, so it erases completely
// when compiled or when TypeScript is run with type stripping. Enums therefore
// render as string-literal unions rather than TypeScript enum declarations.
//
// Schema shaping is shared with the Go model generator: Generator.RenderSchemas
// asks the golang package for its language-neutral IR (see
// golang.Generator.SchemaIRs) and prints TypeScript from it. Component schema
// names are kept verbatim when they are valid TypeScript identifiers, so
// generated names match the names used in the OpenAPI document. Nested inline
// schemas render as inline type literals rather than as separate named
// declarations.
package typescript
