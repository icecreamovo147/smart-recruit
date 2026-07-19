package persistence

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"smart-recruit-commons/oss"
	commonsquota "smart-recruit-commons/quota"
	"smart-recruit-platform-go/errs"
	"smart-recruit-platform-go/metadata"
	"smart-recruit-proto/recruitment/pb"
	"smart-recruit-recruitment-service/internal/application/service"
	domainmodel "smart-recruit-recruitment-service/internal/domain/model"
	domainpolicy "smart-recruit-recruitment-service/internal/domain/policy"
)

type NativeBundle struct {
	Job                      JobAPI
	JobTaxonomy              JobTaxonomyAPI
	TaxonomyAdmin            TaxonomyAdminAPI
	Admin                    RecruitmentAdminAPI
	UsageStats               UsageStatsAPI
	Candidate                CandidateAPI
	Application              ApplicationAPI
	ApplicationOwnerContract pb.ApplicationOwnerServiceServer
	Collaboration            pb.CollaborationServiceServer
}

type NativeOptions struct {
	DB      *gorm.DB
	OSS     oss.Storage
	Redis   *redis.Client
	NowFunc func() time.Time
}

type JobAPI interface {
	CreateJob(context.Context, *pb.CreateJobRequest) (*pb.CreateJobResponse, error)
	UpdateJob(context.Context, *pb.UpdateJobRequest) (*pb.CommonResponse, error)
	OfflineJob(context.Context, *pb.OfflineJobRequest) (*pb.CommonResponse, error)
	OnlineJob(context.Context, *pb.OfflineJobRequest) (*pb.CommonResponse, error)
	ListHRJobs(context.Context, *pb.ListHRJobsRequest) (*pb.ListJobsResponse, error)
	ListPublicJobs(context.Context, *pb.ListPublicJobsRequest) (*pb.ListJobsResponse, error)
	GetJobDetail(context.Context, *pb.GetJobDetailRequest) (*pb.GetJobDetailResponse, error)
}

type JobTaxonomyAPI interface {
	ListJobOptions(context.Context, *pb.ListJobOptionsRequest) (*pb.ListJobOptionsResponse, error)
	ListDepartmentLocations(context.Context, *pb.ListDepartmentLocationsRequest) (*pb.ListDepartmentLocationsResponse, error)
}

type TaxonomyAdminAPI interface {
	ListDepartments(context.Context, *pb.ListDepartmentsRequest) (*pb.ListDepartmentsResponse, error)
	CreateDepartment(context.Context, *pb.CreateDepartmentRequest) (*pb.DepartmentResponse, error)
	UpdateDepartment(context.Context, *pb.UpdateDepartmentRequest) (*pb.DepartmentResponse, error)
	UpdateDepartmentStatus(context.Context, *pb.UpdateDepartmentStatusRequest) (*pb.CommonResponse, error)
	DeleteDepartment(context.Context, *pb.DeleteDepartmentRequest) (*pb.CommonResponse, error)
	ListJobLocations(context.Context, *pb.ListJobLocationsRequest) (*pb.ListJobLocationsResponse, error)
	CreateJobLocation(context.Context, *pb.CreateJobLocationRequest) (*pb.JobLocationResponse, error)
	UpdateJobLocation(context.Context, *pb.UpdateJobLocationRequest) (*pb.JobLocationResponse, error)
	UpdateJobLocationStatus(context.Context, *pb.UpdateJobLocationStatusRequest) (*pb.CommonResponse, error)
	DeleteJobLocation(context.Context, *pb.DeleteJobLocationRequest) (*pb.CommonResponse, error)
	GetDepartmentLocationConfig(context.Context, *pb.GetDepartmentLocationConfigRequest) (*pb.DepartmentLocationConfigResponse, error)
	UpdateDepartmentLocationConfig(context.Context, *pb.UpdateDepartmentLocationConfigRequest) (*pb.DepartmentLocationConfigResponse, error)
	ListDepartmentsLocationMap(context.Context, *pb.ListDepartmentsLocationMapRequest) (*pb.ListDepartmentsLocationMapResponse, error)
}

type RecruitmentAdminAPI interface {
	CreateInviteCode(context.Context, *pb.CreateInviteCodeRequest) (*pb.CreateInviteCodeResponse, error)
	ListInviteCodes(context.Context, *pb.ListInviteCodesRequest) (*pb.ListInviteCodesResponse, error)
	ExtendInviteCode(context.Context, *pb.ExtendInviteCodeRequest) (*pb.CommonResponse, error)
	RevokeInviteCode(context.Context, *pb.RevokeInviteCodeRequest) (*pb.CommonResponse, error)
	ReactivateInviteCode(context.Context, *pb.ReactivateInviteCodeRequest) (*pb.CommonResponse, error)
	ValidateInviteCode(context.Context, *pb.ValidateInviteCodeRequest) (*pb.ValidateInviteCodeResponse, error)
	QueryUsageLogs(context.Context, *pb.QueryUsageLogsRequest) (*pb.QueryUsageLogsResponse, error)
}

type UsageStatsAPI interface {
	GetUsageStats(context.Context, *pb.GetUsageStatsRequest) (*pb.GetUsageStatsResponse, error)
	GetUsageTrend(context.Context, *pb.GetUsageTrendRequest) (*pb.GetUsageTrendResponse, error)
}

type CandidateAPI interface {
	GetProfile(context.Context, *pb.GetProfileRequest) (*pb.GetProfileResponse, error)
	UpdateProfile(context.Context, *pb.UpdateProfileRequest) (*pb.GetProfileResponse, error)
	GetResume(context.Context, *pb.GetResumeRequest) (*pb.GetResumeResponse, error)
	PresignResumeUpload(context.Context, *pb.PresignResumeUploadRequest) (*pb.PresignResumeUploadResponse, error)
	ConfirmResumeUpload(context.Context, *pb.ConfirmResumeUploadRequest) (*pb.ConfirmResumeUploadResponse, error)
}

type ApplicationAPI interface {
	ApplyJob(context.Context, *pb.ApplyJobRequest) (*pb.CommonResponse, error)
	ListMyApplications(context.Context, *pb.ListMyApplicationsRequest) (*pb.ListMyApplicationsResponse, error)
	ListJobApplications(context.Context, *pb.ListJobApplicationsRequest) (*pb.ListJobApplicationsResponse, error)
	UpdateApplicationStatus(context.Context, *pb.UpdateApplicationStatusRequest) (*pb.CommonResponse, error)
	ListApplicationStatusTransitions(context.Context, *pb.ListApplicationStatusTransitionsRequest) (*pb.ListApplicationStatusTransitionsResponse, error)
}

func NewNativeBundle(options NativeOptions) (*NativeBundle, error) {
	if options.DB == nil {
		return nil, fmt.Errorf("recruitment native db is required")
	}
	if options.OSS == nil {
		return nil, fmt.Errorf("recruitment native oss storage is required")
	}
	now := options.NowFunc
	if now == nil {
		now = time.Now
	}
	store := &nativeStore{db: options.DB, storage: options.OSS, redis: options.Redis, now: now, quota: commonsquota.NewChecker(options.DB)}
	application := &applicationAdapter{nativeStore: store}
	return &NativeBundle{
		Job:                      &jobAdapter{nativeStore: store},
		JobTaxonomy:              &jobTaxonomyAdapter{nativeStore: store},
		TaxonomyAdmin:            &taxonomyAdminAdapter{nativeStore: store},
		Admin:                    &adminAdapter{nativeStore: store},
		UsageStats:               &usageStatsAdapter{nativeStore: store},
		Candidate:                &candidateAdapter{nativeStore: store},
		Application:              application,
		ApplicationOwnerContract: &applicationOwnerAdapter{applicationAdapter: application},
		Collaboration:            &collaborationAdapter{nativeStore: store},
	}, nil
}

type nativeStore struct {
	db      *gorm.DB
	storage oss.Storage
	redis   *redis.Client
	now     func() time.Time
	quota   *commonsquota.Checker
}

type jobAdapter struct{ *nativeStore }

type jobTaxonomyAdapter struct{ *nativeStore }

type taxonomyAdminAdapter struct{ *nativeStore }

type adminAdapter struct{ *nativeStore }

type usageStatsAdapter struct{ *nativeStore }

type candidateAdapter struct{ *nativeStore }

type applicationAdapter struct{ *nativeStore }

type applicationOwnerAdapter struct {
	pb.UnimplementedApplicationOwnerServiceServer
	*applicationAdapter
}

type collaborationAdapter struct {
	pb.UnimplementedCollaborationServiceServer
	*nativeStore
}

var (
	_ JobAPI                           = (*jobAdapter)(nil)
	_ JobTaxonomyAPI                   = (*jobTaxonomyAdapter)(nil)
	_ TaxonomyAdminAPI                 = (*taxonomyAdminAdapter)(nil)
	_ RecruitmentAdminAPI              = (*adminAdapter)(nil)
	_ UsageStatsAPI                    = (*usageStatsAdapter)(nil)
	_ CandidateAPI                     = (*candidateAdapter)(nil)
	_ ApplicationAPI                   = (*applicationAdapter)(nil)
	_ pb.ApplicationOwnerServiceServer = (*applicationOwnerAdapter)(nil)
	_ pb.CollaborationServiceServer    = (*collaborationAdapter)(nil)
)

