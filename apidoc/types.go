package main

// RuntimeAPI is the top-level structure of the Factorio runtime-api.json.
type RuntimeAPI struct {
	Application        string     `json:"application"`
	ApplicationVersion string     `json:"application_version"`
	APIVersion         int        `json:"api_version"`
	Stage              string     `json:"stage"`
	Classes            []Class    `json:"classes"`
	Events             []Event    `json:"events"`
	Concepts           []Concept  `json:"concepts"`
	Defines            []Define   `json:"defines"`
	GlobalObjects      []Parameter `json:"global_objects"`
	GlobalFunctions    []Method   `json:"global_functions"`
}

// Class represents a Factorio Lua class (LuaObject).
type Class struct {
	BasicMember
	Visibility []string    `json:"visibility,omitempty"`
	Parent     string      `json:"parent,omitempty"`
	Abstract   bool        `json:"abstract"`
	Methods    []Method    `json:"methods"`
	Attributes []Attribute `json:"attributes"`
	Operators  []any       `json:"operators"`
}

// Event represents a Factorio event.
type Event struct {
	BasicMember
	Data   []Parameter `json:"data"`
	Filter string      `json:"filter,omitempty"`
}

// Concept represents a Factorio concept type.
type Concept struct {
	BasicMember
	Type any `json:"type"`
}

// Define represents a Factorio define (enum-like).
type Define struct {
	BasicMember
	Values  []DefineValue `json:"values,omitempty"`
	Subkeys []Define      `json:"subkeys,omitempty"`
}

// DefineValue is a single value within a Define.
type DefineValue struct {
	Name        string `json:"name"`
	Order       int    `json:"order"`
	Description string `json:"description"`
}

// BasicMember contains fields common to most API members.
type BasicMember struct {
	Name        string   `json:"name"`
	Order       int      `json:"order"`
	Description string   `json:"description"`
	Lists       []string `json:"lists,omitempty"`
	Examples    []string `json:"examples,omitempty"`
}

// Method represents a class method or global function.
type Method struct {
	BasicMember
	Visibility []string     `json:"visibility,omitempty"`
	Parameters []Parameter  `json:"parameters"`
	Format     MethodFormat `json:"format"`
	ReturnValues []Parameter `json:"return_values"`
}

// MethodFormat describes how method arguments are structured.
type MethodFormat struct {
	TakesTable    bool `json:"takes_table"`
	TableOptional bool `json:"table_optional,omitempty"`
}

// Attribute represents a class attribute.
type Attribute struct {
	BasicMember
	Visibility []string `json:"visibility,omitempty"`
	ReadType   any      `json:"read_type,omitempty"`
	WriteType  any      `json:"write_type,omitempty"`
	Optional   bool     `json:"optional"`
}

// Parameter represents a method parameter, return value, or global object.
type Parameter struct {
	Name        string `json:"name"`
	Order       int    `json:"order"`
	Description string `json:"description"`
	Type        any    `json:"type"`
	Optional    bool   `json:"optional"`
}
