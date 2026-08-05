package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPascalCase(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"entity_died", "EntityDied"},
		{"move", "Move"},
		{"player_joined", "PlayerJoined"},
		{"game_tick", "GameTick"},
		{"resource_depleted", "ResourceDepleted"},
		{"surface_created", "SurfaceCreated"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.want, pascalCase(tc.input))
		})
	}
}

func TestParseExpr(t *testing.T) {
	cases := []struct {
		name     string
		field    FieldSchema
		contains string
	}{
		{
			name:     "string field",
			field:    FieldSchema{Name: "Name", Type: "string", Position: 0},
			contains: "result.Name = parts[0]",
		},
		{
			name:     "int field",
			field:    FieldSchema{Name: "Index", Type: "int", Position: 1},
			contains: "strconv.Atoi",
		},
		{
			name:     "float64 field",
			field:    FieldSchema{Name: "X", Type: "float64", Position: 2},
			contains: "strconv.ParseFloat",
		},
		{
			name:     "uint64 field",
			field:    FieldSchema{Name: "UnitNumber", Type: "uint64", Position: 3},
			contains: "strconv.ParseUint",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseExpr(tc.field)
			assert.Contains(t, got, tc.contains)
		})
	}
}

func TestFormatExpr(t *testing.T) {
	cases := []struct {
		name     string
		field    FieldSchema
		contains string
	}{
		{
			name:     "string field",
			field:    FieldSchema{Name: "Name", Type: "string"},
			contains: "m.Name",
		},
		{
			name:     "int field",
			field:    FieldSchema{Name: "Index", Type: "int"},
			contains: "strconv.Itoa(m.Index)",
		},
		{
			name:     "float64 field",
			field:    FieldSchema{Name: "X", Type: "float64"},
			contains: "strconv.FormatFloat",
		},
		{
			name:     "float64 precision",
			field:    FieldSchema{Name: "X", Type: "float64"},
			contains: "'f', 1, 64",
		},
		{
			name:     "uint64 field",
			field:    FieldSchema{Name: "Tick", Type: "uint64"},
			contains: "strconv.FormatUint",
		},
		{
			name:     "uint64 base 10",
			field:    FieldSchema{Name: "Tick", Type: "uint64"},
			contains: ", 10)",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := formatExpr(tc.field)
			assert.Contains(t, got, tc.contains)
		})
	}
}

func TestFieldCount(t *testing.T) {
	cases := []struct {
		name   string
		fields []FieldSchema
		want   int
	}{
		{
			name: "all non-tag non-optional",
			fields: []FieldSchema{
				{Name: "Name", Type: "string", Position: 0},
				{Name: "Index", Type: "int", Position: 1},
				{Name: "X", Type: "float64", Position: 2},
			},
			want: 3,
		},
		{
			name: "mix of tag and non-tag",
			fields: []FieldSchema{
				{Name: "Event", Type: "string", Position: -1, IsTag: true},
				{Name: "Name", Type: "string", Position: 0},
				{Name: "Index", Type: "int", Position: 1},
			},
			want: 2,
		},
		{
			name: "fields with optional excluded",
			fields: []FieldSchema{
				{Name: "Name", Type: "string", Position: 0},
				{Name: "Index", Type: "int", Position: 1},
				{Name: "Cause", Type: "string", Position: 2, Optional: true},
			},
			want: 2,
		},
		{
			name: "tag and optional excluded",
			fields: []FieldSchema{
				{Name: "Event", Type: "string", Position: -1, IsTag: true},
				{Name: "Name", Type: "string", Position: 0},
				{Name: "Cause", Type: "string", Position: 1, Optional: true},
			},
			want: 1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, fieldCount(tc.fields))
		})
	}
}

func TestTotalFields(t *testing.T) {
	cases := []struct {
		name   string
		fields []FieldSchema
		want   int
	}{
		{
			name: "all non-tag including optional",
			fields: []FieldSchema{
				{Name: "Name", Type: "string", Position: 0},
				{Name: "Index", Type: "int", Position: 1},
				{Name: "Cause", Type: "string", Position: 2, Optional: true},
			},
			want: 3,
		},
		{
			name: "mix of tag and non-tag",
			fields: []FieldSchema{
				{Name: "Event", Type: "string", Position: -1, IsTag: true},
				{Name: "Name", Type: "string", Position: 0},
				{Name: "Index", Type: "int", Position: 1},
				{Name: "Cause", Type: "string", Position: 2, Optional: true},
			},
			want: 3,
		},
		{
			name: "only tag field excluded",
			fields: []FieldSchema{
				{Name: "Event", Type: "string", Position: -1, IsTag: true},
				{Name: "Name", Type: "string", Position: 0},
			},
			want: 1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, totalFields(tc.fields))
		})
	}
}