type jobRecord struct {
	ID           int64 `gorm:"primaryKey"`
	TenantID     int64
	HrID         int64 `gorm:"column:hr_id"`
	Title        string
	Department   string
	DepartmentID *int64 `gorm:"column:department_id"`
	Location     string
	LocationID   *int64 `gorm:"column:location_id"`
	SalaryRange  string
	Description  string
	Requirements string
	Status       int32
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (jobRecord) TableName() string { return "jobs" }

type departmentRecord struct {
	ID               int64 `gorm:"primaryKey"`
	TenantID         int64
	ParentID         int64
	Name             string
	FullName         string
	Path             string
	Depth            int
	SortOrder        int
	IsActive         int32
	InheritLocations int32
	CreatedBy        *int64
	UpdatedBy        *int64
	DeletedAt        *time.Time
	DeletedBy        *int64
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (departmentRecord) TableName() string { return "departments" }

type jobLocationRecord struct {
	ID        int64 `gorm:"primaryKey"`
	TenantID  int64
	Name      string
	Code      *string
	SortOrder int
	IsActive  int32
	CreatedBy *int64
	UpdatedBy *int64
	DeletedAt *time.Time
	DeletedBy *int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (jobLocationRecord) TableName() string { return "job_locations" }

type candidateProfileRecord struct {
	ID             int64 `gorm:"primaryKey"`
	UserID         int64
	RealName       string
	Phone          string
	Education      string
	School         string
	WorkExperience string
	Skills         string
	IsComplete     int32
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (candidateProfileRecord) TableName() string { return "candidate_profiles" }

type resumeRecord struct {
	ID         int64 `gorm:"primaryKey"`
	UserID     int64
	OSSKey     string
	FileName   string
	FileType   string
	FileSize   int64
	ParsedText string
	ParsedAt   *time.Time
	IsValid    int32
	UploadedAt time.Time
}

func (resumeRecord) TableName() string { return "resumes" }

type applicationRecord struct {
	ID        int64 `gorm:"primaryKey"`
	TenantID  int64
	UserID    int64
	JobID     int64
	ResumeID  int64
	Status    int32
	StatusKey string
	RoundNo   int32
	IsCurrent int32
	AppliedAt time.Time
	UpdatedAt time.Time
}

func (applicationRecord) TableName() string { return "applications" }

type applicationDetailRow struct {
	ApplicationID int64
	TenantID      int64
	UserID        int64
	JobID         int64
	JobTitle      string
	Department    string
	Location      string
	RealName      string
	Phone         string
	Education     string
	School        string
	Skills        string
	ResumeID      int64
	OSSKey        string
	FileName      string
	FileType      string
	Status        int32
	StatusKey     string
	RoundNo       int32
	IsCurrent     int32
	AppliedAt     time.Time
	UpdatedAt     time.Time
}

type applicationTransitionRecord struct {
	ID               uint64 `gorm:"primaryKey"`
	TenantID         int64
	ApplicationID    int64
	FromStatus       string
	ToStatus         string
	ActorUserID      int64
	ActorAccountType string
	Reason           string
	CreatedAt        time.Time
}

func (applicationTransitionRecord) TableName() string { return "application_status_transitions" }

type candidateWorkspaceInterviewRow struct {
	InterviewID                 int64
	ApplicationID               int64
	Title                       string
	Mode                        string
	Status                      string
	ScheduledAt                 *time.Time
	InterviewerID               int64
	JobTitle                    string
	RoundNo                     int32
	FeedbackRecommendation      string
	FeedbackScore               int32
	FeedbackDimensionScoresJSON string
	FeedbackComments            string
	HasFeedback                 bool
	CreatedAt                   time.Time
	UpdatedAt                   time.Time
}

type candidateWorkspaceOfferRow struct {
	OfferID       int64
	ApplicationID int64
	Title         string
	Status        string
	SalaryRange   string
	Level         string
	WorkLocation  string
	StartDate     string
	JobTitle      string
	CreatedBy     int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DecidedAt     *time.Time
}

type inviteCodeRecord struct {
	ID        int64 `gorm:"primaryKey"`
	TenantID  int64
	Code      string
	CreatedBy int64
	ExpiresAt *time.Time
	IsActive  int32
	CreatedAt time.Time
}

func (inviteCodeRecord) TableName() string { return "invite_codes" }

type usageLogRecord struct {
	ID              int64 `gorm:"primaryKey"`
	TenantID        *int64
	UserID          int64
	Role            int32
	ServiceType     string
	Endpoint        string
	Provider        string
	Model           string
	RequestChars    int32
	ResponseChars   int32
	EstimatedTokens int32
	ObjectKey       string
	ObjectSize      int64
	Status          string
	ErrorCode       string
	CostMs          int32
	RequestID       string
	IP              string
	CreatedAt       time.Time
}

func (usageLogRecord) TableName() string { return "third_party_usage_logs" }

type departmentLocationRecord struct {
	ID           int64 `gorm:"primaryKey"`
	TenantID     int64
	DepartmentID int64
	LocationID   int64
	IsActive     int32
	CreatedBy    *int64
	UpdatedBy    *int64
	DeletedAt    *time.Time
	DeletedBy    *int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (departmentLocationRecord) TableName() string { return "department_locations" }

type candidateNoteRecord struct {
	ID              uint64 `gorm:"primaryKey"`
	TenantID        int64
	CandidateUserID uint64
	ApplicationID   *uint64
	AuthorUserID    uint64
	Content         string
	Visibility      string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (candidateNoteRecord) TableName() string { return "candidate_notes" }

type candidateTagRecord struct {
	ID        uint64 `gorm:"primaryKey"`
	TenantID  int64
	Name      string
	Color     string
	CreatedBy *uint64
	CreatedAt time.Time
}

func (candidateTagRecord) TableName() string { return "candidate_tags" }

type candidateTagAssignmentRecord struct {
	ID              uint64 `gorm:"primaryKey"`
	TenantID        int64
	TagID           uint64
	CandidateUserID uint64
	CreatedBy       *uint64
	CreatedAt       time.Time
}

func (candidateTagAssignmentRecord) TableName() string { return "candidate_tag_assignments" }

type followUpTaskRecord struct {
	ID              uint64 `gorm:"primaryKey"`
	TenantID        int64
	CandidateUserID uint64
	ApplicationID   *uint64
	AssigneeUserID  uint64
	CreatedBy       uint64
	Title           string
	Description     string
	DueAt           *time.Time
	Status          string
	CompletedAt     *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (followUpTaskRecord) TableName() string { return "follow_up_tasks" }

type eventOutboxRecord struct {
	ID             uint64 `gorm:"primaryKey"`
	TenantID       *int64
	EventID        string
	SchemaVersion  string
	EventType      string
	AggregateType  string
	AggregateID    uint64
	RoutingKey     string
	Producer       string
	IdempotencyKey string
	Payload        string
	Metadata       string
	Status         int32
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (eventOutboxRecord) TableName() string { return "event_outbox" }

func (a *jobAdapter) CreateJob(ctx context.Context, req *pb.CreateJobRequest) (*pb.CreateJobResponse, error) {
	if req.HrId == 0 || strings.TrimSpace(req.Title) == "" {
		return &pb.CreateJobResponse{Code: errs.ErrBadRequest, Msg: "岗位名称不能为空"}, nil
	}
	if _, err := a.quota.Check(ctx, metadata.GetAuthTenantID(ctx), "jobs.published.max", 1); err != nil {
		if errors.Is(err, commonsquota.ErrLimitExceeded) {
			return &pb.CreateJobResponse{Code: errs.ErrForbidden, Msg: "已达到当前套餐的在线岗位上限"}, nil
		}
		return nil, err
	}
	department, location := strings.TrimSpace(req.Department), strings.TrimSpace(req.Location)
	if req.DepartmentId > 0 {
		dep, _ := a.lookupDepartment(ctx, req.DepartmentId)
		if dep != nil {
			department = dep.FullName
		}
	}
	if req.LocationId > 0 {
		loc, _ := a.lookupLocation(ctx, req.LocationId)
		if loc != nil {
			location = loc.Name
		}
	}
	if department == "" {
		return &pb.CreateJobResponse{Code: errs.ErrBadRequest, Msg: "请选择部门"}, nil
	}
	if location == "" {
		return &pb.CreateJobResponse{Code: errs.ErrBadRequest, Msg: "请选择地点"}, nil
	}
	if req.DepartmentId > 0 && req.LocationId > 0 {
		if err := a.validateDepartmentLocation(ctx, req.DepartmentId, req.LocationId); err != nil {
			return &pb.CreateJobResponse{Code: errs.ErrBadRequest, Msg: err.Error()}, nil
		}
	}
	job := &jobRecord{
		HrID:         req.HrId,
		Title:        strings.TrimSpace(req.Title),
		Department:   department,
		DepartmentID: positivePtr(req.DepartmentId),
		Location:     location,
		LocationID:   positivePtr(req.LocationId),
		SalaryRange:  req.SalaryRange,
		Description:  req.Description,
		Requirements: req.Requirements,
		Status:       1,
	}
	if err := a.db.WithContext(ctx).Create(job).Error; err != nil {
		return nil, err
	}
	return &pb.CreateJobResponse{Code: errs.OK, Msg: "success", JobId: job.ID}, nil
}

func (a *jobAdapter) UpdateJob(ctx context.Context, req *pb.UpdateJobRequest) (*pb.CommonResponse, error) {
	fields := map[string]any{}
	putTrimmed(fields, "title", req.Title)
	putTrimmed(fields, "salary_range", req.SalaryRange)
	putTrimmed(fields, "description", req.Description)
	putTrimmed(fields, "requirements", req.Requirements)
	if req.DepartmentId > 0 {
		dep, _ := a.lookupDepartment(ctx, req.DepartmentId)
		if dep != nil {
			fields["department"] = dep.FullName
			fields["department_id"] = req.DepartmentId
		}
	} else {
		putTrimmed(fields, "department", req.Department)
	}
	if req.LocationId > 0 {
		loc, _ := a.lookupLocation(ctx, req.LocationId)
		if loc != nil {
			fields["location"] = loc.Name
			fields["location_id"] = req.LocationId
		}
	} else {
		putTrimmed(fields, "location", req.Location)
	}
	if len(fields) == 0 {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "没有可更新字段"}, nil
	}
	scope, err := a.checkRecruitmentJobScope(ctx, req.HrId, req.JobId)
	if err != nil {
		return nil, err
	}
	if !scope.allowed() {
		return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: "无权限操作该岗位"}, nil
	}
	query := a.db.WithContext(ctx).Model(&jobRecord{}).Where("id = ?", req.JobId)
	query = applyRecruitmentScopeToJobMutationQuery(query, scope)
	result := query.Updates(fields)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: "无权限操作该岗位"}, nil
	}
	return &pb.CommonResponse{Code: errs.OK, Msg: "success"}, nil
}

func (a *jobAdapter) OfflineJob(ctx context.Context, req *pb.OfflineJobRequest) (*pb.CommonResponse, error) {
	return a.setJobStatus(ctx, req.HrId, req.JobId, 0, "岗位已下架")
}

func (a *jobAdapter) OnlineJob(ctx context.Context, req *pb.OfflineJobRequest) (*pb.CommonResponse, error) {
	return a.setJobStatus(ctx, req.HrId, req.JobId, 1, "岗位已上线")
}

func (a *jobAdapter) setJobStatus(ctx context.Context, hrID, jobID int64, status int32, msg string) (*pb.CommonResponse, error) {
	scope, err := a.checkRecruitmentJobScope(ctx, hrID, jobID)
	if err != nil {
		return nil, err
	}
	if !scope.allowed() {
		return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: "无权限操作该岗位"}, nil
	}
	if status == 1 && a.quota != nil {
		var job jobRecord
		query := applyRecruitmentScopeToJobsQuery(a.db.WithContext(ctx).Model(&jobRecord{}).Where("id = ?", jobID), scope)
		if err := query.First(&job).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: "无权限操作该岗位"}, nil
			}
			return nil, err
		}
		if job.Status == 1 {
			return &pb.CommonResponse{Code: errs.OK, Msg: msg}, nil
		}
		if _, err := a.quota.Check(ctx, job.TenantID, "jobs.published.max", 1); err != nil {
			if errors.Is(err, commonsquota.ErrLimitExceeded) {
				return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: "已达到当前套餐的在线岗位上限"}, nil
			}
			return nil, err
		}
	}
	query := a.db.WithContext(ctx).Model(&jobRecord{}).Where("id = ?", jobID)
	query = applyRecruitmentScopeToJobMutationQuery(query, scope)
	result := query.Update("status", status)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: "无权限操作该岗位"}, nil
	}
	return &pb.CommonResponse{Code: errs.OK, Msg: msg}, nil
}

func (a *jobAdapter) ListHRJobs(ctx context.Context, req *pb.ListHRJobsRequest) (*pb.ListJobsResponse, error) {
	scope, err := a.evaluateRecruitmentScope(ctx, req.HrId)
	if err != nil {
		return nil, err
	}
	if !scope.allowed() {
		return &pb.ListJobsResponse{Code: errs.ErrForbidden, Msg: "无数据范围权限"}, nil
	}
	query := a.db.WithContext(ctx).Model(&jobRecord{})
	query = applyRecruitmentScopeToJobsQuery(query, scope)
	return a.listJobs(query, page(req.Page), pageSize(req.PageSize))
}

func (a *jobAdapter) ListPublicJobs(ctx context.Context, req *pb.ListPublicJobsRequest) (*pb.ListJobsResponse, error) {
	query := a.db.WithContext(ctx).Model(&jobRecord{}).Where("status = ?", 1)
	keyword := strings.TrimSpace(req.Keyword)
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("title LIKE ? OR department LIKE ? OR location LIKE ?", like, like, like)
	}
	if ids := positiveIDs(req.GetDepartmentIds()); len(ids) > 0 {
		query = query.Where("department_id IN ?", ids)
	}
	if ids := positiveIDs(req.GetLocationIds()); len(ids) > 0 {
		query = query.Where("location_id IN ?", ids)
	}
	return a.listJobs(query, page(req.Page), pageSize(req.PageSize))
}

func (a *jobAdapter) GetJobDetail(ctx context.Context, req *pb.GetJobDetailRequest) (*pb.GetJobDetailResponse, error) {
	var job jobRecord
	err := a.db.WithContext(ctx).Where("id = ? AND status = ?", req.JobId, 1).First(&job).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &pb.GetJobDetailResponse{Code: errs.ErrBadRequest, Msg: "岗位不存在或已下架"}, nil
	}
	if err != nil {
		return nil, err
	}
	item, err := a.jobToPB(ctx, job)
	if err != nil {
		return nil, err
	}
	return &pb.GetJobDetailResponse{Code: errs.OK, Msg: "success", Job: item}, nil
}

func (a *jobAdapter) listJobs(query *gorm.DB, pageNum, size int32) (*pb.ListJobsResponse, error) {
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}
	var jobs []jobRecord
	if err := query.Order("id DESC").Offset(int((pageNum - 1) * size)).Limit(int(size)).Find(&jobs).Error; err != nil {
		return nil, err
	}
	list := make([]*pb.Job, 0, len(jobs))
	for _, job := range jobs {
		item, err := a.jobToPB(context.Background(), job)
		if err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return &pb.ListJobsResponse{Code: errs.OK, Msg: "success", Total: total, List: list}, nil
}

func (a *jobAdapter) jobToPB(ctx context.Context, job jobRecord) (*pb.Job, error) {
	var applications int64
	if err := a.db.WithContext(ctx).Model(&applicationRecord{}).Where("job_id = ?", job.ID).Count(&applications).Error; err != nil {
		return nil, err
	}
	return &pb.Job{
		JobId:            job.ID,
		HrId:             job.HrID,
		Title:            job.Title,
		Department:       job.Department,
		Location:         job.Location,
		SalaryRange:      job.SalaryRange,
		Description:      job.Description,
		Requirements:     job.Requirements,
		Status:           job.Status,
		ApplicationCount: applications,
		CreatedAt:        formatTime(job.CreatedAt),
		DepartmentId:     ptrValue(job.DepartmentID),
		LocationId:       ptrValue(job.LocationID),
	}, nil
}

func (a *candidateAdapter) GetProfile(ctx context.Context, req *pb.GetProfileRequest) (*pb.GetProfileResponse, error) {
	var profile candidateProfileRecord
	err := a.db.WithContext(ctx).Where("user_id = ?", req.UserId).First(&profile).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &pb.GetProfileResponse{Code: errs.OK, Msg: "success", Profile: &pb.CandidateProfile{}}, nil
	}
	if err != nil {
		return nil, err
	}
	return &pb.GetProfileResponse{Code: errs.OK, Msg: "success", Profile: profileToPB(profile)}, nil
}

func (a *candidateAdapter) UpdateProfile(ctx context.Context, req *pb.UpdateProfileRequest) (*pb.GetProfileResponse, error) {
	profile := candidateProfileRecord{
		UserID:         req.UserId,
		RealName:       req.RealName,
		Phone:          req.Phone,
		Education:      req.Education,
		School:         req.School,
		WorkExperience: req.WorkExperience,
		Skills:         req.Skills,
		IsComplete:     completeFlag(req.RealName, req.Phone, req.Education, req.School, req.WorkExperience, req.Skills),
	}
	err := a.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing candidateProfileRecord
		err := tx.Where("user_id = ?", req.UserId).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tx.Create(&profile).Error
		}
		if err != nil {
			return err
		}
		profile.ID = existing.ID
		return tx.Model(&candidateProfileRecord{}).Where("id = ?", existing.ID).Updates(map[string]any{
			"real_name": profile.RealName, "phone": profile.Phone, "education": profile.Education,
			"school": profile.School, "work_experience": profile.WorkExperience, "skills": profile.Skills,
			"is_complete": profile.IsComplete,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return &pb.GetProfileResponse{Code: errs.OK, Msg: "保存成功", Profile: profileToPB(profile)}, nil
}

func (a *candidateAdapter) GetResume(ctx context.Context, req *pb.GetResumeRequest) (*pb.GetResumeResponse, error) {
	if code, msg, ok := validateCandidateActor(ctx, req.UserId); !ok {
		return &pb.GetResumeResponse{Code: code, Msg: msg}, nil
	}
	var resume resumeRecord
	err := a.db.WithContext(ctx).Where("user_id = ? AND is_valid = ?", req.UserId, 1).Order("uploaded_at DESC, id DESC").First(&resume).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &pb.GetResumeResponse{Code: errs.OK, Msg: "success"}, nil
	}
	if err != nil {
		return nil, err
	}
	url, err := a.storage.GeneratePresignedGetURL(resume.OSSKey)
	if err != nil {
		return nil, err
	}
	return &pb.GetResumeResponse{Code: errs.OK, Msg: "success", Resume: &pb.CandidateResume{
		ResumeId: resume.ID, FileName: resume.FileName, FileType: resume.FileType, FileSize: resume.FileSize,
		UploadedAt: formatTime(resume.UploadedAt), ResumeUrl: url,
	}}, nil
}

func (a *candidateAdapter) PresignResumeUpload(ctx context.Context, req *pb.PresignResumeUploadRequest) (*pb.PresignResumeUploadResponse, error) {
	if code, msg, ok := validateCandidateActor(ctx, req.UserId); !ok {
		return &pb.PresignResumeUploadResponse{Code: code, Msg: msg}, nil
	}
	if !allowedResumeFile(req.FileName, req.FileType) {
		return &pb.PresignResumeUploadResponse{Code: errs.ErrBadRequest, Msg: "仅支持 PDF、DOCX 格式"}, nil
	}
	uploadID, err := oss.GenerateUploadID()
	if err != nil {
		return nil, err
	}
	ossKey := fmt.Sprintf("resumes/tmp/%d/%s/%s", req.UserId, uploadID, sanitizeFileName(req.FileName))
	contentType := contentTypeFromFileType(req.FileType)
	uploadURL, expireAt, err := a.storage.GeneratePresignedPutURL(ossKey, contentType)
	if err != nil {
		return nil, err
	}
	if err := a.storage.SavePresignSessionWithID(ctx, uploadID, oss.PresignSession{
		UserID: req.UserId, OssKey: ossKey, FileName: req.FileName, FileType: req.FileType, ContentType: contentType, MaxSize: oss.MaxResumeSizeBytes, Status: "pending",
	}); err != nil {
		return nil, err
	}
	_ = a.writeUsageLog(ctx, usageLogRecord{UserID: req.UserId, Role: 1, ServiceType: "oss_presign", Endpoint: "/candidate/resume/presign", Provider: a.storage.ProviderName(), ObjectKey: ossKey, ObjectSize: oss.MaxResumeSizeBytes, Status: "ok"})
	return &pb.PresignResumeUploadResponse{Code: errs.OK, Msg: "success", UploadUrl: uploadURL, OssKey: ossKey, ExpireAt: formatTime(expireAt), UploadId: uploadID}, nil
}

