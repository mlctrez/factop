package tiles

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []Tile
		wantErr bool
	}{
		{name: "empty string", input: "", want: nil},
		{name: "single tile", input: "concrete:5:10", want: []Tile{{Name: "concrete", X: 5, Y: 10}}},
		{
			name:  "multiple tiles",
			input: "grass-1:0:0,concrete:1:2,stone-path:-3:-4",
			want: []Tile{
				{Name: "grass-1", X: 0, Y: 0},
				{Name: "concrete", X: 1, Y: 2},
				{Name: "stone-path", X: -3, Y: -4},
			},
		},
		{name: "whitespace trimmed", input: "  concrete:1:2  ", want: []Tile{{Name: "concrete", X: 1, Y: 2}}},
		{name: "missing y", input: "concrete:1", wantErr: true},
		{name: "bad x", input: "concrete:abc:1", wantErr: true},
		{name: "bad y", input: "concrete:1:abc", wantErr: true},
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
	a := Area{X1: -10, Y1: -20, X2: 10, Y2: 20}
	assert.Equal(t, "-10,-20,10,20", a.String())
}
