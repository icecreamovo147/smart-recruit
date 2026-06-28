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
	Nodes []AgentSkillFlowNode `json:"nodes"`
}

type AgentSkillFlowNode struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Order   int32  `json:"order"`
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
	var flow AgentSkillFlow
	if err := json.Unmarshal([]byte(flowJSON), &flow); err != nil {
		return nil, "", fmt.Errorf("invalid flow_json: %w", err)
	}
	if len(flow.Nodes) == 0 {
		return nil, "", fmt.Errorf("flow_json.nodes is required")
	}
	for _, node := range flow.Nodes {
		if _, ok := agentSkillNodeSections[node.Type]; !ok {
			return nil, "", fmt.Errorf("unsupported flow node type %q", node.Type)
		}
		if strings.TrimSpace(node.Content) == "" && strings.TrimSpace(node.Title) == "" {
			return nil, "", fmt.Errorf("flow node %q must include content or title", node.ID)
		}
		if err := ValidateAgentSkillMarkdown(node.Content); err != nil {
			return nil, "", err
		}
	}
	canonical, _ := json.Marshal(flow)
	return &flow, string(canonical), nil
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
