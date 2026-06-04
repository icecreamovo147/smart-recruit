package repository

import (
	"context"
	"testing"
	"time"

	"gorm.io/gorm"

	"logic-grpc-service/model"
)

// setupInterviewTestData creates seed data for interview repo tests.
// Returns the created job, user (interviewer), application, and candidate user.
func setupInterviewTestData(t *testing.T, db *gorm.DB) (job *model.Job, interviewer *model.User, app *model.Application, candidate *model.User) {
	t.Helper()

	candidate = &model.User{Username: "candidate1", Password: "hash", AccountType: "candidate"}
	interviewer = &model.User{Username: "interviewer1", Password: "hash", AccountType: "staff"}
	if err := db.Create([]*model.User{candidate, interviewer}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}

	job = &model.Job{HrID: interviewer.ID, Title: "Software Engineer", Status: 1}
	if err := db.Create(job).Error; err != nil {
		t.Fatalf("create job: %v", err)
	}

	app = &model.Application{
		UserID:    candidate.ID,
		JobID:     job.ID,
		ResumeID:  1,
		Status:    0,
		IsCurrent: 1,
	}
	if err := db.Create(app).Error; err != nil {
		t.Fatalf("create application: %v", err)
	}

	return job, interviewer, app, candidate
}

func createTestInterview(t *testing.T, db *gorm.DB, app *model.Application, interviewer *model.User, roundNo int32) *model.InterviewSchedule {
	t.Helper()
	now := time.Now()
	interview := &model.InterviewSchedule{
		ApplicationID:   app.ID,
		InterviewerID:   interviewer.ID,
		RoundNo:         roundNo,
		Title:           "Round 1",
		Mode:            "video",
		MeetingURL:      "https://meet.example.com/1",
		Location:        "",
		DurationMinutes: 60,
		CandidateNote:   "candidate note",
		InternalNote:    "internal note",
		ScheduledAt:     &now,
		Status:          "scheduled",
		CreatedBy:       &interviewer.ID,
	}
	if err := db.Create(interview).Error; err != nil {
		t.Fatalf("create interview: %v", err)
	}
	return interview
}

// ── Interview Schedule CRUD ────────────────────────────────────────────────

func TestInterviewRepo_CreateAndGetByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInterviewRepo(db)
	ctx := context.Background()

	_, interviewer, app, _ := setupInterviewTestData(t, db)
	interview := createTestInterview(t, db, app, interviewer, 1)

	// GetByID should return the interview with joined fields
	got, err := repo.GetByID(ctx, interview.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got == nil {
		t.Fatalf("GetByID returned nil")
	}
	if got.ID != interview.ID {
		t.Errorf("ID = %d, want %d", got.ID, interview.ID)
	}
	if got.ApplicationID != app.ID {
		t.Errorf("ApplicationID = %d, want %d", got.ApplicationID, app.ID)
	}
	if got.InterviewerID != interviewer.ID {
		t.Errorf("InterviewerID = %d, want %d", got.InterviewerID, interviewer.ID)
	}
	if got.Title != interview.Title {
		t.Errorf("Title = %q, want %q", got.Title, interview.Title)
	}
	if got.Status != "scheduled" {
		t.Errorf("Status = %q, want 'scheduled'", got.Status)
	}
	if got.InterviewerName == "" {
		t.Errorf("InterviewerName should not be empty (joined field)")
	}
	if got.ApplicationStatusKey == "" {
		t.Errorf("ApplicationStatusKey should not be empty (joined field)")
	}
}

func TestInterviewRepo_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInterviewRepo(db)
	ctx := context.Background()

	got, err := repo.GetByID(ctx, 9999)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for non-existent interview, got %+v", got)
	}
}

func TestInterviewRepo_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInterviewRepo(db)
	ctx := context.Background()

	_, interviewer, app, _ := setupInterviewTestData(t, db)
	interview := createTestInterview(t, db, app, interviewer, 1)

	// Get the model for update
	model, err := repo.GetModelByID(ctx, interview.ID)
	if err != nil {
		t.Fatalf("GetModelByID: %v", err)
	}
	if model == nil {
		t.Fatalf("GetModelByID returned nil")
	}

	// Update fields
	model.Title = "Updated Title"
	model.Mode = "phone"
	model.Location = "Room A"
	model.Status = "cancelled"
	model.CancelReason = "Schedule conflict"

	if err := repo.Update(ctx, model); err != nil {
		t.Fatalf("Update: %v", err)
	}

	// Verify
	updated, err := repo.GetByID(ctx, interview.ID)
	if err != nil {
		t.Fatalf("GetByID after update: %v", err)
	}
	if updated.Title != "Updated Title" {
		t.Errorf("Title = %q, want 'Updated Title'", updated.Title)
	}
	if updated.Mode != "phone" {
		t.Errorf("Mode = %q, want 'phone'", updated.Mode)
	}
	if updated.Status != "cancelled" {
		t.Errorf("Status = %q, want 'cancelled'", updated.Status)
	}
	if updated.CancelReason != "Schedule conflict" {
		t.Errorf("CancelReason = %q, want 'Schedule conflict'", updated.CancelReason)
	}
}