func (a *candidateAdapter) ConfirmResumeUpload(ctx context.Context, req *pb.ConfirmResumeUploadRequest) (*pb.ConfirmResumeUploadResponse, error) {
	if code, msg, ok := validateCandidateActor(ctx, req.UserId); !ok {
		return &pb.ConfirmResumeUploadResponse{Code: code, Msg: msg}, nil
	}
	if !allowedResumeFile(req.FileName, req.FileType) {
		return &pb.ConfirmResumeUploadResponse{Code: errs.ErrBadRequest, Msg: "仅支持 PDF、DOCX 格式"}, nil
	}
	if req.UploadId == "" {
		return &pb.ConfirmResumeUploadResponse{Code: errs.ErrBadRequest, Msg: domainpolicy.ErrResumeUploadIDRequired.Error()}, nil
	}
	session, err := a.storage.GetAndDeletePresignSession(ctx, req.UploadId)
	if err != nil {
		return &pb.ConfirmResumeUploadResponse{Code: errs.ErrBadRequest, Msg: domainpolicy.ErrResumeSessionInvalid.Error()}, nil
	}
	if err := domainpolicy.ValidateStrictResumeSession(req.UserId, req.UploadId, req.OssKey, req.FileType, req.FileSize, ossPresignSessionToDomain(session)); err != nil {
		return &pb.ConfirmResumeUploadResponse{Code: errs.ErrBadRequest, Msg: err.Error()}, nil
	}
	if err := a.storage.VerifyObject(ctx, req.OssKey); err != nil {
		return &pb.ConfirmResumeUploadResponse{Code: errs.ErrBadRequest, Msg: "未在 OSS 中找到已上传的简历文件"}, nil
	}
	if err := a.storage.VerifyObjectSize(ctx, req.OssKey, oss.MaxResumeSizeBytes); err != nil {
		return &pb.ConfirmResumeUploadResponse{Code: errs.ErrBadRequest, Msg: "简历文件大小超过限制（最大 20MB）"}, nil
	}
	permanentKey := fmt.Sprintf("resumes/%d/%d_%s", req.UserId, a.now().Unix(), sanitizeFileName(req.FileName))
	if err := a.storage.CopyObject(ctx, req.OssKey, permanentKey); err != nil {
		return &pb.ConfirmResumeUploadResponse{Code: errs.ErrInternal, Msg: "简历保存失败，请重新上传"}, nil
	}
	_ = a.storage.DeleteObject(ctx, req.OssKey)
	resume := &resumeRecord{UserID: req.UserId, OSSKey: permanentKey, FileName: req.FileName, FileType: strings.ToLower(req.FileType), FileSize: req.FileSize, IsValid: 1, UploadedAt: a.now()}
	if err := a.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&resumeRecord{}).Where("user_id = ?", req.UserId).Update("is_valid", 0).Error; err != nil {
			return err
		}
		if err := tx.Create(resume).Error; err != nil {
			return err
		}
		return a.writeOutboxTx(tx, "resume.parse", "resume", uint64(resume.ID), "resume.parse", map[string]any{"resume_id": resume.ID, "file_type": resume.FileType, "oss_key": resume.OSSKey})
	}); err != nil {
		return nil, err
	}
	_ = a.writeUsageLog(ctx, usageLogRecord{UserID: req.UserId, Role: 1, ServiceType: "oss_confirm", Endpoint: "/candidate/resume/confirm", Provider: a.storage.ProviderName(), ObjectKey: req.OssKey, ObjectSize: req.FileSize, Status: "ok"})
	return &pb.ConfirmResumeUploadResponse{Code: errs.OK, Msg: "success", ResumeId: resume.ID}, nil
}

func validateCandidateActor(ctx context.Context, userID int64) (int32, string, bool) {
	if userID <= 0 {
		return errs.ErrBadRequest, "用户信息无效", false
	}
	if metadata.GetAuthAccountType(ctx) != "candidate" || metadata.GetAuthUserID(ctx) != userID {
		return errs.ErrForbidden, "无权限访问该候选人资源", false
	}
	return errs.OK, "", true
}

func ossPresignSessionToDomain(session *oss.PresignSession) *domainmodel.PresignSession {
	if session == nil {
		return nil
	}
	return &domainmodel.PresignSession{
		UserID:      session.UserID,
		OSSKey:      session.OssKey,
		FileName:    session.FileName,
		FileType:    session.FileType,
		ContentType: session.ContentType,
		MaxSize:     session.MaxSize,
		Status:      session.Status,
	}
}

func (a *applicationAdapter) ApplyJob(ctx context.Context, req *pb.ApplyJobRequest) (*pb.CommonResponse, error) {
	var profile candidateProfileRecord
	if err := a.db.WithContext(ctx).Where("user_id = ?", req.UserId).First(&profile).Error; err != nil || profile.IsComplete != 1 {
		return &pb.CommonResponse{Code: errs.ErrProfileIncomplete, Msg: "请先完善个人资料后再投递"}, nil
	}
	var resume resumeRecord
	if err := a.db.WithContext(ctx).Where("user_id = ? AND is_valid = ?", req.UserId, 1).Order("uploaded_at DESC, id DESC").First(&resume).Error; err != nil {
		return &pb.CommonResponse{Code: errs.ErrResumeNotFound, Msg: "请先上传简历后再投递"}, nil
	}
	var job jobRecord
	if err := a.db.WithContext(ctx).Where("id = ? AND status = ?", req.JobId, 1).First(&job).Error; err != nil {
		return &pb.CommonResponse{Code: errs.ErrJobNotAvailable, Msg: "该岗位已下架或不存在，无法投递"}, nil
	}
	if _, err := a.quota.Check(ctx, job.TenantID, "applications.monthly.max", 1); err != nil {
		if errors.Is(err, commonsquota.ErrLimitExceeded) {
			return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: "该企业已达到当前套餐的月投递上限"}, nil
		}
		return nil, err
	}
	var resumeAlreadyCounted int64
	if err := a.db.WithContext(ctx).Table("applications").Where("tenant_id = ? AND resume_id = ?", job.TenantID, resume.ID).Count(&resumeAlreadyCounted).Error; err != nil {
		return nil, err
	}
	if resumeAlreadyCounted == 0 {
		if _, err := a.quota.Check(ctx, job.TenantID, "resumes.storage.max", 1); err != nil {
			if errors.Is(err, commonsquota.ErrLimitExceeded) {
				return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: "该企业已达到当前套餐的简历存储上限"}, nil
			}
			return nil, err
		}
	}
	now := a.now()
	app := &applicationRecord{TenantID: job.TenantID, UserID: req.UserId, JobID: req.JobId, ResumeID: resume.ID, Status: 0, StatusKey: domainmodel.StatusKeyApplied, RoundNo: 1, IsCurrent: 1, AppliedAt: now, UpdatedAt: now}
	err := a.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(app).Error; err != nil {
			return err
		}
		content := fmt.Sprintf("%s 投递了「%s」岗位，请及时查看简历。", candidateDisplayName(profile.RealName, req.UserId), job.Title)
		if err := a.writeOutboxTx(tx, "application.notification_requested", "application", uint64(app.ID), "notification.create", notificationPayload(job.HrID, "staff", "new_application", "新的岗位投递", content, fmt.Sprintf("/hr/jobs/%d/applications", job.ID), "application", app.ID, job.Title, "")); err != nil {
			return err
		}
		return a.writeOutboxTx(tx, "application.email_requested", "application", uint64(app.ID), "email.send", notificationPayload(job.HrID, "staff", "new_application", "新的岗位投递", content, fmt.Sprintf("/hr/jobs/%d/applications", job.ID), "application", app.ID, job.Title, ""))
	})
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return &pb.CommonResponse{Code: errs.ErrDuplicateApply, Msg: "您已投递过该岗位，当前流程结束前不能重复投递"}, nil
	}
	if err != nil {
		return nil, err
	}
	return &pb.CommonResponse{Code: errs.OK, Msg: "投递成功"}, nil
}

func (a *applicationAdapter) ListMyApplications(ctx context.Context, req *pb.ListMyApplicationsRequest) (*pb.ListMyApplicationsResponse, error) {
	var rows []applicationDetailRow
	query := a.applicationDetails().Where("a.user_id = ?", req.UserId)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}
	if err := query.Order("a.applied_at DESC, a.id DESC").Offset(int((page(req.Page) - 1) * pageSize(req.PageSize))).Limit(int(pageSize(req.PageSize))).Scan(&rows).Error; err != nil {
		return nil, err
	}
	list := make([]*pb.MyApplication, 0, len(rows))
	for _, row := range rows {
		list = append(list, &pb.MyApplication{ApplicationId: row.ApplicationID, JobId: row.JobID, JobTitle: row.JobTitle, Status: row.Status, StatusKey: row.StatusKey, AppliedAt: formatTime(row.AppliedAt), RoundNo: row.RoundNo, IsCurrent: row.IsCurrent})
	}
	return &pb.ListMyApplicationsResponse{Code: errs.OK, Msg: "success", Total: total, List: list}, nil
}

func (a *applicationAdapter) ListJobApplications(ctx context.Context, req *pb.ListJobApplicationsRequest) (*pb.ListJobApplicationsResponse, error) {
	scope, err := a.checkRecruitmentJobScope(ctx, req.HrId, req.JobId)
	if err != nil {
		return nil, err
	}
	if !scope.allowed() {
		return &pb.ListJobApplicationsResponse{Code: errs.ErrForbidden, Msg: "无权限查看该岗位"}, nil
	}
	var rows []applicationDetailRow
	query := a.applicationDetails().Where("a.job_id = ?", req.JobId)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}
	if err := query.Order("a.applied_at DESC, a.id DESC").Offset(int((page(req.Page) - 1) * pageSize(req.PageSize))).Limit(int(pageSize(req.PageSize))).Scan(&rows).Error; err != nil {
		return nil, err
	}
	list := make([]*pb.JobApplication, 0, len(rows))
	for _, row := range rows {
		url := ""
		if row.OSSKey != "" {
			url, _ = a.storage.GeneratePresignedGetURL(row.OSSKey)
		}
		list = append(list, &pb.JobApplication{
			ApplicationId: row.ApplicationID, UserId: row.UserID, RealName: row.RealName, Phone: row.Phone, Education: row.Education, School: row.School,
			Skills: splitSkills(row.Skills), AppliedAt: formatTime(row.AppliedAt), ResumeUrl: url, Status: row.Status, StatusKey: row.StatusKey,
			RoundNo: row.RoundNo, IsCurrent: row.IsCurrent, FileName: row.FileName, FileType: row.FileType,
		})
	}
	return &pb.ListJobApplicationsResponse{Code: errs.OK, Msg: "success", Total: total, List: list}, nil
}

func (a *applicationAdapter) UpdateApplicationStatus(ctx context.Context, req *pb.UpdateApplicationStatusRequest) (*pb.CommonResponse, error) {
	actorAccountType, err := normalizePublicActorAccountType("")
	if err != nil {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: err.Error()}, nil
	}
	_, resp, err := a.applyApplicationStatusChange(ctx, applicationStatusChangeCommand{
		actorUserID:        req.HrId,
		actorAccountType:   actorAccountType,
		applicationID:      req.ApplicationId,
		targetStatusKey:    req.StatusKey,
		legacyTargetStatus: req.Status,
		reason:             req.Reason,
	})
	return resp, err
}

type applicationStatusChangeCommand struct {
	actorUserID        int64
	actorAccountType   string
	trustedOwnerCall   bool
	applicationID      int64
	targetStatusKey    string
	legacyTargetStatus int32
	reason             string
	closeCurrentRound  bool
}

type applicationStatusChangeResult struct {
	FromStatusKey    string
	CurrentStatusKey string
}

func (a *applicationAdapter) applyApplicationStatusChange(ctx context.Context, cmd applicationStatusChangeCommand) (applicationStatusChangeResult, *pb.CommonResponse, error) {
	targetKey, err := domainpolicy.TargetStatusKey(cmd.targetStatusKey, cmd.legacyTargetStatus)
	if err != nil {
		return applicationStatusChangeResult{}, &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: err.Error()}, nil
	}
	actorAccountType, err := normalizeActorAccountType(cmd.actorAccountType)
	if err != nil {
		return applicationStatusChangeResult{}, &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: err.Error()}, nil
	}
	detail, err := a.getApplicationDetail(ctx, cmd.applicationID)
	if err != nil {
		return applicationStatusChangeResult{}, nil, err
	}
	if detail == nil {
		return applicationStatusChangeResult{}, &pb.CommonResponse{Code: errs.ErrForbidden, Msg: "该投递记录不存在或无权限访问"}, nil
	}
	var staffScope recruitmentScope
	enforceStaffScope := actorAccountType == "staff" && !cmd.trustedOwnerCall
	if enforceStaffScope {
		staffScope, err = a.checkRecruitmentJobScope(ctx, cmd.actorUserID, detail.JobID)
		if err != nil {
			return applicationStatusChangeResult{}, nil, err
		}
		if !staffScope.allowed() {
			return applicationStatusChangeResult{}, &pb.CommonResponse{Code: errs.ErrForbidden, Msg: "无权限操作投递状态"}, nil
		}
	}
	domainDetail := domainmodel.ApplicationDetail{
		ApplicationID: detail.ApplicationID,
		UserID:        detail.UserID,
		JobID:         detail.JobID,
		JobTitle:      detail.JobTitle,
		RealName:      detail.RealName,
		ResumeID:      detail.ResumeID,
		Status:        detail.Status,
		StatusKey:     detail.StatusKey,
		RoundNo:       detail.RoundNo,
		IsCurrent:     detail.IsCurrent,
	}
	currentKey, isRePass, legacyStatus, err := domainpolicy.ValidateStatusChange(domainDetail, targetKey, cmd.reason)
	if err != nil {
		return applicationStatusChangeResult{}, statusChangeValidationResponse(err), nil
	}
	var rowsAffected int64
	err = a.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{"status": legacyStatus, "status_key": targetKey, "updated_at": a.now()}
		if isRePass {
			updates["round_no"] = gorm.Expr("round_no + 1")
			updates["is_current"] = 1
		} else if domainmodel.TerminalStatusKeys[targetKey] || cmd.closeCurrentRound {
			updates["is_current"] = 0
		}
		query := tx.Model(&applicationRecord{}).Where("id = ?", cmd.applicationID)
		if enforceStaffScope {
			query = applyRecruitmentScopeToApplicationMutationQuery(query, staffScope)
		}
		if detail.StatusKey == "" {
			query = query.Where("(status_key = ? OR status_key = '')", currentKey)
		} else {
			query = query.Where("status_key = ?", currentKey)
		}
		result := query.Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		rowsAffected = result.RowsAffected
		if result.RowsAffected == 0 {
			return service.ErrApplicationConflict
		}
		if targetKey == domainmodel.StatusKeyRejected || targetKey == domainmodel.StatusKeyWithdrawn {
			cancelReason := strings.TrimSpace(cmd.reason)
			if cancelReason == "" {
				cancelReason = "投递状态变更为" + domainmodel.HRStatusLabels[targetKey]
			}
			if err := a.cancelActiveInterviewsTx(tx, detail.TenantID, cmd.applicationID, cancelReason); err != nil {
				return err
			}
		}
		transition := &applicationTransitionRecord{
			TenantID:         detail.TenantID,
			ApplicationID:    cmd.applicationID,
			FromStatus:       currentKey,
			ToStatus:         targetKey,
			ActorUserID:      cmd.actorUserID,
			ActorAccountType: actorAccountType,
			Reason:           cmd.reason,
			CreatedAt:        a.now(),
		}
		if err := tx.Create(transition).Error; err != nil {
			return err
		}
		notifyType, content := domainpolicy.BuildApplicationNotification(targetKey, isRePass, domainDetail)
		if notifyType == "" {
			return nil
		}
		payload := notificationPayload(detail.UserID, "candidate", notifyType, "投递进展更新", content, "/applications", "application", cmd.applicationID, detail.JobTitle, detail.RealName)
		if err := a.writeOutboxTx(tx, "application.notification_requested", "application", uint64(cmd.applicationID), "notification.create", payload); err != nil {
			return err
		}
		return a.writeOutboxTx(tx, "application.email_requested", "application", uint64(cmd.applicationID), "email.send", payload)
	})
	if errors.Is(err, service.ErrApplicationConflict) {
		return applicationStatusChangeResult{}, &pb.CommonResponse{Code: errs.ErrConflict, Msg: service.ErrApplicationConflict.Error()}, nil
	}
	if err != nil {
		return applicationStatusChangeResult{}, nil, err
	}
	if rowsAffected == 0 {
		return applicationStatusChangeResult{}, &pb.CommonResponse{Code: errs.ErrConflict, Msg: service.ErrApplicationConflict.Error()}, nil
	}
	return applicationStatusChangeResult{FromStatusKey: currentKey, CurrentStatusKey: targetKey}, &pb.CommonResponse{Code: errs.OK, Msg: "投递状态已更新"}, nil
}

