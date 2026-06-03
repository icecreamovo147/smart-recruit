package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"logic-grpc-service/model"
	"logic-grpc-service/pkg/authz"
	"logic-grpc-service/pkg/cache"
	"logic-grpc-service/pkg/errs"
	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/recruitment/pb"
	"logic-grpc-service/repository"
)

type JobService struct {
	jobs      *repository.JobRepo
	jobCache  *cache.JobCache
	authzRepo *repository.AuthzRepo
	validator DepartmentLocationValidator
	scopeEval *scopeEvaluator
}

func NewJobService(jobs *repository.JobRepo, jobCache *cache.JobCache, authzRepo *repository.AuthzRepo, validator DepartmentLocationValidator, scopeEval *scopeEvaluator) *JobService {
	return &JobService{jobs: jobs, jobCache: jobCache, authzRepo: authzRepo, validator: validator, scopeEval: scopeEval}
}

// ScopeLevel indicates the caller's effective data access level after scope check.
type ScopeLevel int

const (
	scopeDenied               ScopeLevel = iota // no valid scope
	scopeOwned                                  // own_jobs — must filter by hr_id
	scopeDepartmentOrLocation                   // department or location scope — filter by dept/loc IDs
	scopeFull                                   // recruiting_all or system_all — no hr_id filter
)

// checkJobScope verifies scope and returns the effective access level.
// Delegates to the shared scopeEvaluator to avoid duplication with ApplicationService.
func (s *JobService) checkJobScope(ctx context.Context, userID int64, _ string, jobID int64) (ScopeLevel, error) {
	// Defensive nil guard: scopeEval is wired in production via services.go, but
	// missing in some tests. Fail-closed rather than panicking.
	if s.scopeEval == nil {
		return scopeDenied, fmt.Errorf("scope evaluator not configured")
	}
	var jobGetter func() (*jobScopeTarget, error)
	if jobID > 0 {
		jobGetter = func() (*jobScopeTarget, error) {
			job, err := s.jobs.GetByID(ctx, jobID)
			if err != nil {
				return nil, fmt.Errorf("job lookup failed: %w", err)
			}
			if job == nil {
				return nil, fmt.Errorf("job %d not found", jobID)
			}
			return &jobScopeTarget{
				ID:           job.ID,
				HrID:         job.HrID,
				DepartmentID: job.DepartmentID,
				LocationID:   job.LocationID,
			}, nil
		}
	}
	return s.scopeEval.evalScope(ctx, uint64(userID), jobGetter)
}

