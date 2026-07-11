package infrastructure

import (
	"gorm.io/gorm"

	"logic-grpc-service/repository"
)

// InterviewRepository adapts the current GORM interview repository to Interview application ports.
type InterviewRepository = repository.InterviewRepo

// NewInterviewRepository creates an Interview-owned repository adapter.
func NewInterviewRepository(db *gorm.DB) *InterviewRepository {
	return repository.NewInterviewRepo(db)
}
