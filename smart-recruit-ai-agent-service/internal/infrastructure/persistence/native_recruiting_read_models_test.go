package persistence

import (
	"context"
	"database/sql"
	"reflect"
	"sort"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	aiagentgrpc "smart-recruit-ai-agent-service/internal/interfaces/grpc"
)

func TestNativeStoreRecruitingResumeProfileSnapshot(t *testing.T) {
	ctx := context.Background()
	db := newRecruitingReadModelTestDB(t)
	store := NewNativeStore(db)
	seedRecruitingReadModels(t, db)

	application, found, err := store.GetLatestRecruitingApplicationByResumeID(ctx, 8001)
	if err != nil {
		t.Fatalf("GetLatestRecruitingApplicationByResumeID error = %v", err)
	}
	if !found {
		t.Fatal("expected application context for resume 8001")
	}
	if application.ApplicationID != 7001 || application.JobID != 9001 || application.CandidateUserID != 3001 || application.CandidateName != "Ada Lovelace" || application.ResumeID != 8001 {
		t.Fatalf("application context = %+v, want app/resume/candidate identifiers", application)
	}

	current, found, err := store.GetCurrentRecruitingResumeProfileByResumeID(ctx, 8001)
	if err != nil {
		t.Fatalf("GetCurrentRecruitingResumeProfileByResumeID error = %v", err)
	}
	if !found || current.ID != 6001 || current.ResumeID != 8001 {
		t.Fatalf("current profile = %+v found=%v, want profile 6001 resume 8001", current, found)
	}

	snapshot, found, err := store.GetRecruitingResumeProfileSnapshot(ctx, 6001)
	if err != nil {
		t.Fatalf("GetRecruitingResumeProfileSnapshot error = %v", err)
	}
	if !found {
		t.Fatal("expected resume profile snapshot")
	}
	if snapshot.ParseRun.ID != 5001 || snapshot.ParseRun.ResumeID != 8001 || snapshot.Profile.ID != 6001 || snapshot.Profile.ParseRunID != 5001 {
		t.Fatalf("snapshot identifiers inconsistent: %+v", snapshot)
	}
	if snapshot.Profile.FullName != "Ada Lovelace" || snapshot.Profile.RawJSON != `{"name":"Ada"}` || snapshot.Profile.TotalExperienceYears != 6.5 {
		t.Fatalf("profile fields = %+v, want seeded values", snapshot.Profile)
	}
	if got := len(snapshot.Educations); got != 1 || snapshot.Educations[0].School != "Analytical University" {
		t.Fatalf("educations = %+v, want seeded education", snapshot.Educations)
	}
	if got := len(snapshot.Experiences); got != 1 || snapshot.Experiences[0].AchievementsJSON != `["compiler"]` {
		t.Fatalf("experiences = %+v, want seeded experience", snapshot.Experiences)
	}
	if got := len(snapshot.Projects); got != 1 || snapshot.Projects[0].TechnologiesJSON != `["go","sql"]` {
		t.Fatalf("projects = %+v, want seeded project", snapshot.Projects)
	}
	if got := len(snapshot.Skills); got != 1 || snapshot.Skills[0].Name != "Go" || snapshot.Skills[0].Years != 4.5 {
		t.Fatalf("skills = %+v, want seeded skill", snapshot.Skills)
	}
}

