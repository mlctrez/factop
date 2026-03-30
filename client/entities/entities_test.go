package entities

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
