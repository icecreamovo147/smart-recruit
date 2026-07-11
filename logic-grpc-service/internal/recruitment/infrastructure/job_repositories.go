package infrastructure

import (
	"gorm.io/gorm"

	"logic-grpc-service/repository"
)

// JobRepository adapts the current GORM job repository to Recruitment application ports.
type JobRepository = repository.JobRepo

// DepartmentRepository adapts the current GORM department repository to Recruitment application ports.
type DepartmentRepository = repository.DepartmentRepo

// LocationRepository adapts the current GORM job-location repository to Recruitment application ports.
type LocationRepository = repository.JobLocationRepo

// NewJobRepository creates a Recruitment-owned job repository adapter.
func NewJobRepository(db *gorm.DB) *JobRepository {
	return repository.NewJobRepo(db)
}

// NewDepartmentRepository creates a Recruitment-owned department repository adapter.
func NewDepartmentRepository(db *gorm.DB) *DepartmentRepository {
	return repository.NewDepartmentRepo(db)
}

// NewLocationRepository creates a Recruitment-owned job-location repository adapter.
func NewLocationRepository(db *gorm.DB) *LocationRepository {
	return repository.NewJobLocationRepo(db)
}
