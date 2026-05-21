package service

import (
	"context"
	"encoding/json"
	"strings"

	"go.uber.org/zap"

	"logic-grpc-service/model"
	"logic-grpc-service/pkg/cache"
	"logic-grpc-service/pkg/errs"
	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/recruitment/pb"
	"logic-grpc-service/repository"
)

type JobService struct {
	jobs      *repository.JobRepo
	jobCache  *cache.JobCache
	validator DepartmentLocationValidator
}

func NewJobService(jobs *repository.JobRepo, jobCache *cache.JobCache, validator DepartmentLocationValidator) *JobService {
	return &JobService{jobs: jobs, jobCache: jobCache, validator: validator}
}

func (s *JobService) CreateJob(ctx context.Context, req *pb.CreateJobRequest) (*pb.CreateJobResponse, error) {
	if req.HrId == 0 || strings.TrimSpace(req.Title) == "" {
		return &pb.CreateJobResponse{Code: errs.ErrBadRequest, Msg: "岗位名称不能为空"}, nil
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
	rows, err := s.jobs.UpdateOwned(ctx, req.HrId, req.JobId, fields)
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
	rows, err := s.jobs.OfflineOwned(ctx, req.HrId, req.JobId)
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
	rows, err := s.jobs.OnlineOwned(ctx, req.HrId, req.JobId)
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
	if req.Cursor != "" || req.Page <= 0 {
		jobs, cursor, hasMore, err := s.jobs.ListByHRCursor(ctx, req.HrId, req.Cursor, pageSize(req.PageSize))
		if err != nil {
			logger.L().Error("list hr jobs cursor failed", zap.Int64("hr_id", req.HrId), zap.Error(err))
			return nil, err
		}
		return &pb.ListJobsResponse{Code: errs.OK, Msg: "success", List: jobsToPB(ctx, jobs, s.jobs), NextCursor: cursor, HasMore: hasMore}, nil
	}
	jobs, total, err := s.jobs.ListByHR(ctx, req.HrId, page(req.Page), pageSize(req.PageSize))
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

func (s *JobService) invalidateJobCache(ctx context.Context) {
	if s.jobCache != nil {
		s.jobCache.InvalidatePublicFirstPage(ctx)
	}
}

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
