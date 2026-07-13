package repository

import (
	"context"

	"smart-recruit-recruitment-service/internal/domain/model"
)

type JobScopeLevel int

const (
	JobScopeDenied JobScopeLevel = iota
	JobScopeOwned
	JobScopeDepartmentOrLocation
	JobScopeFull
)

type JobScope struct {
	Level         JobScopeLevel
	DepartmentIDs []uint64
	LocationIDs   []uint64
}

type JobRepository interface {
	Create(ctx context.Context, job *model.Job) error
	SetStatusOwned(ctx context.Context, hrID, jobID int64, status int32) (int64, error)
	SetStatusAny(ctx context.Context, jobID int64, status int32) (int64, error)
	SetStatusInScope(ctx context.Context, jobID int64, departmentIDs, locationIDs []uint64, status int32) (int64, error)
	LookupDepartment(ctx context.Context, departmentID int64) (*model.Department, error)
	LookupLocation(ctx context.Context, locationID int64) (*model.JobLocation, error)
}

type JobScopeChecker interface {
	CheckJobScope(ctx context.Context, hrID, jobID int64) (JobScope, error)
}

type DepartmentLocationValidator interface {
	ValidateDepartmentLocation(ctx context.Context, departmentID, locationID int64) error
}

type CandidateProfileRepository interface {
	GetByUserID(ctx context.Context, userID int64) (*model.CandidateProfile, error)
	Upsert(ctx context.Context, profile *model.CandidateProfile) error
}

type ResumeRepository interface {
	GetValidByUserID(ctx context.Context, userID int64) (*model.Resume, error)
	ConfirmUpload(ctx context.Context, resume *model.Resume, afterCreate func(resumeID int64) error) error
}

type ObjectStorage interface {
	GeneratePresignedPutURL(ossKey, contentType string) (uploadURL string, expireAt int64, err error)
	SavePresignSession(ctx context.Context, uploadID string, session model.PresignSession) error
	GetAndDeletePresignSession(ctx context.Context, uploadID string) (*model.PresignSession, error)
	VerifyObject(ctx context.Context, ossKey string) error
	VerifyObjectSize(ctx context.Context, ossKey string, maxBytes int64) error
	CopyObject(ctx context.Context, srcKey, dstKey string) error
	DeleteObject(ctx context.Context, ossKey string) error
	ProviderName() string
}

type OutboxPublisher interface {
	WriteEvent(ctx context.Context, event model.OutboxEvent) error
	Signal()
}

type UsageLogRepository interface {
	CreateUsageLog(ctx context.Context, entry model.UsageLogEntry) error
}
