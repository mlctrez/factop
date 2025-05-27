package tests

import (
	"testing"
)

func TestMerlin(t *testing.T) {
	pushSoftModFile(t, "merlin")
	t.Run("follow", testCase(func(t *testing.T, c *testContext) {
		// todo: determine how to test parts that rely on storage, player, surface, etc
	}))
}
