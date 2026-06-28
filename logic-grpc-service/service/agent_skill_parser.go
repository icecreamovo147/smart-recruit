package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type AgentSkillFlow struct {
	Format  string               `json:"format,omitempty"`
	Version string               `json:"version,omitempty"`
	Type    string               `json:"type,omitempty"`
	Nodes   []AgentSkillFlowNode `json:"nodes"`
}

type AgentSkillFlowNode struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Order   int32  `json:"order"`
}

type agentSkillCanvasFlow struct {
	Format   string                     `json:"format"`
	Version  string                     `json:"version"`
	Type     string                     `json:"type"`
	Nodes    []agentSkillCanvasNode     `json:"nodes"`
	Edges    []agentSkillCanvasEdge     `json:"edges"`
	Viewport map[string]json.RawMessage `json:"viewport"`
}

type agentSkillCanvasNode struct {
	ID       string               `json:"id"`
	Type     string               `json:"type"`
	Position map[string]float64   `json:"position"`
	Data     agentSkillCanvasData `json:"data"`
}

type agentSkillCanvasData struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

type agentSkillCanvasEdge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
	Label  string `json:"label"`
}

type AgentSkillDocument struct {
	Frontmatter map[string]any
	Body        string
}

var (
	agentSkillNamePattern   = regexp.MustCompile(`^[a-z][a-z0-9_-]{1,127}$`)
	agentSkillVersionRegexp = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
	markdownLinkRegexp      = regexp.MustCompile(`!?\[[^\]]*\]\(([^)]+)\)`)
)

var agentSkillNodeSections = map[string]string{
	"trigger":     "When to use",
	"context":     "Required context",
	"instruction": "Instructions",
	"condition":   "Decision rules",
	"output":      "Output format",
	"constraint":  "Constraints",
}

var agentSkillSectionOrder = []string{"trigger", "context", "instruction", "condition", "output", "constraint"}

func GenerateAgentSkillMarkdown(name, description, flowJSON string) (skillMD, frontmatterJSON, bodyMarkdown string, err error) {
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	if !agentSkillNamePattern.MatchString(name) {
		return "", "", "", fmt.Errorf("invalid agent skill name")
	}
	if description == "" {
		return "", "", "", fmt.Errorf("description is required")
	}
	flow, canonicalFlow, err := parseAgentSkillFlow(flowJSON)
	if err != nil {
		return "", "", "", err
	}
	_ = canonicalFlow
	frontmatter := map[string]string{
		"name":        name,
		"description": description,
	}
	fmBytes, _ := json.Marshal(frontmatter)

	var body bytes.Buffer
	for _, nodeType := range agentSkillSectionOrder {
		items := nodesByType(flow.Nodes, nodeType)
		body.WriteString("## " + agentSkillNodeSections[nodeType] + "\n\n")
		if len(items) == 0 {
			body.WriteString("None.\n\n")
			continue
		}
		for _, item := range items {
			content := strings.TrimSpace(item.Content)
			if content == "" {
				content = strings.TrimSpace(item.Title)
			}
			if content == "" {
				continue
			}
			body.WriteString("- " + normalizeMarkdownListText(content) + "\n")
		}
		body.WriteString("\n")
	}

	bodyText := strings.TrimSpace(body.String()) + "\n"
	if err := ValidateAgentSkillMarkdown(bodyText); err != nil {
		return "", "", "", err
	}
	var md bytes.Buffer
	md.WriteString("---\n")
	md.WriteString("name: " + name + "\n")
	md.WriteString("description: " + yamlScalar(description) + "\n")
	md.WriteString("---\n\n")
	md.WriteString(bodyText)
	skillMD = md.String()
	if _, err := ParseAgentSkillMarkdown(skillMD); err != nil {
		return "", "", "", err
	}
	return skillMD, string(fmBytes), bodyText, nil
}

