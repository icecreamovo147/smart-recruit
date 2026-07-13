package service

import (
	"math"
	"regexp"
	"strings"

	"smart-recruit-ai-agent-service/internal/legacydomain/model"
	"smart-recruit-ai-agent-service/internal/legacydomain/repository"
)

const (
	maxEvidenceSnippetLen = 240
	maxTotalSnippets      = 100
)

var (
	phoneRegex = regexp.MustCompile(`1[3-9]\d{9}`)
	emailRegex = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
)

type EvidenceUnit struct {
	SourceTable     string   `json:"source_table"`
	SourceID        uint64   `json:"source_id"`
	SourceType      string   `json:"source_type"`
	Snippet         string   `json:"snippet"`
	NormalizedTerms []string `json:"normalized_terms"`
	Metadata        string   `json:"metadata,omitempty"`
}

type EvidenceIndex struct {
	Units []EvidenceUnit `json:"units"`
}

func BuildEvidenceIndex(snapshot *repository.ResumeProfileSnapshot, candidateProfile *model.CandidateProfile, resume *model.Resume) *EvidenceIndex {
	index := &EvidenceIndex{}

	index.addSkillEvidence(snapshot.Skills)
	index.addExperienceEvidence(snapshot.Experiences)
	index.addProjectEvidence(snapshot.Projects)
	index.addEducationEvidence(snapshot.Educations)
	index.addResumeTextEvidence(resume)
	index.addCandidateProfileEvidence(candidateProfile)

	if len(index.Units) > maxTotalSnippets {
		index.Units = index.Units[:maxTotalSnippets]
	}

	return index
}

func (idx *EvidenceIndex) addSkillEvidence(skills []model.ResumeSkill) {
	for _, skill := range skills {
		name := strings.TrimSpace(skill.Name)
		if name == "" {
			continue
		}
		if isEvidenceSensitiveField(name) {
			continue
		}
		snippet := name
		if skill.Evidence != "" {
			snippet = name + ": " + skill.Evidence
		}
		if skill.Level != "" {
			snippet = snippet + " (" + skill.Level + ")"
		}
		id := skill.ID
		idx.Units = append(idx.Units, EvidenceUnit{
			SourceTable:     "resume_skills",
			SourceID:        id,
			SourceType:      "skill",
			Snippet:         truncateEvidenceSnippet(snippet),
			NormalizedTerms: normalizeEvidenceTerms(name),
		})
	}
}

func (idx *EvidenceIndex) addExperienceEvidence(experiences []model.ResumeExperience) {
	for _, exp := range experiences {
		terms := []string{exp.Title, exp.Company}
		parts := []string{exp.Title, exp.Company, exp.Description}
		if exp.AchievementsJSON != "" && exp.AchievementsJSON != "[]" && exp.AchievementsJSON != "null" {
			parts = append(parts, exp.AchievementsJSON)
		}
		snippet := strings.Join(parts, " | ")
		id := exp.ID
		idx.Units = append(idx.Units, EvidenceUnit{
			SourceTable:     "resume_experiences",
			SourceID:        id,
			SourceType:      "experience",
			Snippet:         truncateEvidenceSnippet(snippet),
			NormalizedTerms: normalizeEvidenceTerms(strings.Join(terms, " ")),
		})
	}
}

func (idx *EvidenceIndex) addProjectEvidence(projects []model.ResumeProject) {
	for _, proj := range projects {
		terms := []string{proj.Name, proj.Role}
		parts := []string{proj.Name, proj.Role, proj.Description}
		if proj.TechnologiesJSON != "" && proj.TechnologiesJSON != "[]" && proj.TechnologiesJSON != "null" {
			parts = append(parts, proj.TechnologiesJSON)
		}
		if proj.HighlightsJSON != "" && proj.HighlightsJSON != "[]" && proj.HighlightsJSON != "null" {
			parts = append(parts, proj.HighlightsJSON)
		}
		snippet := strings.Join(parts, " | ")
		id := proj.ID
		idx.Units = append(idx.Units, EvidenceUnit{
			SourceTable:     "resume_projects",
			SourceID:        id,
			SourceType:      "project",
			Snippet:         truncateEvidenceSnippet(snippet),
			NormalizedTerms: normalizeEvidenceTerms(strings.Join(terms, " ")),
		})
	}
}

func (idx *EvidenceIndex) addEducationEvidence(educations []model.ResumeEducation) {
	for _, edu := range educations {
		terms := []string{edu.School, edu.Degree, edu.Major}
		parts := []string{edu.School, edu.Degree, edu.Major}
		if edu.Description != "" {
			parts = append(parts, edu.Description)
		}
		snippet := strings.Join(parts, " | ")
		id := edu.ID
		idx.Units = append(idx.Units, EvidenceUnit{
			SourceTable:     "resume_educations",
			SourceID:        id,
			SourceType:      "education",
			Snippet:         truncateEvidenceSnippet(snippet),
			NormalizedTerms: normalizeEvidenceTerms(strings.Join(terms, " ")),
		})
	}
}