func statusChangeValidationResponse(err error) *pb.CommonResponse {
	var transitionErr *domainpolicy.TransitionError
	switch {
	case errors.As(err, &transitionErr), errors.Is(err, domainpolicy.ErrReasonRequired):
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: err.Error()}
	case errors.Is(err, domainpolicy.ErrApplicationNotCurrent):
		return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: err.Error()}
	default:
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: err.Error()}
	}
}

func normalizePublicActorAccountType(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "staff", nil
	}
	return normalizeActorAccountType(value)
}

func normalizeOwnerActorAccountType(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("actor_account_type 不能为空")
	}
	return normalizeActorAccountType(value)
}

func normalizeActorAccountType(value string) (string, error) {
	value = strings.TrimSpace(value)
	switch value {
	case "candidate", "staff", "service":
		return value, nil
	default:
		return "", fmt.Errorf("actor_account_type 不合法")
	}
}

func (a *nativeStore) cancelActiveInterviewsTx(tx *gorm.DB, tenantID, applicationID int64, reason string) error {
	return tx.Table("interview_schedules").
		Where("tenant_id = ? AND application_id = ? AND status IN ? AND deleted_at IS NULL", tenantID, applicationID, []string{"pending", "scheduled"}).
		Updates(map[string]any{
			"status":        "cancelled",
			"cancel_reason": reason,
		}).Error
}

func (a *applicationAdapter) ListApplicationStatusTransitions(ctx context.Context, req *pb.ListApplicationStatusTransitionsRequest) (*pb.ListApplicationStatusTransitionsResponse, error) {
	detail, err := a.getApplicationDetail(ctx, req.ApplicationId)
	if err != nil {
		return nil, err
	}
	if detail == nil {
		return &pb.ListApplicationStatusTransitionsResponse{Code: errs.ErrForbidden, Msg: "该投递记录不存在"}, nil
	}
	scope, err := a.checkRecruitmentJobScope(ctx, req.HrId, detail.JobID)
	if err != nil {
		return nil, err
	}
	if !scope.allowed() {
		return &pb.ListApplicationStatusTransitionsResponse{Code: errs.ErrForbidden, Msg: "无权限查看投递状态变更记录"}, nil
	}
	var rows []applicationTransitionRecord
	if err := a.db.WithContext(ctx).Where("application_id = ?", req.ApplicationId).Order("created_at ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	list := make([]*pb.ApplicationStatusTransition, 0, len(rows))
	for _, row := range rows {
		list = append(list, &pb.ApplicationStatusTransition{Id: int64(row.ID), ApplicationId: row.ApplicationID, FromStatus: row.FromStatus, ToStatus: row.ToStatus, ActorUserId: row.ActorUserID, ActorAccountType: row.ActorAccountType, Reason: row.Reason, CreatedAt: row.CreatedAt.Format("2006-01-02 15:04:05")})
	}
	return &pb.ListApplicationStatusTransitionsResponse{Code: errs.OK, Msg: "success", List: list}, nil
}

func (a *applicationOwnerAdapter) GetApplicationSnapshot(ctx context.Context, req *pb.GetApplicationSnapshotRequest) (*pb.GetApplicationSnapshotResponse, error) {
	detail, err := a.getApplicationDetail(ctx, req.ApplicationId)
	if err != nil {
		return nil, err
	}
	if detail == nil {
		return &pb.GetApplicationSnapshotResponse{Code: errs.ErrBadRequest, Msg: "application not found"}, nil
	}
	var job jobRecord
	if err := a.db.WithContext(ctx).Where("id = ?", detail.JobID).First(&job).Error; err != nil {
		return nil, err
	}
	ok, err := a.contextStaffJobScope(ctx, job.ID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return &pb.GetApplicationSnapshotResponse{Code: errs.ErrForbidden, Msg: "无权限访问该投递"}, nil
	}
	statusKey := detail.StatusKey
	if statusKey == "" {
		statusKey = domainmodel.LegacyStatusToKey[detail.Status]
	}
	return &pb.GetApplicationSnapshotResponse{
		Code: errs.OK, Msg: "success", ApplicationId: detail.ApplicationID, CandidateUserId: detail.UserID, JobId: detail.JobID, JobTitle: detail.JobTitle,
		CandidateName: detail.RealName, ResumeId: detail.ResumeID, LegacyStatus: detail.Status, StatusKey: statusKey, RoundNo: detail.RoundNo,
		IsCurrent: detail.IsCurrent == 1, JobHrId: job.HrID, DepartmentId: ptrValue(job.DepartmentID), LocationId: ptrValue(job.LocationID),
	}, nil
}

func (a *applicationOwnerAdapter) ApplyApplicationLifecycleTransition(ctx context.Context, req *pb.ApplyApplicationLifecycleTransitionRequest) (*pb.ApplyApplicationLifecycleTransitionResponse, error) {
	actorAccountType, err := normalizeOwnerActorAccountType(req.ActorAccountType)
	if err != nil {
		return &pb.ApplyApplicationLifecycleTransitionResponse{Code: errs.ErrBadRequest, Msg: err.Error()}, nil
	}
	if req.ExpectedStatusKey != "" {
		if err := domainpolicy.ValidateStatusKey(req.ExpectedStatusKey); err != nil {
			return &pb.ApplyApplicationLifecycleTransitionResponse{Code: errs.ErrBadRequest, Msg: err.Error()}, nil
		}
	}
	snapshot, err := a.GetApplicationSnapshot(ctx, &pb.GetApplicationSnapshotRequest{ApplicationId: req.ApplicationId})
	if err != nil || snapshot.Code != errs.OK {
		return &pb.ApplyApplicationLifecycleTransitionResponse{Code: snapshot.GetCode(), Msg: snapshot.GetMsg()}, err
	}
	if req.ExpectedStatusKey != "" && snapshot.StatusKey != req.ExpectedStatusKey {
		return &pb.ApplyApplicationLifecycleTransitionResponse{Code: errs.ErrConflict, Msg: "投递状态已变化，请刷新后重试"}, nil
	}
	result, resp, err := a.applyApplicationStatusChange(ctx, applicationStatusChangeCommand{
		actorUserID:        req.ActorUserId,
		actorAccountType:   actorAccountType,
		trustedOwnerCall:   isTrustedLifecycleOwner(ctx),
		applicationID:      req.ApplicationId,
		targetStatusKey:    req.TargetStatusKey,
		legacyTargetStatus: req.LegacyTargetStatus,
		reason:             req.Reason,
		closeCurrentRound:  req.CloseCurrentRound,
	})
	if err != nil {
		return nil, err
	}
	if resp.Code != errs.OK {
		return &pb.ApplyApplicationLifecycleTransitionResponse{Code: resp.Code, Msg: resp.Msg}, nil
	}
	return &pb.ApplyApplicationLifecycleTransitionResponse{Code: errs.OK, Msg: "success", Changed: true, FromStatusKey: result.FromStatusKey, CurrentStatusKey: result.CurrentStatusKey}, nil
}

func isTrustedLifecycleOwner(ctx context.Context) bool {
	return metadata.GetAuthAccountType(ctx) == "service" &&
		metadata.GetAuthClientApp(ctx) == "interview-service"
}

func (a *nativeStore) applicationDetails() *gorm.DB {
	return a.db.Table("applications a").
		Select(`a.id AS application_id, a.tenant_id, a.user_id, a.job_id, j.title AS job_title, COALESCE(j.department, '') AS department, COALESCE(j.location, '') AS location, COALESCE(cp.real_name, CONCAT('候选人', a.user_id)) AS real_name,
			COALESCE(cp.phone, '') AS phone, COALESCE(cp.education, '') AS education, COALESCE(cp.school, '') AS school, COALESCE(cp.skills, '') AS skills,
			a.resume_id, COALESCE(r.oss_key, '') AS oss_key, COALESCE(r.file_name, '') AS file_name, COALESCE(r.file_type, '') AS file_type,
			a.status, a.status_key, a.round_no, a.is_current, a.applied_at, a.updated_at`).
		Joins("JOIN jobs j ON j.id = a.job_id").
		Joins("LEFT JOIN candidate_profiles cp ON cp.user_id = a.user_id").
		Joins("LEFT JOIN resumes r ON r.id = a.resume_id")
}

func (a *nativeStore) getApplicationDetail(ctx context.Context, applicationID int64) (*applicationDetailRow, error) {
	var detail applicationDetailRow
	err := a.applicationDetails().WithContext(ctx).Where("a.id = ?", applicationID).Scan(&detail).Error
	if err != nil {
		return nil, err
	}
	if detail.ApplicationID == 0 {
		return nil, nil
	}
	return &detail, nil
}

func (a *jobTaxonomyAdapter) ListJobOptions(ctx context.Context, _ *pb.ListJobOptionsRequest) (*pb.ListJobOptionsResponse, error) {
	departments, err := a.listDepartmentRecords(ctx)
	if err != nil {
		return nil, err
	}
	locations, err := a.listLocationRecords(ctx, false)
	if err != nil {
		return nil, err
	}
	maps, err := a.listDepartmentLocationMaps(ctx)
	if err != nil {
		return nil, err
	}
	return &pb.ListJobOptionsResponse{Code: errs.OK, Msg: "success", DepartmentTree: departmentTree(departments), Locations: locationsToPB(locations), DepartmentLocationMap: maps}, nil
}

func (a *jobTaxonomyAdapter) ListDepartmentLocations(ctx context.Context, req *pb.ListDepartmentLocationsRequest) (*pb.ListDepartmentLocationsResponse, error) {
	locations, err := a.effectiveLocations(ctx, req.DepartmentId)
	if err != nil {
		return nil, err
	}
	return &pb.ListDepartmentLocationsResponse{Code: errs.OK, Msg: "success", DepartmentId: req.DepartmentId, Locations: locationsToPB(locations)}, nil
}

func (a *taxonomyAdminAdapter) ListDepartments(ctx context.Context, _ *pb.ListDepartmentsRequest) (*pb.ListDepartmentsResponse, error) {
	departments, err := a.listDepartmentRecords(ctx)
	if err != nil {
		return nil, err
	}
	return &pb.ListDepartmentsResponse{Code: errs.OK, Msg: "success", List: departmentTree(departments)}, nil
}

func (a *taxonomyAdminAdapter) CreateDepartment(ctx context.Context, req *pb.CreateDepartmentRequest) (*pb.DepartmentResponse, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return &pb.DepartmentResponse{Code: errs.ErrBadRequest, Msg: "部门名称不能为空"}, nil
	}
	dep := &departmentRecord{ParentID: req.ParentId, Name: name, SortOrder: int(req.SortOrder), IsActive: 1, InheritLocations: 1, CreatedBy: positivePtr(req.AdminId)}
	if err := a.fillDepartmentPath(ctx, dep); err != nil {
		return nil, err
	}
	if err := a.db.WithContext(ctx).Create(dep).Error; err != nil {
		return nil, err
	}
	return &pb.DepartmentResponse{Code: errs.OK, Msg: "success", Department: departmentToPB(*dep)}, nil
}

func (a *taxonomyAdminAdapter) UpdateDepartment(ctx context.Context, req *pb.UpdateDepartmentRequest) (*pb.DepartmentResponse, error) {
	var dep departmentRecord
	err := a.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", req.Id).First(&dep).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &pb.DepartmentResponse{Code: errs.ErrBadRequest, Msg: "部门不存在"}, nil
	}
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Name) != "" {
		dep.Name = strings.TrimSpace(req.Name)
	}
	dep.ParentID = req.ParentId
	dep.SortOrder = int(req.SortOrder)
	dep.UpdatedBy = positivePtr(req.AdminId)
	if err := a.fillDepartmentPath(ctx, &dep); err != nil {
		return nil, err
	}
	if err := a.db.WithContext(ctx).Save(&dep).Error; err != nil {
		return nil, err
	}
	return &pb.DepartmentResponse{Code: errs.OK, Msg: "success", Department: departmentToPB(dep)}, nil
}

func (a *taxonomyAdminAdapter) UpdateDepartmentStatus(ctx context.Context, req *pb.UpdateDepartmentStatusRequest) (*pb.CommonResponse, error) {
	result := a.db.WithContext(ctx).Model(&departmentRecord{}).Where("id = ? AND deleted_at IS NULL", req.Id).Updates(map[string]any{"is_active": req.IsActive, "updated_by": req.AdminId})
	return rowsCommon(result, "success", "部门不存在")
}

func (a *taxonomyAdminAdapter) DeleteDepartment(ctx context.Context, req *pb.DeleteDepartmentRequest) (*pb.CommonResponse, error) {
	now := a.now()
	result := a.db.WithContext(ctx).Model(&departmentRecord{}).Where("id = ? AND deleted_at IS NULL", req.Id).Updates(map[string]any{"deleted_at": now, "deleted_by": req.AdminId})
	return rowsCommon(result, "success", "部门不存在")
}

func (a *taxonomyAdminAdapter) ListJobLocations(ctx context.Context, _ *pb.ListJobLocationsRequest) (*pb.ListJobLocationsResponse, error) {
	locations, err := a.listLocationRecords(ctx, true)
	if err != nil {
		return nil, err
	}
	return &pb.ListJobLocationsResponse{Code: errs.OK, Msg: "success", List: locationsToPB(locations)}, nil
}

func (a *taxonomyAdminAdapter) CreateJobLocation(ctx context.Context, req *pb.CreateJobLocationRequest) (*pb.JobLocationResponse, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return &pb.JobLocationResponse{Code: errs.ErrBadRequest, Msg: "地点名称不能为空"}, nil
	}
	loc := &jobLocationRecord{Name: name, Code: stringPtr(strings.TrimSpace(req.Code)), SortOrder: int(req.SortOrder), IsActive: 1, CreatedBy: positivePtr(req.AdminId)}
	if err := a.db.WithContext(ctx).Create(loc).Error; err != nil {
		return nil, err
	}
	return &pb.JobLocationResponse{Code: errs.OK, Msg: "success", Location: locationToPB(*loc)}, nil
}

