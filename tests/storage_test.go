package tests

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBuiltin(t *testing.T) {
	pushSoftModFile(t, "storage")
	clearTestTable := func(c *testContext) {
		assert.Equal(t, "true", c.rconSc(
			`storage["table_test"]=nil; rcon.print(storage["table_test"]==nil)`))
	}
	t.Run("builtin", testCase(func(t *testing.T, c *testContext) {
		clearTestTable(c)
		assert.Equal(t, "", c.rconSc(`storage["table_test"]="saved"`))
		assert.Equal(t, "saved", c.rconSc(`rcon.print(storage["table_test"])`))
		assert.Equal(t, "", c.rconSc(`storage["table_test"]=nil`))
		assert.Equal(t, "nil", c.rconSc(`rcon.print(storage["table_test"])`))
	}))
	t.Run("get", testCase(func(t *testing.T, c *testContext) {
		clearTestTable(c)
		assert.Equal(t, "factop_storage.get - tableKey and key required",
			c.errorString(c.rconSc(`factop_storage.get()`)))
		assert.Equal(t, "true", c.rconSc(`rcon.print(factop_storage.get("table_test","key")==nil)`))
		assert.Equal(t, "true", c.rconSc(`rcon.print(storage["table_test"]~=nil)`))
		assert.Equal(t, "table", c.rconSc(`rcon.print(type(storage["table_test"]))`))
	}))
	t.Run("put", testCase(func(t *testing.T, c *testContext) {
		clearTestTable(c)
		assert.Equal(t, "factop_storage.put - tableKey and key required",
			c.errorString(c.rconSc(`factop_storage.put()`)))
		assert.Equal(t, "", c.rconSc(`factop_storage.put("table_test","key","value")`))
		assert.Equal(t, "true", c.rconSc(`rcon.print(storage["table_test"]~=nil)`))
		assert.Equal(t, "value", c.rconSc(`rcon.print(factop_storage.get("table_test","key"))`))
		assert.Equal(t, "", c.rconSc(`factop_storage.put("table_test","key")`))
		assert.Equal(t, "nil", c.rconSc(`rcon.print(factop_storage.get("table_test","key"))`))
	}))
	t.Run("keys", testCase(func(t *testing.T, c *testContext) {
		clearTestTable(c)
		assert.Equal(t, "", c.rconSc(`factop_storage.put("table_test","key","value")`))
		assert.Equal(t, "", c.rconSc(`factop_storage.put("table_test","key2","value2")`))
		assert.Contains(t,
			c.rconSc(`rcon.print(helpers.table_to_json(factop_storage.keys()))`), `"table_test"`)
		assert.Equal(t, "2", c.rconSc(`rcon.print(#factop_storage.keys(storage["table_test"]))`))
		assert.Equal(t, "value", c.rconSc(`rcon.print(storage["table_test"]["key"])`))
		assert.Equal(t, "value2", c.rconSc(`rcon.print(storage["table_test"]["key2"])`))
	}))
}
