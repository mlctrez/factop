package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const stateFileName = "campaign.json"

// Stage identifiers for the First Night arc (and the base plugin).
const (
	StageIntro    = "intro"
	StageExplore  = "explore"
	StageIndustry = "industry"
	StageSiege    = "siege"
	StageResolve  = "resolve"
	StageDone     = "done"
)

// Campaign flags persisted across plugin restarts.
const (
	FlagWelcomed       = "welcomed"
	FlagExplored       = "explored"
	FlagIndustry       = "industry"
	FlagSiegeSpawned   = "siege_spawned"
	FlagFirstDeath     = "first_death"
	FlagFirstNightDone = "first_night_done"
)

// State is the durable campaign snapshot stored under the plugin DataDir.
type State struct {
	Stage      string          `json:"stage"`
	Flags      map[string]bool `json:"flags"`
	OriginX    float64         `json:"origin_x"`
	OriginY    float64         `json:"origin_y"`
	HasOrigin  bool            `json:"has_origin"`
	SpawnCount int             `json:"spawn_count"` // biters spawned this siege
	KillCount  int             `json:"kill_count"`  // enemy kills during siege
}

// NewState returns a fresh campaign at the intro stage.
func NewState() *State {
	return &State{
		Stage: StageIntro,
		Flags: make(map[string]bool),
	}
}

// Store loads and saves campaign state with a mutex for event-handler safety.
type Store struct {
	mu   sync.Mutex
	path string
	s    *State
}

// OpenStore loads state from dataDir/campaign.json, or creates defaults.
func OpenStore(dataDir string) (*Store, error) {
	if dataDir == "" {
		return nil, errors.New("plugin data-dir is empty; cannot persist campaign state")
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	path := filepath.Join(dataDir, stateFileName)
	st, err := loadState(path)
	if err != nil {
		return nil, err
	}
	return &Store{path: path, s: st}, nil
}

func loadState(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return NewState(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("read state: %w", err)
	}
	st := NewState()
	if err := json.Unmarshal(data, st); err != nil {
		return nil, fmt.Errorf("parse state: %w", err)
	}
	if st.Flags == nil {
		st.Flags = make(map[string]bool)
	}
	if st.Stage == "" {
		st.Stage = StageIntro
	}
	return st, nil
}

// Snapshot returns a copy of the current state for read-only use.
func (st *Store) Snapshot() State {
	st.mu.Lock()
	defer st.mu.Unlock()
	return cloneState(st.s)
}

// Flag reports whether a named flag is set.
func (st *Store) Flag(name string) bool {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.s.Flags[name]
}

// Update applies fn under lock and persists on success.
// If fn returns false, the change is discarded and nothing is written.
func (st *Store) Update(fn func(s *State) (changed bool, err error)) error {
	st.mu.Lock()
	defer st.mu.Unlock()

	// Work on a clone so a failed update does not corrupt in-memory state.
	next := cloneState(st.s)
	changed, err := fn(&next)
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}
	if err := saveState(st.path, &next); err != nil {
		return err
	}
	st.s = &next
	return nil
}

func cloneState(s *State) State {
	out := *s
	out.Flags = make(map[string]bool, len(s.Flags))
	for k, v := range s.Flags {
		out.Flags[k] = v
	}
	return out
}

func saveState(path string, s *State) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write state temp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename state: %w", err)
	}
	return nil
}
