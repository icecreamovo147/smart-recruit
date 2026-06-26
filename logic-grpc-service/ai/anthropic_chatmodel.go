package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	chatmodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// anthropicChatModel implements chatmodel.ToolCallingChatModel using the
// Anthropic Messages API protocol (x-api-key auth, /v1/messages endpoint).
type anthropicChatModel struct {
	apiKey     string
	baseURL    string
	model      string
	tools      []*schema.ToolInfo
	maxTokens  int
	httpClient *http.Client
}

// AnthropicChatModelConfig holds configuration for creating an Anthropic chat model.
type AnthropicChatModelConfig struct {
	APIKey  string
	BaseURL string
	Model   string
	Timeout int // seconds, 0 uses default
}

// newAnthropicChatModel creates a new Anthropic chat model.
func newAnthropicChatModel(config AnthropicChatModelConfig) *anthropicChatModel {
	timeout := config.Timeout
	if timeout <= 0 {
		timeout = 60
	}
	return &anthropicChatModel{
		apiKey:     config.APIKey,
		baseURL:    strings.TrimRight(config.BaseURL, "/"),
		model:      config.Model,
		maxTokens:  4096,
		httpClient: &http.Client{},
	}
}

// WithTools returns a copy of the model with the specified tools bound.
func (m *anthropicChatModel) WithTools(tools []*schema.ToolInfo) (chatmodel.ToolCallingChatModel, error) {
	copied := *m
	copied.tools = tools
	return &copied, nil
}

