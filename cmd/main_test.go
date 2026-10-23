package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveRconSource(t *testing.T) {
	dir := t.TempDir()
	luaFile := filepath.Join(dir, "tick.lua")
	require.NoError(t, os.WriteFile(luaFile, []byte("rcon.print(game.tick)\n"), 0644))
	dashFile := filepath.Join(dir, "-e")
	require.NoError(t, os.WriteFile(dashFile, []byte("rcon.print(1)"), 0644))

	stdinLua := "-- comment\nrcon.print(#game.connected_players)\n"

	cases := []struct {
		name    string
		args    []string
		stdin   string
		piped   bool
		want    string
		wantErr string
	}{
		{name: "file path", args: []string{luaFile}, want: "rcon.print(game.tick)\n"},
		{name: "file path ignores piped stdin", args: []string{luaFile}, stdin: "ignored", piped: true, want: "rcon.print(game.tick)\n"},
		{name: "file path named -e via ./ prefix", args: []string{dashFile}, want: "rcon.print(1)"},
		{name: "missing file", args: []string{filepath.Join(dir, "nope.lua")}, wantErr: "no such file"},
		{name: "dash reads stdin", args: []string{"-"}, stdin: stdinLua, want: stdinLua},
		{name: "dash reads stdin even from terminal", args: []string{"-"}, stdin: stdinLua, piped: false, want: stdinLua},
		{name: "dash with empty stdin", args: []string{"-"}, stdin: " \n", wantErr: "no Lua received"},
		{name: "no args piped reads stdin", stdin: stdinLua, piped: true, want: stdinLua},
		{name: "no args terminal is usage error", wantErr: "usage: rcon"},
		{name: "inline single arg", args: []string{"-e", "rcon.print(game.tick)"}, want: "rcon.print(game.tick)"},
		{name: "inline multiple args joined", args: []string{"-e", "rcon.print(", "game.tick", ")"}, want: "rcon.print( game.tick )"},
		{name: "inline ignores stdin", args: []string{"-e", "rcon.print(1)"}, stdin: "other", piped: true, want: "rcon.print(1)"},
		{name: "inline missing lua", args: []string{"-e"}, wantErr: "requires inline Lua"},
		{name: "inline blank lua", args: []string{"-e", "  "}, wantErr: "inline Lua is empty"},
		{name: "extra args after path", args: []string{luaFile, "extra"}, wantErr: "usage: rcon"},
		{name: "extra args after dash", args: []string{"-", "extra"}, wantErr: "usage: rcon"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveRconSource(tc.args, strings.NewReader(tc.stdin), tc.piped)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, string(got))
		})
	}
}

func TestResolveRconSourceStdinReadError(t *testing.T) {
	_, err := resolveRconSource([]string{"-"}, iotest.ErrReader(errors.New("boom")), true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading stdin: boom")
}

func TestRconPayload(t *testing.T) {
	cases := []struct {
		name, src, want string
	}{
		{"single line", "rcon.print(game.tick)", "/sc rcon.print(game.tick)"},
		{"comments and blanks dropped", "-- header\n\n  local t = game.tick  \n-- note\nrcon.print(t)\n", "/sc local t = game.tick\nrcon.print(t)"},
		{"crlf input", "rcon.print(1)\r\nrcon.print(2)\r\n", "/sc rcon.print(1)\nrcon.print(2)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, string(rconPayload([]byte(tc.src))))
		})
	}
}

func TestConsoleCommand(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		want    string
		wantErr string
	}{
		{name: "single", args: []string{"/players"}, want: "/players"},
		{name: "server save", args: []string{"/server-save"}, want: "/server-save"},
		{name: "kick with reason", args: []string{"/kick", "bob", "griefing"}, want: "/kick bob griefing"},
		{name: "time", args: []string{"/time"}, want: "/time"},
		{name: "multiple args joined", args: []string{"/admins", "list"}, want: "/admins list"},
		{name: "quoted single arg", args: []string{" /version "}, want: "/version"},
		{name: "no args", wantErr: "usage: console"},
		{name: "blank", args: []string{" "}, wantErr: "usage: console"},
		{name: "chat text rejected", args: []string{"hello"}, wantErr: "must start with /"},
		{name: "multi-line rejected", args: []string{"/players\n/version"}, wantErr: "single line"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := consoleCommand(tc.args)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
