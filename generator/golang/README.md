# generator/golang

`generator/golang` is a library package for model-only generation:

- OpenAPI schema/component models to Go model source.
- Go reflection types to OpenAPI schema/component models.
- No CLI, client generation, server generation, validation runtime, or generated runtime dependency.

## OpenAPI To Go

Use `RenderSchema` for one schema or `Generator.RenderSchemas` for an ordered component map.

```go
source, err := golang.RenderSchema("Pet", schemaProxy)
if err != nil {
    return err
}
fmt.Println(string(source))
```

`RenderSchemas` returns a `*GeneratedFile` with:

- `PackageName`: generated package name.
- `Source`: gofmt-formatted Go source.
- `SchemaMetadata`: optional `schema_metadata.go` sidecar source when metadata sidecar generation is enabled.
- `Types`: top-level generated type names and kinds.
- `Diagnostics`: notable generator decisions.

## Go To OpenAPI

Use `SchemaFromType` for one schema or `SchemasFromTypes` for a reusable component graph.

```go
set, err := golang.SchemasFromTypes(reflect.TypeOf(Customer{}))
if err != nil {
    return err
}
root := set.Root
components := set.Components
```

`SchemaSet.Root` is the first requested root. `SchemaSet.Roots` contains every requested root keyed by generated type name. Named structs, registered interface unions, and reusable model shapes are emitted into `SchemaSet.Components`; nested named model references are rendered as `#/components/schemas/...` refs.

Nullable reflected values render with JSON Schema 2020-12 native nullability: `type: [T, "null"]` for direct schemas, or `anyOf` around `$ref` plus `{type: "null"}` for nullable component references. The generator does not emit OpenAPI 3.0 `nullable: true`.

Package-level graph helpers that need options use slice-based variants:

```go
set, err := golang.SchemasFromTypesWithOptions(
    []reflect.Type{reflect.TypeOf(Customer{})},
    golang.WithOneOfTypes((*PaymentMethod)(nil), Card{}, Bank{}),
)
```

Custom scalar aliases can be mapped without adding methods to the type:

```go
gen := golang.NewGenerator(
    golang.WithTypeSchema(reflect.TypeOf(CustomerID("")), customerIDSchema),
)
```

## Metadata Hooks

Reflection metadata is layered from lightweight to exact:

- Field tags for simple metadata: `openapi:"format=uuid;nullable=false;readOnly;minLength=3;maxLength=4"`.
- External registry overrides: `WithTypeSchema`, `WithFieldSchema`, and `WithFieldSchemaByJSONName`.
- Type-level providers: `OpenAPISchema() *base.SchemaProxy`, dependency-free `OpenAPISchemaMetadata() any`, or legacy `OpenAPISchemaYAML() string`.

Use `WithOpenAPITags(true)` when generating Go models to include compact `openapi` tags for metadata that Go reflection cannot infer from type shape alone. Tags support `format`, `title`, `description`, `nullable`, `readOnly`, `writeOnly`, `deprecated`, scalar/object/array constraints, `enum`, and `const`.

Use `WithSchemaMetadataSidecar(true)` when generated models should carry exact source schemas for high-fidelity reflection. The generated sidecar is a separate `schema_metadata.go` source file containing typed Go data exposed through `OpenAPISchemaMetadata() any`, so model packages do not need to import `libopenapi` or carry escaped YAML strings just to preserve metadata.

Leave the metadata sidecar disabled when generated model source should stay lean and the reverse path only needs canonical Go-shape output. In that mode `GeneratedFile.SchemaMetadata` is nil and no `schema_metadata.go` file should be written. This is explicitly lossy for OpenAPI -> Go -> OpenAPI reconstruction: validation-only keywords, exact source ordering, and other non-Go-shape schema details may not be recreated from reflection alone.

For exact per-field shapes without modifying model source, use field schema overrides:

```go
gen := golang.NewGenerator(
    golang.WithFieldSchema(reflect.TypeOf(BookingPayment{}), "Source", sourceSchema),
    golang.WithFieldSchemaByJSONName(reflect.TypeOf(BookingPayment{}), "status", statusSchema),
)
```

## Polymorphism

OpenAPI `oneOf` renders as a typed union when:

