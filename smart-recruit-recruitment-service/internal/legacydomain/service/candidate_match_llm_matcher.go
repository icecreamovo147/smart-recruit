package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
	"go.uber.org/zap"

	"smart-recruit-platform-go/logger"
	"smart-recruit-recruitment-service/internal/legacydomain/ai"
	"smart-recruit-recruitment-service/internal/legacydomain/repository"
)

const (
	llmMatcherVersion       = "candidate-match-llm-v1"
	llmMatcherPromptAgent   = "candidate_match_evaluator"
	llmMatcherPromptRole    = "system"
	llmMatcherSaveReserve   = 7 * time.Second
	llmMatcherMinCallWindow = 1 * time.Second
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

	systemPrompt, promptKey, promptVersion, err := m.loadPrompt(ctx)
	if err != nil {
		return RequirementMatchResult{}, err
	}

	evidenceText := buildEvidenceTextForLLM(index, req)
	candidateInfo := buildCandidateInfoForLLM(snapshot)

	userMsg := fmt.Sprintf(`岗位要求：
ID：%s
名称：%s
说明：%s
优先级：%s
类别：%s
别名：%s

候选人证据：
%s

候选人画像：
%s`,
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

func (m *LLMRequirementMatcher) loadPrompt(ctx context.Context) (prompt string, key string, version int32, err error) {
	if m.promptRepo == nil {
		return "", "", 0, fmt.Errorf("llm matcher: DB prompt repo is not configured for agent_type=%s role=%s", llmMatcherPromptAgent, llmMatcherPromptRole)
	}
	tmpl, err := m.promptRepo.GetActiveByAgentType(ctx, llmMatcherPromptAgent, llmMatcherPromptRole)
	if err != nil {
		return "", "", 0, fmt.Errorf("llm matcher: load DB prompt template for agent_type=%s role=%s: %w", llmMatcherPromptAgent, llmMatcherPromptRole, err)
	}
	if strings.TrimSpace(tmpl.Content) == "" {
		return "", "", 0, fmt.Errorf("llm matcher: active DB prompt template is empty for agent_type=%s role=%s", llmMatcherPromptAgent, llmMatcherPromptRole)
	}
	return tmpl.Content, tmpl.Name, tmpl.Version, nil
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
