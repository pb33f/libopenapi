# AGENTS.md

`github.com/pb33f/libopenapi` is a Go library for parsing, indexing, mutating, bundling, diffing, overlaying, rendering, and mocking OpenAPI/OAS-adjacent documents. It is the engine behind vacuum, wiretap, openapi-changes, printing press, and the pb33f platform. When code, tests, and external docs disagree, code is canonical.

## Context

This repo is a library, not an app. The root package exposes the public entry points (`NewDocument`, `CompareDocuments`, overlay/arazzo helpers), while most real behavior lives in subsystem packages. The highest-risk edits are in `index/`: it owns reference extraction, cross-document lookup, circular analysis, schema-id resolution, and performance-sensitive caches.

## Packages

| Package | Purpose |
|---|---|
| `.` | Public API surface: document creation, model building, render/reload, compare, cache clearing, overlay/arazzo entry points |
| `index/` | Core indexing engine: `SpecIndex`, `Rolodex`, lookup, resolver, `$id` registry, origin search, local/remote file systems |
| `datamodel/low/` | YAML-backed low-level models and generic builders; source of truth for comments, line/column, refs, hashing |
| `datamodel/high/` | Mutable high-level facades plus node builders/rendering back to YAML/JSON |
| `datamodel/` | Spec parsing/version detection, schemas, and `DocumentConfiguration` |
| `bundler/` | Inline and composed bundling, ref rewrite/composition, origin tracking |
| `what-changed/` | Unified diff engine for OAS2 and OAS3+, plus breaking-change rule config/report helpers |
| `overlay/` | OpenAPI Overlay application engine |
| `arazzo/` | Arazzo parsing, resolution, validation, and workflow execution engine |
| `renderer/` | Schema/mock sample generation |
| `orderedmap/` | Stable insertion-ordered map wrapper used throughout models/rendering |
| `json/` | YAML-node to ordered JSON conversion |
| `tests/` | Cross-package integration and benchmark coverage, especially sibling-ref behavior |
| `test_specs/` | Realistic fixtures and regression specs used across packages |

## Key Paths

| Path | Purpose |
|---|---|
| `document.go` | Root orchestration layer; keep it thin |
| `index/doc.go` | Best summary of `index` subsystem boundaries and invariants |
| `index/index_model.go` | `SpecIndex`, config, caches, release lifecycle |
| `index/spec_index_build.go` | Index construction/build sequencing |
| `index/rolodex.go` | Cross-document lookup ownership and lifecycle |
| `index/extract_refs.go` | Reference discovery entry point |
| `index/find_component_entry.go` | Component lookup entry path |
| `index/search_index.go` | Reference search flow, cache usage, schema-id lookup |
| `index/resolver_entry.go` | Circular detection and destructive resolution entry point |
| `datamodel/document_config.go` | Canonical config surface for documents/index/bundler behavior |
| `datamodel/spec_info.go` | Spec parsing, version detection, JSON conversion, `$self` handling |
| `datamodel/low/v3/create_document.go` | V3 document/index/rolodex assembly |
| `datamodel/low/model_builder.go` | Reflection-driven low-model population |
| `datamodel/high/node_builder.go` | High-model re-rendering/mutation path |
| `bundler/bundler.go` | Public bundling entry points/config |
| `bundler/bundler_composer.go` | Composed bundling and component lifting |
| `what-changed/model/document.go` | Unified change model and compare flow |
| `what-changed/model/breaking_rules.go` | Default/custom breaking-change policy |
| `.github/workflows/build.yaml` | CI shape: Linux + Windows `go test ./...`, coverage upload |

## Commands

| Command | Purpose |
|---|---|
| `go test ./...` | Canonical full test suite |
| `go test -coverprofile=coverage.out ./...` | CI-style coverage run |
| `go test ./index ./bundler ./what-changed/...` | Fast pass over the most coupled subsystems |
| `go test ./tests -run SiblingRefs` | Target sibling-ref integration surface |
| `go test ./index -run TestSpecIndex` | Target index-heavy regressions |
| `go test ./bundler -run TestBundle` | Target bundler regressions |
| `go test ./what-changed/... -run Test` | Target diff/breaking-rule regressions |
| `go test -bench . ./index ./datamodel/low/... ./what-changed/...` | Run benchmarks in hot paths |
| `GOCACHE=/tmp/go-build go test ./...` | Useful in restricted sandboxes where default Go build cache is not writable |

## Testing Caveats

- `go.mod` is the authoritative toolchain target: Go `1.25.0`. CI still shows `1.23`; prefer `go.mod` when they conflict.
- Some tests use `httptest.NewServer` and require local loopback socket binding.
- Some bundler/index tests clone or fetch pinned external specs (notably DigitalOcean and `raw.githubusercontent.com` fixtures).
- In network-restricted or socket-restricted sandboxes, prefer targeted offline package tests and report environment-caused failures explicitly instead of treating them as code regressions.

## Rules & Patterns