func (a *taxonomyAdminAdapter) UpdateJobLocation(ctx context.Context, req *pb.UpdateJobLocationRequest) (*pb.JobLocationResponse, error) {
	fields := map[string]any{"updated_by": req.AdminId}
	putTrimmed(fields, "name", req.Name)
	fields["code"] = strings.TrimSpace(req.Code)
	fields["sort_order"] = req.SortOrder
	result := a.db.WithContext(ctx).Model(&jobLocationRecord{}).Where("id = ? AND deleted_at IS NULL", req.Id).Updates(fields)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return &pb.JobLocationResponse{Code: errs.ErrBadRequest, Msg: "地点不存在"}, nil
	}
	loc, err := a.lookupLocation(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &pb.JobLocationResponse{Code: errs.OK, Msg: "success", Location: locationToPB(*loc)}, nil
}

func (a *taxonomyAdminAdapter) UpdateJobLocationStatus(ctx context.Context, req *pb.UpdateJobLocationStatusRequest) (*pb.CommonResponse, error) {
	result := a.db.WithContext(ctx).Model(&jobLocationRecord{}).Where("id = ? AND deleted_at IS NULL", req.Id).Updates(map[string]any{"is_active": req.IsActive, "updated_by": req.AdminId})
	return rowsCommon(result, "success", "地点不存在")
}

func (a *taxonomyAdminAdapter) DeleteJobLocation(ctx context.Context, req *pb.DeleteJobLocationRequest) (*pb.CommonResponse, error) {
	now := a.now()
	result := a.db.WithContext(ctx).Model(&jobLocationRecord{}).Where("id = ? AND deleted_at IS NULL", req.Id).Updates(map[string]any{"deleted_at": now, "deleted_by": req.AdminId})
	return rowsCommon(result, "success", "地点不存在")
}

func (a *taxonomyAdminAdapter) GetDepartmentLocationConfig(ctx context.Context, req *pb.GetDepartmentLocationConfigRequest) (*pb.DepartmentLocationConfigResponse, error) {
	dep, err := a.lookupDepartment(ctx, req.DepartmentId)
	if err != nil {
		return nil, err
	}
	if dep == nil {
		return &pb.DepartmentLocationConfigResponse{Code: errs.ErrBadRequest, Msg: "部门不存在"}, nil
	}
	direct, err := a.directLocationIDs(ctx, req.DepartmentId)
	if err != nil {
		return nil, err
	}
	effective, err := a.effectiveLocationIDs(ctx, *dep)
	if err != nil {
		return nil, err
	}
	locations, err := a.locationsByIDs(ctx, effective)
	if err != nil {
		return nil, err
	}
	available, err := a.allActiveLocationIDs(ctx)
	if err != nil {
		return nil, err
	}
	return &pb.DepartmentLocationConfigResponse{Code: errs.OK, Msg: "success", DepartmentId: req.DepartmentId, InheritLocations: dep.InheritLocations, DirectLocationIds: direct, EffectiveLocationIds: effective, Locations: locationsToPB(locations), AvailableLocationIds: available}, nil
}

func (a *taxonomyAdminAdapter) UpdateDepartmentLocationConfig(ctx context.Context, req *pb.UpdateDepartmentLocationConfigRequest) (*pb.DepartmentLocationConfigResponse, error) {
	err := a.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&departmentRecord{}).Where("id = ? AND deleted_at IS NULL", req.DepartmentId).Updates(map[string]any{"inherit_locations": req.InheritLocations, "updated_by": req.AdminId}).Error; err != nil {
			return err
		}
		now := a.now()
		if err := tx.Model(&departmentLocationRecord{}).Where("department_id = ? AND deleted_at IS NULL", req.DepartmentId).Updates(map[string]any{"deleted_at": now, "deleted_by": req.AdminId, "is_active": 0}).Error; err != nil {
			return err
		}
		for _, locationID := range req.LocationIds {
			if locationID <= 0 {
				continue
			}
			row := &departmentLocationRecord{DepartmentID: req.DepartmentId, LocationID: locationID, IsActive: 1, CreatedBy: positivePtr(req.AdminId), UpdatedBy: positivePtr(req.AdminId)}
			if err := tx.Create(row).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return a.GetDepartmentLocationConfig(ctx, &pb.GetDepartmentLocationConfigRequest{DepartmentId: req.DepartmentId})
}

func (a *taxonomyAdminAdapter) ListDepartmentsLocationMap(ctx context.Context, _ *pb.ListDepartmentsLocationMapRequest) (*pb.ListDepartmentsLocationMapResponse, error) {
	items, err := a.listDepartmentLocationMaps(ctx)
	if err != nil {
		return nil, err
	}
	return &pb.ListDepartmentsLocationMapResponse{Code: errs.OK, Msg: "success", Items: items}, nil
}

func (a *adminAdapter) CreateInviteCode(ctx context.Context, req *pb.CreateInviteCodeRequest) (*pb.CreateInviteCodeResponse, error) {
	expiresAt, err := parseOptionalTime(req.ExpiresAt)
	if err != nil {
		return &pb.CreateInviteCodeResponse{Code: errs.ErrBadRequest, Msg: "过期时间格式不正确"}, nil
	}
	const maxAttempts = 8
	for attempt := 0; attempt < maxAttempts; attempt++ {
		code, err := randomInviteCode()
		if err != nil {
			return nil, err
		}
		row := &inviteCodeRecord{Code: code, CreatedBy: req.CreatedBy, ExpiresAt: expiresAt, IsActive: 1, CreatedAt: a.now()}
		if err := a.db.WithContext(ctx).Create(row).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(err.Error(), "Duplicate") {
				continue
			}
			return nil, err
		}
		return &pb.CreateInviteCodeResponse{Code: errs.OK, Msg: "success", InviteCode: inviteCodeToPB(*row)}, nil
	}
	return nil, fmt.Errorf("failed to generate unique invite code after %d attempts", maxAttempts)
}

func (a *adminAdapter) ListInviteCodes(ctx context.Context, req *pb.ListInviteCodesRequest) (*pb.ListInviteCodesResponse, error) {
	query := a.db.WithContext(ctx).Model(&inviteCodeRecord{})
	if req.CreatedBy > 0 {
		query = query.Where("created_by = ?", req.CreatedBy)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []inviteCodeRecord
	if err := query.Order("id DESC").Offset(int((page(req.Page) - 1) * pageSize(req.PageSize))).Limit(int(pageSize(req.PageSize))).Find(&rows).Error; err != nil {
		return nil, err
	}
	list := make([]*pb.InviteCodeInfo, 0, len(rows))
	for _, row := range rows {
		list = append(list, inviteCodeToPB(row))
	}
	return &pb.ListInviteCodesResponse{Code: errs.OK, Msg: "success", Total: total, List: list}, nil
}

func (a *adminAdapter) ExtendInviteCode(ctx context.Context, req *pb.ExtendInviteCodeRequest) (*pb.CommonResponse, error) {
	expiresAt, err := parseOptionalTime(req.NewExpiresAt)
	if err != nil {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "过期时间格式不正确"}, nil
	}
	result := a.db.WithContext(ctx).Model(&inviteCodeRecord{}).Where("id = ?", req.Id).Update("expires_at", expiresAt)
	return rowsCommon(result, "success", "邀请码不存在")
}

func (a *adminAdapter) RevokeInviteCode(ctx context.Context, req *pb.RevokeInviteCodeRequest) (*pb.CommonResponse, error) {
	result := a.db.WithContext(ctx).Model(&inviteCodeRecord{}).Where("id = ?", req.Id).Update("is_active", 0)
	return rowsCommon(result, "success", "邀请码不存在")
}

func (a *adminAdapter) ReactivateInviteCode(ctx context.Context, req *pb.ReactivateInviteCodeRequest) (*pb.CommonResponse, error) {
	result := a.db.WithContext(ctx).Model(&inviteCodeRecord{}).Where("id = ?", req.Id).Update("is_active", 1)
	return rowsCommon(result, "success", "邀请码不存在")
}

func (a *adminAdapter) ValidateInviteCode(ctx context.Context, req *pb.ValidateInviteCodeRequest) (*pb.ValidateInviteCodeResponse, error) {
	var row inviteCodeRecord
	err := a.db.WithContext(ctx).Where("code = ? AND is_active = ?", strings.TrimSpace(req.InviteCode), 1).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &pb.ValidateInviteCodeResponse{Code: errs.OK, Msg: "success", Valid: false}, nil
	}
	if err != nil {
		return nil, err
	}
	valid := row.ExpiresAt == nil || row.ExpiresAt.After(a.now())
	return &pb.ValidateInviteCodeResponse{Code: errs.OK, Msg: "success", Valid: valid}, nil
}

