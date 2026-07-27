// Tool registration with middleware chain
package registry

import (
	"context"
	"fmt"
	"go-ai-tools/tool"
	"strings"
	"sync"
	"time"
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

// New creates an empty registry
func New() *Registry {
	return &Registry{tools: make(map[string]tool.Tool)}
}

// Register adds a tool
func (r *Registry) Register(t tool.Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[t.Name()] = t
}

// Use appends middleware
func (r *Registry) Use(m Middleware) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.middleware = append(r.middleware, m)
}

// Get returns a tool by name
func (r *Registry) Get(name string) (tool.Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("tool %q not found", name)
	}
	return t, nil
}

// Definitions returns tool definitions for the LLM
func (r *Registry) Definitions() []map[string]any {
	r.mu.RLock()
	defer r.mu.Unlock()
	var defs []map[string]any

	for _, t := range r.tools {
		defs = append(defs, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Name(),
				"description": t.Description(),
				"parameters":  t.InputSchema(),
			},
		})
	}

	return defs
}

// List returns all tool names
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var names []string
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// jsonEx is a type alias for json.RawMessage to avoid import issues
type jsonEx = []byte

// Call invokes a tool with all middleware applied
func (r *Registry) Call(ctx context.Context, name string, raw jsonEx) (*tool.Result, error) {
	t, err := r.Get(name)
	if err != nil {
		return nil, err
	}

	// Validate required args
	schema := t.InputSchema()
	if schema != nil {
		for _, req := range schema.Required {
			if !strings.Contains(string(raw), `"`+req+`"`) {
				return nil, fmt.Errorf("validation: missing required field %q", req)
			}
		}
	}

	// Build the call chain
	var call ToolCallFunc = func(ctx context.Context, t tool.Tool, args string) (*tool.Result, error) {
		return t.Call(ctx, []byte(args))
	}

	// Apply middleware in reverse order
	for i := len(r.middleware) - 1; i >= 0; i-- {
		call = r.middleware[i](call)
	}

	return call(ctx, t, string(raw))
}

// LoggingMiddleware logs every tool call
func LoggingMiddleware(logFn func(name string, dur time.Duration, err error)) Middleware {
	return func(next ToolCallFunc) ToolCallFunc {
		return func(ctx context.Context, t tool.Tool, args string) (*tool.Result, error) {
			start := time.Now()
			result, err := next(ctx, t, args)
			dur := time.Since(start)
			if logFn != nil {
				logFn(t.Name(), dur, err)
			}
			return result, err
		}
	}
}

// TimeoutMiddleware applies a context timeout
func TimeoutMiddleware(timeout time.Duration) Middleware {
	return func(next ToolCallFunc) ToolCallFunc {
		return func(ctx context.Context, t tool.Tool, args string) (*tool.Result, error) {
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			return next(ctx, t, args)
		}
	}
}

// RetryMiddleware retries on a failure
func RetryMiddleware(maxRetries int, backoff time.Duration) Middleware {
	return func(next ToolCallFunc) ToolCallFunc {
		return func(ctx context.Context, t tool.Tool, args string) (*tool.Result, error) {
			var lastErr error
			for i := 0; i <= maxRetries; i++ {
				result, err := next(ctx, t, args)
				if err == nil && result != nil && result.Success {
					return result, nil
				}
				if err != nil {
					lastErr = err
				} else {
					lastErr = fmt.Errorf("tool execution failed: %s", result.Error)
				}
				if i < maxRetries {
					time.Sleep(backoff)
				}
			}
			return nil, fmt.Errorf("after %d retries: %w", maxRetries, lastErr)
		}
	}
}
