package service

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"logic-grpc-service/model"
	"logic-grpc-service/pkg/errs"
	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/recruitment/pb"
	"logic-grpc-service/repository"
)

const maxDepartmentDepth = 2

type JobTaxonomyService struct {
	departments *repository.DepartmentRepo
	locations   *repository.JobLocationRepo
	jobs        *repository.JobRepo
	deptLocs    *repository.DepartmentLocationRepo
}

func NewJobTaxonomyService(
	departments *repository.DepartmentRepo,
	locations *repository.JobLocationRepo,
	jobs *repository.JobRepo,
	deptLocs *repository.DepartmentLocationRepo,
) *JobTaxonomyService {
	return &JobTaxonomyService{
		departments: departments,
		locations:   locations,
		jobs:        jobs,
		deptLocs:    deptLocs,
	}
}

// ── Job Options (shared by all HR users) ────────────────────────────

func (s *JobTaxonomyService) ListJobOptions(ctx context.Context, _ *pb.ListJobOptionsRequest) (*pb.ListJobOptionsResponse, error) {
	allDeps, err := s.departments.ListActive(ctx)
	if err != nil {
		logger.L().Error("list active departments failed", zap.Error(err))
		return nil, err
	}
	tree := buildDepartmentTree(allDeps)
	locs, err := s.locations.ListActive(ctx)
	if err != nil {
		logger.L().Error("list active locations failed", zap.Error(err))
		return nil, err
	}

	// Compute department-location map for frontend linkage.
	deptLocMap, err := s.DepartmentLocationMap(ctx)
	if err != nil {
		logger.L().Error("compute department location map failed", zap.Error(err))
		// Non-fatal: return empty map so existing functionality still works.
		deptLocMap = map[int64][]int64{}
	}
	pbMap := make([]*pb.DepartmentLocationMap, 0, len(deptLocMap))
	for deptID, locIDs := range deptLocMap {
		pbMap = append(pbMap, &pb.DepartmentLocationMap{
			DepartmentId: deptID,
			LocationIds:  locIDs,
		})
	}

	return &pb.ListJobOptionsResponse{
		Code:                 errs.OK,
		Msg:                  "success",
		DepartmentTree:       tree,
		Locations:            toPBLocationOptions(locs),
		DepartmentLocationMap: pbMap,
	}, nil
}

// ── Department CRUD ─────────────────────────────────────────────────

func (s *JobTaxonomyService) ListDepartments(ctx context.Context, _ *pb.ListDepartmentsRequest) (*pb.ListDepartmentsResponse, error) {
	allDeps, err := s.departments.ListAll(ctx)
	if err != nil {
		logger.L().Error("list all departments failed", zap.Error(err))
		return nil, err
	}
	return &pb.ListDepartmentsResponse{
		Code: errs.OK,
		Msg:  "success",
		List: buildDepartmentTree(allDeps),
	}, nil
}