func TestInterviewRepo_ListByApplication(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInterviewRepo(db)
	ctx := context.Background()

	_, interviewer, app, _ := setupInterviewTestData(t, db)

	// Create multiple interviews for the same application
	_ = createTestInterview(t, db, app, interviewer, 1)
	_ = createTestInterview(t, db, app, interviewer, 2)
	_ = createTestInterview(t, db, app, interviewer, 3)

	rows, err := repo.ListByApplication(ctx, app.ID)
	if err != nil {
		t.Fatalf("ListByApplication: %v", err)
	}
	if len(rows) != 3 {
		t.Errorf("got %d interviews, want 3", len(rows))
	}

	// Verify ordering: round_no ASC
	for i := 1; i < len(rows); i++ {
		if rows[i].RoundNo < rows[i-1].RoundNo {
			t.Errorf("interviews not ordered by round_no ASC: row %d RoundNo=%d, row %d RoundNo=%d",
				i-1, rows[i-1].RoundNo, i, rows[i].RoundNo)
		}
	}
}

func TestInterviewRepo_ListByInterviewer(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInterviewRepo(db)
	ctx := context.Background()

	_, interviewer, app, _ := setupInterviewTestData(t, db)
	_ = createTestInterview(t, db, app, interviewer, 1)
	_ = createTestInterview(t, db, app, interviewer, 2)

	// List without status filter
	rows, err := repo.ListByInterviewer(ctx, interviewer.ID, "")
	if err != nil {
		t.Fatalf("ListByInterviewer: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("got %d interviews, want 2", len(rows))
	}

	// List with status filter
	rows, err = repo.ListByInterviewer(ctx, interviewer.ID, "scheduled")
	if err != nil {
		t.Fatalf("ListByInterviewer with status: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("got %d interviews with status 'scheduled', want 2", len(rows))
	}

	// List with non-matching status
	rows, err = repo.ListByInterviewer(ctx, interviewer.ID, "completed")
	if err != nil {
		t.Fatalf("ListByInterviewer with non-matching status: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("got %d interviews with status 'completed', want 0", len(rows))
	}
}

func TestInterviewRepo_ListByCandidate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInterviewRepo(db)
	ctx := context.Background()

	_, interviewer, app, candidate := setupInterviewTestData(t, db)

	// Create a scheduled interview
	_ = createTestInterview(t, db, app, interviewer, 1)

	// Create a cancelled interview — should not appear in candidate list
	now := time.Now()
	cancelledInterview := &model.InterviewSchedule{
		ApplicationID:   app.ID,
		InterviewerID:   interviewer.ID,
		RoundNo:         2,
		Title:           "Cancelled Round",
		Mode:            "video",
		ScheduledAt:     &now,
		Status:          "cancelled",
		CreatedBy:       &interviewer.ID,
	}
	if err := db.Create(cancelledInterview).Error; err != nil {
		t.Fatalf("create cancelled interview: %v", err)
	}

	rows, err := repo.ListByCandidate(ctx, candidate.ID)
	if err != nil {
		t.Fatalf("ListByCandidate: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("got %d interviews, want 1 (cancelled should be filtered out)", len(rows))
	}
	if len(rows) > 0 && rows[0].ID == cancelledInterview.ID {
		t.Errorf("cancelled interview should not appear in candidate list")
	}
}

func TestInterviewRepo_SoftDelete_NotReturned(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInterviewRepo(db)
	ctx := context.Background()

	_, interviewer, app, _ := setupInterviewTestData(t, db)
	interview := createTestInterview(t, db, app, interviewer, 1)

	// Soft delete (gorm does this via DeletedAt)
	if err := db.Delete(interview).Error; err != nil {
		t.Fatalf("soft delete: %v", err)
	}

	// GetByID should not return the deleted interview
	got, err := repo.GetByID(ctx, interview.ID)
	if err != nil {
		t.Fatalf("GetByID after delete: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for deleted interview, got %+v", got)
	}

	// GetModelByID should also not return it
	model, err := repo.GetModelByID(ctx, interview.ID)
	if err != nil {
		t.Fatalf("GetModelByID after delete: %v", err)
	}
	if model != nil {
		t.Errorf("expected nil for deleted interview (GetModelByID), got %+v", model)
	}
}

// ── Interview Feedback ────────────────────────────────────────────────────

func TestInterviewRepo_CreateAndGetFeedback(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInterviewRepo(db)
	ctx := context.Background()

	_, interviewer, app, _ := setupInterviewTestData(t, db)
	interview := createTestInterview(t, db, app, interviewer, 1)

	feedback := &model.InterviewFeedback{
		InterviewID:         interview.ID,
		ApplicationID:       app.ID,
		InterviewerID:       interviewer.ID,
		Recommendation:      "positive",
		Score:               8,
		DimensionScoresJSON: `{"communication": 8, "technical": 8}`,
		Comments:            "Good candidate",
		SubmittedAt:         time.Now(),
	}

	if err := repo.CreateFeedback(ctx, feedback); err != nil {
		t.Fatalf("CreateFeedback: %v", err)
	}
	if feedback.ID == 0 {
		t.Errorf("expected feedback ID to be set after insert")
	}

	// Get by interview
	got, err := repo.GetFeedbackByInterview(ctx, interview.ID)
	if err != nil {
		t.Fatalf("GetFeedbackByInterview: %v", err)
	}
	if got == nil {
		t.Fatalf("GetFeedbackByInterview returned nil")
	}
	if got.InterviewID != interview.ID {
		t.Errorf("InterviewID = %d, want %d", got.InterviewID, interview.ID)
	}
	if got.Recommendation != "positive" {
		t.Errorf("Recommendation = %q, want 'positive'", got.Recommendation)
	}
	if got.Score != 8 {
		t.Errorf("Score = %d, want 8", got.Score)
	}

	// Get by interview and interviewer
	got2, err := repo.GetFeedbackByInterviewAndInterviewer(ctx, interview.ID, interviewer.ID)
	if err != nil {
		t.Fatalf("GetFeedbackByInterviewAndInterviewer: %v", err)
	}
	if got2 == nil {
		t.Fatalf("GetFeedbackByInterviewAndInterviewer returned nil")
	}
	if got2.ID != got.ID {
		t.Errorf("ID mismatch: %d vs %d", got2.ID, got.ID)
	}
}

func TestInterviewRepo_FeedbackNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInterviewRepo(db)
	ctx := context.Background()

	got, err := repo.GetFeedbackByInterview(ctx, 9999)
	if err != nil {
		t.Fatalf("GetFeedbackByInterview: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for non-existent feedback, got %+v", got)
	}
}

func TestInterviewRepo_FeedbackExistsByInterviewer(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInterviewRepo(db)
	ctx := context.Background()

	_, interviewer, app, _ := setupInterviewTestData(t, db)
	interview := createTestInterview(t, db, app, interviewer, 1)

	// Before feedback
	exists, err := repo.FeedbackExistsByInterviewer(ctx, interview.ID, interviewer.ID)
	if err != nil {
		t.Fatalf("FeedbackExistsByInterviewer before: %v", err)
	}
	if exists {
		t.Errorf("expected false before feedback creation")
	}

	// Create feedback
	feedback := &model.InterviewFeedback{
		InterviewID:    interview.ID,
		ApplicationID:  app.ID,
		InterviewerID:  interviewer.ID,
		Recommendation: "positive",
		Score:          7,
		SubmittedAt:    time.Now(),
	}
	if err := repo.CreateFeedback(ctx, feedback); err != nil {
		t.Fatalf("CreateFeedback: %v", err)
	}

	// After feedback
	exists, err = repo.FeedbackExistsByInterviewer(ctx, interview.ID, interviewer.ID)
	if err != nil {
		t.Fatalf("FeedbackExistsByInterviewer after: %v", err)
	}
	if !exists {
		t.Errorf("expected true after feedback creation")
	}
}

func TestInterviewRepo_HasFeedback(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInterviewRepo(db)
	ctx := context.Background()

	_, interviewer, app, _ := setupInterviewTestData(t, db)
	interview := createTestInterview(t, db, app, interviewer, 1)

	// Before
	has, err := repo.HasFeedback(ctx, interview.ID)
	if err != nil {
		t.Fatalf("HasFeedback before: %v", err)
	}
	if has {
		t.Errorf("expected false before feedback")
	}

	// Create feedback
	feedback := &model.InterviewFeedback{
		InterviewID:    interview.ID,
		ApplicationID:  app.ID,
		InterviewerID:  interviewer.ID,
		Recommendation: "positive",
		Score:          7,
		SubmittedAt:    time.Now(),
	}
	if err := repo.CreateFeedback(ctx, feedback); err != nil {
		t.Fatalf("CreateFeedback: %v", err)
	}

	// After
	has, err = repo.HasFeedback(ctx, interview.ID)
	if err != nil {
		t.Fatalf("HasFeedback after: %v", err)
	}
	if !has {
		t.Errorf("expected true after feedback")
	}
}

func TestInterviewRepo_PreventDuplicateFeedback(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInterviewRepo(db)
	ctx := context.Background()

	_, interviewer, app, _ := setupInterviewTestData(t, db)
	interview := createTestInterview(t, db, app, interviewer, 1)

	// First feedback should succeed
	feedback1 := &model.InterviewFeedback{
		InterviewID:    interview.ID,
		ApplicationID:  app.ID,
		InterviewerID:  interviewer.ID,
		Recommendation: "positive",
		Score:          8,
		SubmittedAt:    time.Now(),
	}
	if err := repo.CreateFeedback(ctx, feedback1); err != nil {
		t.Fatalf("first CreateFeedback: %v", err)
	}

	// Second feedback for the same interview should fail due to unique constraint
	feedback2 := &model.InterviewFeedback{
		InterviewID:    interview.ID,
		ApplicationID:  app.ID,
		InterviewerID:  interviewer.ID,
		Recommendation: "negative",
		Score:          3,
		SubmittedAt:    time.Now(),
	}
	err := repo.CreateFeedback(ctx, feedback2)
	if err == nil {
		t.Errorf("expected error for duplicate feedback, got nil")
	}
}
