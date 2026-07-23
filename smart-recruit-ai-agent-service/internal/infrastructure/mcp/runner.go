package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"smart-recruit-ai-agent-service/internal/domain/model"
	"smart-recruit-ai-agent-service/internal/domain/policy"
)

const defaultTimeout = 30 * time.Second

type ServerConfig struct {
	ID                  int64
	Name                string
	Transport           string
	CommandOrURL        string
	ArgsJSON            string
	EnvVarsJSON         string
	TimeoutSeconds      int
	Enabled             bool
	AllowPrivateNetwork bool
	AllowedCommands     []string
}

type Tool struct {
	Name        string
	Description string
	SchemaJSON  string
}

type TestResult struct {
	Success    bool
	Detail     string
	ToolsFound int
	DurationMs int64
}

type CallResult struct {
	Content    string
	Error      string
	DurationMs int64
}

type ToolLog struct {
	ServerID       int64
	ToolName       string
	ArgsJSON       string
	ResultContent  string
	DurationMs     int64
	ErrorMsg       string
	CalledByHRID   int64
	SessionID      int64
	PolicyID       int64
	PolicyDecision string
	PolicyReason   string
}

type Runner interface {
	Test(context.Context, ServerConfig) (TestResult, error)
	ListTools(context.Context, ServerConfig) ([]Tool, error)
	CallTool(context.Context, ServerConfig, string, map[string]any) (CallResult, error)
}

type RunnerFunc struct {
	TestFunc      func(context.Context, ServerConfig) (TestResult, error)
	ListToolsFunc func(context.Context, ServerConfig) ([]Tool, error)
	CallToolFunc  func(context.Context, ServerConfig, string, map[string]any) (CallResult, error)
}

func (f RunnerFunc) Test(ctx context.Context, cfg ServerConfig) (TestResult, error) {
	return f.TestFunc(ctx, cfg)
}

func (f RunnerFunc) ListTools(ctx context.Context, cfg ServerConfig) ([]Tool, error) {
	return f.ListToolsFunc(ctx, cfg)
}

func (f RunnerFunc) CallTool(ctx context.Context, cfg ServerConfig, tool string, args map[string]any) (CallResult, error) {
	return f.CallToolFunc(ctx, cfg, tool, args)
}

type DefaultRunner struct {
	HTTPClient *http.Client
}

func NewRunner() *DefaultRunner {
	return &DefaultRunner{HTTPClient: &http.Client{}}
}

func (r *DefaultRunner) Test(ctx context.Context, cfg ServerConfig) (TestResult, error) {
	start := time.Now()
	tools, err := r.ListTools(ctx, cfg)
	duration := time.Since(start).Milliseconds()
	if err != nil {
		return TestResult{Success: false, Detail: err.Error(), DurationMs: duration}, err
	}
	return TestResult{Success: true, Detail: "connection ok", ToolsFound: len(tools), DurationMs: duration}, nil
}

func (r *DefaultRunner) ListTools(ctx context.Context, cfg ServerConfig) ([]Tool, error) {
	if err := validateRuntimeConfig(cfg); err != nil {
		return nil, err
	}
	switch strings.ToLower(strings.TrimSpace(cfg.Transport)) {
	case model.MCPTransportHTTP:
		var result struct {
			Tools []mcpToolPayload `json:"tools"`
		}
		if err := r.httpRPC(ctx, cfg, "tools/list", map[string]any{}, &result); err != nil {
			return nil, err
		}
		return mapTools(result.Tools), nil
	case model.MCPTransportStdio:
		return r.stdioListTools(ctx, cfg)
	case model.MCPTransportSSE:
		return nil, fmt.Errorf("mcp transport sse is not supported by native runner")
	default:
		return nil, fmt.Errorf("unsupported mcp transport %q", cfg.Transport)
	}
}

func (r *DefaultRunner) CallTool(ctx context.Context, cfg ServerConfig, tool string, args map[string]any) (CallResult, error) {
	if err := validateRuntimeConfig(cfg); err != nil {
		return CallResult{}, err
	}
	start := time.Now()
	switch strings.ToLower(strings.TrimSpace(cfg.Transport)) {
	case model.MCPTransportHTTP:
		var result mcpCallPayload
		err := r.httpRPC(ctx, cfg, "tools/call", map[string]any{"name": tool, "arguments": args}, &result)
		return callResultFromPayload(result, time.Since(start).Milliseconds(), err)
	case model.MCPTransportStdio:
		result, err := r.stdioCallTool(ctx, cfg, tool, args)
		result.DurationMs = time.Since(start).Milliseconds()
		return result, err
	case model.MCPTransportSSE:
		return CallResult{DurationMs: time.Since(start).Milliseconds()}, fmt.Errorf("mcp transport sse is not supported by native runner")
	default:
		return CallResult{DurationMs: time.Since(start).Milliseconds()}, fmt.Errorf("unsupported mcp transport %q", cfg.Transport)
	}
}

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type rpcResponse struct {
	ID     int64           `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *rpcError       `json:"error"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type mcpToolPayload struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
	Schema      json.RawMessage `json:"schema"`
}

type mcpCallPayload struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	IsError bool            `json:"isError"`
	Result  json.RawMessage `json:"result"`
}

