package repository

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"logic-grpc-service/model"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("failed to open in-memory SQLite: %v", err)
	}

	err = db.AutoMigrate(
		&model.User{},
		&model.Job{},
		&model.CandidateProfile{},
		&model.Resume{},
		&model.Application{},
		&model.AIChatSession{},
		&model.AIChatHistory{},
		&model.AISessionSummary{},
		&model.AIToolTrace{},
		&model.AIMemory{},
		&model.Notification{},
		&model.EventOutbox{},
		&model.RefreshToken{},
	)
	if err != nil {
		t.Fatalf("auto-migrate failed: %v", err)
	}

	// SQLite 3.39+ partial unique indexes (equivalent to MySQL generated-column approach)
	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS uk_active_application ON applications (job_id, user_id) WHERE is_current = 1 AND status <> 3")
	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS uk_user_valid_resume ON resumes (user_id) WHERE is_valid = 1")

	return db
}