func ParseAgentSkillMarkdown(skillMD string) (*AgentSkillDocument, error) {
	text := strings.ReplaceAll(skillMD, "\r\n", "\n")
	if !strings.HasPrefix(text, "---\n") {
		return nil, fmt.Errorf("SKILL.md frontmatter is required")
	}
	end := strings.Index(text[4:], "\n---\n")
	if end < 0 {
		return nil, fmt.Errorf("SKILL.md frontmatter is not closed")
	}
	rawFM := text[4 : 4+end]
	body := strings.TrimSpace(text[4+end+5:]) + "\n"
	fm := map[string]any{}
	if err := yaml.Unmarshal([]byte(rawFM), &fm); err != nil {
		return nil, fmt.Errorf("invalid SKILL.md frontmatter: %w", err)
	}
	name := frontmatterString(fm, "name")
	description := frontmatterString(fm, "description")
	if name == "" || description == "" {
		return nil, fmt.Errorf("frontmatter name and description are required")
	}
	if !agentSkillNamePattern.MatchString(name) {
		return nil, fmt.Errorf("frontmatter name is invalid")
	}
	if err := ValidateAgentSkillMarkdown(body); err != nil {
		return nil, err
	}
	return &AgentSkillDocument{Frontmatter: fm, Body: body}, nil
}

func frontmatterString(fm map[string]any, key string) string {
	value, ok := fm[key]
	if !ok || value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func ValidateAgentSkillMarkdown(markdown string) error {
	lower := strings.ToLower(markdown)
	for _, blocked := range []string{"scripts/", "./scripts/", "../scripts/", "references/", "./references/", "../references/"} {
		if strings.Contains(lower, blocked) {
			return fmt.Errorf("agent SKILL.md cannot reference %s", strings.TrimPrefix(blocked, "./"))
		}
	}
	for _, match := range markdownLinkRegexp.FindAllStringSubmatch(markdown, -1) {
		target := strings.TrimSpace(match[1])
		if target == "" || strings.HasPrefix(target, "#") {
			continue
		}
		if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") || strings.HasPrefix(target, "mailto:") {
			continue
		}
		return fmt.Errorf("agent SKILL.md cannot contain relative reference %q", target)
	}
	return nil
}

func parseAgentSkillFlow(flowJSON string) (*AgentSkillFlow, string, error) {
	if strings.TrimSpace(flowJSON) == "" {
		return nil, "", fmt.Errorf("flow_json is required")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(flowJSON), &raw); err != nil {
		return nil, "", fmt.Errorf("invalid flow_json: %w", err)
	}
	var flow *AgentSkillFlow
	canonicalSource := any(nil)
	var err error
	if isCanvasV1Flow(raw) {
		flow, canonicalSource, err = parseCanvasAgentSkillFlow(flowJSON)
	} else {
		flow, err = parseLegacyAgentSkillFlow(flowJSON)
		canonicalSource = flow
	}
	if err != nil {
		return nil, "", err
	}
	if err := validateAgentSkillFlow(flow); err != nil {
		return nil, "", err
	}
	canonical, _ := json.Marshal(canonicalSource)
	return flow, string(canonical), nil
}

func parseLegacyAgentSkillFlow(flowJSON string) (*AgentSkillFlow, error) {
	var flow AgentSkillFlow
	if err := json.Unmarshal([]byte(flowJSON), &flow); err != nil {
		return nil, fmt.Errorf("invalid flow_json: %w", err)
	}
	return &flow, nil
}

func parseCanvasAgentSkillFlow(flowJSON string) (*AgentSkillFlow, *agentSkillCanvasFlow, error) {
	var canvas agentSkillCanvasFlow
	if err := json.Unmarshal([]byte(flowJSON), &canvas); err != nil {
		return nil, nil, fmt.Errorf("invalid flow_json: %w", err)
	}
	if len(canvas.Nodes) == 0 {
		return nil, nil, fmt.Errorf("flow_json.nodes is required")
	}
	canvas.Format = "canvas.v1"
	if strings.TrimSpace(canvas.Version) == "" {
		canvas.Version = "1.0.0"
	}
	if strings.TrimSpace(canvas.Type) == "" {
		canvas.Type = "agent-skill"
	}
	nodeOrder := canvasNodeOrder(canvas.Nodes, canvas.Edges)
	flow := &AgentSkillFlow{
		Format: "canvas.v1",
		Nodes:  make([]AgentSkillFlowNode, 0, len(canvas.Nodes)),
	}
	for _, node := range canvas.Nodes {
		order := nodeOrder[node.ID]
		flow.Nodes = append(flow.Nodes, AgentSkillFlowNode{
			ID:      node.ID,
			Type:    node.Type,
			Title:   node.Data.Title,
			Content: node.Data.Content,
			Order:   int32(order),
		})
	}
	return flow, &canvas, nil
}