func (s *JobTaxonomyService) CreateDepartment(ctx context.Context, req *pb.CreateDepartmentRequest) (*pb.DepartmentResponse, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return &pb.DepartmentResponse{Code: errs.ErrBadRequest, Msg: "部门名称不能为空"}, nil
	}

	// Validate parent exists and is active (unless creating root)
	if req.ParentId > 0 {
		parent, err := s.departments.GetByID(ctx, req.ParentId)
		if err != nil {
			return nil, err
		}
		if parent == nil {
			return &pb.DepartmentResponse{Code: errs.ErrBadRequest, Msg: "父部门不存在"}, nil
		}
		if parent.IsActive == 0 {
			return &pb.DepartmentResponse{Code: errs.ErrBadRequest, Msg: "父部门已停用，无法在其下新增子部门"}, nil
		}
		if parent.Depth >= maxDepartmentDepth {
			return &pb.DepartmentResponse{Code: errs.ErrBadRequest, Msg: fmt.Sprintf("部门层级不能超过 %d 层", maxDepartmentDepth)}, nil
		}
	}

	// Check if a soft-deleted department with the same parent+name exists — reactivate it.
	if existing, _ := s.departments.FindDeletedByParentAndName(ctx, req.ParentId, name); existing != nil {
		if err := s.departments.Reactivate(ctx, existing.ID, req.AdminId, name, int(req.SortOrder)); err != nil {
			return nil, err
		}
		logger.L().Info("department reactivated", zap.Int64("id", existing.ID), zap.String("name", name))
		return &pb.DepartmentResponse{
			Code:       errs.OK,
			Msg:        "success",
			Department: toPBDepartmentNode(existing, nil),
		}, nil
	}

	// Root departments: inherit_locations=0 with all active locations as default config.
	// Child departments:  inherit_locations=1 (inherit from parent).
	inherit := int32(1)
	if req.ParentId == 0 {
		inherit = 0
	}

	dep := &model.Department{
		ParentID:         req.ParentId,
		Name:             name,
		SortOrder:        int(req.SortOrder),
		IsActive:         1,
		InheritLocations: inherit,
		CreatedBy:        &req.AdminId,
	}
	if err := s.departments.Create(ctx, dep); err != nil {
		logger.L().Error("create department failed", zap.Error(err))
		// Check for duplicate name under same parent
		if strings.Contains(err.Error(), "uk_department_parent_name") || strings.Contains(err.Error(), "Duplicate") {
			return &pb.DepartmentResponse{Code: errs.ErrBadRequest, Msg: "同一父部门下已存在同名部门"}, nil
		}
		return nil, err
	}

	// Root departments get all active locations as default config.
	if req.ParentId == 0 {
		activeLocs, err := s.locations.ListActive(ctx)
		if err == nil {
			locIDs := make([]int64, len(activeLocs))
			for i, l := range activeLocs {
				locIDs[i] = l.ID
			}
			_ = s.deptLocs.ReplaceDepartmentLocations(ctx, req.AdminId, dep.ID, locIDs)
		}
	}

	// Compute parent info (capture before any DB updates)
	parentPath := "/"
	parentDepth := 0
	if req.ParentId > 0 {
		parent, _ := s.departments.GetByID(ctx, req.ParentId)
		if parent != nil {
			parentPath = parent.Path
			parentDepth = parent.Depth
		}
	}
	path := fmt.Sprintf("%s%d/", parentPath, dep.ID)
	depth := parentDepth + 1

	// Write path and depth first — BuildFullNameFromDB reads path from DB
	_ = s.departments.UpdateFields(ctx, dep.ID, map[string]any{
		"path":  path,
		"depth": depth,
	})

	// Now path is persisted, build the full name from the DB-stored path
	fullName, _ := s.departments.BuildFullNameFromDB(ctx, dep.ID)
	_ = s.departments.UpdateFields(ctx, dep.ID, map[string]any{
		"full_name": fullName,
	})
	dep.Path = path
	dep.Depth = depth
	dep.FullName = fullName

	logger.L().Info("department created", zap.Int64("id", dep.ID), zap.String("name", dep.Name))
	return &pb.DepartmentResponse{
		Code:       errs.OK,
		Msg:        "success",
		Department: toPBDepartmentNode(dep, nil),
	}, nil
}

