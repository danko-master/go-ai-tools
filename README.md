# Unified Agent Framework

```
./
|-- tool/      # Core interfaces (Tool, Schema, Result)
|-- llm/       # OpenAI-compatible LLM client
|-- registry/  # Tool registartion with middleware chain
|-- scheduler/ # Plan execution with fallback
|-- ratelimit/ # Token bucket, sliding window
|-- mcp/       # MCP server
|-- config/    # Config loader and parser
|-- configs/   # Example JSON configs
```
