package softmod

import (
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