func TestNativeStoreRecruitingCandidateMatchSnapshotAndComparisonInputs(t *testing.T) {
	ctx := context.Background()
	db := newRecruitingReadModelTestDB(t)
	store := NewNativeStore(db)
	seedRecruitingReadModels(t, db)

	latest, found, err := store.GetLatestRecruitingCandidateMatchEvaluationSnapshotByApplicationID(ctx, 7001)
	if err != nil {
		t.Fatalf("GetLatestRecruitingCandidateMatchEvaluationSnapshotByApplicationID error = %v", err)
	}
	if !found {
		t.Fatal("expected latest candidate match snapshot")
	}
	if latest.Evaluation.ID != 9102 || latest.Evaluation.ApplicationID != 7001 || latest.Evaluation.EvaluationVersion != 2 || latest.Evaluation.OverallScore != 92.5 {
		t.Fatalf("latest evaluation = %+v, want version 2 score 92.5", latest.Evaluation)
	}
	if len(latest.Evidence) != 1 || latest.Evidence[0].SourceTable != "resume_skills" || latest.Evidence[0].SourceID == nil || *latest.Evidence[0].SourceID != 6401 {
		t.Fatalf("latest evidence = %+v, want seeded evidence", latest.Evidence)
	}

	version, found, err := store.GetRecruitingCandidateMatchEvaluationSnapshotByApplicationVersion(ctx, 7001, 1)
	if err != nil {
		t.Fatalf("GetRecruitingCandidateMatchEvaluationSnapshotByApplicationVersion error = %v", err)
	}
	if !found || version.Evaluation.ID != 9101 || version.Evaluation.IsLatest != 0 {
		t.Fatalf("version snapshot = %+v found=%v, want historical evaluation 9101", version, found)
	}

	byID, found, err := store.GetRecruitingCandidateMatchEvaluationSnapshot(ctx, 9102)
	if err != nil {
		t.Fatalf("GetRecruitingCandidateMatchEvaluationSnapshot error = %v", err)
	}
	if !found || byID.Evaluation.ApplicationID != 7001 {
		t.Fatalf("by id snapshot = %+v found=%v, want application 7001", byID, found)
	}

	byRun, found, err := store.GetRecruitingCandidateMatchEvaluationSnapshotByApplicationAgentRunID(ctx, 7001, 9901)
	if err != nil {
		t.Fatalf("GetRecruitingCandidateMatchEvaluationSnapshotByApplicationAgentRunID error = %v", err)
	}
	if !found || byRun.Evaluation.ID != 9102 || byRun.Evaluation.AgentRunID == nil || *byRun.Evaluation.AgentRunID != 9901 {
		t.Fatalf("by agent run snapshot = %+v found=%v, want evaluation 9102 for run 9901", byRun, found)
	}
	crossApplicationRun, found, err := store.GetRecruitingCandidateMatchEvaluationSnapshotByApplicationAgentRunID(ctx, 7002, 9901)
	if err != nil {
		t.Fatalf("GetRecruitingCandidateMatchEvaluationSnapshotByApplicationAgentRunID cross app error = %v", err)
	}
	if found {
		t.Fatalf("cross application agent run snapshot = %+v, want not found", crossApplicationRun)
	}

	applications, err := store.ListCurrentRecruitingApplicationsByJobID(ctx, 9001)
	if err != nil {
		t.Fatalf("ListCurrentRecruitingApplicationsByJobID error = %v", err)
	}
	if got, want := recruitingApplicationIDs(applications), []int64{7001, 7002, 7003}; !reflect.DeepEqual(got, want) {
		t.Fatalf("current application ids = %#v, want %#v", got, want)
	}
	evaluations, err := store.ListLatestRecruitingCandidateMatchEvaluationsByApplicationIDs(ctx, []int64{7001, 7002, 7003})
	if err != nil {
		t.Fatalf("ListLatestRecruitingCandidateMatchEvaluationsByApplicationIDs error = %v", err)
	}
	gotOrder, missing := recruitingComparisonOrder(applications, evaluations)
	if want := []int64{7003, 7001, 7002}; !reflect.DeepEqual(gotOrder, want) {
		t.Fatalf("comparison order = %#v, want %#v", gotOrder, want)
	}
	if want := []int64{7002}; !reflect.DeepEqual(missing, want) {
		t.Fatalf("missing applications = %#v, want %#v", missing, want)
	}
}

