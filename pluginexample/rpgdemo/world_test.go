package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLuaString(t *testing.T) {
	assert.Equal(t, `'hello'`, luaString("hello"))
	assert.Equal(t, `'it\'s fine'`, luaString("it's fine"))
	assert.Equal(t, `'a\\b'`, luaString(`a\b`))
	assert.Equal(t, `'line\nbreak'`, luaString("line\nbreak"))
}
