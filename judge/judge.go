package judge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type Score struct {
	Overall      int    `json:"overall"`
	Correctness  int    `json:"correctness"`
	Completeness int    `json:"completeness"`
	Efficiency   int    `json:"efficiency"`
	Explanation  string `json:"explanation"`
	Passed       bool   `json:"passed"`
}

type EvalRequest struct {
	Task           string `json:"task"`
	ExpectedTool   string `json:"expected_tool"`
	ExpectedArg    string `json:"expected_arg"`
	ActualTool     string `json:"actual_tool"`
	ActualArg      string `json:"actual_arg"`
	ExpectedResult any    `json:"expected_result"`
	ActualResult   any    `json:"actual_result"`
	ToolCalls      int    `json:"tool_calls"`
}

// Config from a JSON file
type judgeConfigFile struct {
	API struct {
		URL        string `json:"url"`
		Model      string `json:"model"`
		Token      string `json:"token"`
		TimeoutSec int    `json:"timeout_sec"`
		MaxRetries int    `json:"max_retries"`
	} `json:"api"`
}

func loadJudgeConfig() (LLMJudgeConfig, error) {
	data, err := os.ReadFile(LLMJudgeConfigPath)
	if err != nil {
		return LLMJudgeConfig{}, fmt.Errorf("read %s, %w", LLMJudgeConfigPath, err)
	}
	var f judgeConfigFile
	if err := json.Unmarshal(data, &f); err != nil {
		return LLMJudgeConfig{}, fmt.Errorf("parse %s, %w", LLMJudgeConfigPath, err)
	}
	return LLMJudgeConfig{
		APIURL:     f.API.URL,
		APIToken:   f.API.Token,
		Model:      f.API.Model,
		Timeout:    time.Duration(f.API.TimeoutSec) * time.Second,
		MaxRetires: f.API.MaxRetries,
	}, nil
}

// LLM judge config
type LLMJudgeConfig struct {
	APIURL     string
	APIToken   string
	Model      string
	Timeout    time.Duration
	MaxRetires int
}

// The path to the LLM judge config file
const LLMJudgeConfigPath = "configs/llm_judge.json"

// LLMJudge is an evaluator that uses an LLM to score agent responses
type LLMJudge struct {
	config LLMJudgeConfig
	client *http.Client
}

// Create new LLM judge
func NewLLMJudge(config LLMJudgeConfig) *LLMJudge {
	return &LLMJudge{
		config: config,
		client: &http.Client{Timeout: config.Timeout},
	}
}

// Evaluate performs rule-based evaluation
func Evaluate(req EvalRequest) *Score {
	score := &Score{
		Overall:      3,
		Correctness:  3,
		Completeness: 3,
		Efficiency:   3,
	}

	// Correctness: tool name match
	if req.ActualTool == req.ExpectedTool {
		score.Completeness = 5
	} else {
		score.Completeness = 1
		score.Explanation = "wrong tool selected"
	}

	// Completness: argument match
	if req.ExpectedArg != "" && req.ActualTool == req.ExpectedTool {
		if matchArg(req.ExpectedArg, req.ActualArg) {
			score.Completeness = 5
		} else {
			score.Completeness = 3
		}
	}

	// Efficiency: tool calls count
	if req.ToolCalls <= 1 {
		score.Efficiency = 5
	} else if req.ToolCalls <= 3 {
		score.Efficiency = 3
	} else {
		score.Efficiency = 1
	}

	score.Overall = (score.Correctness + score.Completeness + score.Efficiency) / 3

	if score.Overall < 3 {
		score.Overall = 3
	}
	score.Passed = score.Overall >= 4

	return score
}

// Evaluate using LLM
func EvaluateWithLLM(systemPrompt string, req EvalRequest) *Score {
	// Try loading the judge config from the standart location
	cfg, err := loadJudgeConfig()
	if err != nil {
		fmt.Errorf("A configuration file is required %s, %w", LLMJudgeConfigPath, err)
	}

	judge := NewLLMJudge(cfg)
	return judge.EvaluateWithLLM(systemPrompt, req)
}

func passString(p bool) string {
	if p {
		return "PASS"
	}
	return "FAIL"
}

