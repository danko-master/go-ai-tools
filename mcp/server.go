// MCP HTTP Server

// Example:
// curl -X POST http://localhost:8080/mcp \
// 	-d '{"jsonrpc":"","method":"tools/list","id":1}'

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"go-ai-tools/registry"
	"io"
	"net/http"
)

// MCP HTTP
type Server struct {
	registry *registry.Registry
	agent    AgentRunner
	port     int
}

// Agent interface
type AgentRunner interface {
	Run(ctx context.Context, query string) (string, error)
}

// Create MCP-server
func New(reg *registry.Registry, port int) *Server {
	return &Server{
		registry: reg,
		port:     port,
	}
}

// Connect agent
func (s *Server) WithAgent(a AgentRunner) *Server {
	s.agent = a
	return s
}

// Listen HTTP
func (s *Server) Listen() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", s.handleMCP)
	mux.HandleFunc("/health", s.handleHealth)

	addr := fmt.Sprintf(":%d", s.port)
	fmt.Printf(" MCP server listening on %s\n", addr)
	return http.ListenAndServe(addr, mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleMCP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, nil, -32700, "parse error")
		return
	}

	var req Request
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, nil, -32700, "invalid request")
		return
	}

	switch req.Method {
	case MethodToolsList:
		s.handelToolsList(w, req)
	case MethodToolsCall:
		s.handelToolsCall(w, req)
	case MethodAgentRun:
		s.handelAgentRun(w, req)
	default:
		writeError(w, req.ID, -32601, fmt.Sprintf("method not found: %s", req.Method))
	}
}

// Tools list
func (s *Server) handelToolsList(w http.ResponseWriter, req Request) {
	defs := s.registry.Definitions()
	var tools []ToolDef
	for _, d := range defs {
		if fn, ok := d["function"].(map[string]any); ok {
			tools = append(tools, ToolDef{
				Name:        toString(fn["name"]),
				Description: toString(fn["description"]),
				InputSchema: fn["parameters"],
			})
		}
	}

	resp := Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  ListToolsResult{Tools: tools},
	}
	writeJSON(w, resp)
}

// Tools call by name and args
func (s *Server) handelToolsCall(w http.ResponseWriter, req Request) {
	var params CallToolsParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		writeError(w, req.ID, -32602, "invalid params")
		return
	}

	ctx := context.Background()
	result, err := s.registry.Call(ctx, params.Name, params.Arguments)
	if err != nil {
		writeError(w, req.ID, -32603, err.Error())
		return
	}

	content := "ok"
	if result.Success {
		data, _ := json.Marshal(result.Data)
		content = string(data)
	} else {
		content = result.Error
	}

	resp := Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: CallToolsResult{
			Content: []ContentItem{{Type: "text", Text: content}},
			IsError: !result.Success,
		},
	}
	writeJSON(w, resp)
}

// ReAct agent
func (s *Server) handelAgentRun(w http.ResponseWriter, req Request) {
	if s.agent == nil {
		writeError(w, req.ID, -32001, "agent not configured")
		return
	}

	var params AgentRunParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		writeError(w, req.ID, -32602, "invalid params")
		return
	}

	ctx := context.Background()
	output, err := s.agent.Run(ctx, params.Query)
	if err != nil {
		writeError(w, req.ID, -32603, err.Error())
		return
	}

	resp := Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: AgentRunResult{
			Output:   output,
			Duration: "simulated",
		},
	}

	writeJSON(w, resp)
}

func writeError(w http.ResponseWriter, id any, code int, msg string) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &ErrorObj{Code: code, Message: msg},
	}
	writeJSON(w, resp)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
