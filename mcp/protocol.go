// Model Context Protocol integration

package mcp

import "encoding/json"

// Error JSON-RPC
type ErrorObj struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Request - input from LLM
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response LLM
type Response struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id"`
	Result  any       `json:"result"`
	Error   *ErrorObj `json:"error,omitempty"`
}

// MCP method constants
const (
	MethodToolsList = "tools/list"
	MethodToolsCall = "tools/call"
	MethodAgentRun  = "agent/run"
)

// ToolDef - MCP-tool definition
type ToolDef struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"inputSchema"`
}

// Content item
type ContentItem struct {
	Type string `json:"type"` // "text", "image", "resource"
	Text string `json:"text,omitempty"`
}

// tools/list
// Response
type ListToolsResult struct {
	Tools []ToolDef `json:"tools"`
}

// tools/call
// Tool call params
type CallToolsParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// The result of calling the tool
type CallToolsResult struct {
	Content []ContentItem `json:"content"`
	IsError bool          `json:"isError"`
}

// agent/run
type AgentRunParams struct {
	Query string `json:"query"`
	Mode  string `json:"mode,omitempty"` // "react", "plan"
}

// Agent result
type AgentRunResult struct {
	Output   string `json:"output"`
	Duration string `json:"duration"`
}
