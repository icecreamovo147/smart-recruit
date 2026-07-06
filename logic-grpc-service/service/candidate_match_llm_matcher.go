package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"logic-grpc-service/ai"
	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/repository"
)

const (
	llmMatcherVersion       = "candidate-match-llm-v1"
	llmMatcherPromptAgent   = "candidate_match_evaluator"
	llmMatcherPromptRole    = "system"
	llmMatcherSaveReserve   = 7 * time.Second
	llmMatcherMinCallWindow = 1 * time.Second
	llmMatcherDefaultPrompt = `You are a candidate-job requirement matching evaluator. Given a job requirement and candidate evidence, determine how well the candidate satisfies the requirement.

You MUST output ONLY a JSON object — no explanation, no markdown, no code fences, no extra text.
The JSON object MUST conform to this exact schema:
{
  "status": "strong_match|match|partial_match|weak_evidence|missing|conflict",
  "score": 0-100,
  "confidence": 0.0-1.0,
  "risk": "string or empty string",
  "evidence": [
    {
      "source_table": "resume_skills|resume_experiences|resume_projects|resume_educations|resumes|candidate_profiles",
      "source_id": 0,
      "snippet": "the matching evidence text",
      "reason": "explain why this evidence supports or contradicts the requirement"
    }
  ]
}

Rules:
- strong_match: candidate clearly satisfies this requirement with strong evidence
- match: candidate satisfies this requirement
- partial_match: candidate partially satisfies or has related experience
- weak_evidence: some evidence exists but is insufficient for a confident match
- missing: no evidence found; candidate does not satisfy this requirement
- conflict: evidence suggests the candidate contradicts this requirement
- score 0-100: higher = better match
- confidence 0-1: how confident you are in this assessment
- risk: describe what is missing or concerning (empty string if strong_match/match)
- evidence: list ALL relevant evidence snippets that support your assessment
- Each evidence must include the source_table and source_id from the candidate evidence marker (e.g., [resume_skills#3] → source_table="resume_skills", source_id=3)
- Use source_id from the evidence marker [table#id]; use 0 if no id is available
- snippet: the actual matching text from the candidate evidence
- reason: explain why this evidence supports or contradicts the requirement
- For "missing" status, evidence can be empty
- For "match" or "strong_match", at least one evidence entry is REQUIRED
- Be objective: do not overstate weak evidence as a strong match
- If the candidate's evidence is tangentially related but not a direct match, use "partial_match"
- Output ONLY valid JSON, no other text.`
)

type LLMRequirementMatcher struct {
	llmConfigSvc *LlmConfigService
	promptRepo   *repository.PromptTemplateRepo
}

func NewLLMRequirementMatcher(llmConfigSvc *LlmConfigService, promptRepo *repository.PromptTemplateRepo) *LLMRequirementMatcher {
	return &LLMRequirementMatcher{
		llmConfigSvc: llmConfigSvc,
		promptRepo:   promptRepo,
	}
}

func (m *LLMRequirementMatcher) Match(ctx context.Context, req JobRequirementItem, index *EvidenceIndex, snapshot *repository.ResumeProfileSnapshot) (RequirementMatchResult, error) {
	if m == nil || m.llmConfigSvc == nil {
		return RequirementMatchResult{}, fmt.Errorf("llm matcher not configured")
	}

	start := time.Now()
	cfg, err := m.llmConfigSvc.GetDefaultModelRuntimeConfig(ctx)
	if err != nil {
		return RequirementMatchResult{}, fmt.Errorf("llm matcher: get model config: %w", err)
	}
	if cfg.APIKey == "" || cfg.ModelName == "" {
		return RequirementMatchResult{}, fmt.Errorf("llm matcher: model not fully configured")
	}

	systemPrompt, promptKey, promptVersion := m.loadPrompt(ctx)

	evidenceText := buildEvidenceTextForLLM(index, req)
	candidateInfo := buildCandidateInfoForLLM(snapshot)

	userMsg := fmt.Sprintf(`Requirement:
ID: %s
Label: %s
Description: %s
Priority: %s
Category: %s
Aliases: %s

Candidate Evidence:
%s

Candidate Profile:
%s

Evaluate the match between this requirement and the candidate evidence.
Output ONLY valid JSON matching the required schema.`,
		req.ID, req.Label, req.Description, req.Priority, req.Category,
		strings.Join(req.Aliases, ", "),
		evidenceText, candidateInfo)

	msgs := []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage(userMsg),
	}

	timeout := m.resolveTimeout(ctx, cfg)
	logger.L().Info("llm requirement matcher request started",
		zap.String("requirement_id", req.ID),
		zap.String("model_name", cfg.ModelName),
		zap.Duration("timeout", timeout),
		zap.Int("input_chars", len(userMsg)+len(systemPrompt)),
		zap.String("prompt_key", promptKey),
		zap.Int32("prompt_version", promptVersion),
	)

	cm, err := ai.NewChatModelWithParams(ctx, cfg.ProviderType, cfg.APIKey, cfg.ModelName, cfg.BaseURL, timeout, cfg.Params)
	if err != nil {
		return RequirementMatchResult{}, fmt.Errorf("llm matcher: create chat model: %w", err)
	}

	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resp, err := cm.Generate(callCtx, msgs)
	if err != nil {
		logger.L().Warn("llm requirement matcher request failed",
			zap.String("requirement_id", req.ID),
			zap.String("model_name", cfg.ModelName),
			zap.Duration("duration", time.Since(start)),
			zap.Error(err),
		)
		return RequirementMatchResult{}, fmt.Errorf("llm matcher: generate: %w", err)
	}

	content := strings.TrimSpace(resp.Content)
	if content == "" {
		return RequirementMatchResult{}, fmt.Errorf("llm matcher: empty response")
	}

	cleaned := cleanLLMJSONOutput(content)
	if cleaned == "" {
		return RequirementMatchResult{}, fmt.Errorf("llm matcher: no valid JSON found")
	}

	var matchResult RequirementMatchResult
	if err := json.Unmarshal([]byte(cleaned), &matchResult); err != nil {
		return RequirementMatchResult{}, fmt.Errorf("llm matcher: invalid JSON: %w", err)
	}

	matchResult.RequirementID = req.ID

	if err := matchResult.Validate(); err != nil {
		logger.L().Warn("llm matcher: output failed validation, degrading to weak_evidence",
			zap.String("requirement_id", req.ID),
			zap.Error(err),
		)
		matchResult.Status = MatchStatusWeakEvidence
		matchResult.Score = 30
		matchResult.Confidence = 0.3
		matchResult.Evidence = nil
		matchResult.Risk = "LLM 输出不合法，已降级"
	}

	duration := time.Since(start)
	logger.L().Info("llm requirement matcher succeeded",
		zap.String("requirement_id", req.ID),
		zap.String("status", matchResult.Status),
		zap.Float64("score", matchResult.Score),
		zap.Duration("duration", duration),
	)

	return matchResult, nil
}

