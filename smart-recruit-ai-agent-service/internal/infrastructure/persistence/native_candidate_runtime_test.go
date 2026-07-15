package persistence

import (
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	aiagentgrpc "smart-recruit-ai-agent-service/internal/interfaces/grpc"
)

func TestNativeStoreLoadCandidateRuntimeContextScopesCandidateData(t *testing.T) {
	ctx := context.Background()
	db := newCandidateRuntimeTestDB(t)
	store := NewNativeStore(db)
	now := time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC)
	seedCandidateRuntimeRows(t, db, now)

	snapshot, err := store.LoadCandidateRuntimeContext(ctx, 55, 10)
	if err != nil {
		t.Fatalf("LoadCandidateRuntimeContext error = %v", err)
	}
	if len(snapshot.Applications) != 1 || snapshot.Applications[0].ApplicationID != 7001 || snapshot.Applications[0].JobTitle != "Backend Engineer" {
		t.Fatalf("applications = %#v, want only candidate 55 application", snapshot.Applications)
	}
	if !snapshot.Resume.Available || snapshot.Resume.ResumeID != 8001 || snapshot.Resume.FileName != "ada.pdf" || snapshot.Resume.Summary == "" {
		t.Fatalf("resume = %#v, want candidate 55 resume summary", snapshot.Resume)
	}
	if len(snapshot.Interviews) != 1 || snapshot.Interviews[0].InterviewID != 6001 || snapshot.Interviews[0].CandidateNote != "Bring portfolio" {
		t.Fatalf("interviews = %#v, want only candidate-visible interview", snapshot.Interviews)
	}
	if len(snapshot.Offers) != 1 || snapshot.Offers[0].OfferID != 5001 || snapshot.Offers[0].Status != "sent" {
		t.Fatalf("offers = %#v, want only candidate 55 offer", snapshot.Offers)
	}
	if len(snapshot.Jobs) != 2 {
		t.Fatalf("jobs = %#v, want active jobs for recommendation", snapshot.Jobs)
	}
	for _, app := range snapshot.Applications {
		if app.ApplicationID == 7002 {
			t.Fatalf("snapshot leaked another candidate application: %#v", snapshot)
		}
	}
}

func TestNativeStoreRecordCandidateUsageAuditWritesUsageAndAuthContext(t *testing.T) {
	ctx := context.Background()
	db := newCandidateRuntimeTestDB(t)
	store := NewNativeStore(db)

	usageID, err := store.RecordCandidateUsageAudit(ctx, aiagentgrpc.CandidateUsageAuditRow{
		UserID:          55,
		ServiceType:     "ai_chat",
		Endpoint:        "/candidate/ai/chat/stream",
		Provider:        "openai_compatible",
		Model:           "test-model",
		RequestChars:    12,
		ResponseChars:   34,
		EstimatedTokens: 12,
		Status:          "ok",
		RequestID:       "req-1",
		IP:              "127.0.0.1",
		RoleKeys:        []string{"candidate"},
		PermissionKey:   "ai.candidate.use",
		ScopeKeys:       []string{"self"},
	})
	if err != nil {
		t.Fatalf("RecordCandidateUsageAudit error = %v", err)
	}
	if usageID == 0 {
		t.Fatal("usage id = 0, want created row")
	}
	var usage thirdPartyUsageLogRecord
	if err := db.First(&usage, usageID).Error; err != nil {
		t.Fatalf("load usage: %v", err)
	}
	if usage.UserID != 55 || usage.Role != 1 || usage.RequestChars != 12 || usage.ResponseChars != 34 || usage.RequestID != "req-1" {
		t.Fatalf("usage = %#v", usage)
	}
	var authCtx aiUsageAuthContextRecord
	if err := db.Where("usage_log_id = ?", usageID).First(&authCtx).Error; err != nil {
		t.Fatalf("load auth context: %v", err)
	}
	if authCtx.ActorUserID != 55 || authCtx.AccountType != "candidate" || authCtx.PermissionKey != "ai.candidate.use" || authCtx.ScopeKeys != "self" {
		t.Fatalf("auth context = %#v", authCtx)
	}
}

