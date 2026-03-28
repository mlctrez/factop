package service

import (
	"encoding/json"
	"math/rand"
	"os"
	"reflect"
	"testing"
	"testing/quick"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// settingsTestDir creates a temp directory and overrides SettingsDirectory-related
// paths for isolated testing. Returns a cleanup function.
func settingsTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return dir
}

// writeSettingsJSON writes a JSON map to the given path.
func writeSettingsJSON(t *testing.T, path string, data map[string]any) {
	t.Helper()
	b, err := json.MarshalIndent(data, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, b, 0644))
}

// partialSettings generates a random partial FactOpSettings as a JSON map.
// Each field is independently included or excluded based on random bits.
type partialSettings struct {
	Port            string
	SaveLocation    string
	FactorioVersion string
	GamePort        int
	RconBind        string
	UDPIncoming     int
	UDPOutgoing     int
	// bitmask: which fields are present
	Mask uint8
}

func (p partialSettings) Generate(rand *rand.Rand, size int) reflect.Value {
	ps := partialSettings{
		Mask:            uint8(rand.Intn(128)),
		Port:            "12345",
		SaveLocation:    "/tmp/saves",
		FactorioVersion: "2.0.28",
		GamePort:        rand.Intn(65535) + 1,
		RconBind:        "10.0.0.1:5000",
		UDPIncoming:     rand.Intn(65535) + 1,
		UDPOutgoing:     rand.Intn(65535) + 1,
	}
	return reflect.ValueOf(ps)
}

func (p partialSettings) toJSON() map[string]any {
	m := make(map[string]any)
	if p.Mask&1 != 0 {
		m["Port"] = p.Port
	}
	if p.Mask&2 != 0 {
		m["SaveLocation"] = p.SaveLocation
	}
	if p.Mask&4 != 0 {
		m["FactorioVersion"] = p.FactorioVersion
	}
	if p.Mask&8 != 0 {
		m["GamePort"] = p.GamePort
	}
	if p.Mask&16 != 0 {
		m["RconBind"] = p.RconBind
	}
	if p.Mask&32 != 0 {
		m["UDPIncoming"] = p.UDPIncoming
	}
	if p.Mask&64 != 0 {
		m["UDPOutgoing"] = p.UDPOutgoing
	}
	return m
}

// TestSettingsRoundTripWithDefaults - Property 1
// **Validates: Requirements 1.1, 1.3, 1.4, 1.5, 1.6**
//
// For any subset of FactOpSettings fields present in a JSON file, loading and
// saving should produce a complete JSON with all fields, preserving original
// values and filling defaults for missing ones.
func TestSettingsRoundTripWithDefaults(t *testing.T) {
	f := func(ps partialSettings) bool {
		dir := t.TempDir()
		settingsPath := dir + "/factop-settings.json"

		// Write partial JSON
		inputMap := ps.toJSON()
		b, err := json.MarshalIndent(inputMap, "", "  ")
		if err != nil {
			return false
		}
		if err = os.WriteFile(settingsPath, b, 0644); err != nil {
			return false
		}

		// Load via Settings (using direct file read + unmarshal + applyDefaults)
		var data FactOpSettings
		raw, err := os.ReadFile(settingsPath)
		if err != nil {
			return false
		}
		if err = json.Unmarshal(raw, &data); err != nil {
			return false
		}

		s := &Settings{Data: data}
		changed := s.applyDefaults()

		// Save back
		out, err := json.MarshalIndent(s.Data, "", "  ")
		if err != nil {
			return false
		}
		if err = os.WriteFile(settingsPath, out, 0644); err != nil {
			return false
		}

		// Re-read and verify completeness
		raw2, err := os.ReadFile(settingsPath)
		if err != nil {
			return false
		}
		var result map[string]any
		if err = json.Unmarshal(raw2, &result); err != nil {
			return false
		}

		// All fields with defaults must be present
		if _, ok := result["GamePort"]; !ok {
			return false
		}
		if _, ok := result["RconBind"]; !ok {
			return false
		}
		if _, ok := result["UDPIncoming"]; !ok {
			return false
		}
		if _, ok := result["UDPOutgoing"]; !ok {
			return false
		}

		// Verify original values preserved
		if ps.Mask&8 != 0 {
			// GamePort was set
			if s.Data.GamePort != ps.GamePort {
				return false
			}
		} else {
			// GamePort should be default
			if s.Data.GamePort != 34198 {
				return false
			}
			if !changed {
				return false
			}
		}

		if ps.Mask&16 != 0 {
			if s.Data.RconBind != ps.RconBind {
				return false
			}
		} else {
			if s.Data.RconBind != "127.0.0.1:3000" {
				return false
			}
		}

		if ps.Mask&32 != 0 {
			if s.Data.UDPIncoming != ps.UDPIncoming {
				return false
			}
		} else {
			if s.Data.UDPIncoming != 4000 {
				return false
			}
		}

		if ps.Mask&64 != 0 {
			if s.Data.UDPOutgoing != ps.UDPOutgoing {
				return false
			}
		} else {
			if s.Data.UDPOutgoing != 4001 {
				return false
			}
		}

		return true
	}

	require.NoError(t, quick.Check(f, &quick.Config{MaxCount: 200}))
}

// TestApplyDefaultsSpecificValues verifies each specific default value.
func TestApplyDefaultsSpecificValues(t *testing.T) {
	s := &Settings{}
	changed := s.applyDefaults()

	assert.True(t, changed)
	assert.Equal(t, 34198, s.Data.GamePort)
	assert.Equal(t, "127.0.0.1:3000", s.Data.RconBind)
	assert.Equal(t, 4000, s.Data.UDPIncoming)
	assert.Equal(t, 4001, s.Data.UDPOutgoing)
}

// TestApplyDefaultsPreservesExistingValues verifies that existing field values
// are not overwritten when defaults are applied.
func TestApplyDefaultsPreservesExistingValues(t *testing.T) {
	s := &Settings{
		Data: FactOpSettings{
			GamePort:    55555,
			RconBind:    "192.168.1.1:9999",
			UDPIncoming: 7000,
			UDPOutgoing: 7001,
		},
	}
	changed := s.applyDefaults()

	assert.False(t, changed)
	assert.Equal(t, 55555, s.Data.GamePort)
	assert.Equal(t, "192.168.1.1:9999", s.Data.RconBind)
	assert.Equal(t, 7000, s.Data.UDPIncoming)
	assert.Equal(t, 7001, s.Data.UDPOutgoing)
}

// TestFactOpSettingsOmitzeroTags verifies that all fields have omitzero JSON tags.
func TestFactOpSettingsOmitzeroTags(t *testing.T) {
	typ := reflect.TypeOf(FactOpSettings{})
	for i := range typ.NumField() {
		field := typ.Field(i)
		tag := field.Tag.Get("json")
		assert.Contains(t, tag, "omitzero",
			"field %s should have omitzero in json tag, got %q", field.Name, tag)
	}
}
