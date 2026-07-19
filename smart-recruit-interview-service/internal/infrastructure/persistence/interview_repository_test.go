package persistence

import (
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"smart-recruit-interview-service/internal/domain/model"
)

func TestFeedbackRepositoryUsesSingularInterviewFeedbackTable(t *testing.T) {
	db := openFeedbackRepositoryTestDB(t)
	repo := NewInterviewRepository(db)
	ctx := context.Background()
	submittedAt := time.Date(2026, 7, 14, 9, 30, 0, 0, time.UTC)

	if !db.Migrator().HasTable("interview_feedback") {
		t.Fatal("expected singular interview_feedback table to exist")
	}
	if db.Migrator().HasTable("interview_feedbacks") {
		t.Fatal("did not expect plural interview_feedbacks table to exist")
	}

	feedback := &model.Feedback{
		InterviewID:         101,
		ApplicationID:       202,
		InterviewerID:       303,
		Recommendation:      "recommend",
		Score:               8,
		DimensionScoresJSON: `{"technical":8}`,
		Comments:            "solid technical screen",
		SubmittedAt:         submittedAt,
	}
	if err := repo.CreateFeedback(ctx, feedback); err != nil {
		t.Fatalf("CreateFeedback returned error: %v", err)
	}
	if feedback.ID == 0 {
		t.Fatal("CreateFeedback did not populate feedback ID")
	}

	exists, err := repo.FeedbackExistsByInterviewer(ctx, feedback.InterviewID, feedback.InterviewerID)
	if err != nil {
		t.Fatalf("FeedbackExistsByInterviewer returned error: %v", err)
	}
	if !exists {
		t.Fatal("FeedbackExistsByInterviewer returned false, want true")
	}

	found, err := repo.FindFeedbackByInterviewAndInterviewer(ctx, feedback.InterviewID, feedback.InterviewerID)
	if err != nil {
		t.Fatalf("FindFeedbackByInterviewAndInterviewer returned error: %v", err)
	}
	if found == nil {
		t.Fatal("FindFeedbackByInterviewAndInterviewer returned nil feedback")
	}
	if found.ID != feedback.ID || found.ApplicationID != feedback.ApplicationID || found.Recommendation != feedback.Recommendation {
		t.Fatalf("found feedback = %+v, want ID=%d applicationID=%d recommendation=%q", found, feedback.ID, feedback.ApplicationID, feedback.Recommendation)
	}

	var count int64
	if err := db.Table("interview_feedback").Where("id = ?", feedback.ID).Count(&count).Error; err != nil {
		t.Fatalf("count singular interview_feedback rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("singular interview_feedback row count=%d, want 1", count)
	}
}

func openFeedbackRepositoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql database: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := sqlDB.Close(); closeErr != nil {
			t.Fatalf("close sqlite database: %v", closeErr)
		}
	})

	if err := createSingularInterviewFeedbackTable(db); err != nil {
		t.Fatalf("create singular interview_feedback table: %v", err)
	}
	return db
}

func createSingularInterviewFeedbackTable(db *gorm.DB) error {
	return db.Exec(`
CREATE TABLE interview_feedback (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	tenant_id INTEGER NOT NULL DEFAULT 0,
	interview_id INTEGER NOT NULL,
	application_id INTEGER NOT NULL,
	interviewer_id INTEGER NOT NULL,
	recommendation TEXT NOT NULL,
	score INTEGER NOT NULL,
	dimension_scores_json TEXT,
	comments TEXT,
	submitted_at DATETIME NOT NULL,
	updated_at DATETIME
)`).Error
}
