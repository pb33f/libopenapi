# AGENTS.md

AI agent context for `github.com/pb33f/libopenapi` — a Go library for parsing, indexing, mutating, bundling, diffing, overlaying, rendering, and executing OpenAPI/OAS-adjacent documents. Optimize for code-first maintenance: trust implementation and tests over README or published docs when they drift.

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
| [`document.go`](/Users/dashanle/work/libopenapi/libopenapi/document.go) | Root orchestration layer; keep it thin |
| [`index/doc.go`](/Users/dashanle/work/libopenapi/libopenapi/index/doc.go) | Best summary of `index` subsystem boundaries and invariants |
| [`index/index_model.go`](/Users/dashanle/work/libopenapi/libopenapi/index/index_model.go) | `SpecIndex`, config, caches, release lifecycle |
| [`index/spec_index_build.go`](/Users/dashanle/work/libopenapi/libopenapi/index/spec_index_build.go) | Index construction/build sequencing |
| [`index/rolodex.go`](/Users/dashanle/work/libopenapi/libopenapi/index/rolodex.go) | Cross-document lookup ownership and lifecycle |
| [`index/extract_refs.go`](/Users/dashanle/work/libopenapi/libopenapi/index/extract_refs.go) | Reference discovery entry point |
| [`index/find_component_entry.go`](/Users/dashanle/work/libopenapi/libopenapi/index/find_component_entry.go) | Component lookup entry path |
| [`index/search_index.go`](/Users/dashanle/work/libopenapi/libopenapi/index/search_index.go) | Reference search flow, cache usage, schema-id lookup |
| [`index/resolver_entry.go`](/Users/dashanle/work/libopenapi/libopenapi/index/resolver_entry.go) | Circular detection and destructive resolution entry point |
| [`datamodel/document_config.go`](/Users/dashanle/work/libopenapi/libopenapi/datamodel/document_config.go) | Canonical config surface for documents/index/bundler behavior |
| [`datamodel/spec_info.go`](/Users/dashanle/work/libopenapi/libopenapi/datamodel/spec_info.go) | Spec parsing, version detection, JSON conversion, `$self` handling |
| [`datamodel/low/v3/create_document.go`](/Users/dashanle/work/libopenapi/libopenapi/datamodel/low/v3/create_document.go) | V3 document/index/rolodex assembly |
| [`datamodel/low/model_builder.go`](/Users/dashanle/work/libopenapi/libopenapi/datamodel/low/model_builder.go) | Reflection-driven low-model population |
| [`datamodel/high/node_builder.go`](/Users/dashanle/work/libopenapi/libopenapi/datamodel/high/node_builder.go) | High-model re-rendering/mutation path |
| [`bundler/bundler.go`](/Users/dashanle/work/libopenapi/libopenapi/bundler/bundler.go) | Public bundling entry points/config |
| [`bundler/bundler_composer.go`](/Users/dashanle/work/libopenapi/libopenapi/bundler/bundler_composer.go) | Composed bundling and component lifting |
| [`what-changed/model/document.go`](/Users/dashanle/work/libopenapi/libopenapi/what-changed/model/document.go) | Unified change model and compare flow |
| [`what-changed/model/breaking_rules.go`](/Users/dashanle/work/libopenapi/libopenapi/what-changed/model/breaking_rules.go) | Default/custom breaking-change policy |
| [`.github/workflows/build.yaml`](/Users/dashanle/work/libopenapi/libopenapi/.github/workflows/build.yaml) | CI shape: Linux + Windows `go test ./...`, coverage upload |

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

- Keep [`document.go`](/Users/dashanle/work/libopenapi/libopenapi/document.go) thin. Parsing/version detection belongs in `datamodel`, indexing/lookup/resolution in `index`, and diff logic in `what-changed`.
- Trust code and tests before README or `pb33f.io` docs.
- Prefer existing `index` seams over adding more orchestration: `extract_refs*` for discovery, `find_component*`/`search_*` for lookup, `resolver_*` for resolution, `rolodex*` for external docs, `schema_id*` for JSON Schema `$id`.
- Preserve ownership boundaries: one `SpecIndex` owns one parsed document; `Rolodex` owns shared file/remote lookup and cross-document indexes.
- Treat lifecycle work carefully. `Document.Release()` intentionally does not release the underlying `SpecIndex`; `SpecIndex.Release()` and `Rolodex.Release()` are separate cleanup steps for long-lived processes.
- Protect hot paths in `index` and schema resolution. The package explicitly optimizes direct component lookup, caches, pooled nodes, and reduced JSONPath usage on common paths.
- Add focused regression tests beside the behavior you change. This repo has a strong “surgical tests + high coverage” culture; preserve it.
- If you touch sibling refs, merge semantics, quick-hash behavior, or schema proxy resolution, run both [`tests/`](/Users/dashanle/work/libopenapi/libopenapi/tests) and the relevant `what-changed` coverage/tests because these behaviors interact.
- High-level models are mutable render facades over low-level YAML-backed models. Rendering or mutation fixes usually need checks in both `datamodel/high/*` and `datamodel/low/*`.
- `bundler` mutates models and depends on precise rolodex/index semantics. Ref rewrite or composition changes need bundler-specific tests, especially around discriminator mappings and external refs.
- `what-changed` is intentionally unified across OAS2 and OAS3+. Preserve both default breaking rules and override/config validation behavior.
- Use realistic fixtures from [`test_specs/`](/Users/dashanle/work/libopenapi/libopenapi/test_specs) and package-local fixture dirs instead of inventing toy specs when reproducing parser/indexer bugs.

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
