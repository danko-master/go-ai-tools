package main

import (
	"flag"
	"fmt"
	"go-ai-tools/config"
	"os"
)

func main() {
	configPath := flag.String("config", "", "Path to JSON config file")
	mcpMode := flag.Bool("mcp", false, "Run as MCP server (JSON-RPC HTTP)")
	mcpPort := flag.Int("mcp-port", 8080, "MCP server port")
	flag.Parse()

	if *mcpMode {
		runMCP(*mcpPort)
		return
	}

	if *configPath != "" {
		runWithConfig(*configPath)
		return
	}

	fmt.Println("AI tools")
	demoLevel()
}

func runMCP(port int) {
	fmt.Println("MCP Server")

	// TODO
	_ = port

	// Строим реест tools
	// reg := builDemoRegistry()
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
	// TODO

}

func runProduction(cfg *config.Config) {
	// TODO

}

func demoLevel() {
	// TODO
}
