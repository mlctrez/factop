package tests

import (
	_ "embed"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

//go:embed crafting.lua
var craftingLua string

func scriptCase(function string) string {
	return fmt.Sprintf("%s\n%s()", craftingLua, function)
}

// TestBasics verifies the basic functionality of the rcon / lua interface
func TestBasics(t *testing.T) {
	t.Run("base", testCase(func(t *testing.T, c *testContext) {
		assert.Equal(t, "testBase", c.rconSc(scriptCase("testBase")))
	}))
	t.Run("error", testCase(func(t *testing.T, c *testContext) {
		assert.Equal(t, "testError", c.errorString(c.rconSc(scriptCase("testError"))))
	}))
}

// TestGhost validates operations on crafting machine ghosts
func TestGhost(t *testing.T) {
	t.Run("ghost", testCase(func(t *testing.T, c *testContext) {
		assert.Equal(t, "", c.rconSc(scriptCase("testGhost")))
	}))
}
