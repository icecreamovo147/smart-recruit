package service

import (
	"strings"
	"testing"
)

func TestGenerateAgentSkillMarkdownCanvasV1(t *testing.T) {
	flowJSON := `{
		"format":"canvas.v1",
		"nodes":[
			{"id":"instruction-2","type":"instruction","position":{"x":500,"y":80},"data":{"title":"Second instruction","content":"Then send the result."}},
			{"id":"trigger-1","type":"trigger","position":{"x":0,"y":0},"data":{"title":"Trigger","content":"Use when handling inbound candidates."}},
			{"id":"constraint-1","type":"constraint","position":{"x":750,"y":0},"data":{"title":"Constraint","content":"Do not expose private notes."}},
			{"id":"output-1","type":"output","position":{"x":1000,"y":0},"data":{"title":"Output","content":"Return a concise summary."}},
			{"id":"instruction-1","type":"instruction","position":{"x":250,"y":80},"data":{"title":"First instruction","content":"First review the candidate profile."}}
		],
		"edges":[
			{"id":"e1","source":"trigger-1","target":"instruction-1","label":"start"},
			{"id":"e2","source":"instruction-1","target":"instruction-2","label":"next"},
			{"id":"e3","source":"instruction-2","target":"constraint-1","label":"guard"},
			{"id":"e4","source":"constraint-1","target":"output-1","label":"finish"}
		],
		"viewport":{"x":0,"y":0,"zoom":1}
	}`

	skillMD, frontmatterJSON, bodyMarkdown, err := GenerateAgentSkillMarkdown("candidate-flow", "Candidate flow", flowJSON)
	if err != nil {
		t.Fatalf("GenerateAgentSkillMarkdown returned error: %v", err)
	}
	if !strings.Contains(skillMD, "name: candidate-flow") {
		t.Fatalf("skillMD missing frontmatter name:\n%s", skillMD)
	}
	if !strings.Contains(frontmatterJSON, `"name":"candidate-flow"`) {
		t.Fatalf("frontmatterJSON missing name: %s", frontmatterJSON)
	}
	first := strings.Index(bodyMarkdown, "- First review the candidate profile.")
	second := strings.Index(bodyMarkdown, "- Then send the result.")
	if first < 0 || second < 0 {
		t.Fatalf("bodyMarkdown missing instruction content:\n%s", bodyMarkdown)
	}
	if first > second {
		t.Fatalf("canvas edge order was not preserved:\n%s", bodyMarkdown)
	}
	for _, want := range []string{
		"## When to use",
		"## Workflow",
		"1. When to use: Use when handling inbound candidates.",
		"1. Instructions: First review the candidate profile.",
		"1. Instructions: Then send the result.",
		"## Instructions",
		"## Output format",
		"## Constraints",
	} {
		if !strings.Contains(bodyMarkdown, want) {
			t.Fatalf("bodyMarkdown missing section %q:\n%s", want, bodyMarkdown)
		}
	}
}

func TestGenerateAgentSkillMarkdownWorkflowBranches(t *testing.T) {
	flowJSON := `{
		"format":"canvas.v1",
		"nodes":[
			{"id":"trigger","type":"trigger","position":{"x":0,"y":0},"data":{"title":"Trigger","content":"Use when comparing candidates."}},
			{"id":"context","type":"context","position":{"x":1,"y":0},"data":{"title":"Context","content":"Read resume and job description."}},
			{"id":"condition","type":"condition","position":{"x":2,"y":0},"data":{"title":"Information check","content":"Check whether required evidence is available."}},
			{"id":"instruction","type":"instruction","position":{"x":3,"y":0},"data":{"title":"Compare evidence","content":"Compare evidence against the job requirements."}},
			{"id":"output","type":"output","position":{"x":4,"y":0},"data":{"title":"Output","content":"Return match summary and risks."}},
			{"id":"constraint","type":"constraint","position":{"x":5,"y":0},"data":{"title":"Constraint","content":"Do not invent missing evidence."}}
		],
		"edges":[
			{"id":"e1","source":"trigger","target":"context","label":"start"},
			{"id":"e2","source":"context","target":"condition","label":"next"},
			{"id":"e3","source":"condition","target":"instruction","label":"evidence is available"},
			{"id":"e4","source":"condition","target":"output","label":"evidence is missing"},
			{"id":"e5","source":"instruction","target":"output","label":"finish"},
			{"id":"e6","source":"output","target":"constraint","label":"guard"}
		]
	}`

	_, _, bodyMarkdown, err := GenerateAgentSkillMarkdown("branch-flow", "Branch flow", flowJSON)
	if err != nil {
		t.Fatalf("GenerateAgentSkillMarkdown returned error: %v", err)
	}
	for _, want := range []string{
		"## Workflow",
		`If evidence is available, continue to "Compare evidence".`,
		`If evidence is missing, continue to "Output".`,
		"1. Decision rules: Check whether required evidence is available.",
	} {
		if !strings.Contains(bodyMarkdown, want) {
			t.Fatalf("bodyMarkdown missing workflow content %q:\n%s", want, bodyMarkdown)
		}
	}
	workflow := bodyMarkdown[strings.Index(bodyMarkdown, "## Workflow"):]
	contextIndex := strings.Index(workflow, "1. Required context: Read resume and job description.")
	conditionIndex := strings.Index(workflow, "1. Decision rules: Check whether required evidence is available.")
	instructionIndex := strings.Index(workflow, "1. Instructions: Compare evidence against the job requirements.")
	if contextIndex < 0 || conditionIndex < 0 || instructionIndex < 0 {
		t.Fatalf("workflow missing expected ordered steps:\n%s", workflow)
	}
	if !(contextIndex < conditionIndex && conditionIndex < instructionIndex) {
		t.Fatalf("workflow did not follow edge order:\n%s", workflow)
	}
}

