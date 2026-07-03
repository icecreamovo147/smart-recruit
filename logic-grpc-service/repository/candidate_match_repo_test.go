package repository

import (
	"context"
	"testing"

	"logic-grpc-service/model"
)

func TestCandidateMatchRepoSaveEvaluationVersionCreatesLatestWithEvidence(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCandidateMatchRepo(db)
	ctx := context.Background()
	agentRunID := uint64(77)

	snapshot := &CandidateMatchSnapshot{
		Evaluation: model.CandidateMatchEvaluation{
			ApplicationID:      100,
			JobID:              200,
			CandidateUserID:    300,
			ResumeProfileID:    400,
			AgentRunID:         &agentRunID,
			OverallScore:       86.5,
			Recommendation:     "strong_match",
			Summary:            "Relevant Go backend experience.",
			ScoreBreakdownJSON: `{"skills":90}`,
			ModelName:          "test-model",
		},
		Evidence: []model.CandidateMatchEvidence{
			{EvidenceType: "skill", Dimension: "must_have", SourceTable: "resume_skills", Snippet: "Go", Weight: 0.6, ScoreImpact: 12},
			{EvidenceType: "experience", Dimension: "backend", SourceTable: "resume_experiences", Snippet: "APIs", Weight: 0.4, ScoreImpact: 8},
		},
	}

	if err := repo.SaveEvaluationVersion(ctx, snapshot); err != nil {
		t.Fatalf("SaveEvaluationVersion failed: %v", err)
	}
	if snapshot.Evaluation.ID == 0 {
		t.Fatal("expected evaluation ID to be populated")
	}
	if snapshot.Evaluation.EvaluationVersion != 1 || snapshot.Evaluation.IsLatest != 1 {
		t.Fatalf("expected version 1 latest evaluation, got version=%d latest=%d", snapshot.Evaluation.EvaluationVersion, snapshot.Evaluation.IsLatest)
	}
	if snapshot.Evaluation.EvaluatedAt.IsZero() {
		t.Fatal("expected evaluated_at to be set")
	}

	latest, err := repo.GetLatestByApplicationID(ctx, 100)
	if err != nil {
		t.Fatalf("GetLatestByApplicationID failed: %v", err)
	}
	if latest == nil || latest.ID != snapshot.Evaluation.ID {
		t.Fatalf("unexpected latest evaluation: %+v", latest)
	}

	loaded, err := repo.GetSnapshot(ctx, snapshot.Evaluation.ID)
	if err != nil {
		t.Fatalf("GetSnapshot failed: %v", err)
	}
	if len(loaded.Evidence) != 2 {
		t.Fatalf("expected 2 evidence rows, got %+v", loaded.Evidence)
	}
	for _, evidence := range loaded.Evidence {
		if evidence.MetadataJSON != "{}" {
			t.Fatalf("expected empty metadata to be normalized to {}, got %q", evidence.MetadataJSON)
		}
	}
}

func TestCandidateMatchRepoNormalizesEvidenceMetadataJSON(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "empty", raw: "", want: "{}"},
		{name: "blank", raw: "  ", want: "{}"},
		{name: "object", raw: `{"source":"test"}`, want: `{"source":"test"}`},
		{name: "invalid", raw: "plain text", want: `{"value":"plain text"}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeCandidateMatchEvidenceMetadata(tc.raw); got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestCandidateMatchRepoSaveEvaluationVersionBumpsLatestAndReplacesEvidence(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCandidateMatchRepo(db)
	ctx := context.Background()

	first := &CandidateMatchSnapshot{
		Evaluation: model.CandidateMatchEvaluation{
			ApplicationID:   101,
			JobID:           201,
			CandidateUserID: 301,
			ResumeProfileID: 401,
			OverallScore:    70,
			Recommendation:  "possible_match",
		},
		Evidence: []model.CandidateMatchEvidence{{EvidenceType: "skill", Snippet: "SQL"}},
	}
	if err := repo.SaveEvaluationVersion(ctx, first); err != nil {
		t.Fatalf("first SaveEvaluationVersion failed: %v", err)
	}

	second := &CandidateMatchSnapshot{
		Evaluation: model.CandidateMatchEvaluation{
			ApplicationID:   101,
			JobID:           201,
			CandidateUserID: 301,
			ResumeProfileID: 401,
			OverallScore:    91,
			Recommendation:  "strong_match",
		},
		Evidence: []model.CandidateMatchEvidence{{EvidenceType: "project", Snippet: "Search ranking"}},
	}
	if err := repo.SaveEvaluationVersion(ctx, second); err != nil {
		t.Fatalf("second SaveEvaluationVersion failed: %v", err)
	}
	if second.Evaluation.EvaluationVersion != 2 || second.Evaluation.IsLatest != 1 {
		t.Fatalf("expected second evaluation version 2 latest, got version=%d latest=%d", second.Evaluation.EvaluationVersion, second.Evaluation.IsLatest)
	}

	var old model.CandidateMatchEvaluation
	if err := db.First(&old, first.Evaluation.ID).Error; err != nil {
		t.Fatalf("load old evaluation failed: %v", err)
	}
	if old.IsLatest != 0 {
		t.Fatalf("expected old evaluation to be non-latest, got %d", old.IsLatest)
	}

	second.Evaluation.Summary = "Updated summary"
	second.Evidence = []model.CandidateMatchEvidence{
		{EvidenceType: "skill", Snippet: "Go"},
		{EvidenceType: "risk", Snippet: "No leadership evidence"},
	}
	if err := repo.SaveEvaluationVersion(ctx, second); err != nil {
		t.Fatalf("update SaveEvaluationVersion failed: %v", err)
	}
	if second.Evaluation.EvaluationVersion != 2 {
		t.Fatalf("existing evaluation should preserve version 2, got %d", second.Evaluation.EvaluationVersion)
	}

	loaded, err := repo.GetSnapshot(ctx, second.Evaluation.ID)
	if err != nil {
		t.Fatalf("GetSnapshot failed: %v", err)
	}
	if len(loaded.Evidence) != 2 || loaded.Evidence[0].Snippet != "Go" {
		t.Fatalf("expected replaced evidence, got %+v", loaded.Evidence)
	}
}