func newRecruitingReadModelTestDB(t *testing.T) *gorm.DB {
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
	if err := db.AutoMigrate(
		&recruitingApplicationTestRecord{},
		&recruitingCandidateProfileTestRecord{},
		&recruitingResumeParseRunRecord{},
		&recruitingResumeProfileRecord{},
		&recruitingResumeEducationRecord{},
		&recruitingResumeExperienceRecord{},
		&recruitingResumeProjectRecord{},
		&recruitingResumeSkillRecord{},
		&recruitingCandidateMatchEvaluationRecord{},
		&recruitingCandidateMatchEvidenceRecord{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func seedRecruitingReadModels(t *testing.T, db *gorm.DB) {
	t.Helper()
	now := time.Date(2026, 7, 3, 10, 0, 0, 0, time.UTC)
	applications := []recruitingApplicationTestRecord{
		{ID: 7001, JobID: 9001, UserID: 3001, ResumeID: 8001, StatusKey: "applied", IsCurrent: 1, AppliedAt: now, UpdatedAt: now},
		{ID: 7002, JobID: 9001, UserID: 3002, ResumeID: 8002, StatusKey: "applied", IsCurrent: 1, AppliedAt: now.Add(time.Minute), UpdatedAt: now},
		{ID: 7003, JobID: 9001, UserID: 3003, ResumeID: 8003, StatusKey: "applied", IsCurrent: 1, AppliedAt: now.Add(2 * time.Minute), UpdatedAt: now},
		{ID: 7004, JobID: 9001, UserID: 3004, ResumeID: 8004, StatusKey: "withdrawn", IsCurrent: 0, AppliedAt: now.Add(3 * time.Minute), UpdatedAt: now},
	}
	if err := db.Create(&applications).Error; err != nil {
		t.Fatalf("seed applications: %v", err)
	}
	profiles := []recruitingCandidateProfileTestRecord{
		{ID: 1, UserID: 3001, RealName: "Ada Lovelace"},
		{ID: 2, UserID: 3002, RealName: "Grace Hopper"},
		{ID: 3, UserID: 3003, RealName: "Katherine Johnson"},
	}
	if err := db.Create(&profiles).Error; err != nil {
		t.Fatalf("seed candidate profiles: %v", err)
	}
	completedAt := now.Add(5 * time.Minute)
	parseRun := recruitingResumeParseRunRecord{ID: 5001, ResumeID: 8001, UserID: 3001, Status: "succeeded", ParserVersion: ns("v2"), InputHash: ns("hash"), StartedAt: now, CompletedAt: &completedAt, CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&parseRun).Error; err != nil {
		t.Fatalf("seed parse run: %v", err)
	}
	profile := recruitingResumeProfileRecord{
		ID: 6001, ResumeID: 8001, UserID: 3001, ParseRunID: 5001, Version: 2, IsCurrent: 1,
		FullName: ns("Ada Lovelace"), Email: ns("ada@example.com"), Phone: ns("123"), Location: ns("London"),
		Headline: ns("Engineer"), Summary: ns("Analytical engine"), TotalExperienceYears: nf(6.5), HighestDegree: ns("BS"), RawJSON: ns(`{"name":"Ada"}`),
		CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&profile).Error; err != nil {
		t.Fatalf("seed profile: %v", err)
	}
	startDate := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := db.Create(&recruitingResumeEducationRecord{ID: 6101, ResumeProfileID: 6001, School: "Analytical University", Degree: ns("BS"), Major: ns("Math"), StartDate: &startDate, EndDate: &endDate, Description: ns("Computing"), SortOrder: 1, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatalf("seed education: %v", err)
	}
	if err := db.Create(&recruitingResumeExperienceRecord{ID: 6201, ResumeProfileID: 6001, Company: "Engines", Title: ns("Engineer"), Location: ns("London"), StartDate: &startDate, IsCurrent: 1, Description: ns("Built systems"), AchievementsJSON: ns(`["compiler"]`), SortOrder: 1, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatalf("seed experience: %v", err)
	}
	if err := db.Create(&recruitingResumeProjectRecord{ID: 6301, ResumeProfileID: 6001, Name: "Compiler", Role: ns("Lead"), StartDate: &startDate, EndDate: &endDate, Description: ns("Tooling"), TechnologiesJSON: ns(`["go","sql"]`), HighlightsJSON: ns(`["fast"]`), SortOrder: 1, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatalf("seed project: %v", err)
	}
	if err := db.Create(&recruitingResumeSkillRecord{ID: 6401, ResumeProfileID: 6001, Name: "Go", Category: ns("language"), Level: ns("senior"), Years: nf(4.5), Evidence: ns("Compiler"), SortOrder: 1, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatalf("seed skill: %v", err)
	}

	matchRunID := uint64(9901)
	evaluations := []recruitingCandidateMatchEvaluationRecord{
		{ID: 9101, ApplicationID: 7001, JobID: 9001, CandidateUserID: 3001, ResumeProfileID: 6001, EvaluationVersion: 1, IsLatest: 0, OverallScore: nf(80), Recommendation: ns("possible_match"), Summary: ns("Historical"), ScoreBreakdownJSON: ns(`{"dimensions":[]}`), ModelName: ns("model-a"), EvaluatedAt: now, CreatedAt: now, UpdatedAt: now},
		{ID: 9102, ApplicationID: 7001, JobID: 9001, CandidateUserID: 3001, ResumeProfileID: 6001, AgentRunID: &matchRunID, EvaluationVersion: 2, IsLatest: 1, OverallScore: nf(92.5), Recommendation: ns("strong_match"), Summary: ns("Latest"), StrengthsJSON: ns(`["go"]`), RisksJSON: ns(`[]`), ScoreBreakdownJSON: ns(`{"dimensions":[{"name":"backend","score":92.5}]}`), ModelName: ns("model-b"), EvaluatedAt: now.Add(time.Hour), CreatedAt: now, UpdatedAt: now},
		{ID: 9103, ApplicationID: 7003, JobID: 9001, CandidateUserID: 3003, ResumeProfileID: 6001, EvaluationVersion: 1, IsLatest: 1, OverallScore: nf(97), Recommendation: ns("strong_match"), Summary: ns("Top"), ScoreBreakdownJSON: ns(`{"dimensions":[{"name":"backend","score":97}]}`), ModelName: ns("model-b"), EvaluatedAt: now.Add(2 * time.Hour), CreatedAt: now, UpdatedAt: now},
	}
	if err := db.Create(&evaluations).Error; err != nil {
		t.Fatalf("seed evaluations: %v", err)
	}
	sourceID := uint64(6401)
	evidence := recruitingCandidateMatchEvidenceRecord{ID: 9201, EvaluationID: 9102, EvidenceType: "skill", Dimension: ns("backend"), SourceTable: ns("resume_skills"), SourceID: &sourceID, Snippet: ns("Go"), Weight: nf(0.8), ScoreImpact: nf(5.5), MetadataJSON: ns(`{"source":"resume"}`), CreatedAt: now}
	if err := db.Create(&evidence).Error; err != nil {
		t.Fatalf("seed evidence: %v", err)
	}
}

func recruitingApplicationIDs(rows []aiagentgrpc.RecruitingApplicationContext) []int64 {
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ApplicationID)
	}
	return ids
}

func recruitingComparisonOrder(applications []aiagentgrpc.RecruitingApplicationContext, evaluations []aiagentgrpc.RecruitingCandidateMatchEvaluationRow) ([]int64, []int64) {
	type item struct {
		applicationID int64
		hasEvaluation bool
		score         float64
	}
	byApplicationID := make(map[int64]float64, len(evaluations))
	for _, evaluation := range evaluations {
		byApplicationID[evaluation.ApplicationID] = evaluation.OverallScore
	}
	items := make([]item, 0, len(applications))
	missing := make([]int64, 0)
	for _, application := range applications {
		score, ok := byApplicationID[application.ApplicationID]
		items = append(items, item{applicationID: application.ApplicationID, hasEvaluation: ok, score: score})
		if !ok {
			missing = append(missing, application.ApplicationID)
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].hasEvaluation != items[j].hasEvaluation {
			return items[i].hasEvaluation
		}
		if items[i].score != items[j].score {
			return items[i].score > items[j].score
		}
		return items[i].applicationID < items[j].applicationID
	})
	order := make([]int64, 0, len(items))
	for _, item := range items {
		order = append(order, item.applicationID)
	}
	return order, missing
}

func ns(value string) sql.NullString {
	return sql.NullString{String: value, Valid: true}
}

func nf(value float64) sql.NullFloat64 {
	return sql.NullFloat64{Float64: value, Valid: true}
}

type recruitingApplicationTestRecord struct {
	ID        int64     `gorm:"primaryKey"`
	JobID     int64     `gorm:"column:job_id"`
	UserID    int64     `gorm:"column:user_id"`
	ResumeID  int64     `gorm:"column:resume_id"`
	StatusKey string    `gorm:"column:status_key"`
	IsCurrent int32     `gorm:"column:is_current"`
	AppliedAt time.Time `gorm:"column:applied_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (recruitingApplicationTestRecord) TableName() string { return "applications" }

type recruitingCandidateProfileTestRecord struct {
	ID       int64  `gorm:"primaryKey"`
	UserID   int64  `gorm:"column:user_id"`
	RealName string `gorm:"column:real_name"`
}

func (recruitingCandidateProfileTestRecord) TableName() string { return "candidate_profiles" }