func TestParseAgentSkillFlowCanvasCanonicalKeepsCanvasShape(t *testing.T) {
	flowJSON := `{
		"format":"canvas.v1",
		"nodes":[
			{"id":"trigger","type":"trigger","position":{"x":10,"y":20},"data":{"title":"Trigger","content":"Use for candidate updates."}},
			{"id":"instruction","type":"instruction","position":{"x":240,"y":20},"data":{"title":"Instruction","content":"Review the update."}},
			{"id":"output","type":"output","position":{"x":470,"y":20},"data":{"title":"Output","content":"Return a summary."}},
			{"id":"constraint","type":"constraint","position":{"x":700,"y":20},"data":{"title":"Constraint","content":"Keep it factual."}}
		],
		"edges":[
			{"id":"e1","source":"trigger","target":"instruction","sourceHandle":"right","targetHandle":"left","curvature":0.25,"label":"next"},
			{"id":"e2","source":"instruction","target":"output","label":"next"},
			{"id":"e3","source":"output","target":"constraint","label":"guard"}
		]
	}`

	_, canonical, err := parseAgentSkillFlow(flowJSON)
	if err != nil {
		t.Fatalf("parseAgentSkillFlow returned error: %v", err)
	}
	for _, want := range []string{
		`"format":"canvas.v1"`,
		`"position":{"x":10,"y":20}`,
		`"edges":[`,
		`"source":"trigger"`,
		`"target":"instruction"`,
		`"sourceHandle":"right"`,
		`"targetHandle":"left"`,
		`"curvature":0.25`,
	} {
		if !strings.Contains(canonical, want) {
			t.Fatalf("canonical flow missing %q: %s", want, canonical)
		}
	}
}

func TestGenerateAgentSkillMarkdownLegacyNodesCompatible(t *testing.T) {
	flowJSON := `{
		"nodes":[
			{"id":"out","type":"output","title":"Output","content":"Return next steps.","order":40},
			{"id":"trigger","type":"trigger","title":"Trigger","content":"Use for interview scheduling.","order":10},
			{"id":"constraint","type":"constraint","title":"Constraint","content":"Confirm times before booking.","order":30},
			{"id":"instruction","type":"instruction","title":"Instruction","content":"Collect interviewer availability.","order":20}
		]
	}`

	_, _, bodyMarkdown, err := GenerateAgentSkillMarkdown("schedule-flow", "Schedule flow", flowJSON)
	if err != nil {
		t.Fatalf("GenerateAgentSkillMarkdown returned error: %v", err)
	}
	for _, want := range []string{
		"- Use for interview scheduling.",
		"- Collect interviewer availability.",
		"- Return next steps.",
		"- Confirm times before booking.",
	} {
		if !strings.Contains(bodyMarkdown, want) {
			t.Fatalf("bodyMarkdown missing %q:\n%s", want, bodyMarkdown)
		}
	}
}

func TestGenerateAgentSkillMarkdownMissingRequiredNode(t *testing.T) {
	flowJSON := `{
		"format":"canvas.v1",
		"nodes":[
			{"id":"trigger","type":"trigger","position":{"x":0,"y":0},"data":{"title":"Trigger","content":"Use for candidate updates."}},
			{"id":"instruction","type":"instruction","position":{"x":1,"y":0},"data":{"title":"Instruction","content":"Review the update."}},
			{"id":"output","type":"output","position":{"x":2,"y":0},"data":{"title":"Output","content":"Return a summary."}}
		],
		"edges":[
			{"id":"e1","source":"trigger","target":"instruction","label":""},
			{"id":"e2","source":"instruction","target":"output","label":""}
		]
	}`

	_, _, _, err := GenerateAgentSkillMarkdown("missing-constraint", "Missing constraint", flowJSON)
	if err == nil {
		t.Fatal("GenerateAgentSkillMarkdown returned nil error")
	}
	if !strings.Contains(err.Error(), "missing required node content for: constraint") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGenerateAgentSkillMarkdownRejectsReferencesInCanvas(t *testing.T) {
	flowJSON := `{
		"format":"canvas.v1",
		"nodes":[
			{"id":"trigger","type":"trigger","position":{"x":0,"y":0},"data":{"title":"Trigger","content":"Use for candidate updates."}},
			{"id":"instruction","type":"instruction","position":{"x":1,"y":0},"data":{"title":"Instruction","content":"Read references/private.md first."}},
			{"id":"output","type":"output","position":{"x":2,"y":0},"data":{"title":"Output","content":"Return a summary."}},
			{"id":"constraint","type":"constraint","position":{"x":3,"y":0},"data":{"title":"Constraint","content":"Keep it brief."}}
		],
		"edges":[
			{"id":"e1","source":"trigger","target":"instruction","label":""},
			{"id":"e2","source":"instruction","target":"output","label":""},
			{"id":"e3","source":"output","target":"constraint","label":""}
		]
	}`

	_, _, _, err := GenerateAgentSkillMarkdown("reject-references", "Reject references", flowJSON)
	if err == nil {
		t.Fatal("GenerateAgentSkillMarkdown returned nil error")
	}
	if !strings.Contains(err.Error(), "cannot reference references/") {
		t.Fatalf("unexpected error: %v", err)
	}
}
