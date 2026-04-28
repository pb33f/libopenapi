// Copyright 2026 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

// Package golang generates Go model types from OpenAPI schemas and generates
// OpenAPI schemas from Go runtime types.
//
// The package is intentionally library-only. It does not provide a CLI,
// generated client or server code, a validation runtime, or runtime helper
// package. Callers provide libopenapi schema models or Go reflection types and
// receive generated Go source or OpenAPI schema proxies.
//
// OpenAPI to Go model generation starts with RenderSchema for a single schema
// or Generator.RenderSchemas for component maps. The generated source is
// gofmt-formatted and diagnostics report schema shapes that do not map directly
// to plain Go model fields.
//
// Go to OpenAPI generation starts with SchemaFromType for a single schema or
// Generator.SchemasFromTypes for a reusable component graph. Package-level
// graph helpers also have WithOptions variants for callers that do not need to
// keep a Generator instance. Named reflected structs, enums, and registered
// interface unions are emitted as components, nested named model references are
// rendered as component $refs, and SchemaSet.Roots exposes every requested
// root. WithTypeSchema maps reflected project scalar aliases to explicit
// OpenAPI schema models without adding methods to the scalar type, and
// WithFieldSchema/WithFieldSchemaByJSONName map individual struct fields to
// exact schema models while keeping the surrounding type reflected normally.
// Reflected nullable values use JSON Schema 2020-12 native nullability rather
// than OpenAPI 3.0 nullable: direct schemas use type arrays that include
// "null", and nullable component references use anyOf wrappers.
//
// Reflection metadata is layered. Field-level openapi struct tags handle
// compact scalar metadata such as format, constraints, enum, const,
// readOnly/writeOnly/deprecated, and nullable overrides. SchemaProvider,
// SchemaMetadataProvider, and SchemaYAMLProvider methods handle exact
// type-level schemas. OpenAPI-to-Go generation can opt into WithOpenAPITags and
// WithSchemaMetadataSidecar to emit those hooks into a separate
// schema_metadata.go source file for higher-fidelity Go-to-OpenAPI round trips.
// Disabling the metadata sidecar leaves GeneratedFile.SchemaMetadata nil and
// keeps generated code leaner, but recreating the original OpenAPI input from
// reflected Go types becomes intentionally lossy.
//
// Polymorphic oneOf schemas with an explicit discriminator, or an inferable
// required const discriminator, render as typed union wrappers. Ambiguous oneOf
// and anyOf schemas render as json.RawMessage wrappers so the generated model
// remains dependency-free and does not embed validation behavior.
//
// Schema-valued additionalProperties can round-trip unknown JSON object fields
// through generated marshal/unmarshal methods. WithAdditionalPropertiesMethods
// disables those methods when callers want to provide JSON behavior themselves.
//
// Inline schema type names use "_" as the default parent/child delimiter.
// WithNestedTypeNameDelimiter changes that delimiter, including to an empty
// string for compact names. Name collisions use "__" before the numeric suffix.
// Component names are collision-resolved before local refs are rendered, so
// generated fields point at the final Go type names.
package golang