func (s *JobService) CreateJob(ctx context.Context, req *pb.CreateJobRequest) (*pb.CreateJobResponse, error) {
	if req.HrId == 0 || strings.TrimSpace(req.Title) == "" {
		return &pb.CreateJobResponse{Code: errs.ErrBadRequest, Msg: "岗位名称不能为空"}, nil
	}

	// Scope check: verify user has permission to create jobs in their scope.
	scopeLevel, err := s.checkJobScope(ctx, req.HrId, authz.PermJobCreate, 0)
	if err != nil {
		logger.L().Warn("job create scope denied", zap.Int64("hr_id", req.HrId), zap.Error(err))
		return &pb.CreateJobResponse{Code: errs.ErrForbidden, Msg: "无权限创建岗位：" + err.Error()}, nil
	}

	// For department/location-scoped users, validate the requested dept/loc is in scope.
	// If the user has department/location scope, they MUST provide a valid
	// department_id/location_id within their scope. Text-only is not sufficient.
	if scopeLevel < scopeFull && s.authzRepo != nil {
		deptIDs, err := s.authzRepo.GetUserDepartmentIDs(ctx, uint64(req.HrId))
		if err != nil {
			logger.L().Warn("failed to get user department IDs, treating as empty",
				zap.Int64("hr_id", req.HrId), zap.Error(err))
			deptIDs = nil
		}
		if len(deptIDs) > 0 {
			if req.DepartmentId <= 0 {
				return &pb.CreateJobResponse{Code: errs.ErrForbidden, Msg: "您的权限仅限于指定部门，请选择部门"}, nil
			}
			found := false
			for _, dID := range deptIDs {
				if uint64(req.DepartmentId) == dID {
					found = true
					break
				}
			}
			if !found {
				return &pb.CreateJobResponse{Code: errs.ErrForbidden, Msg: "无权限在该部门创建岗位"}, nil
			}
		}
		locIDs, err := s.authzRepo.GetUserLocationIDs(ctx, uint64(req.HrId))
		if err != nil {
			logger.L().Warn("failed to get user location IDs, treating as empty",
				zap.Int64("hr_id", req.HrId), zap.Error(err))
			locIDs = nil
		}
		if len(locIDs) > 0 {
			if req.LocationId <= 0 {
				return &pb.CreateJobResponse{Code: errs.ErrForbidden, Msg: "您的权限仅限于指定地点，请选择地点"}, nil
			}
			found := false
			for _, lID := range locIDs {
				if uint64(req.LocationId) == lID {
					found = true
					break
				}
			}
			if !found {
				return &pb.CreateJobResponse{Code: errs.ErrForbidden, Msg: "无权限在该地点创建岗位"}, nil
			}
		}
	}

	department := strings.TrimSpace(req.Department)
	location := strings.TrimSpace(req.Location)

	// If department_id is provided, look up the department and fill department text snapshot
	if req.DepartmentId > 0 {
		if dep, err := s.jobs.LookupDepartment(ctx, req.DepartmentId); err == nil && dep != nil {
			department = dep.FullName
		}
	}
	// If location_id is provided, look up the location and fill location text snapshot
	if req.LocationId > 0 {
		if loc, err := s.jobs.LookupLocation(ctx, req.LocationId); err == nil && loc != nil {
			location = loc.Name
		}
	}

	if department == "" {
		return &pb.CreateJobResponse{Code: errs.ErrBadRequest, Msg: "请选择部门"}, nil
	}
	if location == "" {
		return &pb.CreateJobResponse{Code: errs.ErrBadRequest, Msg: "请选择地点"}, nil
	}

	// Validate department-location combination when both IDs are explicitly provided.
	if req.DepartmentId > 0 && req.LocationId > 0 {
		if err := s.validator.ValidateDepartmentLocation(ctx, req.DepartmentId, req.LocationId); err != nil {
			return &pb.CreateJobResponse{Code: errs.ErrBadRequest, Msg: err.Error()}, nil
		}
	}

	var departmentID, locationID *int64
	if req.DepartmentId > 0 {
		departmentID = &req.DepartmentId
	}
	if req.LocationId > 0 {
		locationID = &req.LocationId
	}

	job := &model.Job{
		HrID: req.HrId, Title: req.Title,
		Department: department, DepartmentID: departmentID,
		Location: location, LocationID: locationID,
		SalaryRange: req.SalaryRange, Description: req.Description,
		Requirements: req.Requirements, Status: 1,
	}
	if err := s.jobs.Create(ctx, job); err != nil {
		logger.L().Error("create job failed", zap.Int64("hr_id", req.HrId), zap.String("title", req.Title), zap.Error(err))
		return nil, err
	}
	logger.L().Info("job created", zap.Int64("job_id", job.ID), zap.Int64("hr_id", req.HrId), zap.String("title", req.Title))
	s.invalidateJobCache(ctx)
	return &pb.CreateJobResponse{Code: errs.OK, Msg: "success", JobId: job.ID}, nil
}