- Keep `document.go` thin. Parsing/version detection belongs in `datamodel`, indexing/lookup/resolution in `index`, and diff logic in `what-changed`.
- Trust code and tests before README or `pb33f.io` docs.
- Prefer existing `index` seams over adding more orchestration: `extract_refs*` for discovery, `find_component*`/`search_*` for lookup, `resolver_*` for resolution, `rolodex*` for external docs, `schema_id*` for JSON Schema `$id`.
- Preserve ownership boundaries: one `SpecIndex` owns one parsed document; `Rolodex` owns shared file/remote lookup and cross-document indexes.
- Treat lifecycle work carefully. `Document.Release()` intentionally does not release the underlying `SpecIndex`; `SpecIndex.Release()` and `Rolodex.Release()` are separate cleanup steps for long-lived processes.
- Protect hot paths in `index` and schema resolution. The package explicitly optimizes direct component lookup, caches, pooled nodes, and reduced JSONPath usage on common paths.
- Add focused regression tests beside the behavior you change. This repo has a strong “surgical tests + high coverage” culture; preserve it.
- If you touch sibling refs, merge semantics, quick-hash behavior, or schema proxy resolution, run both `tests/` and the relevant `what-changed` coverage/tests because these behaviors interact.
- High-level models are mutable render facades over low-level YAML-backed models. Rendering or mutation fixes usually need checks in both `datamodel/high/*` and `datamodel/low/*`.
- `bundler` mutates models and depends on precise rolodex/index semantics. Ref rewrite or composition changes need bundler-specific tests, especially around discriminator mappings and external refs.
- `what-changed` is intentionally unified across OAS2 and OAS3+. Preserve both default breaking rules and override/config validation behavior.
- Use realistic fixtures from `test_specs/` and package-local fixture dirs instead of inventing toy specs when reproducing parser/indexer bugs.

## Common Failure Modes

- **Hash contract**: every schema field must appear in `Schema.hash()` (`datamodel/low/base/schema_hash.go`). A missing field means equality and diff silently ignore it. Call `ClearSchemaQuickHashMap()` between document lifecycles or the global `sync.Map` cache returns stale hashes.
- **Circular refs**: `resolver_circular.go` detects loops by comparing `FullDefinition` strings. If ref rewriting (bundler, resolver) changes these inconsistently, loops go undetected and the resolver hangs or overflows the depth limit (500).
- **Reference cache staleness**: `index.cache` (`sync.Map`) is never cleared after bundler mutations. Lookups after bundling can return stale pre-rewrite refs pointing to external files that no longer apply.
- **Bundler irreversibility**: `BundleDocument` / `BundleDocumentComposed` mutates the model in-place permanently. Never compare, re-bundle, or re-index a document after bundling — parse fresh from the rendered output instead.
- **Sibling ref idempotency**: `CreateAllOfStructure()` in `datamodel/low/base/sibling_ref_transformer.go` is not idempotent. Running it twice (e.g., bundle then re-index) produces nested `allOf` wrappers that break schema validity.
- **Resolver state leak**: `IgnorePoly` and `IgnoreArray` flags on the resolver persist between parses. Reusing a resolver across documents causes the second document's polymorphic circular refs to be silently missed.

## Mutation & Rendering

The library uses a dual-model architecture:

- **Low-level models** (`datamodel/low/`): YAML-backed structs that preserve line numbers, column numbers, comments, raw `*yaml.Node` references, and `$ref` metadata. These are the source of truth for document structure.
- **High-level models** (`datamodel/high/`): Mutable Go structs that wrap a low model. Every high model stores a `low` field and exposes `GoLow()` to access it.

**Mutation flow**:

1. Modify fields on the high-level model (e.g., `doc.Info.Title = "New Title"`)
2. Call `Render()` or `MarshalYAML()` on the model
3. `MarshalYAML()` creates a `NodeBuilder(highModel, lowModel)` — the builder uses reflection to read high-model field values and low-model line numbers for ordering
4. `NodeBuilder.Render()` sorts fields by original line number, then calls `AddYAMLNode()` recursively to build a `*yaml.Node` tree
5. `yaml.Marshal()` serializes the node tree to bytes

**Key rendering modes**:

- Default (`Resolve = false`): references render as `$ref: ...` strings
- Inline (`Resolve = true`): references are inlined at point of use
- `RenderingModeBundle`: inlines refs but preserves `$ref` inside discriminator `oneOf`/`anyOf` for bundling compatibility
- `RenderingModeValidation`: fully inlines everything for JSON Schema validation

**`RenderAndReload()`** is destructive — it renders to bytes, then re-parses and rebuilds the entire document model from scratch. The old model is invalid after this call.

**`renderer/` is separate**: the `renderer` package generates mock/example data from schemas (for documentation and testing). It does not serialize models to YAML — that is handled by `NodeBuilder` and `MarshalYAML()`.

## Environment

Required tools beyond Go:

```text
git
```

Optional but commonly needed for full-suite validation:

```text
network access
loopback socket binding
```
