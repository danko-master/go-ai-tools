package registry

import (
	"context"
	"go-ai-tools/tool"
	"sync"
)

// ToolCallFunc is the actual call function
type ToolCallFunc func(ctx context.Context, t tool.Tool, args string) (*tool.Result, error)

// Middleware wraps tool execution
type Middleware func(next ToolCallFunc) ToolCallFunc

// Registry holds tools and middleware
type Registry struct {
	mu         sync.RWMutex
	tools      map[string]tool.Tool
	middleware []Middleware
}
