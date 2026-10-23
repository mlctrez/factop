package softmod

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildControlLuaExcludesCommon(t *testing.T) {
	buf, err := BuildControlLua()
	if err != nil {
		t.Fatal(err)
	}
	content := buf.String()

	if strings.Contains(content, `"factop.common"`) {
		t.Error("control.lua should NOT contain add_lib for factop.common")
	}

	expected := []string{
		`add_lib("factop.companion")`,
		`add_lib("factop.entity_gen")`,
		`add_lib("factop.game_gen")`,
		`add_lib("factop.player")`,
		`add_lib("factop.playerattr_gen")`,
		`add_lib("factop.resource_gen")`,
		`add_lib("factop.surface_gen")`,
		`add_lib("factop.tile_gen")`,
		`add_lib("factop.udp")`,
	}
	for _, e := range expected {
		if !strings.Contains(content, e) {
			t.Errorf("control.lua missing: %s", e)
		}
	}

	// Ensure removed modules are not present
	removed := []string{
		`add_lib("factop.goal")`,
	}
	for _, r := range removed {
		if strings.Contains(content, r) {
			t.Errorf("control.lua should NOT contain removed module: %s", r)
		}
	}

	t.Log(content)
}

// findLua returns the first available tool from names, preferring Lua 5.2
// (the version Factorio embeds). Tests needing it are skipped when absent.
func findLua(t *testing.T, names ...string) string {
	t.Helper()
	for _, name := range names {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	t.Skipf("none of %v on PATH", names)
	return ""
}

// TestLuaSyntax compiles every softmod Lua file with luac -p.
func TestLuaSyntax(t *testing.T) {
	luac := findLua(t, "luac5.2", "luac5.4", "luac5.3", "luac")
	files, err := filepath.Glob("factop/*.lua")
	if err != nil {
		t.Fatal(err)
	}
	files = append(files, "controlHeader.lua", "testdata/companion_test.lua")
	for _, f := range files {
		if out, err := exec.Command(luac, "-p", f).CombinedOutput(); err != nil {
			t.Errorf("%s: %v\n%s", f, err, out)
		}
	}
}

// TestCompanionLua runs the pure-Lua unit tests for the Merlin follow logic in
// factop/companion.lua against stubbed Factorio globals.
func TestCompanionLua(t *testing.T) {
	lua := findLua(t, "lua5.2", "lua5.4", "lua5.3", "lua")
	cmd := exec.Command(lua, "softmod/testdata/companion_test.lua")
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err != nil || strings.TrimSpace(string(out)) != "ok" {
		t.Fatalf("companion_test.lua failed: %v\n%s", err, out)
	}
}
