package application

import (
	"context"

	"logic-grpc-service/model"
	"logic-grpc-service/repository"
)

// JobRepository is the Recruitment-owned job persistence port.
type JobRepository interface {
	Create(ctx context.Context, job *model.Job) error
	UpdateOwned(ctx context.Context, hrID, jobID int64, fields map[string]any) (int64, error)
	UpdateAny(ctx context.Context, jobID int64, fields map[string]any) (int64, error)
	UpdateInScope(ctx context.Context, jobID int64, deptIDs, locIDs []uint64, fields map[string]any) (int64, error)
	OfflineOwned(ctx context.Context, hrID, jobID int64) (int64, error)
	OfflineAny(ctx context.Context, jobID int64) (int64, error)
	OfflineInScope(ctx context.Context, jobID int64, deptIDs, locIDs []uint64) (int64, error)
	OnlineOwned(ctx context.Context, hrID, jobID int64) (int64, error)
	OnlineAny(ctx context.Context, jobID int64) (int64, error)
	OnlineInScope(ctx context.Context, jobID int64, deptIDs, locIDs []uint64) (int64, error)
	GetByID(ctx context.Context, jobID int64) (*model.Job, error)
	GetOwned(ctx context.Context, hrID, jobID int64) (*model.Job, error)
	SearchByHR(ctx context.Context, hrID int64, keyword string, status *int32, page, pageSize int32) ([]model.Job, int64, error)
	ListByHRCursor(ctx context.Context, hrID int64, cursor string, limit int32) ([]model.Job, string, bool, error)
	ListByScopeCursor(ctx context.Context, hrID int64, deptIDs, locIDs []uint64, cursor string, limit int32) ([]model.Job, string, bool, error)
	ListPublicCursor(ctx context.Context, keyword string, cursor string, limit int32) ([]model.Job, string, bool, error)
	ApplicationCount(ctx context.Context, jobID int64) (int64, error)
	BatchApplicationCounts(ctx context.Context, jobIDs []int64) (map[int64]int64, error)
	LookupDepartment(ctx context.Context, id int64) (*model.Department, error)
	LookupLocation(ctx context.Context, id int64) (*model.JobLocation, error)
}

// DepartmentRepository is the Recruitment-owned department catalog port used by job workflows.
type DepartmentRepository interface {
	ListActive(ctx context.Context) ([]model.Department, error)
	GetByID(ctx context.Context, id int64) (*model.Department, error)
	CountJobReferences(ctx context.Context, deptID int64) (int64, error)
	SyncJobDepartmentText(ctx context.Context, deptID int64, fullName string) error
}

// LocationRepository is the Recruitment-owned job-location catalog port used by job workflows.
type LocationRepository interface {
	ListActive(ctx context.Context) ([]model.JobLocation, error)
	GetByID(ctx context.Context, id int64) (*model.JobLocation, error)
	CountJobReferences(ctx context.Context, locID int64) (int64, error)
	SyncJobLocationText(ctx context.Context, locID int64, name string) error
}

// JobReadModelRepository keeps candidate-facing job listing marks inside Recruitment.
type JobReadModelRepository interface {
	ListForCandidateWithApplicationMark(ctx context.Context, userID int64) ([]repository.JobWithApplicationMark, error)
	GetForCandidate(ctx context.Context, userID, jobID int64) (*model.Job, bool, error)
}
