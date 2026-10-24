// Derived from github.com/gorcon/rcon (via github.com/mlctrez/rcon),
// Copyright (c) 2020 Pavel Korotkiy (outdead), MIT License. See LICENSE.

// Package rcon implements the Source RCON Protocol described in
// https://developer.valvesoftware.com/wiki/Source_RCON_Protocol.
//
// It is a vendored copy of the mlctrez/rcon fork (branch command_id_settings)
// of github.com/gorcon/rcon, kept inside factop so no separate checkout or
// go.mod replace is needed. The fork adds SetExeCommandId, which factop uses
// to send commands with a nonzero packet ID so Factorio console messages are
// not mixed into RCON output. The original MIT license is in LICENSE in this
// directory.
//
// The rcontest subpackage holds the fork's test RCON server and is used
// only by tests.
package rcon
