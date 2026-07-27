package tool

import (
	"context"
	"encoding/json"
	"time"
)

// JSON Schema for a tool's argument
type Schema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties,omitempty"`
	Required   []string            `json:"required,omitempty"`
}

// Property is a single field in the schema
type Property struct {
	Type        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Enum        []string `json:"enum,omitempty"`
}

// Definition describes a tool to the LLM
type Definition struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	InputSchema *Schema `json:"input_schema"`
}

// Result is the outcome of a tool call
type Result struct {
	ToolName string        `json:"tool_name"`
	Success  bool          `json:"success"`
	Data     any           `json:"data,omitempty"`
	Error    string        `json:"error,omitempty"`
	Duration time.Duration `json:"duration,omitempty"`
}

// Tool is the core abstraction: a function the agent can call
type Tool interface {
	Name() string
	Description() string
	InputSchema() *Schema
	Call(ctx context.Context, args json.RawMessage) (*Result, error)
}