func (a *adminAdapter) QueryUsageLogs(ctx context.Context, req *pb.QueryUsageLogsRequest) (*pb.QueryUsageLogsResponse, error) {
	query := a.usageLogQuery(ctx, req.StartTime, req.EndTime)
	if strings.TrimSpace(req.ServiceType) != "" {
		query = query.Where("service_type = ?", strings.TrimSpace(req.ServiceType))
	}
	if strings.TrimSpace(req.Provider) != "" {
		query = query.Where("provider = ?", strings.TrimSpace(req.Provider))
	}
	if strings.TrimSpace(req.Status) != "" {
		query = query.Where("status = ?", strings.TrimSpace(req.Status))
	}
	if req.UserId != 0 {
		query = query.Where("user_id = ?", req.UserId)
	}
	if strings.TrimSpace(req.RequestId) != "" {
		query = query.Where("request_id = ?", strings.TrimSpace(req.RequestId))
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []usageLogRecord
	if err := query.Order("created_at DESC, id DESC").Offset(int((page(req.Page) - 1) * pageSize(req.PageSize))).Limit(int(pageSize(req.PageSize))).Find(&rows).Error; err != nil {
		return nil, err
	}
	list := make([]*pb.UsageLogItem, 0, len(rows))
	for _, row := range rows {
		list = append(list, usageLogToPB(row))
	}
	return &pb.QueryUsageLogsResponse{Code: errs.OK, Msg: "success", Total: total, List: list}, nil
}

func (a *usageStatsAdapter) GetUsageStats(ctx context.Context, req *pb.GetUsageStatsRequest) (*pb.GetUsageStatsResponse, error) {
	dimensionCol := usageDimension(req.GetDimension())

	var summaryRow struct {
		TotalTokens  int64   `gorm:"column:total_tokens"`
		CallCount    int64   `gorm:"column:call_count"`
		SuccessCount int64   `gorm:"column:success_count"`
		FailedCount  int64   `gorm:"column:failed_count"`
		AvgCostMs    float64 `gorm:"column:avg_cost_ms"`
	}
	// Independent aggregate (do not reuse a chained query session).
	if err := a.usageLogQuery(ctx, req.GetStartTime(), req.GetEndTime()).
		Select(`COALESCE(SUM(estimated_tokens),0) AS total_tokens,
			COUNT(*) AS call_count,
			COALESCE(SUM(CASE WHEN status = 'ok' THEN 1 ELSE 0 END),0) AS success_count,
			COALESCE(SUM(CASE WHEN status <> 'ok' THEN 1 ELSE 0 END),0) AS failed_count,
			COALESCE(AVG(cost_ms),0) AS avg_cost_ms`).
		Scan(&summaryRow).Error; err != nil {
		return nil, err
	}
	summary := &pb.UsageStatsSummary{
		TotalTokens:   summaryRow.TotalTokens,
		CallCount:     summaryRow.CallCount,
		SuccessCount:  summaryRow.SuccessCount,
		FailedCount:   summaryRow.FailedCount,
		AvgCostMs:     summaryRow.AvgCostMs,
		EstimatedCost: estimateCost(summaryRow.TotalTokens),
	}
	if summaryRow.CallCount > 0 {
		summary.SuccessRate = float64(summaryRow.SuccessCount) * 100 / float64(summaryRow.CallCount)
	}

	var rows []struct {
		DimKey       string  `gorm:"column:dim_key"`
		TotalTokens  int64   `gorm:"column:total_tokens"`
		CallCount    int64   `gorm:"column:call_count"`
		AvgCostMs    float64 `gorm:"column:avg_cost_ms"`
		SuccessCount int64   `gorm:"column:success_count"`
		FailedCount  int64   `gorm:"column:failed_count"`
	}
	// Always alias dimension as dim_key so GORM maps the label reliably (provider/model/etc.).
	selectSQL := fmt.Sprintf(`%s AS dim_key,
			COALESCE(SUM(estimated_tokens),0) AS total_tokens,
			COUNT(*) AS call_count,
			COALESCE(AVG(cost_ms),0) AS avg_cost_ms,
			COALESCE(SUM(CASE WHEN status = 'ok' THEN 1 ELSE 0 END),0) AS success_count,
			COALESCE(SUM(CASE WHEN status <> 'ok' THEN 1 ELSE 0 END),0) AS failed_count`, dimensionCol)
	if err := a.usageLogQuery(ctx, req.GetStartTime(), req.GetEndTime()).
		Select(selectSQL).
		Group(dimensionCol).
		Order("total_tokens DESC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	list := make([]*pb.UsageStatsItem, 0, len(rows))
	for _, row := range rows {
		name := strings.TrimSpace(row.DimKey)
		if name == "" {
			name = "(空)"
		}
		list = append(list, &pb.UsageStatsItem{
			Name:          name,
			TotalTokens:   row.TotalTokens,
			CallCount:     row.CallCount,
			AvgCostMs:     row.AvgCostMs,
			EstimatedCost: estimateCost(row.TotalTokens),
			SuccessCount:  row.SuccessCount,
			FailedCount:   row.FailedCount,
		})
	}
	return &pb.GetUsageStatsResponse{Code: errs.OK, Msg: "success", List: list, Summary: summary}, nil
}

func (a *usageStatsAdapter) GetUsageTrend(ctx context.Context, req *pb.GetUsageTrendRequest) (*pb.GetUsageTrendResponse, error) {
	dateExpr := "DATE(created_at)"
	if req.Granularity == "month" {
		dateExpr = "DATE_FORMAT(created_at, '%Y-%m')"
	} else if req.Granularity == "week" {
		dateExpr = "YEARWEEK(created_at, 3)"
	}
	var rows []struct {
		Date        string
		TotalTokens int64
		CallCount   int64
		AvgCostMs   float64
	}
	err := a.usageLogQuery(ctx, req.StartTime, req.EndTime).
		Select(fmt.Sprintf("%s AS date, COALESCE(SUM(estimated_tokens),0) AS total_tokens, COUNT(*) AS call_count, COALESCE(AVG(cost_ms),0) AS avg_cost_ms", dateExpr)).
		Group("date").
		Order("date ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	list := make([]*pb.UsageTrendPoint, 0, len(rows))
	for _, row := range rows {
		list = append(list, &pb.UsageTrendPoint{Date: row.Date, TotalTokens: row.TotalTokens, CallCount: row.CallCount, AvgCostMs: row.AvgCostMs, EstimatedCost: estimateCost(row.TotalTokens)})
	}
	return &pb.GetUsageTrendResponse{Code: errs.OK, Msg: "success", List: list}, nil
}

const collaborationForbiddenMsg = "无权限访问该候选人协同数据"

func (a *collaborationAdapter) GetCandidateWorkspace(ctx context.Context, req *pb.GetCandidateWorkspaceRequest) (*pb.GetCandidateWorkspaceResponse, error) {
	candidateUserID := uint64(0)
	if req.CandidateUserId > 0 {
		candidateUserID = uint64(req.CandidateUserId)
	}
	allowed, err := a.requireCandidateCollaborationAccess(ctx, req.StaffUserId, candidateUserID, domainmodel.PermissionApplicationRead)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return &pb.GetCandidateWorkspaceResponse{Code: errs.ErrForbidden, Msg: collaborationForbiddenMsg}, nil
	}
	workspace := &pb.CandidateWorkspace{}
	var profile candidateProfileRecord
	if err := a.db.WithContext(ctx).Where("user_id = ?", req.CandidateUserId).First(&profile).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if profile.UserID != 0 {
		workspace.RealName = profile.RealName
		workspace.Phone = profile.Phone
		workspace.Education = profile.Education
		workspace.School = profile.School
		workspace.WorkExperience = profile.WorkExperience
		workspace.Skills = splitSkills(profile.Skills)
	}
	var rows []applicationDetailRow
	if err := a.applicationDetails().WithContext(ctx).Where("a.user_id = ?", req.CandidateUserId).Order("a.applied_at DESC, a.id DESC").Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		workspace.Applications = append(workspace.Applications, &pb.CandidateWorkspaceApplication{
			ApplicationId: row.ApplicationID, JobId: row.JobID, JobTitle: row.JobTitle, Department: row.Department, Location: row.Location, StatusKey: row.StatusKey,
			RoundNo: row.RoundNo, IsCurrent: row.IsCurrent, AppliedAt: formatTime(row.AppliedAt),
		})
		updateLatestActivity(&workspace.LatestActivityAt, row.AppliedAt)
		updateLatestActivity(&workspace.LatestActivityAt, row.UpdatedAt)
	}
	workspace.TotalApplications = int64(len(rows))
	tags, err := a.candidateTags(ctx, uint64(req.CandidateUserId))
	if err != nil {
		return nil, err
	}
	workspace.Tags = tags
	var resume resumeRecord
	if err := a.db.WithContext(ctx).Where("user_id = ? AND is_valid = ?", req.CandidateUserId, 1).Order("uploaded_at DESC, id DESC").First(&resume).Error; err == nil && resume.OSSKey != "" {
		workspace.ResumeUrl, _ = a.storage.GeneratePresignedGetURL(resume.OSSKey)
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	interviews, err := a.candidateWorkspaceInterviews(ctx, int64(req.CandidateUserId))
	if err != nil {
		return nil, err
	}
	if err := a.db.WithContext(ctx).Table("interview_schedules i").Joins("JOIN applications a ON a.id = i.application_id").Where("a.user_id = ? AND i.deleted_at IS NULL", req.CandidateUserId).Count(&workspace.TotalInterviews).Error; err != nil {
		return nil, err
	}
	workspace.Interviews = make([]*pb.CandidateWorkspaceInterview, 0, len(interviews))
	for _, row := range interviews {
		workspace.Interviews = append(workspace.Interviews, &pb.CandidateWorkspaceInterview{
			InterviewId:                 row.InterviewID,
			ApplicationId:               row.ApplicationID,
			Title:                       row.Title,
			Mode:                        row.Mode,
			Status:                      row.Status,
			ScheduledAt:                 formatOptionalTime(row.ScheduledAt),
			InterviewerName:             userDisplayName(row.InterviewerID),
			JobTitle:                    row.JobTitle,
			RoundNo:                     row.RoundNo,
			FeedbackRecommendation:      row.FeedbackRecommendation,
			FeedbackScore:               row.FeedbackScore,
			FeedbackDimensionScoresJson: row.FeedbackDimensionScoresJSON,
			FeedbackComments:            row.FeedbackComments,
			HasFeedback:                 row.HasFeedback,
		})
		updateLatestActivity(&workspace.LatestActivityAt, row.CreatedAt)
		updateLatestActivity(&workspace.LatestActivityAt, row.UpdatedAt)
	}
	offers, err := a.candidateWorkspaceOffers(ctx, int64(req.CandidateUserId))
	if err != nil {
		return nil, err
	}
	if err := a.db.WithContext(ctx).Table("offers").Where("candidate_user_id = ?", req.CandidateUserId).Count(&workspace.TotalOffers).Error; err != nil {
		return nil, err
	}
	workspace.Offers = make([]*pb.CandidateWorkspaceOffer, 0, len(offers))
	for _, row := range offers {
		workspace.Offers = append(workspace.Offers, &pb.CandidateWorkspaceOffer{
			OfferId:       row.OfferID,
			ApplicationId: row.ApplicationID,
			Title:         row.Title,
			Status:        row.Status,
			SalaryRange:   row.SalaryRange,
			Level:         row.Level,
			WorkLocation:  row.WorkLocation,
			StartDate:     row.StartDate,
			JobTitle:      row.JobTitle,
		})
		updateLatestActivity(&workspace.LatestActivityAt, row.CreatedAt)
		updateLatestActivity(&workspace.LatestActivityAt, row.UpdatedAt)
		updateLatestActivityPtr(&workspace.LatestActivityAt, row.DecidedAt)
	}
	latestCollaborationActivity, err := a.latestCandidateCollaborationActivity(ctx, candidateUserID)
	if err != nil {
		return nil, err
	}
	updateLatestActivityPtr(&workspace.LatestActivityAt, latestCollaborationActivity)
	return &pb.GetCandidateWorkspaceResponse{Code: errs.OK, Msg: "success", Workspace: workspace}, nil
}

func (a *collaborationAdapter) CreateNote(ctx context.Context, req *pb.CreateNoteRequest) (*pb.CreateNoteResponse, error) {
	allowed, err := a.requireCandidateCollaborationAccess(ctx, req.StaffUserId, req.CandidateUserId, domainmodel.PermissionApplicationRead, domainmodel.PermissionCollaborationNoteCreate)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return &pb.CreateNoteResponse{Code: errs.ErrForbidden, Msg: collaborationForbiddenMsg}, nil
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return &pb.CreateNoteResponse{Code: errs.ErrBadRequest, Msg: "备注内容不能为空"}, nil
	}
	row := &candidateNoteRecord{CandidateUserID: req.CandidateUserId, ApplicationID: positiveUintPtr(req.ApplicationId), AuthorUserID: uint64(req.StaffUserId), Content: content, Visibility: "internal", CreatedAt: a.now(), UpdatedAt: a.now()}
	if err := a.db.WithContext(ctx).Create(row).Error; err != nil {
		return nil, err
	}
	return &pb.CreateNoteResponse{Code: errs.OK, Msg: "success", Note: noteToPB(*row)}, nil
}

func (a *collaborationAdapter) ListNotes(ctx context.Context, req *pb.ListNotesRequest) (*pb.ListNotesResponse, error) {
	allowed, err := a.requireCandidateCollaborationAccess(ctx, req.StaffUserId, req.CandidateUserId, domainmodel.PermissionApplicationRead, domainmodel.PermissionCollaborationNoteRead)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return &pb.ListNotesResponse{Code: errs.ErrForbidden, Msg: collaborationForbiddenMsg}, nil
	}
	query := a.db.WithContext(ctx).Where("candidate_user_id = ?", req.CandidateUserId)
	if req.ApplicationId > 0 {
		query = query.Where("application_id = ?", req.ApplicationId)
	}
	var rows []candidateNoteRecord
	if err := query.Order("created_at DESC, id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	list := make([]*pb.CandidateNoteInfo, 0, len(rows))
	for _, row := range rows {
		list = append(list, noteToPB(row))
	}
	return &pb.ListNotesResponse{Code: errs.OK, Msg: "success", List: list}, nil
}

func (a *collaborationAdapter) CreateTag(ctx context.Context, req *pb.CreateTagRequest) (*pb.CreateTagResponse, error) {
	allowed, err := a.authorizeUserPermission(ctx, uint64(req.StaffUserId), domainmodel.PermissionCollaborationTagManage)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return &pb.CreateTagResponse{Code: errs.ErrForbidden, Msg: "无权限管理候选人标签"}, nil
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return &pb.CreateTagResponse{Code: errs.ErrBadRequest, Msg: "标签名称不能为空"}, nil
	}
	color := strings.TrimSpace(req.Color)
	if color == "" {
		color = "#409eff"
	}
	row := &candidateTagRecord{Name: name, Color: color, CreatedBy: positiveUintPtr(uint64(req.StaffUserId)), CreatedAt: a.now()}
	if err := a.db.WithContext(ctx).Create(row).Error; err != nil {
		return nil, err
	}
	return &pb.CreateTagResponse{Code: errs.OK, Msg: "success", Tag: tagToPB(*row)}, nil
}

func (a *collaborationAdapter) ListTags(ctx context.Context, req *pb.ListTagsRequest) (*pb.ListTagsResponse, error) {
	allowed, err := a.authorizeUserPermission(ctx, uint64(req.StaffUserId), domainmodel.PermissionCollaborationTagManage)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return &pb.ListTagsResponse{Code: errs.ErrForbidden, Msg: "无权限管理候选人标签"}, nil
	}
	var rows []candidateTagRecord
	if err := a.db.WithContext(ctx).Order("created_at DESC, id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	list := make([]*pb.CandidateTagInfo, 0, len(rows))
	for _, row := range rows {
		list = append(list, tagToPB(row))
	}
	return &pb.ListTagsResponse{Code: errs.OK, Msg: "success", List: list}, nil
}

func (a *collaborationAdapter) AssignTag(ctx context.Context, req *pb.AssignTagRequest) (*pb.CommonResponse, error) {
	allowed, err := a.requireCandidateCollaborationAccess(ctx, req.StaffUserId, req.CandidateUserId, domainmodel.PermissionApplicationRead, domainmodel.PermissionCollaborationTagManage)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: collaborationForbiddenMsg}, nil
	}
	row := &candidateTagAssignmentRecord{TagID: req.TagId, CandidateUserID: req.CandidateUserId, CreatedBy: positiveUintPtr(uint64(req.StaffUserId)), CreatedAt: a.now()}
	err = a.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "tag_id"}, {Name: "candidate_user_id"}}, DoNothing: true}).Create(row).Error
	if err != nil {
		return nil, err
	}
	return &pb.CommonResponse{Code: errs.OK, Msg: "success"}, nil
}

func (a *collaborationAdapter) UnassignTag(ctx context.Context, req *pb.UnassignTagRequest) (*pb.CommonResponse, error) {
	allowed, err := a.requireCandidateCollaborationAccess(ctx, req.StaffUserId, req.CandidateUserId, domainmodel.PermissionApplicationRead, domainmodel.PermissionCollaborationTagManage)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: collaborationForbiddenMsg}, nil
	}
	result := a.db.WithContext(ctx).Where("tag_id = ? AND candidate_user_id = ?", req.TagId, req.CandidateUserId).Delete(&candidateTagAssignmentRecord{})
	return rowsCommon(result, "success", "标签未分配")
}

func (a *collaborationAdapter) ListCandidateTags(ctx context.Context, req *pb.ListCandidateTagsRequest) (*pb.ListCandidateTagsResponse, error) {
	allowed, err := a.requireCandidateCollaborationAccess(ctx, req.StaffUserId, req.CandidateUserId, domainmodel.PermissionApplicationRead)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return &pb.ListCandidateTagsResponse{Code: errs.ErrForbidden, Msg: collaborationForbiddenMsg}, nil
	}
	tags, err := a.candidateTags(ctx, req.CandidateUserId)
	if err != nil {
		return nil, err
	}
	return &pb.ListCandidateTagsResponse{Code: errs.OK, Msg: "success", List: tags}, nil
}

func (a *collaborationAdapter) CreateFollowUpTask(ctx context.Context, req *pb.CreateFollowUpTaskRequest) (*pb.CreateFollowUpTaskResponse, error) {
	allowed, err := a.requireCandidateCollaborationAccess(ctx, req.StaffUserId, req.CandidateUserId, domainmodel.PermissionApplicationRead, domainmodel.PermissionCollaborationTaskManage)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return &pb.CreateFollowUpTaskResponse{Code: errs.ErrForbidden, Msg: collaborationForbiddenMsg}, nil
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return &pb.CreateFollowUpTaskResponse{Code: errs.ErrBadRequest, Msg: "任务标题不能为空"}, nil
	}
	dueAt, err := parseOptionalTime(req.DueAt)
	if err != nil {
		return &pb.CreateFollowUpTaskResponse{Code: errs.ErrBadRequest, Msg: "截止时间格式不正确"}, nil
	}
	row := &followUpTaskRecord{CandidateUserID: req.CandidateUserId, ApplicationID: positiveUintPtr(req.ApplicationId), AssigneeUserID: req.AssigneeUserId, CreatedBy: uint64(req.StaffUserId), Title: title, Description: req.Description, DueAt: dueAt, Status: "pending", CreatedAt: a.now(), UpdatedAt: a.now()}
	if err := a.db.WithContext(ctx).Create(row).Error; err != nil {
		return nil, err
	}
	return &pb.CreateFollowUpTaskResponse{Code: errs.OK, Msg: "success", Task: followUpToPB(*row)}, nil
}

func (a *collaborationAdapter) ListFollowUpTasks(ctx context.Context, req *pb.ListFollowUpTasksRequest) (*pb.ListFollowUpTasksResponse, error) {
	if req.CandidateUserId == 0 {
		return &pb.ListFollowUpTasksResponse{Code: errs.ErrForbidden, Msg: "必须指定候选人范围"}, nil
	}
	allowed, err := a.requireCandidateCollaborationAccess(ctx, req.StaffUserId, req.CandidateUserId, domainmodel.PermissionApplicationRead, domainmodel.PermissionCollaborationTaskManage)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return &pb.ListFollowUpTasksResponse{Code: errs.ErrForbidden, Msg: collaborationForbiddenMsg}, nil
	}
	query := a.db.WithContext(ctx).Model(&followUpTaskRecord{})
	if req.CandidateUserId > 0 {
		query = query.Where("candidate_user_id = ?", req.CandidateUserId)
	}
	if req.AssigneeUserId > 0 {
		query = query.Where("assignee_user_id = ?", req.AssigneeUserId)
	}
	if strings.TrimSpace(req.Status) != "" {
		query = query.Where("status = ?", strings.TrimSpace(req.Status))
	}
	var rows []followUpTaskRecord
	if err := query.Order("created_at DESC, id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	list := make([]*pb.FollowUpTaskInfo, 0, len(rows))
	for _, row := range rows {
		list = append(list, followUpToPB(row))
	}
	return &pb.ListFollowUpTasksResponse{Code: errs.OK, Msg: "success", List: list}, nil
}