- The schema has an explicit discriminator.
- The variants share an inferable required `const` discriminator property.
- The variants share an optional `const` discriminator and `WithOptionalConstDiscriminatorUnions(true)` is enabled.

Ambiguous `oneOf` and all `anyOf` unions render as `json.RawMessage` wrappers. This keeps generated models dependency-free and avoids embedding validation behavior.

For Go reflection to OpenAPI, register interface variants:

```go
gen := golang.NewGenerator(
    golang.WithOneOfTypes((*PaymentMethod)(nil), Card{}, Bank{}),
    golang.WithDiscriminatorMapping((*PaymentMethod)(nil), "object", map[string]string{
        "card": "#/components/schemas/Card",
        "bank": "#/components/schemas/Bank",
    }),
)
```

## additionalProperties

Schema-valued `additionalProperties` renders as an `AdditionalProperties map[string]T` field with `json:"-"`.

Generated objects with schema-valued `additionalProperties` also receive `MarshalJSON` and `UnmarshalJSON` methods. Known properties are encoded normally, and unknown properties round-trip through the additional-properties map.

Use `WithAdditionalPropertiesMethods(false)` when callers only want the struct field and will provide JSON behavior themselves.

Boolean `additionalProperties` is preserved when generating OpenAPI from Go/OpenAPI IR, but it does not create a Go field unless a schema value exists.

## External References

External `$ref` values render as Go type names and emit `DiagnosticExternalReference`. By default, the type name is derived from the reference tail. Use `WithExternalRefTypeResolver` when an external reference should map to a different local type name.

## Diagnostics

Diagnostics have a stable `Code`, plus `Path` and human-readable `Message`. Callers should branch on `Code`, not message text.

Current diagnostic codes:

- `DiagnosticComponentNameCollision`
- `DiagnosticAdditionalPropertiesFalse`
- `DiagnosticArrayContains`
- `DiagnosticBooleanItems`
- `DiagnosticConstKeyword`
- `DiagnosticContentSchema`
- `DiagnosticConditionalSchema`
- `DiagnosticDependentRequired`
- `DiagnosticDependentSchemas`
- `DiagnosticDynamicReference`
- `DiagnosticExternalReference`
- `DiagnosticFieldNameCollision`
- `DiagnosticImplicitType`
- `DiagnosticMixedEnum`
- `DiagnosticMultiTypeSchema`
- `DiagnosticNotSchema`
- `DiagnosticNullEnum`
- `DiagnosticOptionalConstDiscriminator`
- `DiagnosticPatternProperties`
- `DiagnosticPrefixItems`
- `DiagnosticPropertyNames`
- `DiagnosticRootNameCollision`
- `DiagnosticSchemaMetadata`
- `DiagnosticStringEncoded`
- `DiagnosticTypeNameCollision`
- `DiagnosticUnevaluatedItems`
- `DiagnosticUnevaluatedProperties`
- `DiagnosticValidationKeyword`

Diagnostics are intentionally not validation errors. They report lossy model-shape choices, unsupported validation-only keywords, naming collisions, and external reference assumptions.

## Naming

The default naming path handles common Go initialisms such as `ID`, `URL`, `UUID`, `CVC`, `IBAN`, and `JWT`.

Inline/nested schema type names use `_` as the default parent/child delimiter, for example `Order_PaymentSource`. Use `WithNestedTypeNameDelimiter` to change it; pass an empty string to produce compact names such as `OrderPaymentSource`.

Component names are resolved through a collision registry before refs are rendered, so colliding OpenAPI component keys such as `user-id`, `user_id`, and `UserID` produce stable Go names like `UserID`, `UserID__2`, and `UserID__3`, and local `$ref` fields point at the resolved names. The double underscore is reserved for collision suffixes, not ordinary nesting.

Use resolvers when project-specific naming is required:

- `WithTypeNameResolver`
- `WithFieldNameResolver`
- `WithEnumValueNameResolver`
- `WithNameResolver` as a broad fallback

## Current Limits

- Validation behavior belongs in `libopenapi-validator`, not generated models.
- External `$ref` values render as Go type names and emit diagnostics; this package does not load or generate external dependency packages.
- Tuple-like `prefixItems` render as `[]any`.
- `patternProperties`, conditional schemas, `not`, `propertyNames`, and dependent schemas are reported as diagnostics because they do not map cleanly to plain Go model fields.