func TestSnakeCase(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"SurfaceName", "surface_name"},
		{"PlayerIndex", "player_index"},
		{"X", "x"},
		{"UnitNumber", "unit_number"},
		{"Name", "name"},
		{"Y", "y"},
		{"Cause", "cause"},
		{"SurfaceIndex", "surface_index"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.want, snakeCase(tc.input))
		})
	}
}

func TestLuaFormatSpec(t *testing.T) {
	cases := []struct {
		name   string
		fields []FieldSchema
		want   string
	}{
		{
			name: "entity fields",
			fields: []FieldSchema{
				{Name: "Event", Type: "string", Position: -1, IsTag: true},
				{Name: "Name", Type: "string", Position: 0},
				{Name: "X", Type: "float64", Position: 1},
				{Name: "Y", Type: "float64", Position: 2},
				{Name: "UnitNumber", Type: "uint64", Position: 3},
				{Name: "SurfaceName", Type: "string", Position: 4},
				{Name: "SurfaceIndex", Type: "int", Position: 5},
				{Name: "PlayerIndex", Type: "int", Position: 6},
				{Name: "Cause", Type: "string", Position: 7, Optional: true},
			},
			want: "%s:%.1f:%.1f:%d:%s:%d:%d:%s",
		},
		{
			name: "single string field",
			fields: []FieldSchema{
				{Name: "Tick", Type: "uint64", Position: 0},
			},
			want: "%d",
		},
		{
			name: "skips tag fields",
			fields: []FieldSchema{
				{Name: "Event", Type: "string", Position: -1, IsTag: true},
				{Name: "Value", Type: "int", Position: 0},
			},
			want: "%d",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, luaFormatSpec(tc.fields))
		})
	}
}

func TestLuaArgs(t *testing.T) {
	cases := []struct {
		name string
		schema EventSchema
		want   string
	}{
		{
			name: "nil lua schema",
			schema: EventSchema{
				Fields: []FieldSchema{
					{Name: "Name", Type: "string", Position: 0},
				},
			},
			want: "",
		},
		{
			name: "entity fields with defaults",
			schema: EventSchema{
				Fields: []FieldSchema{
					{Name: "Event", Type: "string", Position: -1, IsTag: true},
					{Name: "Name", Type: "string", Position: 0},
					{Name: "X", Type: "float64", Position: 1},
					{Name: "Y", Type: "float64", Position: 2},
					{Name: "UnitNumber", Type: "uint64", Position: 3},
					{Name: "SurfaceName", Type: "string", Position: 4},
					{Name: "SurfaceIndex", Type: "int", Position: 5},
					{Name: "PlayerIndex", Type: "int", Position: 6},
					{Name: "Cause", Type: "string", Position: 7, Optional: true},
				},
				Lua: &LuaSchema{
					Fields: map[string]string{
						"Name":         "entity.name",
						"X":            "entity.position.x",
						"Y":            "entity.position.y",
						"UnitNumber":   "entity.unit_number",
						"SurfaceName":  "surface.name",
						"SurfaceIndex": "surface.index",
						"PlayerIndex":  "event.player_index",
						"Cause":        "cause",
					},
					Defaults: map[string]string{
						"PlayerIndex": "0",
						"Cause":       "",
					},
				},
			},
			want: `entity.name, entity.position.x, entity.position.y, entity.unit_number, surface.name, surface.index, event.player_index or 0, cause or ""`,
		},
		{
			name: "simple game tick",
			schema: EventSchema{
				Fields: []FieldSchema{
					{Name: "Tick", Type: "uint64", Position: 0},
				},
				Lua: &LuaSchema{
					Fields: map[string]string{
						"Tick": "game.tick",
					},
					Defaults: map[string]string{},
				},
			},
			want: "game.tick",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, luaArgs(tc.schema))
		})
	}
}