func (a *collaborationAdapter) CompleteFollowUpTask(ctx context.Context, req *pb.CompleteFollowUpTaskRequest) (*pb.CommonResponse, error) {
	var row followUpTaskRecord
	if err := a.db.WithContext(ctx).Where("id = ?", req.TaskId).First(&row).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "任务不存在"}, nil
	} else if err != nil {
		return nil, err
	}
	allowed, err := a.requireCandidateCollaborationAccess(ctx, req.StaffUserId, row.CandidateUserID, domainmodel.PermissionApplicationRead, domainmodel.PermissionCollaborationTaskManage)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: collaborationForbiddenMsg}, nil
	}
	now := a.now()
	result := a.db.WithContext(ctx).Model(&followUpTaskRecord{}).Where("id = ? AND candidate_user_id = ?", req.TaskId, row.CandidateUserID).Updates(map[string]any{"status": "completed", "completed_at": now, "updated_at": now})
	return rowsCommon(result, "success", "任务不存在")
}

func (a *collaborationAdapter) GetFollowUpTask(ctx context.Context, req *pb.GetFollowUpTaskRequest) (*pb.GetFollowUpTaskResponse, error) {
	var row followUpTaskRecord
	err := a.db.WithContext(ctx).Where("id = ?", req.TaskId).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &pb.GetFollowUpTaskResponse{Code: errs.ErrBadRequest, Msg: "任务不存在"}, nil
	}
	if err != nil {
		return nil, err
	}
	allowed, err := a.requireCandidateCollaborationAccess(ctx, req.StaffUserId, row.CandidateUserID, domainmodel.PermissionApplicationRead, domainmodel.PermissionCollaborationTaskManage)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return &pb.GetFollowUpTaskResponse{Code: errs.ErrForbidden, Msg: collaborationForbiddenMsg}, nil
	}
	return &pb.GetFollowUpTaskResponse{Code: errs.OK, Msg: "success", Task: followUpToPB(row)}, nil
}

func (a *collaborationAdapter) ListTimelineEvents(ctx context.Context, req *pb.ListTimelineEventsRequest) (*pb.ListTimelineEventsResponse, error) {
	allowed, err := a.requireCandidateCollaborationAccess(ctx, req.StaffUserId, req.CandidateUserId, domainmodel.PermissionApplicationRead)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return &pb.ListTimelineEventsResponse{Code: errs.ErrForbidden, Msg: collaborationForbiddenMsg}, nil
	}
	events := []*pb.TimelineEventInfo{}
	var notes []candidateNoteRecord
	if err := a.db.WithContext(ctx).Where("candidate_user_id = ?", req.CandidateUserId).Order("created_at DESC, id DESC").Limit(100).Find(&notes).Error; err != nil {
		return nil, err
	}
	for _, note := range notes {
		events = append(events, &pb.TimelineEventInfo{Id: fmt.Sprintf("note-%d", note.ID), EventType: "note", Title: "候选人备注", Description: note.Content, Timestamp: formatTime(note.CreatedAt), ActorName: fmt.Sprintf("用户%d", note.AuthorUserID), ApplicationId: int64(ptrUintValue(note.ApplicationID))})
	}
	var transitions []applicationTransitionRecord
	err = a.db.WithContext(ctx).Table("application_status_transitions ast").
		Select("ast.*").
		Joins("JOIN applications a ON a.id = ast.application_id").
		Where("a.user_id = ?", req.CandidateUserId).
		Order("ast.created_at DESC, ast.id DESC").
		Limit(100).
		Find(&transitions).Error
	if err != nil {
		return nil, err
	}
	for _, transition := range transitions {
		events = append(events, &pb.TimelineEventInfo{Id: fmt.Sprintf("status-%d", transition.ID), EventType: "status_transition", Title: "状态变更", Description: fmt.Sprintf("%s -> %s", transition.FromStatus, transition.ToStatus), Timestamp: formatTime(transition.CreatedAt), ActorName: fmt.Sprintf("用户%d", transition.ActorUserID), ApplicationId: transition.ApplicationID})
	}
	interviews, err := a.candidateWorkspaceInterviews(ctx, int64(req.CandidateUserId))
	if err != nil {
		return nil, err
	}
	for _, interview := range interviews {
		events = append(events, &pb.TimelineEventInfo{
			Id:            fmt.Sprintf("interview-%d", interview.InterviewID),
			EventType:     "interview",
			Title:         candidateWorkspaceInterviewTimelineTitle(interview),
			Description:   candidateWorkspaceInterviewTimelineDescription(interview),
			Timestamp:     formatTime(interview.CreatedAt),
			ActorName:     userDisplayName(interview.InterviewerID),
			ApplicationId: interview.ApplicationID,
		})
	}
	offers, err := a.candidateWorkspaceOffers(ctx, int64(req.CandidateUserId))
	if err != nil {
		return nil, err
	}
	for _, offer := range offers {
		events = append(events, &pb.TimelineEventInfo{
			Id:            fmt.Sprintf("offer-%d", offer.OfferID),
			EventType:     "offer",
			Title:         candidateWorkspaceOfferTimelineTitle(offer),
			Description:   strings.TrimSpace(fmt.Sprintf("薪酬: %s | 职级: %s", offer.SalaryRange, offer.Level)),
			Timestamp:     formatTime(offer.CreatedAt),
			ActorName:     userDisplayName(offer.CreatedBy),
			ApplicationId: offer.ApplicationID,
		})
	}
	sortTimelineEvents(events)
	return &pb.ListTimelineEventsResponse{Code: errs.OK, Msg: "success", Events: events}, nil
}

func (a *collaborationAdapter) candidateWorkspaceInterviews(ctx context.Context, candidateUserID int64) ([]candidateWorkspaceInterviewRow, error) {
	var rows []candidateWorkspaceInterviewRow
	err := a.db.WithContext(ctx).Table("interview_schedules i").
		Select(`i.id AS interview_id,
			i.application_id,
			COALESCE(i.title, '') AS title,
			COALESCE(i.mode, '') AS mode,
			i.status,
			i.scheduled_at,
			i.interviewer_id,
			j.title AS job_title,
			i.round_no,
			COALESCE(f.recommendation, '') AS feedback_recommendation,
			COALESCE(f.score, 0) AS feedback_score,
			COALESCE(f.dimension_scores_json, '') AS feedback_dimension_scores_json,
			COALESCE(f.comments, '') AS feedback_comments,
			CASE WHEN f.id IS NULL THEN 0 ELSE 1 END AS has_feedback,
			i.created_at,
			i.updated_at`).
		Joins("JOIN applications a ON a.id = i.application_id").
		Joins("JOIN jobs j ON j.id = a.job_id").
		Joins("LEFT JOIN interview_feedback f ON f.interview_id = i.id AND f.application_id = i.application_id AND f.interviewer_id = i.interviewer_id").
		Where("a.user_id = ? AND i.deleted_at IS NULL", candidateUserID).
		Where("i.status NOT IN ?", []string{"cancelled"}).
		Order("i.scheduled_at DESC, i.created_at DESC, i.id DESC").
		Scan(&rows).Error
	return rows, err
}

func (a *collaborationAdapter) candidateWorkspaceOffers(ctx context.Context, candidateUserID int64) ([]candidateWorkspaceOfferRow, error) {
	var rows []candidateWorkspaceOfferRow
	err := a.db.WithContext(ctx).Table("offers o").
		Select(`o.id AS offer_id,
			o.application_id,
			o.title,
			o.status,
			COALESCE(o.salary_range, '') AS salary_range,
			COALESCE(o.level, '') AS level,
			COALESCE(o.work_location, '') AS work_location,
			COALESCE(o.start_date, '') AS start_date,
			j.title AS job_title,
			o.created_by,
			o.created_at,
			o.updated_at,
			o.decided_at`).
		Joins("JOIN jobs j ON j.id = o.job_id").
		Joins("JOIN applications a ON a.id = o.application_id").
		Where("o.candidate_user_id = ?", candidateUserID).
		Where("o.status != ?", "draft").
		Order("o.created_at DESC, o.id DESC").
		Limit(100).
		Scan(&rows).Error
	return rows, err
}

func (a *collaborationAdapter) latestCandidateCollaborationActivity(ctx context.Context, candidateUserID uint64) (*time.Time, error) {
	var latest struct {
		ActivityAt sql.NullString
	}
	query := `
		SELECT MAX(activity_at) AS activity_at FROM (
			SELECT applied_at AS activity_at FROM applications WHERE tenant_id = @tenant_id AND user_id = @candidate_id
			UNION ALL
			SELECT updated_at AS activity_at FROM applications WHERE tenant_id = @tenant_id AND user_id = @candidate_id
			UNION ALL
			SELECT i.created_at AS activity_at FROM interview_schedules i JOIN applications a ON a.id = i.application_id AND a.tenant_id = i.tenant_id WHERE i.tenant_id = @tenant_id AND a.user_id = @candidate_id AND i.deleted_at IS NULL
			UNION ALL
			SELECT i.updated_at AS activity_at FROM interview_schedules i JOIN applications a ON a.id = i.application_id AND a.tenant_id = i.tenant_id WHERE i.tenant_id = @tenant_id AND a.user_id = @candidate_id AND i.deleted_at IS NULL
			UNION ALL
			SELECT f.submitted_at AS activity_at FROM interview_feedback f JOIN applications a ON a.id = f.application_id AND a.tenant_id = f.tenant_id WHERE f.tenant_id = @tenant_id AND a.user_id = @candidate_id
			UNION ALL
			SELECT f.updated_at AS activity_at FROM interview_feedback f JOIN applications a ON a.id = f.application_id AND a.tenant_id = f.tenant_id WHERE f.tenant_id = @tenant_id AND a.user_id = @candidate_id
			UNION ALL
			SELECT o.created_at AS activity_at FROM offers o WHERE o.tenant_id = @tenant_id AND o.candidate_user_id = @candidate_id
			UNION ALL
			SELECT o.updated_at AS activity_at FROM offers o WHERE o.tenant_id = @tenant_id AND o.candidate_user_id = @candidate_id
			UNION ALL
			SELECT o.decided_at AS activity_at FROM offers o WHERE o.tenant_id = @tenant_id AND o.candidate_user_id = @candidate_id AND o.decided_at IS NOT NULL
			UNION ALL
			SELECT cn.created_at AS activity_at FROM candidate_notes cn WHERE cn.tenant_id = @tenant_id AND cn.candidate_user_id = @candidate_id
			UNION ALL
			SELECT ast.created_at AS activity_at FROM application_status_transitions ast JOIN applications a ON a.id = ast.application_id AND a.tenant_id = ast.tenant_id WHERE ast.tenant_id = @tenant_id AND a.user_id = @candidate_id
		) candidate_activity
	`
	tenantID := metadata.GetAuthTenantID(ctx)
	if tenantID <= 0 {
		return nil, fmt.Errorf("tenant context is required")
	}
	err := a.db.WithContext(ctx).Raw(query, sql.Named("tenant_id", tenantID), sql.Named("candidate_id", candidateUserID)).Scan(&latest).Error
	if err != nil {
		return nil, err
	}
	if !latest.ActivityAt.Valid || strings.TrimSpace(latest.ActivityAt.String) == "" {
		return nil, nil
	}
	parsed, err := parseDatabaseTime(latest.ActivityAt.String)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func updateLatestActivity(latest *string, candidate time.Time) {
	if latest == nil || candidate.IsZero() {
		return
	}
	current, err := time.Parse(time.RFC3339, *latest)
	if *latest == "" || err != nil || candidate.After(current) {
		*latest = formatTime(candidate)
	}
}

func updateLatestActivityPtr(latest *string, candidate *time.Time) {
	if candidate == nil {
		return
	}
	updateLatestActivity(latest, *candidate)
}

func sortTimelineEvents(events []*pb.TimelineEventInfo) {
	sort.SliceStable(events, func(i, j int) bool {
		left, leftErr := time.Parse(time.RFC3339, events[i].Timestamp)
		right, rightErr := time.Parse(time.RFC3339, events[j].Timestamp)
		if leftErr != nil && rightErr != nil {
			return events[i].Id > events[j].Id
		}
		if leftErr != nil {
			return false
		}
		if rightErr != nil {
			return true
		}
		return left.After(right)
	})
}

func userDisplayName(userID int64) string {
	if userID <= 0 {
		return ""
	}
	return fmt.Sprintf("用户%d", userID)
}

func candidateWorkspaceInterviewTimelineTitle(row candidateWorkspaceInterviewRow) string {
	title := strings.TrimSpace(row.Title)
	if title == "" {
		title = fmt.Sprintf("第 %d 轮面试", row.RoundNo)
	}
	if row.Status == "cancelled" {
		return "面试已取消: " + title
	}
	return "面试: " + title
}

func candidateWorkspaceInterviewTimelineDescription(row candidateWorkspaceInterviewRow) string {
	parts := []string{}
	if name := userDisplayName(row.InterviewerID); name != "" {
		parts = append(parts, "面试官: "+name)
	}
	if scheduledAt := formatOptionalTime(row.ScheduledAt); scheduledAt != "" {
		parts = append(parts, "时间: "+scheduledAt)
	}
	return strings.Join(parts, " | ")
}

func candidateWorkspaceOfferTimelineTitle(row candidateWorkspaceOfferRow) string {
	title := "Offer: " + row.Title
	switch row.Status {
	case "draft":
		title = "Offer已创建: " + row.Title
	case "sent":
		title = "Offer已发送: " + row.Title
	case "accepted":
		title = "Offer已接受: " + row.Title
	case "rejected":
		title = "Offer已拒绝: " + row.Title
	case "withdrawn":
		title = "Offer已撤回: " + row.Title
	}
	return title
}

func parseDatabaseTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
	} {
		parsed, err := time.ParseInLocation(layout, value, time.Local)
		if err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid database time: %s", value)
}

func (a *nativeStore) lookupDepartment(ctx context.Context, id int64) (*departmentRecord, error) {
	var dep departmentRecord
	err := a.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&dep).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &dep, nil
}

func (a *nativeStore) lookupLocation(ctx context.Context, id int64) (*jobLocationRecord, error) {
	var loc jobLocationRecord
	err := a.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&loc).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &loc, nil
}

func (a *nativeStore) validateDepartmentLocation(ctx context.Context, departmentID, locationID int64) error {
	ids, err := a.directLocationIDs(ctx, departmentID)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if id == locationID {
			return nil
		}
	}
	return fmt.Errorf("所选部门不支持该地点")
}

func (a *nativeStore) listDepartmentRecords(ctx context.Context) ([]departmentRecord, error) {
	var rows []departmentRecord
	err := a.db.WithContext(ctx).Where("deleted_at IS NULL").Order("parent_id ASC, sort_order ASC, id ASC").Find(&rows).Error
	return rows, err
}

func (a *nativeStore) listLocationRecords(ctx context.Context, includeInactive bool) ([]jobLocationRecord, error) {
	query := a.db.WithContext(ctx).Where("deleted_at IS NULL")
	if !includeInactive {
		query = query.Where("is_active = ?", 1)
	}
	var rows []jobLocationRecord
	err := query.Order("sort_order ASC, id ASC").Find(&rows).Error
	return rows, err
}

func (a *nativeStore) fillDepartmentPath(ctx context.Context, dep *departmentRecord) error {
	dep.FullName = dep.Name
	dep.Path = "/"
	dep.Depth = 1
	if dep.ParentID > 0 {
		parent, err := a.lookupDepartment(ctx, dep.ParentID)
		if err != nil {
			return err
		}
		if parent == nil {
			return fmt.Errorf("父部门不存在")
		}
		dep.FullName = strings.Trim(parent.FullName+"/"+dep.Name, "/")
		dep.Path = fmt.Sprintf("%s%d/", parent.Path, dep.ParentID)
		dep.Depth = parent.Depth + 1
	}
	return nil
}

