package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"go-ai-tools/config"
	"go-ai-tools/llm"
	"go-ai-tools/mcp"
	"go-ai-tools/ratelimit"
	"go-ai-tools/registry"
	"go-ai-tools/tool"
	"os"
	"time"
)

func main() {
	configPath := flag.String("config", "", "Path to JSON config file")
	mcpPort := flag.Int("mcp-port", 8080, "MCP server port")
	flag.Parse()

	if *configPath != "" {
		runWithConfig(*configPath)
		return
	}

	// Simple demo
	runMCP(*mcpPort)
	fmt.Println("AI tools")
}

func runMCP(port int) {
	fmt.Println("MCP Server")

	// Build demo tools
	reg := builDemoRegistry()

	// MCP server
	srv := mcp.New(reg, port)

	// POST /mcp -d {"method": "tolls/list", "id": 1}
	// POST /mcp -d {"method": "tolls/call", "params": {"name": "get_weather", "arguments": {"location": "Moscow"}}, "id": 2}

	if err := srv.Listen(); err != nil {
		fmt.Printf("MCP server error: %v\n", err)
	}
}

func builDemoRegistry() *registry.Registry {
	reg := registry.New()
	reg.Register(newTool("get_weather", "Get current weather."))
	reg.Register(newTool("get_shops", "Get nearest stores."))
	reg.Register(newTool("local_standarts", "Get the standards of the units of the country of location."))
	reg.Use(registry.LoggingMiddleware(func(name string, dur time.Duration, err error) {}))
	reg.Use(registry.TimeoutMiddleware(30 * time.Second))
	reg.Use(registry.RetryMiddleware(2, 500*time.Millisecond))
	return reg
}

func newTool(name, desc string) tool.Tool {
	return &simpleTool{name: name, desc: desc}
}

type simpleTool struct {
	name string
	desc string
}

func (t *simpleTool) Name() string        { return t.name }
func (t *simpleTool) Description() string { return t.desc }
func (t *simpleTool) InputSchema() *tool.Schema {
	return &tool.Schema{
		Type: "object",
		Properties: map[string]tool.Property{
			"location": {Type: "string", Description: "city name"},
		},
		Required: []string{"location"},
	}
}
func (t *simpleTool) Call(ctx context.Context, args json.RawMessage) (*tool.Result, error) {
	return &tool.Result{
		Success: true,
		Data:    map[string]any{"result": "ok"},
	}, nil
}

func runWithConfig(path string) {
	cfg, err := config.Load(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	cfg.Print()

	switch cfg.Scenario {
	case "basic":
		runBasic(cfg)
	case "production":
		runProduction(cfg)
	default:
		fmt.Printf("Unknow scenario: %s\n", cfg.Scenario)
	}
}

func runBasic(cfg *config.Config) {
	fmt.Println("Starting basic agent")

	// Build tools from config
	reg := buildTools(cfg.Tools)

	reg.Use(registry.LoggingMiddleware(func(name string, dur time.Duration, err error) {
		status := "ok"
		if err != nil {
			status = "err"
		}
		fmt.Printf(" [tool] %s (%v) %s\n", name, dur, status)
	}))

	if cfg.LLM.Endpoint != "" {
		client := llm.New(cfg.LLM.Endpoint, "", cfg.LLM.Model)
		_ = client

		tb := ratelimit.NewTockenBucket(
			cfg.RateLimiting.TockenBucket.Capacity,
			cfg.RateLimiting.TockenBucket.RefillRatePerSec,
		)
		_ = tb

		fmt.Println(" LLM client configured:", cfg.LLM.Provider, "/", cfg.LLM.Model)
		fmt.Println(" Tools:", reg.List())
		fmt.Println(" Rate limit: burst=", cfg.RateLimiting.TockenBucket.Capacity)
	} else {
		fmt.Println(" No LLM endpoint configured - demo mode")
	}

}

func buildTools(toolConfigs []config.ToolConfig) *registry.Registry {
	reg := registry.New()
	ctx := context.Background()
	_ = ctx
	for _, tc := range toolConfigs {
		t := configToTool(tc)
		reg.Register(t)
	}
	return reg
}

func configToTool(tc config.ToolConfig) tool.Tool {
	return &cfgTool{
		cfg: tc,
	}
}

type cfgTool struct {
	cfg config.ToolConfig
}

func (t *cfgTool) Name() string        { return t.cfg.Name }
func (t *cfgTool) Description() string { return t.cfg.Description }
func (t *cfgTool) InputSchema() *tool.Schema {
	return &tool.Schema{
		Type: "object",
		Properties: map[string]tool.Property{
			"location":   {Type: "string", Description: "city name"},
			"expression": {Type: "string", Description: "math expression"},
		},
		Required: t.cfg.RequiredArgs,
	}
}
func (t *cfgTool) Call(ctx context.Context, args json.RawMessage) (*tool.Result, error) {
	return &tool.Result{
		Success: true,
		Data:    map[string]any{"result": "ok"},
	}, nil
}

func runProduction(cfg *config.Config) {
	// Your production mode
}
