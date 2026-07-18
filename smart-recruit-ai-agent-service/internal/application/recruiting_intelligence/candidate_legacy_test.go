package recruiting_intelligence

import (
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"strings"
	"testing"
)

func TestScoreCandidateMatchLegacyRestoresFiveDeterministicDimensions(t *testing.T) {
	index := BuildEvidenceIndex(CandidateEvidenceSource{
		Skills:      []CandidateEvidenceSkill{{ID: 11, Name: "Go", Evidence: "distributed systems"}},
		Experiences: []CandidateEvidenceExperience{{ID: 12, Title: "Backend Engineer", Description: "Built distributed Go services"}},
		Educations:  []CandidateEvidenceEducation{{ID: 13, Degree: "本科"}},
	})
	input := LegacyCandidateMatchInput{
		JobID: 9, Job: JobRequirementSource{JobTitle: "Backend Engineer", Requirements: "Go, distributed systems, 本科"},
		Evidence: index, ProfileID: 7, ResumeID: 8, ResumeParsedText: "Go distributed systems", ApplicationEducation: "本科",
		ResumeText:       "Go distributed systems Candidate Backend Engineer Distributed services 本科 Candidate 本科 University Backend Go University 本科 Go distributed systems Backend Engineer Built distributed Go services Go  distributed systems",
		SkillNames:       []string{"Go"},
		CandidateProfile: &LegacyCandidateProfile{ID: 6, RealName: "Candidate", Phone: "13800000000", Education: "本科", School: "University", WorkExperience: "Backend", Skills: "Go", IsComplete: 1},
		FullName:         "Candidate", Headline: "Backend Engineer", Summary: "Distributed services", HighestDegree: "本科", ExperienceYears: 4,
	}
	first := ScoreCandidateMatchLegacy(input)
	second := ScoreCandidateMatchLegacy(input)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("legacy score is not deterministic: first=%+v second=%+v", first, second)
	}
	wantNames := []string{"skills", "requirements", "experience", "education", "profile"}
	wantWeights := []float64{.35, .25, .20, .10, .10}
	if len(first.Breakdown.Dimensions) != len(wantNames) {
		t.Fatalf("dimensions = %+v", first.Breakdown.Dimensions)
	}
	for index, dimension := range first.Breakdown.Dimensions {
		if dimension.Name != wantNames[index] || dimension.Weight != wantWeights[index] {
			t.Fatalf("dimension[%d] = %+v, want %s/%.2f", index, dimension, wantNames[index], wantWeights[index])
		}
	}
	if first.Breakdown.ScorerVersion != LegacyCandidateMatchScorerVersion || first.OverallScore <= 0 || len(first.Evidence) == 0 {
		t.Fatalf("legacy result = %+v, want scorer version, positive score, and real evidence", first)
	}
}

func TestScoreCandidateMatchLegacyMatchesDevGoldenFixture(t *testing.T) {
	index := BuildEvidenceIndex(CandidateEvidenceSource{
		Skills: []CandidateEvidenceSkill{{ID: 11, Name: "Go"}},
		Resume: &CandidateEvidenceResume{ID: 8, ParsedText: "Go"},
	})
	result := ScoreCandidateMatchLegacy(LegacyCandidateMatchInput{
		JobID: 9, Job: JobRequirementSource{JobTitle: "硕士 Go", Requirements: "Go"}, Evidence: index,
		ProfileID: 7, ResumeID: 8, ResumeParsedText: "Go", ApplicationEducation: "硕士",
		ResumeText: "Go Ada   硕士 Ada 硕士 University 5 years Go Go   ", SkillNames: []string{"Go"},
		CandidateProfile: &LegacyCandidateProfile{ID: 6, RealName: "Ada", Phone: "13800000000", Education: "硕士", School: "University", WorkExperience: "5 years", Skills: "Go", IsComplete: 1},
		FullName:         "Ada", HighestDegree: "硕士", ExperienceYears: 5,
	})
	if result.OverallScore != 100 || result.Recommendation != LegacyRecommendationStrongMatch || len(result.Risks) != 0 {
		t.Fatalf("dev golden result = score %.1f recommendation %q risks %+v, want 100/strong_match/no risks", result.OverallScore, result.Recommendation, result.Risks)
	}
	wantScores := []float64{100, 100, 100, 100, 100}
	for i, dimension := range result.Breakdown.Dimensions {
		if dimension.Score != wantScores[i] {
			t.Fatalf("dimension[%d] = %+v, want score %.1f", i, dimension, wantScores[i])
		}
	}
}