func (r *DefaultRunner) httpRPC(ctx context.Context, cfg ServerConfig, method string, params any, out any) error {
	payload, _ := json.Marshal(rpcRequest{JSONRPC: "2.0", ID: 1, Method: method, Params: params})
	req, err := http.NewRequestWithContext(timeoutContext(ctx, cfg), http.MethodPost, cfg.CommandOrURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := r.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("mcp http status %d", resp.StatusCode)
	}
	var rpcResp rpcResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return fmt.Errorf("mcp response invalid: %w", err)
	}
	if rpcResp.Error != nil {
		return fmt.Errorf("mcp rpc error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(rpcResp.Result, out)
}

func (r *DefaultRunner) stdioListTools(ctx context.Context, cfg ServerConfig) ([]Tool, error) {
	var result struct {
		Tools []mcpToolPayload `json:"tools"`
	}
	if err := stdioRPC(ctx, cfg, "tools/list", map[string]any{}, &result); err != nil {
		return nil, err
	}
	return mapTools(result.Tools), nil
}

func (r *DefaultRunner) stdioCallTool(ctx context.Context, cfg ServerConfig, tool string, args map[string]any) (CallResult, error) {
	var result mcpCallPayload
	err := stdioRPC(ctx, cfg, "tools/call", map[string]any{"name": tool, "arguments": args}, &result)
	return callResultFromPayload(result, 0, err)
}

func stdioRPC(ctx context.Context, cfg ServerConfig, method string, params any, out any) error {
	parts := strings.Fields(cfg.CommandOrURL)
	if len(parts) == 0 {
		return fmt.Errorf("mcp stdio command is required")
	}
	args := append([]string(nil), parts[1:]...)
	args = append(args, parseStringArray(cfg.ArgsJSON)...)
	cmd := exec.CommandContext(timeoutContext(ctx, cfg), parts[0], args...)
	if env := parseStringMap(cfg.EnvVarsJSON); len(env) > 0 {
		for key, value := range env {
			cmd.Env = append(cmd.Env, key+"="+value)
		}
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	enc := json.NewEncoder(stdin)
	dec := json.NewDecoder(bufio.NewReader(stdout))
	if err := enc.Encode(rpcRequest{JSONRPC: "2.0", ID: 1, Method: "initialize", Params: map[string]any{"protocolVersion": "2024-11-05", "capabilities": map[string]any{}, "clientInfo": map[string]any{"name": "smart-recruit-ai-agent-service", "version": "native"}}}); err != nil {
		return finishCommand(cmd, stdin, err)
	}
	var initResp rpcResponse
	if err := dec.Decode(&initResp); err != nil {
		return finishCommand(cmd, stdin, err)
	}
	if initResp.Error != nil {
		return finishCommand(cmd, stdin, fmt.Errorf("mcp initialize error: %s", initResp.Error.Message))
	}
	if err := enc.Encode(map[string]any{"jsonrpc": "2.0", "method": "notifications/initialized"}); err != nil {
		return finishCommand(cmd, stdin, err)
	}
	if err := enc.Encode(rpcRequest{JSONRPC: "2.0", ID: 2, Method: method, Params: params}); err != nil {
		return finishCommand(cmd, stdin, err)
	}
	var callResp rpcResponse
	if err := dec.Decode(&callResp); err != nil {
		return finishCommand(cmd, stdin, err)
	}
	if callResp.Error != nil {
		return finishCommand(cmd, stdin, fmt.Errorf("mcp rpc error %d: %s", callResp.Error.Code, callResp.Error.Message))
	}
	if out != nil {
		if err := json.Unmarshal(callResp.Result, out); err != nil {
			return finishCommand(cmd, stdin, err)
		}
	}
	return finishCommand(cmd, stdin, nil)
}

func finishCommand(cmd *exec.Cmd, stdin io.Closer, err error) error {
	_ = stdin.Close()
	waitErr := cmd.Wait()
	if err != nil {
		return err
	}
	return waitErr
}

func validateRuntimeConfig(cfg ServerConfig) error {
	server := model.MCPServerConfig{ID: uint64(cfg.ID), Name: cfg.Name, Transport: cfg.Transport, TimeoutSeconds: cfg.TimeoutSeconds, Enabled: cfg.Enabled, AllowPrivateNetwork: cfg.AllowPrivateNetwork, AllowedCommands: cfg.AllowedCommands}
	if strings.EqualFold(cfg.Transport, model.MCPTransportStdio) {
		server.Command = cfg.CommandOrURL
	} else {
		server.URL = cfg.CommandOrURL
	}
	return policy.ValidateMCPServerConfig(server)
}

func timeoutContext(ctx context.Context, cfg ServerConfig) context.Context {
	timeout := defaultTimeout
	if cfg.TimeoutSeconds > 0 {
		timeout = time.Duration(cfg.TimeoutSeconds) * time.Second
	}
	child, _ := context.WithTimeout(ctx, timeout)
	return child
}

func mapTools(items []mcpToolPayload) []Tool {
	tools := make([]Tool, 0, len(items))
	for _, item := range items {
		schema := item.InputSchema
		if len(schema) == 0 {
			schema = item.Schema
		}
		tools = append(tools, Tool{Name: item.Name, Description: item.Description, SchemaJSON: string(schema)})
	}
	return tools
}

func callResultFromPayload(payload mcpCallPayload, durationMs int64, err error) (CallResult, error) {
	result := CallResult{DurationMs: durationMs}
	if len(payload.Content) > 0 {
		parts := make([]string, 0, len(payload.Content))
		for _, item := range payload.Content {
			if item.Text != "" {
				parts = append(parts, item.Text)
			}
		}
		result.Content = strings.Join(parts, "\n")
	} else if len(payload.Result) > 0 {
		result.Content = string(payload.Result)
	}
	if payload.IsError {
		result.Error = result.Content
	}
	if err != nil {
		result.Error = err.Error()
	}
	return result, err
}

func parseStringArray(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	var items []string
	if err := json.Unmarshal([]byte(value), &items); err != nil {
		return nil
	}
	return items
}

func parseStringMap(value string) map[string]string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	var items map[string]string
	if err := json.Unmarshal([]byte(value), &items); err != nil {
		return nil
	}
	return items
}