func (s *JobService) UpdateJob(ctx context.Context, req *pb.UpdateJobRequest) (*pb.CommonResponse, error) {
	// Scope check.
	scopeLevel, err := s.checkJobScope(ctx, req.HrId, authz.PermJobUpdate, req.JobId)
	if err != nil {
		logger.L().Warn("job update scope denied", zap.Int64("hr_id", req.HrId), zap.Int64("job_id", req.JobId), zap.Error(err))
		return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: "无权限编辑岗位：" + err.Error()}, nil
	}

	fields := map[string]any{}
	put(fields, "title", req.Title)
	put(fields, "salary_range", req.SalaryRange)
	put(fields, "description", req.Description)
	put(fields, "requirements", req.Requirements)

	// Track whether department or location is being changed for validation.
	changingDept := false
	changingLoc := false
	var newDeptID int64
	var newLocID int64

	// Handle department: if department_id provided, resolve text snapshot
	if req.DepartmentId > 0 {
		if dep, err := s.jobs.LookupDepartment(ctx, req.DepartmentId); err == nil && dep != nil {
			fields["department"] = dep.FullName
			fields["department_id"] = req.DepartmentId
			changingDept = true
			newDeptID = req.DepartmentId
		}
	} else if strings.TrimSpace(req.Department) != "" {
		// Compat: if only text is passed (old frontend or fallback), keep it
		put(fields, "department", req.Department)
	}

	// Handle location: if location_id provided, resolve text snapshot
	if req.LocationId > 0 {
		if loc, err := s.jobs.LookupLocation(ctx, req.LocationId); err == nil && loc != nil {
			fields["location"] = loc.Name
			fields["location_id"] = req.LocationId
			changingLoc = true
			newLocID = req.LocationId
		}
	} else if strings.TrimSpace(req.Location) != "" {
		put(fields, "location", req.Location)
	}

	// Validate department-location combination if either is being changed.
	if changingDept || changingLoc {
		// Load the current job to determine the full resulting combination.
		existing, err := s.jobs.GetByID(ctx, req.JobId)
		if err != nil {
			return nil, err
		}
		if existing == nil {
			return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "岗位不存在"}, nil
		}
		resultDeptID := newDeptID
		if !changingDept && existing.DepartmentID != nil {
			resultDeptID = *existing.DepartmentID
		}
		resultLocID := newLocID
		if !changingLoc && existing.LocationID != nil {
			resultLocID = *existing.LocationID
		}
		if resultDeptID > 0 && resultLocID > 0 {
			if err := s.validator.ValidateDepartmentLocation(ctx, resultDeptID, resultLocID); err != nil {
				return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: err.Error()}, nil
			}
		}
	}

	if len(fields) == 0 {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "没有可更新字段"}, nil
	}
	var rows int64
	if scopeLevel >= scopeFull {
		rows, err = s.jobs.UpdateAny(ctx, req.JobId, fields)
	} else if scopeLevel == scopeDepartmentOrLocation {
		deptIDs, locIDs := s.getScopeDeptAndLocIDs(ctx, uint64(req.HrId))
		rows, err = s.jobs.UpdateInScope(ctx, req.JobId, deptIDs, locIDs, fields)
	} else {
		rows, err = s.jobs.UpdateOwned(ctx, req.HrId, req.JobId, fields)
	}
	if err != nil {
		logger.L().Error("update job failed", zap.Int64("job_id", req.JobId), zap.Error(err))
		return nil, err
	}
	if rows == 0 {
		return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: "无权限操作该岗位"}, nil
	}
	logger.L().Info("job updated", zap.Int64("job_id", req.JobId), zap.Int64("hr_id", req.HrId))
	s.invalidateJobCache(ctx)
	return &pb.CommonResponse{Code: errs.OK, Msg: "success"}, nil
}

func (s *JobService) OfflineJob(ctx context.Context, req *pb.OfflineJobRequest) (*pb.CommonResponse, error) {
	scopeLevel, err := s.checkJobScope(ctx, req.HrId, authz.PermJobPublish, req.JobId)
	if err != nil {
		return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: "无权限操作该岗位：" + err.Error()}, nil
	}
	var rows int64
	if scopeLevel >= scopeFull {
		rows, err = s.jobs.OfflineAny(ctx, req.JobId)
	} else if scopeLevel == scopeDepartmentOrLocation {
		deptIDs, locIDs := s.getScopeDeptAndLocIDs(ctx, uint64(req.HrId))
		rows, err = s.jobs.OfflineInScope(ctx, req.JobId, deptIDs, locIDs)
	} else {
		rows, err = s.jobs.OfflineOwned(ctx, req.HrId, req.JobId)
	}
	if err != nil {
		logger.L().Error("offline job failed", zap.Int64("job_id", req.JobId), zap.Error(err))
		return nil, err
	}
	if rows == 0 {
		return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: "无权限操作该岗位"}, nil
	}
	logger.L().Info("job offline", zap.Int64("job_id", req.JobId))
	s.invalidateJobCache(ctx)
	return &pb.CommonResponse{Code: errs.OK, Msg: "岗位已下架"}, nil
}