func (s *JobTaxonomyService) UpdateDepartment(ctx context.Context, req *pb.UpdateDepartmentRequest) (*pb.DepartmentResponse, error) {
	dep, err := s.departments.GetByID(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if dep == nil {
		return &pb.DepartmentResponse{Code: errs.ErrBadRequest, Msg: "部门不存在"}, nil
	}

	name := strings.TrimSpace(req.Name)
	fields := map[string]any{}
	needsRename := false

	if name != "" && name != dep.Name {
		fields["name"] = name
		needsRename = true
	}
	if req.ParentId != dep.ParentID {
		// Moving department — validate constraints
		if err := s.validateDepartmentMove(ctx, dep.ID, req.ParentId); err != nil {
			return &pb.DepartmentResponse{Code: errs.ErrBadRequest, Msg: err.Error()}, nil
		}
		// Validate target depth: entire subtree must not exceed maxDepartmentDepth
		newParentDepth := 0
		if req.ParentId > 0 {
			parent, _ := s.departments.GetByID(ctx, req.ParentId)
			if parent != nil {
				newParentDepth = parent.Depth
			}
		}
		depthDelta := newParentDepth + 1 - dep.Depth
		maxDepth, _ := s.departments.MaxDepthInSubtree(ctx, dep.Path)
		if maxDepth+depthDelta > maxDepartmentDepth {
			return &pb.DepartmentResponse{Code: errs.ErrBadRequest, Msg: fmt.Sprintf("移动后部门层级将超过 %d 层上限", maxDepartmentDepth)}, nil
		}
		fields["parent_id"] = req.ParentId
		needsRename = true // rebuild full_name after move
	}
	if req.SortOrder != int32(dep.SortOrder) {
		fields["sort_order"] = int(req.SortOrder)
	}
	fields["updated_by"] = &req.AdminId

	if len(fields) == 0 {
		return &pb.DepartmentResponse{Code: errs.ErrBadRequest, Msg: "没有可更新字段"}, nil
	}

	// If moving or renaming, recalculate subtree paths and sync job texts.
	if needsRename {
		newParentPath := "/"
		newParentDepth := 0
		if req.ParentId > 0 {
			parent, _ := s.departments.GetByID(ctx, req.ParentId)
			if parent != nil {
				newParentPath = parent.Path
				newParentDepth = parent.Depth
			}
		}
		// Update just the name/parent first
		if err := s.departments.UpdateFields(ctx, req.Id, fields); err != nil {
			if strings.Contains(err.Error(), "uk_department_parent_name") || strings.Contains(err.Error(), "Duplicate") {
				return &pb.DepartmentResponse{Code: errs.ErrBadRequest, Msg: "同一父部门下已存在同名部门"}, nil
			}
			return nil, err
		}
		// Recalculate subtree paths, depths, full_names
		if err := s.departments.UpdateSubtreePaths(ctx, req.Id, newParentPath, newParentDepth); err != nil {
			logger.L().Error("update subtree paths failed", zap.Int64("dept_id", req.Id), zap.Error(err))
			return nil, err
		}
		// Sync job.department text for all affected jobs.
		// After UpdateSubtreePaths, the subtree lives at the new path.
		// Use the full path (e.g. /1/5/) not just /5/ so LIKE matches nested descendants.
		newRootPath := fmt.Sprintf("%s%d/", newParentPath, req.Id)
		descendantIDs, _ := s.departments.DescendantIDs(ctx, newRootPath)
		allIDs := append([]int64{req.Id}, descendantIDs...)
		if err := s.departments.SyncSubtreeJobDepartmentText(ctx, allIDs); err != nil {
			logger.L().Error("sync job department text failed", zap.Int64("dept_id", req.Id), zap.Error(err))
		}
	} else {
		if err := s.departments.UpdateFields(ctx, req.Id, fields); err != nil {
			if strings.Contains(err.Error(), "uk_department_parent_name") || strings.Contains(err.Error(), "Duplicate") {
				return &pb.DepartmentResponse{Code: errs.ErrBadRequest, Msg: "同一父部门下已存在同名部门"}, nil
			}
			return nil, err
		}
	}

	// Reload
	updated, _ := s.departments.GetByID(ctx, req.Id)
	logger.L().Info("department updated", zap.Int64("id", req.Id))
	return &pb.DepartmentResponse{
		Code:       errs.OK,
		Msg:        "success",
		Department: toPBDepartmentNode(updated, nil),
	}, nil
}

func (s *JobTaxonomyService) UpdateDepartmentStatus(ctx context.Context, req *pb.UpdateDepartmentStatusRequest) (*pb.CommonResponse, error) {
	dep, err := s.departments.GetByID(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if dep == nil {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "部门不存在"}, nil
	}
	if err := s.departments.UpdateFields(ctx, req.Id, map[string]any{
		"is_active":  int(req.IsActive),
		"updated_by": &req.AdminId,
	}); err != nil {
		return nil, err
	}
	action := "停用"
	if req.IsActive == 1 {
		action = "启用"
	}
	logger.L().Info("department status updated", zap.Int64("id", req.Id), zap.String("action", action))
	return &pb.CommonResponse{Code: errs.OK, Msg: "success"}, nil
}

func (s *JobTaxonomyService) DeleteDepartment(ctx context.Context, req *pb.DeleteDepartmentRequest) (*pb.CommonResponse, error) {
	dep, err := s.departments.GetByID(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if dep == nil {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "部门不存在"}, nil
	}

	// Cannot delete if it has children
	childCount, err := s.departments.CountChildren(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if childCount > 0 {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "该部门下存在子部门，请先删除子部门"}, nil
	}

	// Cannot delete if referenced by jobs
	jobCount, err := s.departments.CountJobReferences(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if jobCount > 0 {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "该部门已被岗位使用，请先停用，不能删除"}, nil
	}

	// Soft delete: set deleted_at, deleted_by, and is_active=0
	if err := s.departments.SoftDelete(ctx, req.Id, req.AdminId); err != nil {
		return nil, err
	}
	logger.L().Info("department soft-deleted", zap.Int64("id", req.Id), zap.Int64("admin_id", req.AdminId))
	return &pb.CommonResponse{Code: errs.OK, Msg: "部门已删除"}, nil
}

// validateDepartmentMove checks that a department is not moved to itself or its descendant.
func (s *JobTaxonomyService) validateDepartmentMove(ctx context.Context, deptID, newParentID int64) error {
	if deptID == newParentID {
		return fmt.Errorf("不能将部门移动到自己下面")
	}
	if newParentID == 0 {
		return nil
	}
	// Check that newParentID is not a descendant of deptID
	newParent, err := s.departments.GetByID(ctx, newParentID)
	if err != nil {
		return err
	}
	if newParent == nil {
		return fmt.Errorf("目标父部门不存在")
	}
	// Check that newParent is not a descendant of deptID.
	// Use path segment match: "/deptID/" must appear somewhere inside newParent's path.
	if strings.Contains(newParent.Path, fmt.Sprintf("/%d/", deptID)) {
		return fmt.Errorf("不能将部门移动到自己的子部门下面")
	}
	return nil
}

// ── Location CRUD ───────────────────────────────────────────────────

func (s *JobTaxonomyService) ListJobLocations(ctx context.Context, _ *pb.ListJobLocationsRequest) (*pb.ListJobLocationsResponse, error) {
	locs, err := s.locations.ListAll(ctx)
	if err != nil {
		logger.L().Error("list all locations failed", zap.Error(err))
		return nil, err
	}
	return &pb.ListJobLocationsResponse{
		Code: errs.OK,
		Msg:  "success",
		List: toPBLocationOptions(locs),
	}, nil
}

func (s *JobTaxonomyService) CreateJobLocation(ctx context.Context, req *pb.CreateJobLocationRequest) (*pb.JobLocationResponse, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return &pb.JobLocationResponse{Code: errs.ErrBadRequest, Msg: "地点名称不能为空"}, nil
	}

	// Check if a soft-deleted location with the same name exists — reactivate it.
	if existing, _ := s.locations.FindDeletedByName(ctx, name); existing != nil {
		if err := s.locations.Reactivate(ctx, existing.ID, req.AdminId, name, int(req.SortOrder)); err != nil {
			return nil, err
		}
		logger.L().Info("job location reactivated", zap.Int64("id", existing.ID), zap.String("name", name))
		return &pb.JobLocationResponse{
			Code:     errs.OK,
			Msg:      "success",
			Location: toPBLocationOption(existing),
		}, nil
	}

	loc := &model.JobLocation{
		Name:      name,
		SortOrder: int(req.SortOrder),
		IsActive:  1,
		CreatedBy: &req.AdminId,
	}
	if code := strings.TrimSpace(req.Code); code != "" {
		loc.Code = &code
	}
	if err := s.locations.Create(ctx, loc); err != nil {
		if strings.Contains(err.Error(), "uk_job_location_name") || strings.Contains(err.Error(), "Duplicate") {
			return &pb.JobLocationResponse{Code: errs.ErrBadRequest, Msg: "地点名称已存在"}, nil
		}
		return nil, err
	}
	logger.L().Info("job location created", zap.Int64("id", loc.ID), zap.String("name", loc.Name))
	return &pb.JobLocationResponse{
		Code:     errs.OK,
		Msg:      "success",
		Location: toPBLocationOption(loc),
	}, nil
}

