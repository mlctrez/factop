# AGENTS.md

Repo-specific rules for agents and contributors working on factop.

## Setup

- `go.mod` has `replace github.com/mlctrez/rcon => ../rcon`. The fork's own
  `go.mod` still declares `github.com/gorcon/rcon`, so the module cannot be
  fetched normally; a local checkout is required.
- Clone `mlctrez/rcon` next to this repo (`../rcon`) on branch
  `command_id_settings` (same commit as tag `v1.0.1`). factop uses
  `rcon.SetExeCommandId`, which is not on rcon's `master`.
- The softmod Lua tests need `lua5.2` and `luac5.2` on `PATH` (Factorio embeds
  Lua 5.2; 5.3/5.4 are used as fallbacks). `TestLuaSyntax` and
  `TestCompanionLua` are skipped, not failed, when no Lua is found, so make
  sure they actually ran.

## Build and test

```sh
CGO_ENABLED=0 go vet ./...
CGO_ENABLED=0 go test ./...
CGO_ENABLED=0 go test -v -run 'TestLuaSyntax|TestCompanionLua' ./softmod  # confirm PASS, not SKIP
```

## Rules

- Never modify an existing Factorio save without the repository owner's explicit agreement.
- Never deploy without the repository owner's go-ahead. When asking, say whether the deploy
  includes a softmod deploy.
