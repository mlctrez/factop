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
		`add_lib("factop.entity")`,
		`add_lib("factop.game")`,
		`add_lib("factop.goal")`,
		`add_lib("factop.player")`,
		`add_lib("factop.playerattr")`,
		`add_lib("factop.resource")`,
		`add_lib("factop.surface")`,
		`add_lib("factop.tile")`,
		`add_lib("factop.udp")`,
	}
	for _, e := range expected {
		if !strings.Contains(content, e) {
			t.Errorf("control.lua missing: %s", e)
		}
	}

	t.Log(content)
}