func (idx *EvidenceIndex) addResumeTextEvidence(resume *model.Resume) {
	if resume == nil {
		return
	}
	text := strings.TrimSpace(resume.ParsedText)
	if text == "" {
		return
	}
	cleaned := stripSensitiveFields(text)
	snippet := truncateEvidenceSnippet(cleaned)
	id := uint64(resume.ID)
	idx.Units = append(idx.Units, EvidenceUnit{
		SourceTable:     "resumes",
		SourceID:        id,
		SourceType:      "resume_text",
		Snippet:         snippet,
		NormalizedTerms: normalizeEvidenceTerms(cleaned),
	})
}

func (idx *EvidenceIndex) addCandidateProfileEvidence(profile *model.CandidateProfile) {
	if profile == nil {
		return
	}
	parts := make([]string, 0, 6)
	terms := make([]string, 0, 6)
	if profile.Education != "" {
		parts = append(parts, "教育: "+profile.Education)
		terms = append(terms, profile.Education)
	}
	if profile.School != "" {
		parts = append(parts, "学校: "+profile.School)
		terms = append(terms, profile.School)
	}
	if profile.WorkExperience != "" {
		parts = append(parts, "工作: "+profile.WorkExperience)
		terms = append(terms, profile.WorkExperience)
	}
	if profile.Skills != "" {
		parts = append(parts, "技能: "+profile.Skills)
		terms = append(terms, profile.Skills)
	}
	if len(parts) == 0 {
		return
	}
	snippet := strings.Join(parts, " | ")
	id := uint64(profile.ID)
	idx.Units = append(idx.Units, EvidenceUnit{
		SourceTable:     "candidate_profiles",
		SourceID:        id,
		SourceType:      "candidate_profile",
		Snippet:         truncateEvidenceSnippet(snippet),
		NormalizedTerms: normalizeEvidenceTerms(strings.Join(terms, " ")),
	})
}

func normalizeEvidenceTerms(text string) []string {
	terms := uniqueSortedTokens(text)
	if len(terms) > 20 {
		terms = terms[:20]
	}
	for _, word := range chineseWordExtract(text) {
		found := false
		for _, t := range terms {
			if t == word {
				found = true
				break
			}
		}
		if !found {
			terms = append(terms, word)
		}
	}
	return terms
}

func chineseWordExtract(text string) []string {
	var words []string
	seen := make(map[string]bool)
	runes := []rune(text)
	i := 0
	for i < len(runes) {
		if runes[i] > 0x4E00 && runes[i] < 0x9FFF {
			word := string(runes[i])
			if !seen[word] && !candidateMatchStopwords[word] {
				seen[word] = true
				words = append(words, word)
			}
		}
		i++
	}
	return words
}

func truncateEvidenceSnippet(text string) string {
	text = strings.Join(strings.Fields(text), " ")
	runes := []rune(text)
	if len(runes) <= maxEvidenceSnippetLen {
		return text
	}
	return string(runes[:maxEvidenceSnippetLen]) + "..."
}

func stripSensitiveFields(text string) string {
	text = phoneRegex.ReplaceAllString(text, "[PHONE]")
	text = emailRegex.ReplaceAllString(text, "[EMAIL]")
	return text
}

func isEvidenceSensitiveField(value string) bool {
	lower := strings.ToLower(value)
	if phoneRegex.MatchString(lower) || emailRegex.MatchString(lower) {
		return true
	}
	return false
}

func computeCoverageScore(matched, total int) float64 {
	if total == 0 {
		return 70
	}
	return math.Round(float64(matched)/float64(total)*1000) / 10
}

func findEvidenceByTerm(terms []string, index *EvidenceIndex) []MatchEvidenceUnit {
	var matched []MatchEvidenceUnit
	seen := make(map[string]bool)

	for _, unit := range index.Units {
		for _, term := range terms {
			key := unit.SourceTable + ":" + unit.SourceType + ":" + term
			if seen[key] {
				continue
			}
			if term == "" {
				continue
			}

			lowerTerm := strings.ToLower(term)
			if containsTerm(unit.NormalizedTerms, lowerTerm) {
				seen[key] = true
				matched = append(matched, MatchEvidenceUnit{
					SourceTable: unit.SourceTable,
					SourceID:    unit.SourceID,
					Snippet:     unit.Snippet,
					Reason:      "匹配到关键词: " + term,
				})
			}
		}
	}

	if len(matched) > 10 {
		matched = matched[:10]
	}
	return matched
}

func containsTerm(terms []string, target string) bool {
	for _, t := range terms {
		if strings.EqualFold(t, target) {
			return true
		}
	}
	return false
}