func (a *nativeStore) directLocationIDs(ctx context.Context, departmentID int64) ([]int64, error) {
	var ids []int64
	err := a.db.WithContext(ctx).Model(&departmentLocationRecord{}).
		Where("department_id = ? AND is_active = ? AND deleted_at IS NULL", departmentID, 1).
		Order("location_id ASC").
		Pluck("location_id", &ids).Error
	return ids, err
}

func (a *nativeStore) effectiveLocationIDs(ctx context.Context, dep departmentRecord) ([]int64, error) {
	if dep.InheritLocations == 1 && dep.ParentID > 0 {
		parent, err := a.lookupDepartment(ctx, dep.ParentID)
		if err != nil {
			return nil, err
		}
		if parent != nil {
			return a.effectiveLocationIDs(ctx, *parent)
		}
	}
	return a.directLocationIDs(ctx, dep.ID)
}

func (a *nativeStore) effectiveLocations(ctx context.Context, departmentID int64) ([]jobLocationRecord, error) {
	if departmentID <= 0 {
		return a.listLocationRecords(ctx, false)
	}
	dep, err := a.lookupDepartment(ctx, departmentID)
	if err != nil || dep == nil {
		return nil, err
	}
	ids, err := a.effectiveLocationIDs(ctx, *dep)
	if err != nil {
		return nil, err
	}
	return a.locationsByIDs(ctx, ids)
}

func (a *nativeStore) locationsByIDs(ctx context.Context, ids []int64) ([]jobLocationRecord, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var rows []jobLocationRecord
	err := a.db.WithContext(ctx).Where("id IN ? AND is_active = ? AND deleted_at IS NULL", ids, 1).Order("sort_order ASC, id ASC").Find(&rows).Error
	return rows, err
}

func (a *nativeStore) allActiveLocationIDs(ctx context.Context) ([]int64, error) {
	var ids []int64
	err := a.db.WithContext(ctx).Model(&jobLocationRecord{}).Where("is_active = ? AND deleted_at IS NULL", 1).Order("sort_order ASC, id ASC").Pluck("id", &ids).Error
	return ids, err
}

func (a *nativeStore) listDepartmentLocationMaps(ctx context.Context) ([]*pb.DepartmentLocationMap, error) {
	var rows []departmentLocationRecord
	if err := a.db.WithContext(ctx).Where("is_active = ? AND deleted_at IS NULL", 1).Order("department_id ASC, location_id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	index := map[int64]*pb.DepartmentLocationMap{}
	for _, row := range rows {
		item := index[row.DepartmentID]
		if item == nil {
			item = &pb.DepartmentLocationMap{DepartmentId: row.DepartmentID}
			index[row.DepartmentID] = item
		}
		item.LocationIds = append(item.LocationIds, row.LocationID)
	}
	list := make([]*pb.DepartmentLocationMap, 0, len(index))
	for _, item := range index {
		list = append(list, item)
	}
	return list, nil
}

func (a *nativeStore) usageLogQuery(ctx context.Context, start, end string) *gorm.DB {
	query := a.db.WithContext(ctx).Model(&usageLogRecord{})
	if t, err := parseOptionalTime(start); err == nil && t != nil {
		query = query.Where("created_at >= ?", *t)
	}
	if t, err := parseOptionalTime(end); err == nil && t != nil {
		query = query.Where("created_at <= ?", *t)
	}
	return query
}

func (a *nativeStore) candidateTags(ctx context.Context, candidateUserID uint64) ([]*pb.CandidateTagInfo, error) {
	var rows []candidateTagRecord
	err := a.db.WithContext(ctx).Table("candidate_tags t").
		Select("t.*").
		Joins("JOIN candidate_tag_assignments a ON a.tag_id = t.id").
		Where("a.candidate_user_id = ?", candidateUserID).
		Order("t.created_at DESC, t.id DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	list := make([]*pb.CandidateTagInfo, 0, len(rows))
	for _, row := range rows {
		list = append(list, tagToPB(row))
	}
	return list, nil
}

func (a *nativeStore) writeUsageLog(ctx context.Context, row usageLogRecord) error {
	if row.CreatedAt.IsZero() {
		row.CreatedAt = a.now()
	}
	if row.Status == "" {
		row.Status = "ok"
	}
	return a.db.WithContext(ctx).Create(&row).Error
}

func (a *nativeStore) writeOutboxTx(tx *gorm.DB, eventType, aggregateType string, aggregateID uint64, routingKey string, payload any) error {
	eventID, err := randomCode()
	if err != nil {
		return err
	}
	payloadMap, err := asOutboxPayloadMap(payload)
	if err != nil {
		return err
	}
	// Consumers (email/notification) rely on event_id in the published body for
	// email_logs / inbox idempotency. Keep it in payload, not only on the outbox row.
	idempotencyKey := fmt.Sprintf("%s:%d:%s:%s", aggregateType, aggregateID, eventType, eventID)
	payloadMap["event_id"] = eventID
	payloadMap["event_type"] = eventType
	payloadMap["idempotency_key"] = idempotencyKey
	tenantID := metadata.GetAuthTenantID(tx.Statement.Context)
	if tenantID <= 0 && aggregateType == "application" {
		_ = tx.Model(&applicationRecord{}).Select("tenant_id").Where("id = ?", aggregateID).Scan(&tenantID).Error
	}
	if tenantID > 0 {
		payloadMap["tenant_id"] = tenantID
	}
	data, err := json.Marshal(payloadMap)
	if err != nil {
		return err
	}
	row := &eventOutboxRecord{
		EventID: eventID, SchemaVersion: "1.0", EventType: eventType, AggregateType: aggregateType, AggregateID: aggregateID,
		RoutingKey: routingKey, Producer: "recruitment-service", IdempotencyKey: idempotencyKey,
		Payload: string(data), Metadata: "{}", Status: 0, CreatedAt: a.now(), UpdatedAt: a.now(),
	}
	if tenantID > 0 {
		row.TenantID = &tenantID
	}
	return tx.Create(row).Error
}

func asOutboxPayloadMap(payload any) (map[string]any, error) {
	switch value := payload.(type) {
	case nil:
		return map[string]any{}, nil
	case map[string]any:
		out := make(map[string]any, len(value)+3)
		for key, item := range value {
			out[key] = item
		}
		return out, nil
	default:
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		var out map[string]any
		if err := json.Unmarshal(data, &out); err != nil {
			return nil, err
		}
		if out == nil {
			out = map[string]any{}
		}
		return out, nil
	}
}

func departmentTree(rows []departmentRecord) []*pb.DepartmentNode {
	nodes := make(map[int64]*pb.DepartmentNode, len(rows))
	roots := []*pb.DepartmentNode{}
	for _, row := range rows {
		nodes[row.ID] = departmentToPB(row)
	}
	for _, row := range rows {
		node := nodes[row.ID]
		if row.ParentID == 0 || nodes[row.ParentID] == nil {
			roots = append(roots, node)
			continue
		}
		nodes[row.ParentID].Children = append(nodes[row.ParentID].Children, node)
	}
	return roots
}

func departmentToPB(row departmentRecord) *pb.DepartmentNode {
	return &pb.DepartmentNode{Id: row.ID, ParentId: row.ParentID, Name: row.Name, FullName: row.FullName, IsActive: row.IsActive, SortOrder: int32(row.SortOrder), Depth: int32(row.Depth)}
}

func locationsToPB(rows []jobLocationRecord) []*pb.LocationOption {
	list := make([]*pb.LocationOption, 0, len(rows))
	for _, row := range rows {
		list = append(list, locationToPB(row))
	}
	return list
}

func locationToPB(row jobLocationRecord) *pb.LocationOption {
	code := ""
	if row.Code != nil {
		code = *row.Code
	}
	return &pb.LocationOption{Id: row.ID, Name: row.Name, Code: code, IsActive: row.IsActive, SortOrder: int32(row.SortOrder)}
}

func inviteCodeToPB(row inviteCodeRecord) *pb.InviteCodeInfo {
	return &pb.InviteCodeInfo{Id: row.ID, Code: row.Code, CreatedBy: row.CreatedBy, ExpiresAt: formatOptionalTime(row.ExpiresAt), IsActive: row.IsActive, CreatedAt: formatTime(row.CreatedAt)}
}

func usageLogToPB(row usageLogRecord) *pb.UsageLogItem {
	return &pb.UsageLogItem{
		Id: row.ID, UserId: row.UserID, Role: row.Role, ServiceType: row.ServiceType, Endpoint: row.Endpoint, Provider: row.Provider, Model: row.Model,
		RequestChars: row.RequestChars, ResponseChars: row.ResponseChars, EstimatedTokens: row.EstimatedTokens, ObjectKey: row.ObjectKey, ObjectSize: row.ObjectSize,
		Status: row.Status, ErrorCode: row.ErrorCode, CostMs: row.CostMs, RequestId: row.RequestID, Ip: row.IP, CreatedAt: formatTime(row.CreatedAt),
	}
}

func noteToPB(row candidateNoteRecord) *pb.CandidateNoteInfo {
	return &pb.CandidateNoteInfo{
		Id: row.ID, CandidateUserId: row.CandidateUserID, ApplicationId: ptrUintValue(row.ApplicationID), AuthorUserId: row.AuthorUserID,
		Content: row.Content, Visibility: row.Visibility, CreatedAt: formatTime(row.CreatedAt), UpdatedAt: formatTime(row.UpdatedAt),
		AuthorName: fmt.Sprintf("用户%d", row.AuthorUserID),
	}
}

func tagToPB(row candidateTagRecord) *pb.CandidateTagInfo {
	return &pb.CandidateTagInfo{Id: row.ID, Name: row.Name, Color: row.Color, CreatedBy: ptrUintValue(row.CreatedBy), CreatedAt: formatTime(row.CreatedAt)}
}

func followUpToPB(row followUpTaskRecord) *pb.FollowUpTaskInfo {
	return &pb.FollowUpTaskInfo{
		Id: row.ID, CandidateUserId: row.CandidateUserID, ApplicationId: ptrUintValue(row.ApplicationID), AssigneeUserId: row.AssigneeUserID, CreatedBy: row.CreatedBy,
		Title: row.Title, Description: row.Description, DueAt: formatOptionalTime(row.DueAt), Status: row.Status, CompletedAt: formatOptionalTime(row.CompletedAt),
		CreatedAt: formatTime(row.CreatedAt), UpdatedAt: formatTime(row.UpdatedAt), AssigneeName: fmt.Sprintf("用户%d", row.AssigneeUserID),
	}
}

func profileToPB(row candidateProfileRecord) *pb.CandidateProfile {
	return &pb.CandidateProfile{RealName: row.RealName, Phone: row.Phone, Education: row.Education, School: row.School, WorkExperience: row.WorkExperience, Skills: splitSkills(row.Skills), IsComplete: row.IsComplete == 1}
}

func rowsCommon(result *gorm.DB, okMsg, missingMsg string) (*pb.CommonResponse, error) {
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: missingMsg}, nil
	}
	return &pb.CommonResponse{Code: errs.OK, Msg: okMsg}, nil
}

func positivePtr(v int64) *int64 {
	if v <= 0 {
		return nil
	}
	return &v
}

func positiveIDs(ids []int64) []int64 {
	if len(ids) == 0 {
		return nil
	}
	out := make([]int64, 0, len(ids))
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func positiveUintPtr(v uint64) *uint64 {
	if v == 0 {
		return nil
	}
	return &v
}

func ptrUintValue(v *uint64) uint64 {
	if v == nil {
		return 0
	}
	return *v
}

func ptrValue(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}

func stringPtr(v string) *string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	value := strings.TrimSpace(v)
	return &value
}

func putTrimmed(fields map[string]any, key, value string) {
	if strings.TrimSpace(value) != "" {
		fields[key] = strings.TrimSpace(value)
	}
}

func page(v int32) int32 {
	if v <= 0 {
		return 1
	}
	return v
}

func pageSize(v int32) int32 {
	if v <= 0 {
		return 20
	}
	if v > 100 {
		return 100
	}
	return v
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

func formatOptionalTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return formatTime(*t)
}

func parseOptionalTime(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	// Accept browser ISO strings with fractional seconds (toISOString) and local forms.
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return &parsed, nil
		}
		if parsed, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return &parsed, nil
		}
	}
	return nil, fmt.Errorf("invalid time: %s", value)
}

func randomCode() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return strings.ToUpper(hex.EncodeToString(buf)), nil
}

// randomInviteCode returns a 6-character invite code using mixed-case letters and digits.
func randomInviteCode() (string, error) {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 6
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, length)
	for i, b := range buf {
		out[i] = alphabet[int(b)%len(alphabet)]
	}
	return string(out), nil
}

func usageDimension(value string) string {
	switch strings.TrimSpace(value) {
	case "model":
		return "model"
	case "session":
		return "request_id"
	case "provider":
		return "provider"
	case "service_type":
		return "service_type"
	case "user":
		return "CAST(user_id AS CHAR)"
	default:
		return "CAST(user_id AS CHAR)"
	}
}

func estimateCost(tokens int64) float64 {
	return float64(tokens) * 0.000002
}

func completeFlag(values ...string) int32 {
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return 0
		}
	}
	return 1
}

func allowedResumeFile(fileName, fileType string) bool {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(fileName), "."))
	typ := strings.ToLower(strings.TrimPrefix(fileType, "."))
	return ext == "pdf" || ext == "docx" || typ == "pdf" || typ == "docx"
}

func sanitizeFileName(value string) string {
	value = filepath.Base(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "/", "_")
	value = strings.ReplaceAll(value, "\\", "_")
	if value == "" || value == "." {
		return "resume"
	}
	return value
}

func contentTypeFromFileType(fileType string) string {
	switch strings.ToLower(strings.TrimPrefix(fileType, ".")) {
	case "pdf":
		return "application/pdf"
	case "docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	default:
		return "application/octet-stream"
	}
}

func notificationPayload(userID int64, accountType, category, title, content, link, relatedType string, relatedID int64, relatedTitle, extra string) map[string]any {
	return map[string]any{
		"receiver_id":           userID,
		"receiver_account_type": accountType,
		"type":                  category,
		"title":                 title,
		"content":               content,
		"link":                  link,
		"biz_type":              relatedType,
		"biz_id":                relatedID,
		"job_title":             relatedTitle,
		"user_id":               userID,
		"account_type":          accountType,
		"category":              category,
		"related_type":          relatedType,
		"related_id":            relatedID,
		"related_title":         relatedTitle,
		"extra":                 extra,
	}
}

func candidateDisplayName(realName string, userID int64) string {
	if strings.TrimSpace(realName) != "" {
		return strings.TrimSpace(realName)
	}
	return fmt.Sprintf("候选人%d", userID)
}

func splitSkills(value string) []string {
	parts := strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == '，' || r == ';' || r == '；' || r == '\n' })
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if item := strings.TrimSpace(part); item != "" {
			out = append(out, item)
		}
	}
	return out
}

func (a *nativeStore) hasFullRecruitmentScope(ctx context.Context, actorID int64) bool {
	scope, err := a.evaluateRecruitmentScope(ctx, actorID)
	return err == nil && scope.full()
}
