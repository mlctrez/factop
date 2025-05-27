package tests

import (
	"fmt"
	"github.com/mlctrez/factop/api"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
	"os"
	"regexp"
	"testing"
)

var CannotExecuteRegex = regexp.MustCompile(`Cannot execute command\. Error.*: (.+)`)

// testContext allows setup and teardown of the nats connection and api client
type testContext struct {
	test   *testing.T
	con    *nats.Conn
	client *api.RconClient
}

// testCase creates testContext for executing a test
func testCase(test func(t *testing.T, c *testContext)) func(*testing.T) {
	return func(t *testing.T) {
		context := &testContext{test: t}
		context.beforeEach()
		defer context.afterEach()
		test(t, context)
	}
}

// beforeEach sets up the nats connection and api client
func (c *testContext) beforeEach() {
	var err error
	c.con, err = nats.Connect("nats://factorio")
	if err != nil {
		c.test.Error(fmt.Errorf("nats error: %v", err))
	}
	c.client = api.NewRconClient(c.con)
}

// afterEach tears down the nats connection
func (c *testContext) afterEach() {
	if c.con != nil {
		c.con.Close()
	}
}

/*
convenience methods for each test
*/

func (c *testContext) rconSc(payload string) string {
	scPayload := "/sc " + payload
	rconResult, err := c.client.Execute(&api.RconCommand{Payload: scPayload})
	if err != nil {
		c.test.Fatalf("rcon error: %v", err)
	}
	return rconResult.Payload
}

func (c *testContext) errorString(response string) string {
	matches := CannotExecuteRegex.FindStringSubmatch(response)
	require.Equal(c.test, 2, len(matches), fmt.Sprintf("response was %q", response))
	return matches[1]
}

func pushSoftModFile(t *testing.T, name string) {
	path := fmt.Sprintf("../softmod/factop/%s.lua", name)

	file, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file error: %v", err)
	}

	context := &testContext{test: t}
	context.beforeEach()
	defer context.afterEach()

	result := context.rconSc(string(file))
	if result != "" {
		t.Fatalf("rcon push error: %v", result)
	}

}