func (s *JobTaxonomyService) UpdateJobLocation(ctx context.Context, req *pb.UpdateJobLocationRequest) (*pb.JobLocationResponse, error) {
	loc, err := s.locations.GetByID(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if loc == nil {
		return &pb.JobLocationResponse{Code: errs.ErrBadRequest, Msg: "地点不存在"}, nil
	}

	name := strings.TrimSpace(req.Name)
	fields := map[string]any{}
	if name != "" && name != loc.Name {
		fields["name"] = name
	}
	if req.SortOrder != int32(loc.SortOrder) {
		fields["sort_order"] = int(req.SortOrder)
	}
	if code := strings.TrimSpace(req.Code); code != "" {
		fields["code"] = code
	}
	fields["updated_by"] = &req.AdminId

	if len(fields) == 0 {
		return &pb.JobLocationResponse{Code: errs.ErrBadRequest, Msg: "没有可更新字段"}, nil
	}

	if err := s.locations.UpdateFields(ctx, req.Id, fields); err != nil {
		if strings.Contains(err.Error(), "uk_job_location_name") || strings.Contains(err.Error(), "Duplicate") {
			return &pb.JobLocationResponse{Code: errs.ErrBadRequest, Msg: "地点名称已存在"}, nil
		}
		return nil, err
	}

	// If renamed, sync job.location text
	if name != "" && name != loc.Name {
		if err := s.locations.SyncJobLocationText(ctx, req.Id, name); err != nil {
			logger.L().Error("sync job location text failed", zap.Int64("loc_id", req.Id), zap.Error(err))
		}
	}

	updated, _ := s.locations.GetByID(ctx, req.Id)
	logger.L().Info("job location updated", zap.Int64("id", req.Id))
	return &pb.JobLocationResponse{
		Code:     errs.OK,
		Msg:      "success",
		Location: toPBLocationOption(updated),
	}, nil
}

func (s *JobTaxonomyService) UpdateJobLocationStatus(ctx context.Context, req *pb.UpdateJobLocationStatusRequest) (*pb.CommonResponse, error) {
	loc, err := s.locations.GetByID(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if loc == nil {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "地点不存在"}, nil
	}
	if err := s.locations.UpdateFields(ctx, req.Id, map[string]any{
		"is_active":  int(req.IsActive),
		"updated_by": &req.AdminId,
	}); err != nil {
		return nil, err
	}
	action := "停用"
	if req.IsActive == 1 {
		action = "启用"
	}
	logger.L().Info("job location status updated", zap.Int64("id", req.Id), zap.String("action", action))
	return &pb.CommonResponse{Code: errs.OK, Msg: "success"}, nil
}

func (s *JobTaxonomyService) DeleteJobLocation(ctx context.Context, req *pb.DeleteJobLocationRequest) (*pb.CommonResponse, error) {
	loc, err := s.locations.GetByID(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if loc == nil {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "地点不存在"}, nil
	}
	jobCount, err := s.locations.CountJobReferences(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	if jobCount > 0 {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "该地点已被岗位使用，请先停用，不能删除"}, nil
	}
	// Soft delete: set deleted_at, deleted_by, and is_active=0
	if err := s.locations.SoftDelete(ctx, req.Id, req.AdminId); err != nil {
		return nil, err
	}
	logger.L().Info("job location soft-deleted", zap.Int64("id", req.Id), zap.Int64("admin_id", req.AdminId))
	return &pb.CommonResponse{Code: errs.OK, Msg: "地点已删除"}, nil
}

// ── PB conversion helpers ────────────────────────────────────────────

func buildDepartmentTree(deps []model.Department) []*pb.DepartmentNode {
	idMap := make(map[int64]*pb.DepartmentNode, len(deps))
	var roots []*pb.DepartmentNode

	for _, d := range deps {
		node := &pb.DepartmentNode{
			Id:        d.ID,
			ParentId:  d.ParentID,
			Name:      d.Name,
			FullName:  d.FullName,
			IsActive:  d.IsActive,
			SortOrder: int32(d.SortOrder),
			Depth:     int32(d.Depth),
		}
		idMap[d.ID] = node
	}

	for _, d := range deps {
		node := idMap[d.ID]
		if d.ParentID == 0 {
			roots = append(roots, node)
		} else if parent, ok := idMap[d.ParentID]; ok {
			parent.Children = append(parent.Children, node)
		} else {
			// Orphan — treat as root
			roots = append(roots, node)
		}
	}

	return roots
}

func toPBDepartmentNode(dep *model.Department, children []*pb.DepartmentNode) *pb.DepartmentNode {
	if dep == nil {
		return nil
	}
	return &pb.DepartmentNode{
		Id:        dep.ID,
		ParentId:  dep.ParentID,
		Name:      dep.Name,
		FullName:  dep.FullName,
		IsActive:  dep.IsActive,
		SortOrder: int32(dep.SortOrder),
		Depth:     int32(dep.Depth),
		Children:  children,
	}
}

func toPBLocationOption(loc *model.JobLocation) *pb.LocationOption {
	if loc == nil {
		return nil
	}
	code := ""
	if loc.Code != nil {
		code = *loc.Code
	}
	return &pb.LocationOption{
		Id:       loc.ID,
		Name:     loc.Name,
		Code:     code,
		IsActive: loc.IsActive,
		SortOrder: int32(loc.SortOrder),
	}
}

func toPBLocationOptions(locs []model.JobLocation) []*pb.LocationOption {
	out := make([]*pb.LocationOption, len(locs))
	for i := range locs {
		out[i] = toPBLocationOption(&locs[i])
	}
	return out
}

// ── Department-Location association ────────────────────────────────────

// EffectiveLocationIDs resolves the available location IDs for a department
// by following the inherit_locations chain:
//   - inherit_locations=0 → use direct config for this department.
//   - inherit_locations=1 → walk up to nearest ancestor with inherit_locations=0.
//   - If no configured ancestor is found, returns an empty list.
func (s *JobTaxonomyService) EffectiveLocationIDs(ctx context.Context, departmentID int64) ([]int64, error) {
	dep, err := s.departments.GetByID(ctx, departmentID)
	if err != nil {
		return nil, err
	}
	if dep == nil || dep.DeletedAt != nil {
		return nil, nil
	}

	if dep.InheritLocations == 0 {
		return s.deptLocs.ListDirectLocationIDs(ctx, departmentID)
	}

	// Walk up the path to find nearest ancestor with inherit_locations=0.
	// The path looks like /1/5/8/. Split, reverse, and query each ID.
	pathSegments := strings.Split(strings.Trim(dep.Path, "/"), "/")
	for i := len(pathSegments) - 1; i >= 0; i-- {
		ancestorID, err := parseID64(pathSegments[i])
		if err != nil {
			continue
		}
		if ancestorID == departmentID {
			continue
		}
		ancestor, err := s.departments.GetByID(ctx, ancestorID)
		if err != nil {
			return nil, err
		}
		if ancestor == nil || ancestor.DeletedAt != nil {
			continue
		}
		if ancestor.InheritLocations == 0 {
			return s.deptLocs.ListDirectLocationIDs(ctx, ancestorID)
		}
	}
	return nil, nil
}

// DepartmentLocationMap returns a map from department ID to effective location ID list
// for all active departments. Used by ListJobOptions.
func (s *JobTaxonomyService) DepartmentLocationMap(ctx context.Context) (map[int64][]int64, error) {
	allDeps, err := s.departments.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	result := make(map[int64][]int64, len(allDeps))
	for _, dep := range allDeps {
		ids, err := s.EffectiveLocationIDs(ctx, dep.ID)
		if err != nil {
			return nil, err
		}
		if ids == nil {
			ids = []int64{}
		}
		result[dep.ID] = ids
	}
	return result, nil
}

// ListDepartmentsLocationMap returns the effective location ID map for all active
// departments, exposed as an admin API.
func (s *JobTaxonomyService) ListDepartmentsLocationMap(ctx context.Context, req *pb.ListDepartmentsLocationMapRequest) (*pb.ListDepartmentsLocationMapResponse, error) {
	m, err := s.DepartmentLocationMap(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]*pb.DepartmentLocationMap, 0, len(m))
	for deptID, locIDs := range m {
		if locIDs == nil {
			locIDs = []int64{}
		}
		items = append(items, &pb.DepartmentLocationMap{
			DepartmentId: deptID,
			LocationIds:  locIDs,
		})
	}
	return &pb.ListDepartmentsLocationMapResponse{
		Code:  errs.OK,
		Msg:   "success",
		Items: items,
	}, nil
}

