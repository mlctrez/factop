package surface

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseList(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []SurfaceEntry
		wantErr bool
	}{
		{name: "empty", input: "", want: nil},
		{name: "single", input: "nauvis:1", want: []SurfaceEntry{{Name: "nauvis", Index: 1}}},
		{
			name:  "multiple",
			input: "nauvis:1,vulcanus:2,fulgora:3",
			want: []SurfaceEntry{
				{Name: "nauvis", Index: 1},
				{Name: "vulcanus", Index: 2},
				{Name: "fulgora", Index: 3},
			},
		},
		{name: "bad index", input: "nauvis:abc", wantErr: true},
		{name: "missing index", input: "nauvis", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseList(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseInfo(t *testing.T) {
	input := "name:nauvis,index:1,always_day:false,daytime:0.5000,darkness:0.0000," +
		"freeze_daytime:false,peaceful_mode:true,no_enemies_mode:false," +
		"wind_speed:0.0100,wind_orientation:0.2500,solar_power_multiplier:1.0000," +
		"min_brightness:0.1500,ticks_per_day:25000,show_clouds:true," +
		"generate_with_lab_tiles:false"

	info, err := ParseInfo(input)
	require.NoError(t, err)

	assert.Equal(t, "nauvis", info.Name)
	assert.Equal(t, 1, info.Index)
	assert.False(t, info.AlwaysDay)
	assert.InDelta(t, 0.5, info.Daytime, 0.001)
	assert.InDelta(t, 0.0, info.Darkness, 0.001)
	assert.False(t, info.FreezeDaytime)
	assert.True(t, info.PeacefulMode)
	assert.False(t, info.NoEnemiesMode)
	assert.InDelta(t, 0.01, info.WindSpeed, 0.001)
	assert.InDelta(t, 0.25, info.WindOrientation, 0.001)
	assert.InDelta(t, 1.0, info.SolarPowerMultiplier, 0.001)
	assert.InDelta(t, 0.15, info.MinBrightness, 0.001)
	assert.Equal(t, 25000, info.TicksPerDay)
	assert.True(t, info.ShowClouds)
	assert.False(t, info.GenerateWithLabTiles)
}

func TestParseInfoEmpty(t *testing.T) {
	_, err := ParseInfo("")
	require.Error(t, err)
}
