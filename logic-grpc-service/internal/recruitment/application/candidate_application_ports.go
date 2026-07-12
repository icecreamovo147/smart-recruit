package application

import (
	"context"

	"gorm.io/gorm"

	"logic-grpc-service/model"
	"logic-grpc-service/repository"
)

// CandidateProfileRepository is the Recruitment-owned candidate profile persistence port.
type CandidateProfileRepository interface {
	GetByUserID(ctx context.Context, userID int64) (*model.CandidateProfile, error)
	Upsert(ctx context.Context, profile *model.CandidateProfile) error
}

// ResumeRepository is the Recruitment-owned uploaded resume persistence port.
type ResumeRepository interface {
	ConfirmUpload(ctx context.Context, resume *model.Resume) error
	ConfirmUploadWithTx(ctx context.Context, resume *model.Resume, afterCreate func(tx *gorm.DB) error) error
	GetValidByUserID(ctx context.Context, userID int64) (*model.Resume, error)
	GetByID(ctx context.Context, resumeID int64) (*model.Resume, error)
	UpdateParsedText(ctx context.Context, resumeID int64, text string) error
}

// ApplicationRepository is the Recruitment-owned application lifecycle persistence port.
type ApplicationRepository interface {
	Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error
	ExistsActive(ctx context.Context, userID, jobID int64) (bool, error)
	CreateNewRound(ctx context.Context, application *model.Application) error
	CreateNewRoundWithTx(ctx context.Context, tx *gorm.DB, application *model.Application) error
	GetByID(ctx context.Context, applicationID int64) (*model.Application, error)
	ListMyCursor(ctx context.Context, userID int64, cursor string, limit int32) ([]repository.MyApplicationRow, string, bool, error)
	ListByJob(ctx context.Context, jobID int64, page, pageSize int32) ([]repository.JobApplicationRow, int64, error)
	ListCurrentByJob(ctx context.Context, jobID int64) ([]repository.JobApplicationRow, error)
	UpdateStatusOwnedWithTx(ctx context.Context, tx *gorm.DB, hrID, applicationID int64, currentStatusKey, statusKey string, legacyStatus int32) (int64, error)
	UpdateStatusInScopeWithTx(ctx context.Context, tx *gorm.DB, deptIDs, locIDs []uint64, applicationID int64, currentStatusKey, statusKey string, legacyStatus int32) (int64, error)
	UpdateStatusAnyWithTx(ctx context.Context, tx *gorm.DB, applicationID int64, currentStatusKey, statusKey string, legacyStatus int32) (int64, error)
	RePassWithTx(ctx context.Context, tx *gorm.DB, applicationID int64, currentStatusKey, targetStatusKey string, legacyStatus int32, hrID int64, deptIDs, locIDs []uint64, scopeLevel int) (int64, error)
	CreateTransition(ctx context.Context, tx *gorm.DB, t *model.ApplicationStatusTransition) error
	ListTransitions(ctx context.Context, applicationID int64) ([]model.ApplicationStatusTransition, error)
}
