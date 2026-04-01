package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []Entity
		wantErr bool
	}{
		{name: "empty", input: "", want: nil},
		{
			name:  "single entity",
			input: "iron-chest:5.5:10.5:42",
			want:  []Entity{{Name: "iron-chest", X: 5.5, Y: 10.5, UnitNumber: 42}},
		},
		{
			name:  "multiple entities",
			input: "stone-furnace:0.5:0.5:1,transport-belt:-3.5:4.5:2",
			want: []Entity{
				{Name: "stone-furnace", X: 0.5, Y: 0.5, UnitNumber: 1},
				{Name: "transport-belt", X: -3.5, Y: 4.5, UnitNumber: 2},
			},
		},
		{
			name:  "zero unit number",
			input: "tree-01:10.0:20.0:0",
			want:  []Entity{{Name: "tree-01", X: 10.0, Y: 20.0, UnitNumber: 0}},
		},
		{name: "missing unit_number", input: "chest:1:2", wantErr: true},
		{name: "bad x", input: "chest:abc:2:1", wantErr: true},
		{name: "bad y", input: "chest:1:abc:1", wantErr: true},
		{name: "bad unit_number", input: "chest:1:2:abc", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAreaString(t *testing.T) {
	a := Area{X1: -10.5, Y1: -20, X2: 10.5, Y2: 20}
	assert.Equal(t, "-10.5,-20,10.5,20", a.String())
}

func TestPositionString(t *testing.T) {
	p := Position{X: 5.5, Y: -3.5}
	assert.Equal(t, "5.5,-3.5", p.String())
}

func TestParseEntityEvent(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    EntityEvent
		wantErr bool
	}{
		{
			name:  "entity died with cause no player",
			input: "[entity-died] iron-chest:5.5:10.5:42:nauvis:1:0:small-biter",
			want: EntityEvent{
				Event: "entity-died", Name: "iron-chest",
				X: 5.5, Y: 10.5, UnitNumber: 42,
				SurfaceName: "nauvis", SurfaceIndex: 1,
				PlayerIndex: 0, Cause: "small-biter",
			},
		},
		{
			name:  "entity built by player",
			input: "[entity-built] transport-belt:-3.5:4.5:99:nauvis:1:2",
			want: EntityEvent{
				Event: "entity-built", Name: "transport-belt",
				X: -3.5, Y: 4.5, UnitNumber: 99,
				SurfaceName: "nauvis", SurfaceIndex: 1,
				PlayerIndex: 2,
			},
		},
		{
			name:  "entity mined by player",
			input: "[entity-mined] stone-furnace:0.5:0.5:7:vulcanus:2:1",
			want: EntityEvent{
				Event: "entity-mined", Name: "stone-furnace",
				X: 0.5, Y: 0.5, UnitNumber: 7,
				SurfaceName: "vulcanus", SurfaceIndex: 2,
				PlayerIndex: 1,
			},
		},
		{
			name:  "entity died unknown cause",
			input: "[entity-died] wall:10.0:20.0:0:nauvis:1:0:unknown",
			want: EntityEvent{
				Event: "entity-died", Name: "wall",
				X: 10.0, Y: 20.0, UnitNumber: 0,
				SurfaceName: "nauvis", SurfaceIndex: 1,
				PlayerIndex: 0, Cause: "unknown",
			},
		},
		{name: "missing tag", input: "no-brackets here", wantErr: true},
		{name: "too few fields", input: "[entity-built] chest:1:2", wantErr: true},
		{name: "bad x", input: "[entity-built] chest:abc:2:1:nauvis:1:0", wantErr: true},
		{name: "bad unit_number", input: "[entity-built] chest:1:2:abc:nauvis:1:0", wantErr: true},
		{name: "bad surface index", input: "[entity-built] chest:1:2:3:nauvis:abc:0", wantErr: true},
		{name: "bad player index", input: "[entity-built] chest:1:2:3:nauvis:1:abc", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseEntityEvent(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