func newCandidateRuntimeTestDB(t *testing.T) *gorm.DB {
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
		&candidateRuntimeApplicationTestRecord{},
		&candidateRuntimeJobTestRecord{},
		&candidateRuntimeResumeTestRecord{},
		&candidateRuntimeInterviewTestRecord{},
		&candidateRuntimeOfferTestRecord{},
		&thirdPartyUsageLogRecord{},
		&aiUsageAuthContextRecord{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func seedCandidateRuntimeRows(t *testing.T, db *gorm.DB, now time.Time) {
	t.Helper()
	jobs := []candidateRuntimeJobTestRecord{
		{ID: 9001, Title: "Backend Engineer", Department: "Engineering", Location: "Shanghai", SalaryRange: "20k-30k", Status: 1, CreatedAt: now},
		{ID: 9002, Title: "Product Manager", Department: "Product", Location: "Remote", SalaryRange: "18k-28k", Status: 1, CreatedAt: now.Add(time.Minute)},
	}
	if err := db.Create(&jobs).Error; err != nil {
		t.Fatalf("seed jobs: %v", err)
	}
	applications := []candidateRuntimeApplicationTestRecord{
		{ID: 7001, JobID: 9001, UserID: 55, ResumeID: 8001, Status: 1, StatusKey: "interview", RoundNo: 2, IsCurrent: 1, AppliedAt: now},
		{ID: 7002, JobID: 9002, UserID: 66, ResumeID: 8002, Status: 1, StatusKey: "applied", RoundNo: 1, IsCurrent: 1, AppliedAt: now},
	}
	if err := db.Create(&applications).Error; err != nil {
		t.Fatalf("seed applications: %v", err)
	}
	resumes := []candidateRuntimeResumeTestRecord{
		{ID: 8001, UserID: 55, FileName: "ada.pdf", ParsedText: "Go backend candidate with distributed systems experience", IsValid: 1, UploadedAt: now},
		{ID: 8002, UserID: 66, FileName: "other.pdf", ParsedText: "Other candidate resume", IsValid: 1, UploadedAt: now},
	}
	if err := db.Create(&resumes).Error; err != nil {
		t.Fatalf("seed resumes: %v", err)
	}
	interviews := []candidateRuntimeInterviewTestRecord{
		{ID: 6001, ApplicationID: 7001, RoundNo: 1, Title: "初试", Mode: "video", CandidateNote: "Bring portfolio", ScheduledAt: &now, Status: "scheduled"},
		{ID: 6002, ApplicationID: 7002, RoundNo: 1, Title: "初试", Mode: "video", CandidateNote: "Other note", ScheduledAt: &now, Status: "scheduled"},
	}
	if err := db.Create(&interviews).Error; err != nil {
		t.Fatalf("seed interviews: %v", err)
	}
	offers := []candidateRuntimeOfferTestRecord{
		{ID: 5001, ApplicationID: 7001, CandidateUserID: 55, JobID: 9001, Title: "Backend Engineer", Status: "sent", CreatedAt: now},
		{ID: 5002, ApplicationID: 7002, CandidateUserID: 66, JobID: 9002, Title: "Product Manager", Status: "sent", CreatedAt: now},
	}
	if err := db.Create(&offers).Error; err != nil {
		t.Fatalf("seed offers: %v", err)
	}
}

type candidateRuntimeApplicationTestRecord struct {
	ID        int64     `gorm:"primaryKey"`
	JobID     int64     `gorm:"column:job_id"`
	UserID    int64     `gorm:"column:user_id"`
	ResumeID  int64     `gorm:"column:resume_id"`
	Status    int32     `gorm:"column:status"`
	StatusKey string    `gorm:"column:status_key"`
	RoundNo   int32     `gorm:"column:round_no"`
	IsCurrent int32     `gorm:"column:is_current"`
	AppliedAt time.Time `gorm:"column:applied_at"`
}

func (candidateRuntimeApplicationTestRecord) TableName() string { return "applications" }

type candidateRuntimeJobTestRecord struct {
	ID          int64     `gorm:"primaryKey"`
	Title       string    `gorm:"column:title"`
	Department  string    `gorm:"column:department"`
	Location    string    `gorm:"column:location"`
	SalaryRange string    `gorm:"column:salary_range"`
	Status      int32     `gorm:"column:status"`
	CreatedAt   time.Time `gorm:"column:created_at"`
}

func (candidateRuntimeJobTestRecord) TableName() string { return "jobs" }

type candidateRuntimeResumeTestRecord struct {
	ID         int64     `gorm:"primaryKey"`
	UserID     int64     `gorm:"column:user_id"`
	FileName   string    `gorm:"column:file_name"`
	ParsedText string    `gorm:"column:parsed_text"`
	IsValid    int32     `gorm:"column:is_valid"`
	UploadedAt time.Time `gorm:"column:uploaded_at"`
}

func (candidateRuntimeResumeTestRecord) TableName() string { return "resumes" }

type candidateRuntimeInterviewTestRecord struct {
	ID            int64      `gorm:"primaryKey"`
	ApplicationID int64      `gorm:"column:application_id"`
	RoundNo       int32      `gorm:"column:round_no"`
	Title         string     `gorm:"column:title"`
	Mode          string     `gorm:"column:mode"`
	CandidateNote string     `gorm:"column:candidate_note"`
	ScheduledAt   *time.Time `gorm:"column:scheduled_at"`
	Status        string     `gorm:"column:status"`
	DeletedAt     *time.Time `gorm:"column:deleted_at"`
}

func (candidateRuntimeInterviewTestRecord) TableName() string { return "interview_schedules" }

type candidateRuntimeOfferTestRecord struct {
	ID              int64      `gorm:"primaryKey"`
	ApplicationID   int64      `gorm:"column:application_id"`
	CandidateUserID int64      `gorm:"column:candidate_user_id"`
	JobID           int64      `gorm:"column:job_id"`
	Title           string     `gorm:"column:title"`
	Status          string     `gorm:"column:status"`
	SalaryRange     string     `gorm:"column:salary_range"`
	WorkLocation    string     `gorm:"column:work_location"`
	StartDate       string     `gorm:"column:start_date"`
	ExpiresAt       *time.Time `gorm:"column:expires_at"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
}

func (candidateRuntimeOfferTestRecord) TableName() string { return "offers" }
