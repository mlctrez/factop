---
inclusion: manual
description: Improvement backlog, deferred items, and rejected proposals for the factop codebase.
---

# Improvement Backlog

## Go Modernization

- Use `omitzero` JSON tags (Go 1.25) instead of `omitempty` for types where the zero value is meaningful.
- Use `strings.SplitSeq` in `service/rcon.go` for memory-efficient string splitting where applicable.

## Architecture

- All runtime config flows through `FactOpSettings`. Future config additions should follow the same pattern: add a field with `omitzero` tag, add a default in `applyDefaults()`, and re-save.
- The project uses `mlctrez/bind` which relies on `any` and reflection. Consider explicit wiring in `service/service.go` if IDE navigation becomes a pain point.

## Deferred

- Decoupling save file modification from Factorio process management. Only one consumer exists and the current implementation works. Revisit if additional consumers emerge.

## Rejected

- UDPBridge persistent outgoing connection — the outgoing UDP path is effectively unused. Not worth optimizing.
- NATS message schemas — rejected to keep the NATS command interface language-independent. Raw string payloads allow any NATS client to send commands without Go-specific JSON contracts.
