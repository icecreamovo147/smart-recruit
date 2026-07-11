package infrastructure

import (
	"gorm.io/gorm"

	"logic-grpc-service/repository"
)

// CandidateProfileRepository adapts the current GORM profile repository to Recruitment application ports.
type CandidateProfileRepository = repository.ProfileRepo

// ResumeRepository adapts the current GORM resume repository to Recruitment application ports.
type ResumeRepository = repository.ResumeRepo

// ApplicationRepository adapts the current GORM application repository to Recruitment application ports.
type ApplicationRepository = repository.ApplicationRepo

// NewCandidateProfileRepository creates a Recruitment-owned candidate profile repository adapter.
func NewCandidateProfileRepository(db *gorm.DB) *CandidateProfileRepository {
	return repository.NewProfileRepo(db)
}

// NewResumeRepository creates a Recruitment-owned resume repository adapter.
func NewResumeRepository(db *gorm.DB) *ResumeRepository {
	return repository.NewResumeRepo(db)
}

// NewApplicationRepository creates a Recruitment-owned application repository adapter.
func NewApplicationRepository(db *gorm.DB) *ApplicationRepository {
	return repository.NewApplicationRepo(db)
}
