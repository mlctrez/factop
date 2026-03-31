package player

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []Player
		wantErr bool
	}{
		{name: "empty", input: "", want: nil},
		{
			name:  "single with character",
			input: "alice:1:10.5:20.5:true",
			want:  []Player{{Name: "alice", Index: 1, X: 10.5, Y: 20.5, HasCharacter: true}},
		},
		{
			name:  "multiple mixed",
			input: "alice:1:10.5:20.5:true,bob:2:0.0:0.0:false",
			want: []Player{
				{Name: "alice", Index: 1, X: 10.5, Y: 20.5, HasCharacter: true},
				{Name: "bob", Index: 2, X: 0.0, Y: 0.0, HasCharacter: false},
			},
		},
		{name: "missing fields", input: "alice:1:10.5", wantErr: true},
		{name: "bad index", input: "alice:abc:0:0:true", wantErr: true},
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
