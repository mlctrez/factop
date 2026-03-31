package resource

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCounts(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []ResourceCount
		wantErr bool
	}{
		{name: "empty", input: "", want: nil},
		{name: "single", input: "iron-ore:12345", want: []ResourceCount{{Name: "iron-ore", Count: 12345}}},
		{
			name:  "multiple",
			input: "iron-ore:12345,copper-ore:6789,coal:1000",
			want: []ResourceCount{
				{Name: "iron-ore", Count: 12345},
				{Name: "copper-ore", Count: 6789},
				{Name: "coal", Count: 1000},
			},
		},
		{name: "bad count", input: "iron-ore:abc", wantErr: true},
		{name: "missing count", input: "iron-ore", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCounts(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseResources(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []Resource
		wantErr bool
	}{
		{name: "empty", input: "", want: nil},
		{
			name:  "single",
			input: "iron-ore:10.5:20.5:1500",
			want:  []Resource{{Name: "iron-ore", X: 10.5, Y: 20.5, Amount: 1500}},
		},
		{
			name:  "multiple",
			input: "iron-ore:0.5:0.5:100,copper-ore:-3.5:4.5:200",
			want: []Resource{
				{Name: "iron-ore", X: 0.5, Y: 0.5, Amount: 100},
				{Name: "copper-ore", X: -3.5, Y: 4.5, Amount: 200},
			},
		},
		{name: "missing amount", input: "iron-ore:1:2", wantErr: true},
		{name: "bad x", input: "iron-ore:abc:2:100", wantErr: true},
		{name: "bad amount", input: "iron-ore:1:2:abc", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseResources(tt.input)
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