func (s *JobService) OnlineJob(ctx context.Context, req *pb.OfflineJobRequest) (*pb.CommonResponse, error) {
	scopeLevel, err := s.checkJobScope(ctx, req.HrId, authz.PermJobPublish, req.JobId)
	if err != nil {
		return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: "无权限操作该岗位：" + err.Error()}, nil
	}
	var rows int64
	if scopeLevel >= scopeFull {
		rows, err = s.jobs.OnlineAny(ctx, req.JobId)
	} else if scopeLevel == scopeDepartmentOrLocation {
		deptIDs, locIDs := s.getScopeDeptAndLocIDs(ctx, uint64(req.HrId))
		rows, err = s.jobs.OnlineInScope(ctx, req.JobId, deptIDs, locIDs)
	} else {
		rows, err = s.jobs.OnlineOwned(ctx, req.HrId, req.JobId)
	}
	if err != nil {
		logger.L().Error("online job failed", zap.Int64("job_id", req.JobId), zap.Error(err))
		return nil, err
	}
	if rows == 0 {
		return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: "无权限操作该岗位"}, nil
	}
	logger.L().Info("job online", zap.Int64("job_id", req.JobId))
	s.invalidateJobCache(ctx)
	return &pb.CommonResponse{Code: errs.OK, Msg: "岗位已上线"}, nil
}

func (s *JobService) ListHRJobs(ctx context.Context, req *pb.ListHRJobsRequest) (*pb.ListJobsResponse, error) {
	// Determine scope filters pushed to the database query.
	// - recruiting_all/system_all: no filter (sees everything)
	// - own_jobs: hrID=req.HrId, no dept/loc IDs
	// - department/location: hrID may be req.HrId (own_jobs OR), plus deptIDs/locIDs
	effectiveHrID := req.HrId
	var deptIDs, locIDs []uint64
	isFullAccess := false
	if s.authzRepo != nil {
		scopeKeys, err := s.authzRepo.GetUserScopeKeys(ctx, uint64(req.HrId))
		if err != nil {
			logger.L().Warn("failed to get user scope keys, treating as empty",
				zap.Int64("hr_id", req.HrId), zap.Error(err))
			scopeKeys = nil
		}
		for _, sk := range scopeKeys {
			if sk == authz.ScopeRecruitingAll || sk == authz.ScopeSystemAll {
				isFullAccess = true
				effectiveHrID = 0
				deptIDs, locIDs = nil, nil
				break
			}
		}
		if !isFullAccess {
			deptIDs, err = s.authzRepo.GetUserDepartmentIDs(ctx, uint64(req.HrId))
			if err != nil {
				logger.L().Warn("failed to get user department IDs for listing, treating as empty",
					zap.Int64("hr_id", req.HrId), zap.Error(err))
				deptIDs = nil
			}
			locIDs, err = s.authzRepo.GetUserLocationIDs(ctx, uint64(req.HrId))
			if err != nil {
				logger.L().Warn("failed to get user location IDs for listing, treating as empty",
					zap.Int64("hr_id", req.HrId), zap.Error(err))
				locIDs = nil
			}
			hasOnlyDeptOrLoc := len(deptIDs) > 0 || len(locIDs) > 0
			hasOwnJobs := false
			for _, sk := range scopeKeys {
				if sk == authz.ScopeOwnJobs {
					hasOwnJobs = true
					break
				}
			}
			// No valid data scope at all → forbid (EC-2: no scope = no business data access)
			if !hasOwnJobs && !hasOnlyDeptOrLoc {
				return &pb.ListJobsResponse{Code: errs.ErrForbidden, Msg: "无数据范围权限"}, nil
			}
			if hasOnlyDeptOrLoc && !hasOwnJobs {
				// Department/location-only scope: include jobs from all HRs in those depts/locs
				effectiveHrID = 0
			}
		}
	}
	if req.Cursor != "" || req.Page <= 0 {
		jobs, cursor, hasMore, err := s.jobs.ListByScopeCursor(ctx, effectiveHrID, deptIDs, locIDs, req.Cursor, pageSize(req.PageSize))
		if err != nil {
			logger.L().Error("list hr jobs cursor failed", zap.Int64("hr_id", req.HrId), zap.Error(err))
			return nil, err
		}
		return &pb.ListJobsResponse{Code: errs.OK, Msg: "success", List: jobsToPB(ctx, jobs, s.jobs), NextCursor: cursor, HasMore: hasMore}, nil
	}
	jobs, total, err := s.jobs.ListByScope(ctx, effectiveHrID, deptIDs, locIDs, page(req.Page), pageSize(req.PageSize))
	if err != nil {
		logger.L().Error("list hr jobs failed", zap.Int64("hr_id", req.HrId), zap.Error(err))
		return nil, err
	}
	return &pb.ListJobsResponse{Code: errs.OK, Msg: "success", Total: total, List: jobsToPB(ctx, jobs, s.jobs)}, nil
}

