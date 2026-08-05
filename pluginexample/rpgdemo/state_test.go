package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenStoreCreatesDefault(t *testing.T) {
	dir := t.TempDir()
	store, err := OpenStore(dir)
	require.NoError(t, err)

	snap := store.Snapshot()
	assert.Equal(t, StageIntro, snap.Stage)
	assert.NotNil(t, snap.Flags)
	assert.False(t, store.Flag(FlagIndustry))
}

func TestStoreUpdatePersistsAcrossOpen(t *testing.T) {
	dir := t.TempDir()
	store, err := OpenStore(dir)
	require.NoError(t, err)

	err = store.Update(func(s *State) (bool, error) {
		s.Flags[FlagIndustry] = true
		s.Stage = StageSiege
		s.HasOrigin = true
		s.OriginX = 12
		s.OriginY = -4
		return true, nil
	})
	require.NoError(t, err)

	// File should exist on disk.
	_, err = os.Stat(filepath.Join(dir, stateFileName))
	require.NoError(t, err)

	store2, err := OpenStore(dir)
	require.NoError(t, err)
	snap := store2.Snapshot()
	assert.True(t, snap.Flags[FlagIndustry])
	assert.Equal(t, StageSiege, snap.Stage)
	assert.True(t, snap.HasOrigin)
	assert.Equal(t, 12.0, snap.OriginX)
	assert.Equal(t, -4.0, snap.OriginY)
}

func TestStoreUpdateDiscardedWhenUnchanged(t *testing.T) {
	dir := t.TempDir()
	store, err := OpenStore(dir)
	require.NoError(t, err)

	err = store.Update(func(s *State) (bool, error) {
		s.Stage = StageDone // would be bad if saved
		return false, nil
	})
	require.NoError(t, err)

	assert.Equal(t, StageIntro, store.Snapshot().Stage)
	// No file written on first discarded update from default.
	_, err = os.Stat(filepath.Join(dir, stateFileName))
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestOpenStoreRequiresDataDir(t *testing.T) {
	_, err := OpenStore("")
	require.Error(t, err)
}
