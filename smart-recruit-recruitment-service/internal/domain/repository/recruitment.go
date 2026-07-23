package repository

import (
	"context"
	"time"

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

type ApplicationRepository interface {
	CreateNewRound(ctx context.Context, application *model.Application, afterCreate func(applicationID int64) error) error
	GetDetail(ctx context.Context, applicationID int64) (*model.ApplicationDetail, error)
	UpdateStatus(ctx context.Context, applicationID int64, currentKey, targetKey string, legacyStatus int32, actorUserID int64, scope JobScope, isRePass bool, reason string, afterUpdate func(rows int64) error) (int64, error)
	CloseCurrentRound(ctx context.Context, applicationID int64) error
	ListTransitions(ctx context.Context, applicationID int64) ([]model.ApplicationStatusTransition, error)
}

type CollaborationRepository interface {
	CreateNote(ctx context.Context, note *model.CandidateNote) error
	ListNotes(ctx context.Context, candidateUserID uint64, applicationID *uint64) ([]model.CandidateNote, error)
	CreateTag(ctx context.Context, tag *model.CandidateTag) error
	AssignTag(ctx context.Context, assignment *model.CandidateTagAssignment) error
	CreateTask(ctx context.Context, task *model.FollowUpTask) error
}

type CollaborationAuthorizer interface {
	RequireCandidateAccess(ctx context.Context, staffUserID, candidateUserID uint64) error
	AuthorizePermission(ctx context.Context, staffUserID uint64, permission string) error
}

type TaxonomyRepository interface {
	GetDepartment(ctx context.Context, id int64) (*model.DepartmentNode, error)
	FindDeletedDepartment(ctx context.Context, parentID int64, name string) (*model.DepartmentNode, error)
	ReactivateDepartment(ctx context.Context, departmentID, adminID int64, name string, sortOrder int) error
	CreateDepartment(ctx context.Context, department *model.DepartmentNode) error
	ListActiveLocations(ctx context.Context) ([]model.JobLocation, error)
	ReplaceDepartmentLocations(ctx context.Context, adminID, departmentID int64, locationIDs []int64) error
	UpdateDepartmentFields(ctx context.Context, departmentID int64, fields map[string]any) error
	BuildDepartmentFullName(ctx context.Context, departmentID int64) (string, error)
}

type UsageStatsRepository interface {
	GetStatsByModel(ctx context.Context, startTime, endTime time.Time) ([]model.UsageStatsRow, error)
	GetStatsByUser(ctx context.Context, startTime, endTime time.Time) ([]model.UsageStatsRow, error)
	GetStatsBySession(ctx context.Context, startTime, endTime time.Time) ([]model.UsageStatsRow, error)
	GetTrend(ctx context.Context, startTime, endTime time.Time, granularity string) ([]model.UsageTrendRow, error)
}
