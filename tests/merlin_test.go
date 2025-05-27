package tests

import (
	"testing"
)

func TestMerlin(t *testing.T) {
	pushSoftModFile(t, "merlin")
	t.Run("follow", testCase(func(t *testing.T, c *testContext) {
	}))
}