func TestScoreCandidateMatchLegacyMatchesDevCompleteInputGolden(t *testing.T) {
	resumeText := strings.Repeat("x", 241) + " docker " + strings.Repeat("record ", 100) + " kubernetes"
	input := LegacyCandidateMatchInput{
		Job:              JobRequirementSource{Requirements: "Docker Kubernetes"},
		ResumeParsedText: resumeText,
		ResumeText:       resumeText,
		SkillNames:       []string{"Go"},
		CandidateProfile: &LegacyCandidateProfile{RealName: "Candidate"},
		ExperienceYears:  3,
		HighestDegree:    "本科",
	}
	result := ScoreCandidateMatchLegacy(input)

	wantDimensions := []LegacyScoreDimension{
		{Name: "skills", Weight: .35, Score: 0, Matched: []string{}, Missing: []string{"docker", "kubernetes"}},
		{Name: "requirements", Weight: .25, Score: 100, Matched: []string{"docker", "kubernetes"}, Missing: []string{}},
		{Name: "experience", Weight: .20, Score: 93},
		{Name: "education", Weight: .10, Score: 80},
		{Name: "profile", Weight: .10, Score: 48},
	}
	if !reflect.DeepEqual(result.Breakdown.Dimensions, wantDimensions) {
		t.Fatalf("dimensions = %+v, want dev golden %+v", result.Breakdown.Dimensions, wantDimensions)
	}
	wantRisks := []AggregationSignal{
		{Code: "candidate_profile_incomplete", Message: "Candidate profile is incomplete."},
		{Code: "skills_gap", Message: "Missing skill evidence: docker, kubernetes"},
	}
	if result.OverallScore != 56.4 || result.Recommendation != LegacyRecommendationNeedsReview ||
		!reflect.DeepEqual(result.Risks, wantRisks) ||
		result.Summary != "Overall score 56.4 with recommendation needs_review. 2 risk signal(s) require review." {
		t.Fatalf("result = %+v, want exact dev overall/recommendation/risks/summary", result)
	}
	jobText := "    Docker Kubernetes"
	sum := sha256.Sum256([]byte(strings.Join([]string{jobText, resumeText, LegacyCandidateMatchScorerVersion}, "\n")))
	if wantHash := hex.EncodeToString(sum[:]); result.Breakdown.InputHash != wantHash {
		t.Fatalf("input hash = %q, want complete dev input hash %q", result.Breakdown.InputHash, wantHash)
	}
}

func TestScoreCandidateMatchLegacySkillEvidenceDoesNotBecomeSkillName(t *testing.T) {
	index := BuildEvidenceIndex(CandidateEvidenceSource{Skills: []CandidateEvidenceSkill{{ID: 11, Name: "Go", Evidence: "Docker"}}})
	result := ScoreCandidateMatchLegacy(LegacyCandidateMatchInput{
		Job: JobRequirementSource{Requirements: "Docker"}, Evidence: index,
		ResumeParsedText: "", ResumeText: "Go   Docker", SkillNames: []string{"Go"},
	})
	if skills := result.Breakdown.Dimensions[0]; skills.Score != 0 || !reflect.DeepEqual(skills.Missing, []string{"docker"}) {
		t.Fatalf("skills = %+v, want Docker missing because only ResumeSkill.Name is eligible", skills)
	}
	if requirements := result.Breakdown.Dimensions[1]; requirements.Score != 100 || !reflect.DeepEqual(requirements.Matched, []string{"docker"}) {
		t.Fatalf("requirements = %+v, want dev full resume text to retain skill evidence", requirements)
	}
}

func TestScoreCandidateMatchLegacyPreservesDevMissingInputSemantics(t *testing.T) {
	index := BuildEvidenceIndex(CandidateEvidenceSource{
		Educations: []CandidateEvidenceEducation{{ID: 13, Degree: "硕士"}},
	})
	result := ScoreCandidateMatchLegacy(LegacyCandidateMatchInput{
		JobID: 9, Job: JobRequirementSource{JobTitle: "硕士岗位"}, Evidence: index, ProfileID: 7,
		HighestDegree: "", ApplicationEducation: "", ExperienceYears: 99,
	})
	dimensions := result.Breakdown.Dimensions
	if dimensions[2].Score != 80 || dimensions[3].Score != 45 || dimensions[4].Score != 35 {
		t.Fatalf("dev missing-input dimensions = %+v, want experience=80 education=45 nil-profile=35", dimensions)
	}
	wantRiskCodes := []string{"candidate_profile_missing", "resume_text_missing"}
	if len(result.Risks) < len(wantRiskCodes) {
		t.Fatalf("risks = %+v, want at least %v", result.Risks, wantRiskCodes)
	}
	for i, code := range wantRiskCodes {
		if result.Risks[i].Code != code {
			t.Fatalf("risk[%d] = %+v, want code %q", i, result.Risks[i], code)
		}
	}
}
