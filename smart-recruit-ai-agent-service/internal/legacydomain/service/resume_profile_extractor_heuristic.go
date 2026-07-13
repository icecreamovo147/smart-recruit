package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"smart-recruit-platform-go/logger"
)

const heuristicParserVersion = "resume-profile-heuristic-v1"

type HeuristicResumeProfileExtractor struct{}

func NewHeuristicResumeProfileExtractor() *HeuristicResumeProfileExtractor {
	return &HeuristicResumeProfileExtractor{}
}

func (e *HeuristicResumeProfileExtractor) Extract(ctx context.Context, text string) (string, error) {
	result, err := e.ExtractWithMetadata(ctx, text)
	if err != nil {
		return "", err
	}
	return result.RawJSON, nil
}

func (e *HeuristicResumeProfileExtractor) ExtractWithMetadata(ctx context.Context, text string) (ResumeProfileExtractResult, error) {
	start := time.Now()
	lines := strings.Split(text, "\n")
	var fullName, email, phone string
	var skills []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		lower := strings.ToLower(trimmed)

		if email == "" && strings.Contains(trimmed, "@") && (strings.HasSuffix(lower, ".com") || strings.HasSuffix(lower, ".cn") || strings.HasSuffix(lower, ".org") || strings.HasSuffix(lower, ".net")) {
			fields := strings.Fields(trimmed)
			for _, f := range fields {
				f = strings.Trim(f, ",;()[]<>\"'")
				if strings.Contains(f, "@") {
					email = f
					break
				}
			}
		}

		if phone == "" {
			cleaned := strings.NewReplacer(" ", "", "-", "", "(", "", ")", "", "+", "").Replace(trimmed)
			digits := 0
			for _, ch := range cleaned {
				if ch >= '0' && ch <= '9' {
					digits++
				}
			}
			if digits >= 8 && digits <= 15 && strings.ContainsAny(trimmed, "0123456789") {
				fields := strings.Fields(trimmed)
				for _, f := range fields {
					f = strings.Trim(f, ",;()[]<>\"'")
					f = strings.NewReplacer(" ", "", "-", "", "(", "", ")", "").Replace(f)
					digitCount := 0
					for _, ch := range f {
						if ch >= '0' && ch <= '9' {
							digitCount++
						}
					}
					if digitCount >= 8 && digitCount <= 15 {
						phone = f
						break
					}
				}
			}
		}
	}

	if fullName == "" && len(lines) > 0 {
		first := strings.TrimSpace(lines[0])
		if first != "" && !strings.Contains(first, "@") && len(first) <= 50 {
			fullName = first
		}
	}

	skillKeywords := []string{"golang", "go", "python", "java", "javascript", "typescript", "rust", "c++", "c#", "sql", "kubernetes", "docker", "aws", "gcp", "azure", "redis", "kafka", "grpc", "rest", "react", "vue", "angular", "node.js", "nodejs", "postgresql", "mysql", "mongodb", "git", "ci/cd", "machine learning", "deep learning", "nlp"}
	seenSkill := map[string]bool{}
	for _, line := range lines {
		lower := strings.ToLower(strings.TrimSpace(line))
		for _, kw := range skillKeywords {
			if strings.Contains(lower, kw) && !seenSkill[kw] {
				skills = append(skills, kw)
				seenSkill[kw] = true
			}
		}
	}

	profile := extractedResumeProfile{
		FullName:    fullName,
		Email:       email,
		Phone:       phone,
		Educations:  []extractedResumeEducation{},
		Experiences: []extractedResumeExperience{},
		Projects:    []extractedResumeProject{},
		Skills:      []extractedResumeSkill{},
	}
	zero := 0.0
	profile.TotalExperienceYears = &zero
	for _, s := range skills {
		profile.Skills = append(profile.Skills, extractedResumeSkill{
			Name: s,
		})
	}

	raw, err := json.Marshal(profile)
	if err != nil {
		return ResumeProfileExtractResult{}, fmt.Errorf("heuristic extractor marshal: %w", err)
	}
	rawStr := string(raw)

	meta := ExtractorMetadata{
		ParserVersion: heuristicParserVersion,
		ExtractorType: "heuristic",
		FallbackUsed:  false,
		Duration:      time.Since(start),
	}

	logger.L().Info("heuristic resume profile extractor used",
		zap.String("parser_version", heuristicParserVersion),
		zap.Int("input_chars", len(text)),
	)
	return ResumeProfileExtractResult{RawJSON: rawStr, Metadata: meta}, nil
}
