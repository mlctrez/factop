package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractTag(t *testing.T) {
	cases := []struct {
		name  string
		input []byte
		want  string
	}{
		{"normal message", []byte("[move] player1:1:45.5:-12.0:nauvis:1"), "move"},
		{"uppercase tag", []byte("[MOVE] data"), "move"},
		{"multi-word tag", []byte("[entity_died] data"), "entity_died"},
		{"empty data", []byte{}, "unknown"},
		{"no brackets", []byte("hello world"), "unknown"},
		{"missing closing bracket", []byte("[move data"), "unknown"},
		{"empty brackets", []byte("[] data"), "unknown"},
		{"single char tag", []byte("[x] data"), "x"},
		{"just opening bracket", []byte("["), "unknown"},
		{"two-char message", []byte("[a"), "unknown"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractTag(tc.input)
			assert.Equal(t, tc.want, got)
		})
	}
}
