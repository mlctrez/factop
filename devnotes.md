# Dev Notes

## Recent Development Summary

A comprehensive codebase improvement effort was completed across the `factop` Go service, covering six areas:

1. **Configuration externalization** — Hardcoded ports, RCON bind addresses, and UDP ports were moved into `factop-settings.json` with struct-level defaults and automatic re-save on first load.

2. **Structured log publishing over NATS** — The `LogHandler` now publishes JSON-encoded log records to the `factop.log` NATS subject when a connection is available. `WithAttrs`/`WithGroup` stubs were replaced with safe no-ops. NATS activation is lazy to avoid circular startup dependencies.

3. **Factorio process state machine** — A formal `ServerState` enum (`Stopped`, `Starting`, `Running`, `Stopping`, `Error`) with a validated transition table replaced the previous signal-0 probing approach. A monitor goroutine detects process exits and drives state transitions. The RCON startup marker on stdout triggers `Starting → Running`.

4. **Standard library adoption** — Manual loops were replaced with `slices` and `maps` package functions where the replacement improved clarity.

5. **Consistent error handling** — All `fmt.Errorf` calls now use `%w` for wrapping. Shutdown sequences in `Factorio` and `Nats` use `errors.Join` to collect multiple cleanup errors. `SoftMod` file operations include step-level context in error messages.

6. **Context-aware lifecycle** — `context.WithCancelCause` replaced `context.WithCancel` at the service level. `Stop()` propagates a cause string. `UDPBridge` uses the service-level context directly instead of maintaining its own.

All six areas include property-based tests validating correctness properties (settings round-trip, log handler invariants, state machine transition rules).

---

## Improvement Backlog

### Go Modernization

- Use `omitzero` JSON tags (Go 1.25) instead of `omitempty` for types where the zero value is meaningful.
- Use `strings.SplitSeq` in `service/rcon.go` for memory-efficient string splitting where applicable.

### Architecture

- **Settings injection** — All runtime config now flows through `FactOpSettings`. Future config additions should follow the same pattern: add a field with `omitzero` tag, add a default in `applyDefaults()`, and re-save.
- **Dependency injection** — The project uses `mlctrez/bind` which relies on `any` and reflection. Consider explicit wiring in `service/service.go` if IDE navigation becomes a pain point.
- **Magefile portability** — `magefiles/mage.go` has hardcoded hostnames (`factorio`). Use environment variables or `magefile.env` for deployment targets.

### Deferred

- **SoftMod abstraction** — Decoupling save file modification from Factorio process management was deferred. Only one consumer exists and the current implementation works. Revisit if additional consumers emerge.

### Rejected

- **UDPBridge persistent outgoing connection** — The outgoing UDP path is effectively unused (Lua event handlers are commented out, nothing publishes to `udp.outgoing`). Not worth optimizing.
- **NATS message schemas** — Rejected to keep the NATS command interface language-independent. Raw string payloads allow any NATS client to send commands without Go-specific JSON contracts.
- **Package `doc.go` files** — Low priority. Existing READMEs cover documentation needs. Can be added ad-hoc.