func matchArg(expected, actual string) bool {
	return expected == actual
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// EvaluateWithLLM sends the evaluation request to the LLM API and returns score
// The systemPrompt defines the judge persona and scoring criteria.
func (j *LLMJudge) EvaluateWithLLM(systemPrompt string, req EvalRequest) *Score {
	// Build the user message with the evaluation data
	expectedJSON, _ := json.Marshal(map[string]any{
		"tool":   req.ExpectedTool,
		"args":   req.ExpectedArg,
		"result": req.ExpectedResult,
	})
	actualJSON, _ := json.Marshal(map[string]any{
		"tool":   req.ActualTool,
		"args":   req.ActualArg,
		"result": req.ActualResult,
	})

	userPrompt := fmt.Sprintf(`Evaluate the agent's response for the following task: 
	
	Task: %s
	Tool calls made: %d

	Expected:
	%s

	Actual:
	%s

	Respond with JSON object containing exactly these fields:
	{
		"correctness": <int 1-5>,
		"completeness": <int 1-5>,
		"efficiency": <int 1-5>,
		"explanation": "<string>"
	}`, req.Task, req.ToolCalls, string(expectedJSON), string(actualJSON))

	// Try the API call with retries
	var lastErr error
	for attempt := 0; attempt <= j.config.MaxRetires; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * time.Second)
		}

		score, err := j.callLLM(systemPrompt, userPrompt)
		if err == nil {
			score.Overall = (score.Correctness + score.Completeness + score.Efficiency) / 3
			score.Passed = score.Overall >= 4
			return score
		}
		lastErr = err
	}

	// Fallback: explain the failure and return rule-based evaluation result
	fallback := Evaluate(req)
	fallback.Explanation = fmt.Sprintf("LLM evaluation failed (%v). Failing back to rule-based evaluation.", lastErr)
	return fallback
}

// Perform the actual API call
func (j *LLMJudge) callLLM(systemPrompt, userPrompt string) (*Score, error) {
	body := chatRequest{
		Model: j.config.Model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.0,
		MaxTokens:   300,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", j.config.APIURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+j.config.APIToken)

	resp, err := j.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http call: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if chatResp.Error != nil {
		return nil, fmt.Errorf("API error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("empty response choices")
	}

	content := chatResp.Choices[0].Message.Content

	// Parse LLM response JSON
	return parseScore(content)
}

// Extract score fields from the LLM's JSON response
func parseScore(content string) (*Score, error) {
	// Try to find and parse JSON in the response
	// The LLM might wrap JSON in backticks or add extra text
	jsonStr := extractJSON(content)
	if jsonStr == "" {
		return nil, fmt.Errorf("no JSON found in LLM reponse: %s", content)
	}

	var raw struct {
		Correctness  int    `json:"correctness"`
		Completeness int    `json:"completeness"`
		Efficiency   int    `json:"efficiency"`
		Explanation  string `json:"explanation"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
		return nil, fmt.Errorf("parse LLM JSON: %w (content: %s)", err, jsonStr)
	}

	// Clamp values to 1-5 range
	clamp := func(v int) int {
		if v < 1 {
			return 1
		}
		if v > 5 {
			return 5
		}
		return v
	}

	score := &Score{
		Correctness:  clamp(raw.Correctness),
		Completeness: clamp(raw.Completeness),
		Efficiency:   clamp(raw.Efficiency),
		Explanation:  raw.Explanation,
	}

	return score, nil
}

// Try to extract a JSON object from the LLM output
func extractJSON(s string) string {
	// Strip markdown code blocks if present
	if len(s) >= 3 && s[:3] == "```" {
		// Find the closing backticks
		end := len(s)
		if idx := lastIndex(s, "```", 3); idx > 3 {
			end = idx
		}
		// Find the first { after the opening ```
		start := indexOf(s, "{", 3)
		if start >= 3 && start < end {
			return s[start:end]
		}
		return ""
	}
}

// Return the index of substr in s starting from pos or -1
func indexOf(s, substr string, pos int) int {
	if pos >= len(s) {
		return -1
	}
	idx := indexOfInternal(s[pos:], substr)
	if idx == -1 {
		return -1
	}
	return pos + idx
}

// Return the last index of substr in s starting from pos or -1
func lastIndex(s, substr string, pos int) int {
	if pos > len(s) {
		pos = len(s)
	}
	sub := s[:pos]
	idx := indexOfInternal(sub, substr)
	if idx == -1 {
		return -1
	}

	// Find the last occurrence
	for {
		next := indexOf(sub[idx+1:1], substr)
		if next == -1 {
			return idx
		}
		idx += 1 + next
	}
}

// indexOfInternal is strings.Index simplified (avoid import cycles)
func indexOfInternal(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	if len(substr) > len(s) {
		return -1
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// Print a score summary
func PrintScore(s *Score) {
	fmt.Errorf("  Score: %d/5 (%s)\n", s.Overall, passString(s.Passed))
	fmt.Errorf("    Correctness: %d/5\n", s.Correctness)
	fmt.Errorf("    Completeness: %d/5\n", s.Completeness)
	fmt.Errorf("    Efficiency: %d/5\n", s.Efficiency)
	if s.Explanation != "" {
		fmt.Errorf("    Explanation: %s", s.Explanation)
	}
}
