package grpc_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"smart-recruit-ai-agent-service/internal/infrastructure/persistence"
	aiagentgrpc "smart-recruit-ai-agent-service/internal/interfaces/grpc"
)

type candidateEvaluationTestRow struct {
	ID                 uint64          `gorm:"primaryKey"`
	ApplicationID      int64           `gorm:"column:application_id"`
	JobID              int64           `gorm:"column:job_id"`
	CandidateUserID    int64           `gorm:"column:candidate_user_id"`
	ResumeProfileID    uint64          `gorm:"column:resume_profile_id"`
	AgentRunID         *uint64         `gorm:"column:agent_run_id"`
	EvaluationVersion  int32           `gorm:"column:evaluation_version"`
	IsLatest           int32           `gorm:"column:is_latest"`
	OverallScore       sql.NullFloat64 `gorm:"column:overall_score"`
	Recommendation     sql.NullString  `gorm:"column:recommendation"`
	Summary            sql.NullString  `gorm:"column:summary"`
	StrengthsJSON      sql.NullString  `gorm:"column:strengths_json"`
	RisksJSON          sql.NullString  `gorm:"column:risks_json"`
	ScoreBreakdownJSON sql.NullString  `gorm:"column:score_breakdown_json"`
	ModelName          sql.NullString  `gorm:"column:model_name"`
	EvaluatedAt        time.Time       `gorm:"column:evaluated_at"`
	CreatedAt          time.Time       `gorm:"column:created_at"`
	UpdatedAt          time.Time       `gorm:"column:updated_at"`
}

func (candidateEvaluationTestRow) TableName() string { return "candidate_match_evaluations" }

type candidateEvidenceTestRow struct {
	ID           uint64          `gorm:"primaryKey"`
	EvaluationID uint64          `gorm:"column:evaluation_id"`
	EvidenceType string          `gorm:"column:evidence_type"`
	Dimension    sql.NullString  `gorm:"column:dimension"`
	SourceTable  sql.NullString  `gorm:"column:source_table"`
	SourceID     *uint64         `gorm:"column:source_id"`
	Snippet      sql.NullString  `gorm:"column:snippet"`
	Weight       sql.NullFloat64 `gorm:"column:weight"`
	ScoreImpact  sql.NullFloat64 `gorm:"column:score_impact"`
	MetadataJSON sql.NullString  `gorm:"column:metadata_json"`
	CreatedAt    time.Time       `gorm:"column:created_at"`
}

func (candidateEvidenceTestRow) TableName() string { return "candidate_match_evidence" }

func TestNativeStoreCandidateMatchPersistenceVersionsLatestAgentRunAndEvidence(t *testing.T) {
	db := newCandidatePersistenceDB(t)
	now := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	if err := db.Create(&candidateEvaluationTestRow{ID: 10, ApplicationID: 7001, JobID: 9001, CandidateUserID: 3001, ResumeProfileID: 6001, EvaluationVersion: 1, IsLatest: 1, Summary: sql.NullString{String: "historical", Valid: true}, EvaluatedAt: now, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatalf("seed historical evaluation: %v", err)
	}
	agentRunID, sourceID := uint64(12001), uint64(6401)
	snapshot, err := persistence.NewNativeStore(db).SaveRecruitingCandidateMatchDraft(context.Background(), aiagentgrpc.RecruitingCandidateMatchDraft{
		ApplicationID: 7001, JobID: 9001, CandidateUserID: 3001, ResumeProfileID: 6001, AgentRunID: &agentRunID,
		OverallScore: 88.8, Recommendation: "strong_recommend", Summary: "综合评分 88.8 分。",
		StrengthsJSON: `[]`, RisksJSON: `[]`, ScoreBreakdownJSON: `{"scorer_version":"candidate-match-scorer-v1"}`, ModelName: "candidate-match-scorer-v1",
		Evidence: []aiagentgrpc.RecruitingCandidateMatchEvidenceRow{{EvidenceType: "requirement_match", Dimension: "go", SourceTable: "resume_skills", SourceID: &sourceID, Snippet: "Go", Weight: .6, ScoreImpact: 95, MetadataJSON: `{"requirement_id":"go"}`}},
	})
	if err != nil {
		t.Fatalf("SaveRecruitingCandidateMatchDraft error = %v", err)
	}
	if snapshot.Evaluation.EvaluationVersion != 2 || snapshot.Evaluation.IsLatest != 1 || snapshot.Evaluation.AgentRunID == nil || *snapshot.Evaluation.AgentRunID != agentRunID {
		t.Fatalf("saved evaluation = %+v, want version 2/latest/agent run", snapshot.Evaluation)
	}
	if len(snapshot.Evidence) != 1 || snapshot.Evidence[0].SourceID == nil || *snapshot.Evidence[0].SourceID != sourceID {
		t.Fatalf("saved evidence = %+v, want real source ID", snapshot.Evidence)
	}
	var historical candidateEvaluationTestRow
	if err := db.First(&historical, 10).Error; err != nil {
		t.Fatalf("load historical evaluation: %v", err)
	}
	if historical.IsLatest != 0 || historical.Summary.String != "historical" {
		t.Fatalf("historical evaluation = %+v, want only latest demoted and no backfill", historical)
	}
}

func TestNativeStoreCandidateMatchPersistenceRollsBackLatestAndEvaluationOnEvidenceFailure(t *testing.T) {
	db := newCandidatePersistenceDB(t)
	now := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	if err := db.Create(&candidateEvaluationTestRow{ID: 10, ApplicationID: 7001, EvaluationVersion: 1, IsLatest: 1, Summary: sql.NullString{String: "historical", Valid: true}, EvaluatedAt: now, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatalf("seed historical evaluation: %v", err)
	}
	if err := db.Exec(`CREATE TRIGGER reject_candidate_evidence BEFORE INSERT ON candidate_match_evidence BEGIN SELECT RAISE(ABORT, 'forced evidence failure'); END`).Error; err != nil {
		t.Fatalf("create failure trigger: %v", err)
	}
	sourceID := uint64(6401)
	_, err := persistence.NewNativeStore(db).SaveRecruitingCandidateMatchDraft(context.Background(), aiagentgrpc.RecruitingCandidateMatchDraft{
		ApplicationID: 7001, OverallScore: 90,
		Evidence: []aiagentgrpc.RecruitingCandidateMatchEvidenceRow{{EvidenceType: "requirement_match", SourceTable: "resume_skills", SourceID: &sourceID, Snippet: "Go"}},
	})
	if err == nil {
		t.Fatal("SaveRecruitingCandidateMatchDraft unexpectedly succeeded")
	}
	var count int64
	if err := db.Model(&candidateEvaluationTestRow{}).Count(&count).Error; err != nil {
		t.Fatalf("count evaluations: %v", err)
	}
	var historical candidateEvaluationTestRow
	if err := db.First(&historical, 10).Error; err != nil {
		t.Fatalf("load historical evaluation: %v", err)
	}
	if count != 1 || historical.IsLatest != 1 || historical.Summary.String != "historical" {
		t.Fatalf("rollback count=%d historical=%+v, want unchanged single latest history", count, historical)
	}
}

func newCandidatePersistenceDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&candidateEvaluationTestRow{}, &candidateEvidenceTestRow{}); err != nil {
		t.Fatalf("migrate candidate persistence tables: %v", err)
	}
	return db
}