// Generate sends a non-streaming request to the Anthropic Messages API.
func (m *anthropicChatModel) Generate(ctx context.Context, messages []*schema.Message, opts ...chatmodel.Option) (*schema.Message, error) {
	body, err := m.buildRequest(messages, false, opts)
	if err != nil {
		return nil, err
	}

	httpReq, err := m.newRequest(ctx, body)
	if err != nil {
		return nil, err
	}

	resp, err := m.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("anthropic generate: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("anthropic generate read: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anthropic generate: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return m.parseResponse(respBody)
}

// Stream sends a streaming request and returns a StreamReader of message chunks.
func (m *anthropicChatModel) Stream(ctx context.Context, messages []*schema.Message, opts ...chatmodel.Option) (*schema.StreamReader[*schema.Message], error) {
	body, err := m.buildRequest(messages, true, opts)
	if err != nil {
		return nil, err
	}

	httpReq, err := m.newRequest(ctx, body)
	if err != nil {
		return nil, err
	}

	resp, err := m.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("anthropic stream: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("anthropic stream: HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	sr, sw := schema.Pipe[*schema.Message](1)
	go m.processStream(resp.Body, sw)
	return sr, nil
}

// ── internal helpers ──────────────────────────────────────────────────────

func (m *anthropicChatModel) newRequest(ctx context.Context, body []byte) (*http.Request, error) {
	url := m.baseURL + "/v1/messages"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", m.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

type anthropicReq struct {
	Model     string            `json:"model"`
	MaxTokens int               `json:"max_tokens"`
	System    string            `json:"system,omitempty"`
	Messages  []anthropicMsg    `json:"messages"`
	Tools     []anthropicTool   `json:"tools,omitempty"`
	Stream    bool              `json:"stream"`
}

type anthropicMsg struct {
	Role    string           `json:"role"`
	Content []anthropicBlock `json:"content"`
}

type anthropicBlock struct {
	Type       string          `json:"type"`
	Text       string          `json:"text,omitempty"`
	ID         string          `json:"id,omitempty"`
	Name       string          `json:"name,omitempty"`
	Input      json.RawMessage `json:"input,omitempty"`
	ToolUseID  string          `json:"tool_use_id,omitempty"`
	Content    string          `json:"content,omitempty"`
}

type anthropicTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema"`
}

type anthropicResp struct {
	ID      string           `json:"id"`
	Type    string           `json:"type"`
	Role    string           `json:"role"`
	Content []anthropicBlock `json:"content"`
	Usage   struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type anthropicSSE struct {
	Type  string          `json:"type"`
	Delta json.RawMessage `json:"delta,omitempty"`
	Usage json.RawMessage `json:"usage,omitempty"`
}

func (m *anthropicChatModel) buildRequest(messages []*schema.Message, stream bool, opts []chatmodel.Option) ([]byte, error) {
	options := chatmodel.GetCommonOptions(&chatmodel.Options{}, opts...)
	tools := m.tools
	if len(options.Tools) > 0 {
		tools = options.Tools
	}

	var systemParts []string
	var msgs []anthropicMsg

	for _, msg := range messages {
		if msg.Role == schema.System {
			systemParts = append(systemParts, msg.Content)
			continue
		}
		converted := convertEinoMessage(msg)
		if converted != nil {
			msgs = append(msgs, *converted)
		}
	}

	// Merge consecutive same-role messages to avoid protocol errors
	msgs = mergeConsecutiveRoles(msgs)

	maxTokens := m.maxTokens
	if options.MaxTokens != nil && *options.MaxTokens > 0 {
		maxTokens = *options.MaxTokens
	}

	req := anthropicReq{
		Model:     m.model,
		MaxTokens: maxTokens,
		System:    strings.Join(systemParts, "\n"),
		Messages:  msgs,
		Stream:    stream,
	}

	for _, t := range tools {
		req.Tools = append(req.Tools, convertEinoTool(t))
	}

	return json.Marshal(req)
}

func convertEinoMessage(msg *schema.Message) *anthropicMsg {
	switch msg.Role {
	case schema.User:
		content := msg.Content
		if content == "" {
			content = " "
		}
		return &anthropicMsg{
			Role:    "user",
			Content: []anthropicBlock{{Type: "text", Text: content}},
		}
	case schema.Assistant:
		if len(msg.ToolCalls) > 0 {
			var blocks []anthropicBlock
			if msg.Content != "" {
				blocks = append(blocks, anthropicBlock{Type: "text", Text: msg.Content})
			}
			for _, tc := range msg.ToolCalls {
				input := json.RawMessage(tc.Function.Arguments)
				if input == nil {
					input = json.RawMessage("{}")
				}
				blocks = append(blocks, anthropicBlock{
					Type:  "tool_use",
					ID:    tc.ID,
					Name:  tc.Function.Name,
					Input: input,
				})
			}
			return &anthropicMsg{Role: "assistant", Content: blocks}
		}
		return &anthropicMsg{
			Role:    "assistant",
			Content: []anthropicBlock{{Type: "text", Text: msg.Content}},
		}
	case schema.Tool:
		return &anthropicMsg{
			Role: "user",
			Content: []anthropicBlock{{
				Type:      "tool_result",
				ToolUseID: msg.ToolCallID,
				Content:   msg.Content,
			}},
		}
	default:
		return nil
	}
}

func convertEinoTool(t *schema.ToolInfo) anthropicTool {
	schemaBytes := json.RawMessage(`{"type":"object","properties":{},"required":[]}`)
	if t.ParamsOneOf != nil {
		if js, err := t.ParamsOneOf.ToJSONSchema(); err == nil && js != nil {
			if b, err2 := json.Marshal(js); err2 == nil {
				schemaBytes = b
			}
		}
	}
	return anthropicTool{
		Name:        t.Name,
		Description: t.Desc,
		InputSchema: schemaBytes,
	}
}

// mergeConsecutiveRoles merges adjacent messages with the same role into one,
// combining their content blocks. Anthropic API requires alternating roles.
func mergeConsecutiveRoles(msgs []anthropicMsg) []anthropicMsg {
	if len(msgs) <= 1 {
		return msgs
	}
	result := make([]anthropicMsg, 0, len(msgs))
	result = append(result, msgs[0])
	for i := 1; i < len(msgs); i++ {
		last := &result[len(result)-1]
		if last.Role == msgs[i].Role {
			last.Content = append(last.Content, msgs[i].Content...)
		} else {
			result = append(result, msgs[i])
		}
	}
	return result
}

func (m *anthropicChatModel) parseResponse(body []byte) (*schema.Message, error) {
	var resp anthropicResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse anthropic response: %w", err)
	}
	return anthropicContentToMessage(resp.Content), nil
}

func anthropicContentToMessage(blocks []anthropicBlock) *schema.Message {
	var textParts []string
	var toolCalls []schema.ToolCall

	for _, b := range blocks {
		switch b.Type {
		case "text":
			textParts = append(textParts, b.Text)
		case "tool_use":
			input := string(b.Input)
			if input == "" {
				input = "{}"
			}
			toolCalls = append(toolCalls, schema.ToolCall{
				ID:   b.ID,
				Type: "function",
				Function: schema.FunctionCall{
					Name:      b.Name,
					Arguments: input,
				},
			})
		}
	}

	return &schema.Message{
		Role:      schema.Assistant,
		Content:   strings.Join(textParts, ""),
		ToolCalls: toolCalls,
	}
}

// ── SSE streaming ─────────────────────────────────────────────────────────

func (m *anthropicChatModel) processStream(body io.ReadCloser, sw *schema.StreamWriter[*schema.Message]) {
	defer body.Close()
	defer sw.Close()

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)

	var (
		mu         sync.Mutex
		blocks     = make(map[int]*streamBlock)
		hasContent bool
	)

	type eventData struct {
		Type  string          `json:"type"`
		Delta json.RawMessage `json:"delta,omitempty"`
		Usage json.RawMessage `json:"usage,omitempty"`
	}

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		var ev eventData
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			continue
		}

		switch ev.Type {
		case "content_block_delta":
			var d struct {
				Type  string `json:"type"`
				Text  string `json:"text"`
				Index int    `json:"index"`
			}
			if err := json.Unmarshal(ev.Delta, &d); err != nil {
				continue
			}
			mu.Lock()
			b, ok := blocks[d.Index]
			if !ok {
				b = &streamBlock{index: d.Index}
				blocks[d.Index] = b
			}
			if d.Type == "text_delta" {
				b.text.WriteString(d.Text)
				hasContent = true
			} else if d.Type == "input_json_delta" {
				b.input.WriteString(d.Text)
			}
			mu.Unlock()

			if d.Type == "text_delta" && d.Text != "" {
				sw.Send(&schema.Message{
					Role:    schema.Assistant,
					Content: d.Text,
				}, nil)
			}
		case "content_block_start":
			var d struct {
				Index        int             `json:"index"`
				ContentBlock json.RawMessage `json:"content_block"`
			}
			if err := json.Unmarshal(ev.Delta, &d); err != nil {
				continue
			}
			var cb struct {
				Type string `json:"type"`
				Name string `json:"name"`
				ID   string `json:"id"`
			}
			json.Unmarshal(d.ContentBlock, &cb)
			mu.Lock()
			blocks[d.Index] = &streamBlock{
				index:    d.Index,
				blockType: cb.Type,
				toolName: cb.Name,
				toolID:   cb.ID,
			}
			mu.Unlock()
		}
	}

	// If no delta was ever emitted, send a final message with assembled content/tool calls
	if !hasContent {
		mu.Lock()
		final := assembleStreamBlocks(blocks)
		mu.Unlock()
		if final.Content != "" || len(final.ToolCalls) > 0 {
			sw.Send(final, nil)
		}
	}
}

type streamBlock struct {
	index     int
	blockType string
	toolName  string
	toolID    string
	text      bytes.Buffer
	input     bytes.Buffer
}

func assembleStreamBlocks(blocks map[int]*streamBlock) *schema.Message {
	var textParts []string
	var toolCalls []schema.ToolCall
	for i := 0; ; i++ {
		b, ok := blocks[i]
		if !ok {
			break
		}
		if b.blockType == "tool_use" || b.input.Len() > 0 {
			toolCalls = append(toolCalls, schema.ToolCall{
				ID:   b.toolID,
				Type: "function",
				Function: schema.FunctionCall{
					Name:      b.toolName,
					Arguments: b.input.String(),
				},
			})
		}
		if b.text.Len() > 0 {
			textParts = append(textParts, b.text.String())
		}
	}
	return &schema.Message{
		Role:      schema.Assistant,
		Content:   strings.Join(textParts, ""),
		ToolCalls: toolCalls,
	}
}
