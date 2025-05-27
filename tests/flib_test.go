package tests

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func rconPrint(c *testContext, nilTest string) string {
	return c.rconSc(fmt.Sprintf("rcon.print( %s )", nilTest))
}

func TestFlib(t *testing.T) {
	t.Run("check present", testCase(func(t *testing.T, c *testContext) {
		require.Equal(t, "true", rconPrint(c, "factop_flib ~= nil"))
		assert.Equal(t, "true", rconPrint(c, "factop_flib.bounding_box ~= nil"))
		assert.Equal(t, "true", rconPrint(c, "factop_flib.position ~= nil"))
		assert.Equal(t, "true", rconPrint(c, "factop_flib.orientation ~= nil"))
		assert.Equal(t, "true", rconPrint(c, "factop_flib.direction ~= nil"))
	}))
}
