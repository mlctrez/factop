# ADR 001: Code Generation from Go Struct Annotations as Single Source of Truth

## Status

Accepted

## Context

The factop project uses a colon-separated wire format for UDP messages between Factorio (Lua) and Go. Previously, the Lua emitters in `softmod/factop/` were hand-written, creating synchronization risk with the Go parsers in `client/` packages. Any change to the wire format required coordinated edits in both languages, with no compile-time guarantee of consistency.

Architecture analysis explored several approaches (documented in `docs/archive/arch_analysis.md`). Option F — using Go struct annotations as the single source of truth — was selected after evaluating trade-offs around maintainability, correctness guarantees, and implementation complexity.

## Decision

Use Go struct annotations (`wire:"N"` tags and `// gen:lua` comments) as the single source of truth for the wire format. Both Go parser/format code and Lua emitter code are generated from the same annotated struct definitions using `go run ./gen`.

The implementation follows five phases from the original plan (`docs/archive/arch_option_f_plan.md`):

| # | Phase | Status |
|---|-------|--------|
| 1 | Event Router | Completed (`plugin/` package) |
| 2 | Schema Definition | Completed (`wire` tags on structs in `client/` packages) |
| 3 | Go Code Generator | Completed (`gen/` produces `_gen.go` files) |
| 4 | Lua Code Generator | Completed (`gen/` produces `_gen.lua` files) |
| 5 | UDP Dispatcher | Completed (`softmod/factop/udp.lua` + `service/`) |

A single invocation of `go run ./gen` scans all annotated structs and produces both Go and Lua output files, ensuring the wire format is always consistent across both languages.

## Consequences

### Positive

- Single source of truth eliminates wire format drift between Go parsers and Lua emitters
- Adding new events only requires annotating a Go struct; generated code handles both sides
- The generator enforces consistency: field order, separator, format specifiers, and field count are derived from the same definition
- Property-based tests validate round-trip correctness across generated code

### Negative

- Template complexity in `gen/` requires familiarity with Go's `text/template` and the rendering pipeline
- Developers must understand the annotation DSL (`wire` tags, `// gen:lua` comments) to add new event types
- Debugging generated code requires tracing back to the template rather than editing output directly
