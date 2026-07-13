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
	Edges   []AgentSkillFlowEdge `json:"edges,omitempty"`
}

type AgentSkillFlowNode struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Order   int32  `json:"order"`
}

type AgentSkillFlowEdge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
	Label  string `json:"label,omitempty"`
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
	ID           string   `json:"id"`
	Source       string   `json:"source"`
	Target       string   `json:"target"`
	SourceHandle string   `json:"sourceHandle,omitempty"`
	TargetHandle string   `json:"targetHandle,omitempty"`
	Curvature    *float64 `json:"curvature,omitempty"`
	Label        string   `json:"label,omitempty"`
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
		if nodeType == "trigger" {
			workflow := renderAgentSkillWorkflow(flow)
			if workflow != "" {
				body.WriteString(workflow)
			}
		}
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
		Edges:  make([]AgentSkillFlowEdge, 0, len(canvas.Edges)),
	}
	nodeIDs := make(map[string]bool, len(canvas.Nodes))
	for _, node := range canvas.Nodes {
		nodeIDs[node.ID] = true
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
	for _, edge := range canvas.Edges {
		if !nodeIDs[edge.Source] || !nodeIDs[edge.Target] {
			continue
		}
		flow.Edges = append(flow.Edges, AgentSkillFlowEdge{
			ID:     edge.ID,
			Source: edge.Source,
			Target: edge.Target,
			Label:  strings.TrimSpace(edge.Label),
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

func renderAgentSkillWorkflow(flow *AgentSkillFlow) string {
	if flow == nil || len(flow.Edges) == 0 {
		return ""
	}
	nodesByID := make(map[string]AgentSkillFlowNode, len(flow.Nodes))
	for _, node := range flow.Nodes {
		nodesByID[node.ID] = node
	}
	validEdges := make([]AgentSkillFlowEdge, 0, len(flow.Edges))
	inDegree := make(map[string]int, len(flow.Nodes))
	outgoing := make(map[string][]AgentSkillFlowEdge, len(flow.Nodes))
	for _, node := range flow.Nodes {
		inDegree[node.ID] = 0
	}
	for _, edge := range flow.Edges {
		if _, ok := nodesByID[edge.Source]; !ok {
			continue
		}
		if _, ok := nodesByID[edge.Target]; !ok {
			continue
		}
		validEdges = append(validEdges, edge)
		outgoing[edge.Source] = append(outgoing[edge.Source], edge)
		inDegree[edge.Target]++
	}
	if len(validEdges) == 0 {
		return ""
	}
	for source := range outgoing {
		sort.SliceStable(outgoing[source], func(i, j int) bool {
			left := nodesByID[outgoing[source][i].Target]
			right := nodesByID[outgoing[source][j].Target]
			if left.Order == right.Order {
				if outgoing[source][i].Label == outgoing[source][j].Label {
					return outgoing[source][i].ID < outgoing[source][j].ID
				}
				return outgoing[source][i].Label < outgoing[source][j].Label
			}
			return left.Order < right.Order
		})
	}

	orderedNodes := append([]AgentSkillFlowNode(nil), flow.Nodes...)
	sort.SliceStable(orderedNodes, func(i, j int) bool {
		if orderedNodes[i].Order == orderedNodes[j].Order {
			return orderedNodes[i].ID < orderedNodes[j].ID
		}
		return orderedNodes[i].Order < orderedNodes[j].Order
	})
	starts := workflowStartNodes(orderedNodes, inDegree)
	visited := map[string]bool{}
	visiting := map[string]bool{}
	steps := make([]string, 0, len(flow.Nodes))
	for _, start := range starts {
		appendWorkflowSteps(start.ID, nodesByID, outgoing, visited, visiting, &steps)
	}
	for _, node := range orderedNodes {
		if !visited[node.ID] {
			appendWorkflowSteps(node.ID, nodesByID, outgoing, visited, visiting, &steps)
		}
	}
	if len(steps) == 0 {
		return ""
	}

	var body bytes.Buffer
	body.WriteString("## Workflow\n\n")
	for _, step := range steps {
		body.WriteString(step)
		body.WriteString("\n")
	}
	body.WriteString("\n")
	return body.String()
}

func workflowStartNodes(nodes []AgentSkillFlowNode, inDegree map[string]int) []AgentSkillFlowNode {
	starts := make([]AgentSkillFlowNode, 0)
	for _, node := range nodes {
		if node.Type == "trigger" && inDegree[node.ID] == 0 {
			starts = append(starts, node)
		}
	}
	if len(starts) > 0 {
		return starts
	}
	for _, node := range nodes {
		if node.Type == "trigger" {
			starts = append(starts, node)
		}
	}
	if len(starts) > 0 {
		return starts
	}
	for _, node := range nodes {
		if inDegree[node.ID] == 0 {
			starts = append(starts, node)
		}
	}
	if len(starts) > 0 {
		return starts
	}
	return nodes
}

func appendWorkflowSteps(
	nodeID string,
	nodesByID map[string]AgentSkillFlowNode,
	outgoing map[string][]AgentSkillFlowEdge,
	visited map[string]bool,
	visiting map[string]bool,
	steps *[]string,
) {
	if visited[nodeID] || visiting[nodeID] {
		return
	}
	node, ok := nodesByID[nodeID]
	if !ok {
		return
	}
	visiting[nodeID] = true
	*steps = append(*steps, "1. "+workflowNodeText(node))
	if len(outgoing[nodeID]) > 1 {
		for _, edge := range outgoing[nodeID] {
			target, ok := nodesByID[edge.Target]
			if !ok {
				continue
			}
			label := strings.TrimSpace(edge.Label)
			if label == "" {
				label = "this path"
			}
			*steps = append(*steps, "   - If "+normalizeMarkdownListText(label)+", continue to \""+workflowNodeTitle(target)+"\".")
		}
	}
	for _, edge := range outgoing[nodeID] {
		appendWorkflowSteps(edge.Target, nodesByID, outgoing, visited, visiting, steps)
	}
	visiting[nodeID] = false
	visited[nodeID] = true
}

func workflowNodeText(node AgentSkillFlowNode) string {
	title := workflowNodeTitle(node)
	content := strings.TrimSpace(node.Content)
	if content == "" || content == strings.TrimSpace(node.Title) {
		return fmt.Sprintf("%s: %s.", agentSkillNodeSections[node.Type], title)
	}
	return fmt.Sprintf("%s: %s", agentSkillNodeSections[node.Type], normalizeMarkdownListText(content))
}

func workflowNodeTitle(node AgentSkillFlowNode) string {
	title := strings.TrimSpace(node.Title)
	if title != "" {
		return title
	}
	content := strings.TrimSpace(node.Content)
	if content != "" {
		return content
	}
	if section, ok := agentSkillNodeSections[node.Type]; ok {
		return section
	}
	return node.ID
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
