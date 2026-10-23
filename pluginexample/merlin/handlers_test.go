package main

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/mlctrez/factop/client/companion"
	"github.com/mlctrez/factop/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestMerlin(t *testing.T) *merlin {
	t.Helper()
	return &merlin{
		log:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		companions: make(map[int]CompanionEntry),
		dataDir:    t.TempDir(),
	}
}

func readPersisted(t *testing.T, dir string) map[string]CompanionEntry {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "companions.json"))
	require.NoError(t, err)
	var out map[string]CompanionEntry
	require.NoError(t, json.Unmarshal(data, &out))
	return out
}

// Summon records and persists the mapping; dismissed/died remove it.
func TestLifecycleHandlersTrackAndPersist(t *testing.T) {
	m := newTestMerlin(t)

	m.onSummoned(companion.CompanionEvent{Name: "merlin", PlayerIndex: 1, PlayerName: "a", UnitNumber: 11, SurfaceName: "nauvis"})
	m.onSummoned(companion.CompanionEvent{Name: "merlin", PlayerIndex: 2, PlayerName: "b", UnitNumber: 22, SurfaceName: "nauvis"})
	assert.Equal(t, CompanionEntry{Name: "merlin", UnitNumber: 11, SurfaceName: "nauvis"}, m.companions[1])
	assert.Len(t, readPersisted(t, m.dataDir), 2)

	m.onDismissed(companion.CompanionEvent{PlayerIndex: 1})
	assert.NotContains(t, m.companions, 1)
	assert.Equal(t, map[string]CompanionEntry{"2": {Name: "merlin", UnitNumber: 22, SurfaceName: "nauvis"}}, readPersisted(t, m.dataDir))

	m.onDied(companion.CompanionEvent{PlayerIndex: 2})
	assert.Empty(t, m.companions)
	assert.Empty(t, readPersisted(t, m.dataDir))
}

// fakeCtx implements the parts of plugin.Context that Setup uses.
type fakeCtx struct {
	plugin.Context
	dir string
	log *slog.Logger
}

func (f fakeCtx) DataDir() string      { return f.dir }
func (f fakeCtx) Logger() *slog.Logger { return f.log }

// Setup reloads what persist wrote.
func TestSetupReloadsPersistedMapping(t *testing.T) {
	m := newTestMerlin(t)
	m.onSummoned(companion.CompanionEvent{Name: "merlin", PlayerIndex: 7, UnitNumber: 77, SurfaceName: "nauvis"})

	m2 := &merlin{}
	require.NoError(t, m2.Setup(fakeCtx{dir: m.dataDir, log: m.log}))
	assert.Equal(t, map[int]CompanionEntry{7: {Name: "merlin", UnitNumber: 77, SurfaceName: "nauvis"}}, m2.companions)
}

// Follow now lives in the softmod (companion.lua on_nth_tick). Guard against
// the plugin driving Merlin again, which would fight the softmod: main.go must
// not subscribe to move events or send RCON.
func TestPluginDoesNotDriveMovement(t *testing.T) {
	f, err := parser.ParseFile(token.NewFileSet(), "main.go", nil, 0)
	require.NoError(t, err)
	ast.Inspect(f, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok {
			assert.NotEqual(t, "OnMove", sel.Sel.Name, "merlin plugin must not subscribe to OnMove")
			assert.NotEqual(t, "Rcon", sel.Sel.Name, "merlin plugin must not send RCON to move Merlin")
		}
		return true
	})
}
