# generator/typescript

`generator/typescript` renders OpenAPI component schemas as TypeScript type
declarations. It is a library package: no CLI, no client, no runtime code.

```go
file, err := typescript.RenderSchemas(model.Model.Components.Schemas)
if err != nil {
    return err
}
os.WriteFile("api.gen.ts", file.Source, 0o644)
```

`RenderSchemas` returns a `*GeneratedFile` with the module `Source`, the
exported `Types` in declaration order, and `Diagnostics`.

The output is a type-only module, so it erases entirely when compiled or run
with type stripping (`node --experimental-strip-types`). Shaping comes from the
same IR as `generator/golang` (`golang.Generator.SchemaIRs`), so both targets
read a schema the same way.

## Mapping

| OpenAPI | TypeScript |
|---|---|
| object with properties | `export interface X { … }` at the top level, an inline `{ … }` literal when nested |
| required / optional | `name: T` / `name?: T` |
| `nullable: true`, `type: [T, "null"]` | `T \| null` |
| `enum` | string, number or boolean literal union; never a TypeScript `enum` |
| `const` | literal type |
| `allOf` | intersection of referenced members and the merged inline members |
| `oneOf` / `anyOf` | union |
| `properties` / `required` beside `oneOf` / `anyOf` | an object literal intersected with the union; required names the outer schema does not declare become required `unknown` members |
| `oneOf` with a discriminator mapping | each referenced variant that declares the discriminator inline as a plain string, integer or number is intersected with its mapped values, so the union narrows; variants that declare it as a literal already narrow, and inherited or `$ref`-typed properties are left as they are |
| `additionalProperties: {schema}` with no properties | `Record<string, T>` |
| `additionalProperties: true` or a schema, with properties | `[key: string]: unknown` index signature |
| `required` names with no declared property | required members, typed by the `additionalProperties` schema when there is one (with a matching index signature if nothing else is declared), `unknown` otherwise |
| `type: object` with no properties or required names | `Record<string, unknown>`, or `Record<string, never>` with `additionalProperties: false` |
| no type | `unknown` |
| `$ref` to a component | the component's type name |
| `$ref` into a component (`…/properties/p`, `/items`, `/additionalProperties`) | the type of the schema the pointer names, rendered in place |
| `description`, `deprecated` | JSDoc, `@deprecated` |
| string formats, `int64` | `string`, `number` |

Component names are kept verbatim when they are valid TypeScript identifiers.
Names `tsc` rejects for a type declaration (reserved words such as `await`
and `as`, and predefined types such as `string`), invalid identifiers, and the
globals the output uses (`Array`, `Record`) are renamed and reported with
`invalidTypeScriptIdentifier`. Valid names are claimed before renamed ones, so
a rename that lands on a taken name gets a suffix and a
`componentNameCollision` diagnostic. `WithTypeNameResolver` overrides naming,
and `WithHeaderComment` replaces the generated-code header.

## Known limitations

These come from the shared IR and affect the Go target the same way:

- `properties` or `required` declared beside `allOf` are not merged into the
  result.
- A nullable `oneOf: [$ref, null]` takes the referenced schema's description
  rather than its own, and drops properties or required names declared beside
  it.
