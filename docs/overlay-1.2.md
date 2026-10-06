# Overlay 1.2

The Overlay models and application engine implement the fields and action rules
in [Overlay 1.2.0](https://spec.openapis.org/overlay/v1.2.0.html).

## Upgrade plan

Extend the existing low/high models with `$self`, components, reusable actions,
and action references. Preserve `Actions []*Action`. Resolve each reference
locally before applying it. Reusable fields cannot contain `target` or `$ref`;
each reference supplies its own target. Preserve explicit overrides during
parse, render, and application. No network loader, OpenAPI index, or cache is needed.

Use one merge engine for updates and copies. Validate homogeneous targets,
replace primitives with correct YAML types, append items to arrays, concatenate
arrays, and reject incompatible object property types. Snapshot copy sources
before changes, including when the source is a target. Removal takes precedence.

Verify models, source metadata, hashes, rendering, public entry points, URI
resolution, sequential actions, failure cases, and representative performance.
Run independent correctness and de-slop reviews before publication.

## Compatibility

The existing public entry points and `Actions []*Action` remain available.
Supported version strings are 1.0.x, 1.1.x, and 1.2.x. The engine uses the current
merge rules for all supported versions. Previous behavior that disagreed with
the specification is corrected:

- Primitive targets can be updated. The value and YAML type both change.
- An object or primitive appended to an array remains one item.
- Incompatible property types produce an error instead of replacing containers.
- `remove: true` suppresses copy and update.
- If both copy and update are present, neither changes the target, as stated by
  the two field descriptions. Use separate actions to copy and then update.
- Every action needs a target. Malformed model structures produce errors.

Reusable actions are in `Components.Actions`, with their action content in
`ReusableAction.Fields`. Set `Action.Ref` to `#/components/actions/name`.
JSON Pointer escapes and URI percent escapes are decoded. Same-document relative references are also supported when an absolute `$self`
identifies the overlay. References to other documents are rejected. Reference fields override the corresponding reusable fields;
the update payload is replaced, not merged with the reusable payload.
For a constructed reference that overrides removal with false, call
`action.SetRemove(false)`. Parsed explicit false values are preserved.

`Overlay.ResolveExtends(baseURI)` resolves `extends` using `$self` and the supplied
retrieval, encapsulating, or default base URI. It performs no I/O. Apply functions
continue to use the target bytes supplied by the caller; they do not select or
fetch documents based on `extends`. With no absolute base, `ResolveExtends` returns the unresolved `extends` value. Document identifiers cannot contain fragments.
