package recruiting_intelligence

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

const validResumeProfileJSON = `{
  "full_name":"Ada Lovelace",
  "email":"ada@example.com",
  "phone":"",
  "location":"London",
  "headline":"Engineer",
  "summary":"Builds systems",
  "total_experience_years":5,
  "highest_degree":"BS",
  "educations":[{"school":"University","degree":"BS","major":"Math","start_date":"2015","end_date":"2019","description":"Study"}],
  "experiences":[{"company":"Engines","title":"Engineer","location":"London","start_date":"2020-01","end_date":"present","is_current":true,"description":"Services","achievements":[]}],
  "projects":[{"name":"Engine","role":"Builder","start_date":"2019","end_date":"2020","description":"Compute","technologies":[],"highlights":[]}],
  "skills":[{"name":"golang","category":"language","level":"senior","years":5,"evidence":"services"}]
}`

func TestResumeProfileExtractorUsesDatabaseSystemPromptAndNormalizes(t *testing.T) {
	provider := &resumeStructuredProvider{content: validResumeProfileJSON, model: "model-a"}
	runtime := resumeRuntime(provider, PromptDescriptor{ID: 17, Name: "resume-system", Version: 3, AgentType: AgentTypeResumeProfileExtractor, Role: PromptRoleSystem, Content: "database system prompt"}, DefaultRuntimePolicy())
	result, err := NewResumeProfileExtractor(runtime, DefaultRuntimePolicy()).Extract(context.Background(), ResumeSource{ResumeID: 8, UserID: 3, FileName: "resume.pdf", ParsedText: "Ada uses Go."})
	if err != nil {
		t.Fatalf("Extract error = %v", err)
	}
	if provider.system != "database system prompt" || !strings.Contains(provider.user, "Resume parsed text:\nAda uses Go.") {
		t.Fatalf("messages = system %q user %q", provider.system, provider.user)
	}
	if result.ParserVersion != ResumeLLMParserVersion || result.Prompt.ID != 17 || result.ModelName != "model-a" || result.FallbackUsed {
		t.Fatalf("metadata = %+v", result)
	}
	if len(result.Profile.Skills) != 1 || result.Profile.Skills[0].Name != "Go" || result.InputHash == "" {
		t.Fatalf("normalized result = %+v", result)
	}
}

func TestResumeProfileStrictDecodeRejectsInvalidJSONAndSchema(t *testing.T) {
	tests := []struct {
		name    string
		content string
		kind    ResumeExtractionErrorKind
	}{
		{name: "leading garbage without object", content: "prefix only", kind: ResumeExtractionJSON},
		{name: "not json object", content: `["Go"]`, kind: ResumeExtractionJSON},
		{name: "string skill", content: `{"full_name":"Ada","total_experience_years":1,"educations":[],"experiences":[],"projects":[],"skills":["Go"]}`, kind: ResumeExtractionSchema},
		{name: "invalid date", content: strings.Replace(validResumeProfileJSON, `"start_date":"2020-01"`, `"start_date":"2020-13"`, 1), kind: ResumeExtractionSchema},
		{name: "negative experience", content: strings.Replace(validResumeProfileJSON, `"total_experience_years":5`, `"total_experience_years":-1`, 1), kind: ResumeExtractionSchema},
		{name: "empty useful profile", content: `{"full_name":"","email":"","phone":"","location":"","headline":"x","summary":"x","total_experience_years":0,"highest_degree":"","educations":[],"experiences":[],"projects":[],"skills":[]}`, kind: ResumeExtractionSchema},
	}
	policy := NewRuntimePolicy(RuntimePolicyConfig{StructuredResumeParse: true, Fallbacks: false})
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			provider := &resumeStructuredProvider{content: tc.content}
			_, err := NewResumeProfileExtractor(resumeRuntime(provider, validResumePrompt(), policy), policy).Extract(context.Background(), ResumeSource{ParsedText: "Ada uses Go."})
			var extractionErr *ResumeExtractionError
			if !errors.As(err, &extractionErr) || extractionErr.Kind != tc.kind {
				t.Fatalf("error = %v, want kind %s", err, tc.kind)
			}
		})
	}
}

func TestResumeProfileCanonicalizeRepairsCommonLLMDrift(t *testing.T) {
	fenced := "```json\n" + `{
  "full_name":"Ada Lovelace",
  "email":"ada@example.com",
  "phone":"",
  "location":"London",
  "headline":"Engineer",
  "summary":"Builds systems",
  "total_experience_years":5,
  "highest_degree":"BS",
  "educations":[{"school":"University","degree":"BS","major":"Math","start_date":"2015.09","end_date":"2019.06","description":"Study"}],
  "experiences":[{"company":"Engines","title":"Engineer","location":"London","start_date":"2020.01","end_date":"present","is_current":true,"description":"Services","achievements":[]}],
  "projects":[{"name":"Engine","role":"Builder","start_date":"2019","end_date":"2020","description":"Compute","technologies":[],"highlights":[]}],
  "skills":[{"name":"golang","category":"language","level":"senior","years":5,"evidence":"services"}],
  "age":23,
  "gender":"女"
}` + "\n```"
	tests := []struct {
		name    string
		content string
	}{
		{name: "markdown fence unknown fields and dotted dates", content: fenced},
		{
			name:    "missing phone and null location",
			content: strings.Replace(strings.Replace(validResumeProfileJSON, `  "phone":"",`+"\n", "", 1), `"location":"London"`, `"location":null`, 1),
		},
		{
			name:    "email label prefix",
			content: strings.Replace(validResumeProfileJSON, `"email":"ada@example.com"`, `"email":"邮箱：ada@example.com"`, 1),
		},
	}
	policy := NewRuntimePolicy(RuntimePolicyConfig{StructuredResumeParse: true, Fallbacks: false})
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := NewResumeProfileExtractor(resumeRuntime(&resumeStructuredProvider{content: tc.content}, validResumePrompt(), policy), policy).Extract(context.Background(), ResumeSource{ParsedText: "Ada uses Go."})
			if err != nil {
				t.Fatalf("Extract error = %v", err)
			}
			if result.FallbackUsed || result.ParserVersion != ResumeLLMParserVersion {
				t.Fatalf("result = %+v", result)
			}
			if result.Profile.Email != "ada@example.com" {
				t.Fatalf("email = %q", result.Profile.Email)
			}
		})
	}
}

