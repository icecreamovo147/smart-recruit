package infrastructure

import (
	"logic-grpc-service/oss"
	"logic-grpc-service/repository"
)

// RecruitmentApplicationReader is the current adapter for Recruitment-owned application source data.
type RecruitmentApplicationReader = repository.ApplicationRepo

// RecruitmentJobReader is the current adapter for Recruitment-owned job source data.
type RecruitmentJobReader = repository.JobRepo

// RecruitmentResumeReader is the current adapter for Recruitment-owned resume source data.
type RecruitmentResumeReader = repository.ResumeRepo

// RecruitmentProfileReader is the current adapter for Recruitment-owned candidate profile source data.
type RecruitmentProfileReader = repository.ProfileRepo

// RecruitmentResumeProfileReader adapts AI-derived resume profile projections.
type RecruitmentResumeProfileReader = repository.ResumeProfileRepo

// ObjectStorageReader adapts resume object storage reads used by AI Agent workflows.
type ObjectStorageReader = oss.Storage
