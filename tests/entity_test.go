package tests

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGhostBiter(t *testing.T) {
	t.Run("GhostBiter", testCase(func(t *testing.T, c *testContext) {
		// asserts that small-biter cannot be part of entity ghost
		createGhost := `game.surfaces["nauvis"].create_entity {
			name="entity-ghost", inner_name="small-biter", position={0,0}, force="player"
		}`
		result := c.rconSc(createGhost)
		matches := CannotExecuteRegex.FindStringSubmatch(result)
		require.Equal(t, 2, len(matches))
		assert.Equal(t, "small-biter can not be part a entity ghost.", matches[1])
	}))
}

func TestRemoveGhost(t *testing.T) {
	pushSoftModFile(t, "entity")
	t.Run("RemoveGhost", testCase(func(t *testing.T, c *testContext) {
		// remove existing so test passes
		c.rconSc(`factop_entity.destroy_ghosts({name="spidertron"})`)

		// create
		createGhost := `
		local spider = game.surfaces["nauvis"].create_entity {
			name="entity-ghost", inner_name="spidertron", 
			position={0,0}, force="player"
		}
		if spider ~= nil then
			rcon.print("created spider")
		end
		`
		sc := c.rconSc(createGhost)
		assert.Equal(t, "created spider", sc)

		selectGhost := `
		local found = game.surfaces["nauvis"].find_entities_filtered { 
			name="entity-ghost", position={0,0}, force="player" }
		if #found == 1 and found[1].ghost_name == "spidertron" then
			rcon.print("found ghost")
		end
		`

		assert.Equal(t, "found ghost", c.rconSc(selectGhost))

		r := c.rconSc(`factop_entity.destroy_ghosts({name="spidertron", debug=true})`)
		do := &debugOutput{}
		err := json.Unmarshal([]byte(r), do)
		if err != nil {
			t.Fatal(err)
		}

		// validate that it was deleted
		assert.Equal(t, 1, do.DestroyCount)

	}))
}

func TestDefaults(t *testing.T) {
	t.Run("destroy_ghosts check defaults", testCase(func(t *testing.T, c *testContext) {
		result := c.rconSc(`factop_entity.destroy_ghosts({name="spidertron", debug=true})`)

		do := &debugOutput{}
		err := json.Unmarshal([]byte(result), do)
		if err != nil {
			t.Fatal(err)
		}
		assert.True(t, do.Debug)
		assert.Equal(t, "nauvis", do.Surface)
		assert.Equal(t, "spidertron", do.Name)
		assert.Equal(t, -500, do.Area.LeftTop.X)
		assert.Equal(t, -500, do.Area.LeftTop.Y)
		assert.Equal(t, 500, do.Area.RightBottom.X)
		assert.Equal(t, 500, do.Area.RightBottom.Y)
	}))
	t.Run("destroy_ghosts whole surface", testCase(func(t *testing.T, c *testContext) {
		result := c.rconSc(`factop_entity.destroy_ghosts({name="spidertron", area="surface", debug=true})`)
		fmt.Println(result)
		do := &debugOutput{}
		err := json.Unmarshal([]byte(result), do)
		if err != nil {
			t.Fatal(err)
		}
		assert.True(t, do.Debug)
		assert.Equal(t, "nauvis", do.Surface)
		assert.Equal(t, "spidertron", do.Name)
		assert.Nil(t, do.Area)
	}))
}