func TestResumeProfileFallbackMatrix(t *testing.T) {
	tests := []struct {
		name     string
		prompt   PromptDescriptor
		provider *resumeStructuredProvider
		kind     ResumeExtractionErrorKind
	}{
		{name: "prompt", prompt: PromptDescriptor{}, provider: &resumeStructuredProvider{content: validResumeProfileJSON}, kind: ResumeExtractionPrompt},
		{name: "provider", prompt: validResumePrompt(), provider: &resumeStructuredProvider{err: errors.New("provider unavailable")}, kind: ResumeExtractionProvider},
		{name: "empty", prompt: validResumePrompt(), provider: &resumeStructuredProvider{content: "  "}, kind: ResumeExtractionEmpty},
		{name: "json", prompt: validResumePrompt(), provider: &resumeStructuredProvider{content: "not-json"}, kind: ResumeExtractionJSON},
		{name: "schema", prompt: validResumePrompt(), provider: &resumeStructuredProvider{content: `{"full_name":"Ada","total_experience_years":1,"educations":[],"experiences":[],"projects":[],"skills":["Go"]}`}, kind: ResumeExtractionSchema},
		{name: "timeout", prompt: validResumePrompt(), provider: &resumeStructuredProvider{err: context.DeadlineExceeded}, kind: ResumeExtractionTimeout},
	}
	policy := NewRuntimePolicy(RuntimePolicyConfig{StructuredResumeParse: true, Fallbacks: true, ResumeParseTimeout: time.Second})
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := NewResumeProfileExtractor(resumeRuntime(tc.provider, tc.prompt, policy), policy).Extract(context.Background(), ResumeSource{ParsedText: "Ada Lovelace\nGo engineer"})
			if err != nil {
				t.Fatalf("Extract error = %v", err)
			}
			if !result.FallbackUsed || result.FallbackKind != tc.kind || result.ParserVersion != ResumeHeuristicParserVersion {
				t.Fatalf("fallback result = %+v", result)
			}
		})
	}
}

func TestResumeProfileFallbackDisabledReturnsFailure(t *testing.T) {
	policy := NewRuntimePolicy(RuntimePolicyConfig{StructuredResumeParse: true, Fallbacks: false})
	_, err := NewResumeProfileExtractor(resumeRuntime(&resumeStructuredProvider{content: `{"skills":["Go"]}`}, validResumePrompt(), policy), policy).Extract(context.Background(), ResumeSource{ParsedText: "Ada\nGo"})
	if err == nil {
		t.Fatal("Extract error = nil, want schema failure")
	}
}

func TestResumeProfileMissingProviderFallsBackButDisabledCapabilityDoesNot(t *testing.T) {
	fallbackPolicy := NewRuntimePolicy(RuntimePolicyConfig{StructuredResumeParse: true, Fallbacks: true})
	result, err := NewResumeProfileExtractor(nil, fallbackPolicy).Extract(context.Background(), ResumeSource{ParsedText: "Ada Lovelace\nGo engineer"})
	if err != nil || !result.FallbackUsed || result.FallbackKind != ResumeExtractionProvider {
		t.Fatalf("missing-provider result=%+v error=%v", result, err)
	}

	disabledPolicy := NewRuntimePolicy(RuntimePolicyConfig{StructuredResumeParse: false, Fallbacks: true})
	_, err = NewResumeProfileExtractor(nil, disabledPolicy).Extract(context.Background(), ResumeSource{ParsedText: "Ada Lovelace\nGo engineer"})
	var extractionErr *ResumeExtractionError
	if !errors.As(err, &extractionErr) || extractionErr.Kind != ResumeExtractionPolicy {
		t.Fatalf("disabled error = %v, want policy failure", err)
	}
}

func validResumePrompt() PromptDescriptor {
	return PromptDescriptor{ID: 1, Name: "resume", Version: 1, AgentType: AgentTypeResumeProfileExtractor, Role: PromptRoleSystem, Content: "system"}
}

func resumeRuntime(provider *resumeStructuredProvider, prompt PromptDescriptor, policy RuntimePolicy) *Runtime {
	return NewRuntime(NewPromptLoader(resumePromptStore{prompt: prompt}), provider, policy)
}

type resumePromptStore struct {
	prompt PromptDescriptor
}

func (s resumePromptStore) LoadActiveRecruitingPrompt(context.Context, string, string) (PromptDescriptor, error) {
	return s.prompt, nil
}

type resumeStructuredProvider struct {
	content string
	model   string
	err     error
	system  string
	user    string
}

func (p *resumeStructuredProvider) CompleteStructured(_ context.Context, systemPrompt, userPrompt string) (StructuredCompletionResult, error) {
	p.system = systemPrompt
	p.user = userPrompt
	if p.err != nil {
		return StructuredCompletionResult{}, p.err
	}
	return StructuredCompletionResult{Content: p.content, ModelName: p.model}, nil
}