func validateAgentSkillFlow(flow *AgentSkillFlow) error {
	if len(flow.Nodes) == 0 {
		return fmt.Errorf("flow_json.nodes is required")
	}
	requiredContent := map[string]bool{
		"trigger":     false,
		"instruction": false,
		"output":      false,
		"constraint":  false,
	}
	for _, node := range flow.Nodes {
		if _, ok := agentSkillNodeSections[node.Type]; !ok {
			return fmt.Errorf("unsupported flow node type %q", node.Type)
		}
		if strings.TrimSpace(node.Content) == "" && strings.TrimSpace(node.Title) == "" {
			return fmt.Errorf("flow node %q must include content or title", node.ID)
		}
		if err := ValidateAgentSkillMarkdown(node.Content); err != nil {
			return err
		}
		if _, ok := requiredContent[node.Type]; ok && (strings.TrimSpace(node.Content) != "" || strings.TrimSpace(node.Title) != "") {
			requiredContent[node.Type] = true
		}
	}
	missing := make([]string, 0)
	for _, nodeType := range []string{"trigger", "instruction", "output", "constraint"} {
		if !requiredContent[nodeType] {
			missing = append(missing, nodeType)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("flow_json missing required node content for: %s", strings.Join(missing, ", "))
	}
	return nil
}

func isCanvasV1Flow(raw map[string]json.RawMessage) bool {
	for _, key := range []string{"format", "version", "type"} {
		var value string
		if err := json.Unmarshal(raw[key], &value); err == nil && strings.EqualFold(strings.TrimSpace(value), "canvas.v1") {
			return true
		}
	}
	return false
}

func canvasNodeOrder(nodes []agentSkillCanvasNode, edges []agentSkillCanvasEdge) map[string]int {
	ids := make(map[string]bool, len(nodes))
	inDegree := make(map[string]int, len(nodes))
	adjacent := make(map[string][]string, len(nodes))
	for _, node := range nodes {
		ids[node.ID] = true
		inDegree[node.ID] = 0
	}
	sort.SliceStable(edges, func(i, j int) bool {
		if edges[i].Source == edges[j].Source {
			if edges[i].Target == edges[j].Target {
				return edges[i].ID < edges[j].ID
			}
			return edges[i].Target < edges[j].Target
		}
		return edges[i].Source < edges[j].Source
	})
	for _, edge := range edges {
		if !ids[edge.Source] || !ids[edge.Target] {
			continue
		}
		adjacent[edge.Source] = append(adjacent[edge.Source], edge.Target)
		inDegree[edge.Target]++
	}
	ready := make([]string, 0)
	for _, node := range nodes {
		if inDegree[node.ID] == 0 {
			ready = append(ready, node.ID)
		}
	}
	sort.Strings(ready)
	order := make(map[string]int, len(nodes))
	next := 0
	for len(ready) > 0 {
		id := ready[0]
		ready = ready[1:]
		if _, exists := order[id]; exists {
			continue
		}
		order[id] = next
		next++
		for _, target := range adjacent[id] {
			inDegree[target]--
			if inDegree[target] == 0 {
				ready = append(ready, target)
				sort.Strings(ready)
			}
		}
	}
	remaining := make([]string, 0)
	for _, node := range nodes {
		if _, exists := order[node.ID]; !exists {
			remaining = append(remaining, node.ID)
		}
	}
	sort.Strings(remaining)
	for _, id := range remaining {
		order[id] = next
		next++
	}
	return order
}

func nodesByType(nodes []AgentSkillFlowNode, nodeType string) []AgentSkillFlowNode {
	items := make([]AgentSkillFlowNode, 0)
	for _, node := range nodes {
		if node.Type == nodeType {
			items = append(items, node)
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Order == items[j].Order {
			return items[i].ID < items[j].ID
		}
		return items[i].Order < items[j].Order
	})
	return items
}

func normalizeMarkdownListText(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}
	return strings.Join(lines, "\n  ")
}

func yamlScalar(s string) string {
	if s == "" {
		return `""`
	}
	if strings.ContainsAny(s, ":\n#\"'") {
		b, _ := json.Marshal(s)
		return string(b)
	}
	return s
}