// ValidateDepartmentLocation checks that departmentID + locationID is a valid combination.
// Both must be active, non-deleted, and associated.
func (s *JobTaxonomyService) ValidateDepartmentLocation(ctx context.Context, departmentID, locationID int64) error {
	// Check department exists and is active
	dep, err := s.departments.GetByID(ctx, departmentID)
	if err != nil {
		return fmt.Errorf("查询部门失败: %w", err)
	}
	if dep == nil || dep.DeletedAt != nil || dep.IsActive == 0 {
		return fmt.Errorf("部门不存在或已停用")
	}

	// Check location exists and is active
	loc, err := s.locations.GetByID(ctx, locationID)
	if err != nil {
		return fmt.Errorf("查询地点失败: %w", err)
	}
	if loc == nil || loc.DeletedAt != nil || loc.IsActive == 0 {
		return fmt.Errorf("地点不存在或已停用")
	}

	// Check association
	effectiveIDs, err := s.EffectiveLocationIDs(ctx, departmentID)
	if err != nil {
		return fmt.Errorf("查询部门可用地点失败: %w", err)
	}

	found := false
	for _, id := range effectiveIDs {
		if id == locationID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("该部门不可用此地点")
	}
	return nil
}

// ListDepartmentLocations returns the effective locations for a single department.
func (s *JobTaxonomyService) ListDepartmentLocations(ctx context.Context, req *pb.ListDepartmentLocationsRequest) (*pb.ListDepartmentLocationsResponse, error) {
	ids, err := s.EffectiveLocationIDs(ctx, req.DepartmentId)
	if err != nil {
		return nil, err
	}
	locs, err := s.locations.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	idSet := make(map[int64]bool, len(ids))
	for _, id := range ids {
		idSet[id] = true
	}
	filtered := make([]*pb.LocationOption, 0, len(ids))
	for i := range locs {
		if idSet[locs[i].ID] {
			filtered = append(filtered, toPBLocationOption(&locs[i]))
		}
	}
	return &pb.ListDepartmentLocationsResponse{
		Code:        errs.OK,
		Msg:         "success",
		DepartmentId: req.DepartmentId,
		Locations:   filtered,
	}, nil
}