func (m *LLMRequirementMatcher) loadPrompt(ctx context.Context) (prompt string, key string, version int32) {
	if m.promptRepo == nil {
		return llmMatcherDefaultPrompt, "builtin", 0
	}
	tmpl, err := m.promptRepo.GetActiveByAgentType(ctx, llmMatcherPromptAgent, llmMatcherPromptRole)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return llmMatcherDefaultPrompt, "builtin", 0
		}
		logger.L().Warn("llm matcher: failed to load prompt from DB, using built-in", zap.Error(err))
		return llmMatcherDefaultPrompt, "builtin", 0
	}
	return tmpl.Content, tmpl.Name, tmpl.Version
}

func (m *LLMRequirementMatcher) resolveTimeout(ctx context.Context, cfg *LlmRuntimeModelConfig) time.Duration {
	timeout := 60 * time.Second
	if cfg.Timeout > 0 {
		timeout = cfg.Timeout
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		return timeout
	}
	remaining := time.Until(deadline)
	if remaining <= llmMatcherSaveReserve {
		return llmMatcherMinCallWindow
	}
	available := remaining - llmMatcherSaveReserve
	if available < llmMatcherMinCallWindow {
		return llmMatcherMinCallWindow
	}
	if available < timeout {
		return available
	}
	return timeout
}

func buildEvidenceTextForLLM(index *EvidenceIndex, req JobRequirementItem) string {
	if index == nil || len(index.Units) == 0 {
		return "No candidate evidence available."
	}

	aliases := GenerateRequirementAliases(&req)
	aliasSet := make(map[string]bool)
	for _, a := range aliases {
		aliasSet[strings.ToLower(a)] = true
	}

	var relevant []string
	for _, unit := range index.Units {
		matched := false
		for _, term := range unit.NormalizedTerms {
			if aliasSet[strings.ToLower(term)] {
				matched = true
				break
			}
		}
		if !matched {
			for alias := range aliasSet {
				if strings.Contains(strings.ToLower(unit.Snippet), alias) {
					matched = true
					break
				}
			}
		}
		if matched {
			relevant = append(relevant, fmt.Sprintf("[%s#%d] %s", unit.SourceTable, unit.SourceID, unit.Snippet))
		}
	}

	if len(relevant) == 0 {
		for _, unit := range index.Units {
			relevant = append(relevant, fmt.Sprintf("[%s#%d] %s", unit.SourceTable, unit.SourceID, unit.Snippet))
		}
		if len(relevant) > 5 {
			relevant = relevant[:5]
		}
	}

	if len(relevant) > 10 {
		relevant = relevant[:10]
	}

	if len(relevant) == 0 {
		return "No candidate evidence available."
	}
	return strings.Join(relevant, "\n")
}

func buildCandidateInfoForLLM(snapshot *repository.ResumeProfileSnapshot) string {
	if snapshot == nil {
		return "No profile data available."
	}
	var parts []string
	parts = append(parts, fmt.Sprintf("Total Experience: %.1f years", snapshot.Profile.TotalExperience))
	parts = append(parts, fmt.Sprintf("Highest Degree: %s", snapshot.Profile.HighestDegree))

	if len(snapshot.Skills) > 0 {
		var skills []string
		for _, s := range snapshot.Skills {
			skills = append(skills, s.Name)
		}
		parts = append(parts, "Skills: "+strings.Join(skills, ", "))
	}
	if len(snapshot.Experiences) > 0 {
		var exps []string
		for _, e := range snapshot.Experiences {
			exps = append(exps, fmt.Sprintf("%s at %s", e.Title, e.Company))
		}
		parts = append(parts, "Experience: "+strings.Join(exps, "; "))
	}
	if len(snapshot.Projects) > 0 {
		var projs []string
		for _, p := range snapshot.Projects {
			projs = append(projs, fmt.Sprintf("%s (%s)", p.Name, p.Role))
		}
		parts = append(parts, "Projects: "+strings.Join(projs, "; "))
	}

	return strings.Join(parts, "\n")
}
