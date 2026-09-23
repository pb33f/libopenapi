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
| `nullable: true`, `type: [T, "null"]` | `T \| null`; a 3.1 `enum` or `const` that leaves out null excludes it |
| `enum` | string, number or boolean literal union; never a TypeScript `enum` |
| `const` | literal type |
| `allOf` | intersection of referenced members and one literal of the merged inline members, properties declared beside `allOf`, and required names (undeclared ones as required `unknown` members, which makes an inherited property required) |
| `oneOf` / `anyOf` | union |
| `properties` / `required` beside `oneOf` / `anyOf` | an object literal intersected with the whole union; required names the outer schema does not declare become required `unknown` members |
| `oneOf` with a discriminator | each referenced variant that declares the discriminator inline as a plain string, integer or number is intersected with its literal values — the component name, which OpenAPI falls back to, plus any mapped values — so the union narrows; variants that declare it as a literal already narrow, and inherited or `$ref`-typed properties are left as they are |
| `additionalProperties: {schema}` with no properties | `{ [key: string]: T }`, which unlike `Record` may refer to its own type, so recursive maps compile |
| `additionalProperties: true` or a schema, with properties | `[key: string]: unknown` index signature |
| `patternProperties` | `[key: string]: unknown` index signature, even with `additionalProperties: false` |
| `required` names with no declared property | required members, typed by the `additionalProperties` schema when there is one (with a matching index signature if nothing else is declared), `unknown` otherwise |
| `type: object` with no properties or required names | `{ [key: string]: unknown }`, or `{ [key: string]: never }` with `additionalProperties: false` |
| `items` | `T[]` |
| `prefixItems` | tuple of optional positions, `[A?, B?, ...T[]]`: `items` types the rest, `items: false` closes it, no `items` leaves it `unknown` |
| no type | `unknown` |
| `$ref` to a component | the component's type name |
| `$ref` into a component (`…/properties/p`, `/items`, `/additionalProperties`) | the type of the schema the pointer names, rendered in place |
| `description`, `deprecated` | JSDoc, `@deprecated` |
| string formats, `int64` | `string`, `number` |

Component names are kept verbatim when they are valid TypeScript identifiers.
Names `tsc` rejects for a type declaration (reserved words such as `await`
and `as`, and predefined types such as `string`), invalid identifiers, and
`Array`, which the output uses, are renamed and reported with
`invalidTypeScriptIdentifier`. Valid names are claimed before renamed ones, so
a rename that lands on a taken name gets a suffix and a
`componentNameCollision` diagnostic. `WithTypeNameResolver` overrides naming,
and `WithHeaderComment` replaces the generated-code header.

A nested schema that cannot be built renders as `unknown` and is reported with
`childSchema`, as is a $ref pointer the generator cannot follow with
`unsupportedPointerReference`.

## Known limitations

From the shared IR, which the Go target also has:

- A nullable `oneOf: [$ref, null]` takes the referenced schema's description
  rather than its own, and drops properties or required names declared beside
  it.
- A multi-type schema such as `type: [array, object]` renders as a union of
  the bare types, without its `items` or `properties`.

From TypeScript or this generator's choices:

- `readOnly` and `writeOnly` are not modelled, so one type serves requests and
  responses: a required `readOnly` property is required in request bodies too.
- `additionalProperties` beside `allOf` is ignored.
- Pointers through `allOf/N`, `oneOf/N` or `anyOf/N` render as `unknown`.
- Non-scalar `const` values, and YAML numbers that are not JSON numbers (such
  as `0x1F` or `.inf`), render as `unknown`.
- Integer and number are both `number`, and string formats are not checked.
- A schema with `properties` or `items` but no `type` is treated as an object
  or array, although JSON Schema also accepts other values there.
- An optional property named after a member of `Object` (`constructor`,
  `toString`, `valueOf`) makes `{}` unassignable, because TypeScript checks the
  missing key against `Object`'s own member.
- External `$ref`s render as the referenced name without declaring it; bundle
  the document first.