// GetDepartmentLocationConfig returns the location configuration for admin management.
func (s *JobTaxonomyService) GetDepartmentLocationConfig(ctx context.Context, req *pb.GetDepartmentLocationConfigRequest) (*pb.DepartmentLocationConfigResponse, error) {
	dep, err := s.departments.GetByID(ctx, req.DepartmentId)
	if err != nil {
		return nil, err
	}
	if dep == nil || dep.DeletedAt != nil {
		return &pb.DepartmentLocationConfigResponse{Code: errs.ErrBadRequest, Msg: "部门不存在"}, nil
	}

	directIDs, err := s.deptLocs.ListDirectLocationIDs(ctx, req.DepartmentId)
	if err != nil {
		return nil, err
	}
	if directIDs == nil {
		directIDs = []int64{}
	}

	effectiveIDs, err := s.EffectiveLocationIDs(ctx, req.DepartmentId)
	if err != nil {
		return nil, err
	}
	if effectiveIDs == nil {
		effectiveIDs = []int64{}
	}

	// Load full location data for the effective list
	locs, err := s.locations.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	effSet := make(map[int64]bool, len(effectiveIDs))
	for _, id := range effectiveIDs {
		effSet[id] = true
	}
	pbLocs := make([]*pb.LocationOption, 0, len(effectiveIDs))
	for i := range locs {
		if effSet[locs[i].ID] {
			pbLocs = append(pbLocs, toPBLocationOption(&locs[i]))
		}
	}

	// Available location pool for the admin to pick from:
	//   - Root department → all active locations.
	//   - Child department  → parent's effective locations (subset constraint).
	var availableIDs []int64
	if dep.ParentID == 0 {
		availableIDs = make([]int64, 0, len(locs))
		for i := range locs {
			availableIDs = append(availableIDs, locs[i].ID)
		}
	} else {
		availableIDs, err = s.EffectiveLocationIDs(ctx, dep.ParentID)
		if err != nil {
			return nil, err
		}
		if availableIDs == nil {
			availableIDs = []int64{}
		}
	}

	return &pb.DepartmentLocationConfigResponse{
		Code:                 errs.OK,
		Msg:                  "success",
		DepartmentId:         req.DepartmentId,
		InheritLocations:     dep.InheritLocations,
		DirectLocationIds:    directIDs,
		EffectiveLocationIds: effectiveIDs,
		Locations:            pbLocs,
		AvailableLocationIds: availableIDs,
	}, nil
}