func TestErrorConditions(t *testing.T) {
	t.Run("destroy_ghosts no params", testCase(func(t *testing.T, c *testContext) {
		result := c.rconSc(`factop_entity.destroy_ghosts()`)
		matches := CannotExecuteRegex.FindStringSubmatch(result)
		require.Equal(t, 2, len(matches))
		assert.Equal(t, "no parameters provided", matches[1])

	}))
	t.Run("destroy_ghosts no name", testCase(func(t *testing.T, c *testContext) {
		result := c.rconSc(`factop_entity.destroy_ghosts({})`)
		matches := CannotExecuteRegex.FindStringSubmatch(result)
		require.Equal(t, 2, len(matches))
		assert.Equal(t, "name parameter required", matches[1])

	}))
	t.Run("destroy_ghosts bad surface name", testCase(func(t *testing.T, c *testContext) {
		result := c.rconSc(`factop_entity.destroy_ghosts({name="spidertron", surface="saturn"})`)
		matches := CannotExecuteRegex.FindStringSubmatch(result)
		require.Equal(t, 2, len(matches))
		assert.Equal(t, `surface "saturn" not found`, matches[1])
	}))

	t.Run("destroy_ghosts with name", testCase(func(t *testing.T, c *testContext) {
		result := c.rconSc(`factop_entity.destroy_ghosts({name="spidertron"})`)
		assert.Equal(t, result, "")
	}))
	t.Run("destroy_ghosts check area", testCase(func(t *testing.T, c *testContext) {

		// TODO: check on using additional validation in entity.lua
		// TODO: currently, invalid area just fails with factorio error,
		// TODO: which makes it difficult to tell which part of the area struct was bad

		result := c.rconSc(`factop_entity.destroy_ghosts({name="spidertron", debug=true, area={}})`)
		matches := CannotExecuteRegex.FindStringSubmatch(result)
		require.Equal(t, 2, len(matches))
		assert.Equal(t, "not enough arguments - expected 2 values.", matches[1])

		result = c.rconSc(`factop_entity.destroy_ghosts(
			{name="spidertron", debug=true, area={{},{}} }
		)`)
		matches = CannotExecuteRegex.FindStringSubmatch(result)
		require.Equal(t, 2, len(matches))
		assert.Equal(t, "not enough arguments - expected 2 values.", matches[1])

		result = c.rconSc(`factop_entity.destroy_ghosts(
			{name="spidertron", debug=true, area={{0,0},{0}} }
		)`)
		matches = CannotExecuteRegex.FindStringSubmatch(result)
		require.Equal(t, 2, len(matches))
		assert.Equal(t, "not enough arguments - expected 2 values.", matches[1])

	}))

}

func TestAssembler(t *testing.T) {
	t.Run("cleanup", testCase(func(t *testing.T, c *testContext) {
		result := c.rconSc(`local found = game.surfaces[1].find_entities_filtered{ 
			name={"assembling-machine-3","electric-energy-interface"}}
		for _, e in pairs(found) do e.destroy() end
		`)
		assert.Equal(t, "", result)
	}))
	t.Run("bad name", testCase(func(t *testing.T, c *testContext) {
		result := c.rconSc(`game.surfaces[1].create_entity{ 
			name="bad-entity-name", force="player", position={0,0}}`)
		matches := CannotExecuteRegex.FindStringSubmatch(result)
		require.Equal(t, 2, len(matches))
		assert.Equal(t, "bad-entity-name", matches[1])
	}))
	t.Run("create", testCase(func(t *testing.T, c *testContext) {
		result := c.rconSc(`game.surfaces[1].create_entity{ 
			name="assembling-machine-3", force="player", 
			position={0,0}, create_build_effect_smoke = false }
		game.surfaces[1].create_entity{ 
			name="electric-energy-interface", force="player", 
			position={32,0}, create_build_effect_smoke = false }
		`)
		assert.Equal(t, "", result)
		result = c.rconSc(`local assembler = game.surfaces[1].find_entity("assembling-machine-3", {0,0})
			if assembler ~= nil then
				-- it appears that recipe quality cannot be set on creation of the entity
				assembler.set_recipe("advanced-circuit","legendary")
			end
			rcon.print(assembler ~= nil and assembler.get_recipe() ~= nil)
		`)
		assert.Equal(t, "true", result)

		result = c.rconSc(`local assembler = game.surfaces[1].find_entity("assembling-machine-3", {0,0})
			-- how to insert ingredients based on the recipe
			for _, ing in pairs(assembler.get_recipe().ingredients) do
				assembler.insert({name=ing.name, count=ing.amount*10, quality="legendary"})
			end
		`)
		assert.Equal(t, "", result)

	}))
}

type debugOutput struct {
	Name         string `json:"name"`
	Debug        bool   `json:"debug"`
	Area         *Area  `json:"area,omitempty"`
	Surface      string `json:"surface"`
	DestroyCount int    `json:"destroy_count"`
}

type Area struct {
	LeftTop struct {
		X int `json:"x"`
		Y int `json:"y"`
	} `json:"left_top"`
	RightBottom struct {
		X int `json:"x"`
		Y int `json:"y"`
	} `json:"right_bottom"`
}