func (s *JobService) ListPublicJobs(ctx context.Context, req *pb.ListPublicJobsRequest) (*pb.ListJobsResponse, error) {
	keyword := strings.TrimSpace(req.Keyword)
	ps := pageSize(req.PageSize)

	// Cache only for first page, no keyword, no cursor
	if req.Page == 1 && keyword == "" && req.Cursor == "" && s.jobCache != nil {
		if cached, ok := s.jobCache.GetPublicFirstPage(ctx, ps); ok {
			var resp pb.ListJobsResponse
			if err := json.Unmarshal(cached, &resp); err == nil {
				return &resp, nil
			}
		}
	}

	if req.Cursor != "" || req.Page <= 0 {
		jobs, cursor, hasMore, err := s.jobs.ListPublicCursor(ctx, keyword, req.Cursor, ps)
		if err != nil {
			logger.L().Error("list public jobs cursor failed", zap.String("keyword", keyword), zap.Error(err))
			return nil, err
		}
		resp := &pb.ListJobsResponse{Code: errs.OK, Msg: "success", List: jobsToPB(ctx, jobs, s.jobs), NextCursor: cursor, HasMore: hasMore}
		return resp, nil
	}
	jobs, total, err := s.jobs.ListPublic(ctx, keyword, req.Page, ps)
	if err != nil {
		logger.L().Error("list public jobs failed", zap.String("keyword", keyword), zap.Error(err))
		return nil, err
	}
	resp := &pb.ListJobsResponse{Code: errs.OK, Msg: "success", Total: total, List: jobsToPB(ctx, jobs, s.jobs)}
	if req.Page == 1 && keyword == "" && s.jobCache != nil {
		if data, err := json.Marshal(resp); err == nil {
			s.jobCache.SetPublicFirstPage(ctx, ps, data)
		}
	}
	return resp, nil
}

// getScopeDeptAndLocIDs returns the department and location IDs that a department/location-scoped
// user has access to. Used for InScope repo operations (update/offline/online).
func (s *JobService) getScopeDeptAndLocIDs(ctx context.Context, userID uint64) ([]uint64, []uint64) {
	if s.authzRepo == nil {
		return nil, nil
	}
	deptIDs, _ := s.authzRepo.GetUserDepartmentIDs(ctx, userID)
	locIDs, _ := s.authzRepo.GetUserLocationIDs(ctx, userID)
	return deptIDs, locIDs
}

func (s *JobService) invalidateJobCache(ctx context.Context) {
	if s.jobCache != nil {
		s.jobCache.InvalidatePublicFirstPage(ctx)
	}
}

// GetJobDetail is a public-facing API — it intentionally does not perform scope checks.
// Only published jobs (Status == 1) are returned; drafts, offline, or deleted jobs are
// treated as not found. If this endpoint is later exposed on an internal route, add
// scope enforcement via checkJobScope.
func (s *JobService) GetJobDetail(ctx context.Context, req *pb.GetJobDetailRequest) (*pb.GetJobDetailResponse, error) {
	job, err := s.jobs.GetByID(ctx, req.JobId)
	if err != nil {
		logger.L().Error("get job detail failed", zap.Int64("job_id", req.JobId), zap.Error(err))
		return nil, err
	}
	if job == nil || job.Status != 1 {
		return &pb.GetJobDetailResponse{Code: errs.ErrBadRequest, Msg: "岗位不存在或已下架"}, nil
	}
	return &pb.GetJobDetailResponse{Code: errs.OK, Msg: "success", Job: jobsToPB(ctx, []model.Job{*job}, s.jobs)[0]}, nil
}