// UpdateDepartmentLocationConfig updates a department's location configuration.
func (s *JobTaxonomyService) UpdateDepartmentLocationConfig(ctx context.Context, req *pb.UpdateDepartmentLocationConfigRequest) (*pb.DepartmentLocationConfigResponse, error) {
	dep, err := s.departments.GetByID(ctx, req.DepartmentId)
	if err != nil {
		return nil, err
	}
	if dep == nil || dep.DeletedAt != nil {
		return &pb.DepartmentLocationConfigResponse{Code: errs.ErrBadRequest, Msg: "部门不存在"}, nil
	}

	// Root department cannot inherit from parent; enforce inherit_locations=0.
	if dep.ParentID == 0 && req.InheritLocations == 1 {
		return &pb.DepartmentLocationConfigResponse{
			Code: errs.ErrBadRequest,
			Msg:  "根部门无法继承上级地点，请使用自定义地点配置",
		}, nil
	}

	// Child department with custom config: validate all location_ids are within
	// the parent's effective location pool.
	if dep.ParentID != 0 && req.InheritLocations == 0 && len(req.LocationIds) > 0 {
		parentEffIDs, err := s.EffectiveLocationIDs(ctx, dep.ParentID)
		if err != nil {
			return nil, err
		}
		parentSet := make(map[int64]bool, len(parentEffIDs))
		for _, id := range parentEffIDs {
			parentSet[id] = true
		}
		for _, lid := range req.LocationIds {
			if !parentSet[lid] {
				return &pb.DepartmentLocationConfigResponse{
					Code: errs.ErrBadRequest,
					Msg:  fmt.Sprintf("地点 ID %d 不在上级部门的有效地点范围内，无法配置", lid),
				}, nil
			}
		}
	}

	// Update inherit_locations
	if err := s.departments.UpdateFields(ctx, req.DepartmentId, map[string]any{
		"inherit_locations": int(req.InheritLocations),
		"updated_by":        &req.AdminId,
	}); err != nil {
		return nil, err
	}

	// Replace location assignments (only meaningful when inherit_locations=0,
	// but we still allow the client to send them in either case).
	if err := s.deptLocs.ReplaceDepartmentLocations(ctx, req.AdminId, req.DepartmentId, req.LocationIds); err != nil {
		return nil, err
	}

	logger.L().Info("department location config updated", zap.Int64("dept_id", req.DepartmentId), zap.Int32("inherit", req.InheritLocations), zap.Int64("admin_id", req.AdminId))

	// Return updated config
	return s.GetDepartmentLocationConfig(ctx, &pb.GetDepartmentLocationConfigRequest{DepartmentId: req.DepartmentId})
}

// parseID64 parses a string to int64.
func parseID64(s string) (int64, error) {
	var id int64
	_, err := fmt.Sscanf(s, "%d", &id)
	return id, err
}
