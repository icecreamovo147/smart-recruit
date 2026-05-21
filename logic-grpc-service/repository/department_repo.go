package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"logic-grpc-service/model"
)

type DepartmentRepo struct {
	db *gorm.DB
}

func NewDepartmentRepo(db *gorm.DB) *DepartmentRepo {
	return &DepartmentRepo{db: db}
}

// ListAll returns all non-deleted departments ordered by sort_order, then id.
func (r *DepartmentRepo) ListAll(ctx context.Context) ([]model.Department, error) {
	var deps []model.Department
	err := r.db.WithContext(ctx).Where("deleted_at IS NULL").Order("sort_order ASC, id ASC").Find(&deps).Error
	return deps, err
}

// ListActive returns only active, non-deleted departments.
func (r *DepartmentRepo) ListActive(ctx context.Context) ([]model.Department, error) {
	var deps []model.Department
	err := r.db.WithContext(ctx).Where("is_active = 1 AND deleted_at IS NULL").Order("sort_order ASC, id ASC").Find(&deps).Error
	return deps, err
}

// GetByID returns a single department by id, or nil if not found.
func (r *DepartmentRepo) GetByID(ctx context.Context, id int64) (*model.Department, error) {
	var dep model.Department
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&dep).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &dep, err
}

// ChildrenOf returns immediate children of parentID, ordered by sort_order.
func (r *DepartmentRepo) ChildrenOf(ctx context.Context, parentID int64) ([]model.Department, error) {
	var deps []model.Department
	err := r.db.WithContext(ctx).Where("parent_id = ?", parentID).Order("sort_order ASC, id ASC").Find(&deps).Error
	return deps, err
}

// CountChildren returns the number of direct, non-deleted children under parentID.
func (r *DepartmentRepo) CountChildren(ctx context.Context, parentID int64) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.Department{}).Where("parent_id = ? AND deleted_at IS NULL", parentID).Count(&n).Error
	return n, err
}

// DescendantIDs returns all department ids whose path starts with prefixPath
// (used to find all descendants of a node for moving/renaming).
func (r *DepartmentRepo) DescendantIDs(ctx context.Context, prefixPath string) ([]int64, error) {
	var ids []int64
	err := r.db.WithContext(ctx).Model(&model.Department{}).
		Where("path LIKE ?", prefixPath+"%").
		Pluck("id", &ids).Error
	return ids, err
}

// CountJobReferences returns the number of jobs referencing the given department id.
func (r *DepartmentRepo) CountJobReferences(ctx context.Context, deptID int64) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.Job{}).Where("department_id = ?", deptID).Count(&n).Error
	return n, err
}

// Create inserts a new department.
func (r *DepartmentRepo) Create(ctx context.Context, dep *model.Department) error {
	return r.db.WithContext(ctx).Create(dep).Error
}

// UpdateFields updates specific columns of a department.
func (r *DepartmentRepo) UpdateFields(ctx context.Context, id int64, fields map[string]any) error {
	return r.db.WithContext(ctx).Model(&model.Department{}).Where("id = ?", id).Updates(fields).Error
}

// UpdateSubtreePaths recalculates path, depth, and full_name for a department and its descendants.
// This is called after a department is moved or renamed.
func (r *DepartmentRepo) UpdateSubtreePaths(ctx context.Context, rootID int64, newParentPath string, newParentDepth int) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Get the root department.
		var root model.Department
		if err := tx.Where("id = ?", rootID).First(&root).Error; err != nil {
			return err
		}
		oldPath := root.Path
		newPath := fmt.Sprintf("%s%d/", newParentPath, rootID)
		newDepth := newParentDepth + 1

		// 2. Update the root itself.
		rootUpdates := map[string]any{
			"path":  newPath,
			"depth": newDepth,
		}
		if err := tx.Model(&model.Department{}).Where("id = ?", rootID).Updates(rootUpdates).Error; err != nil {
			return err
		}

		// 3. Update all descendants: replace oldPath prefix with newPath, adjust depth.
		if err := tx.Exec(`
			UPDATE departments
			SET path = CONCAT(?, SUBSTRING(path, ?)),
			    depth = depth + ?
			WHERE path LIKE ? AND id != ?
		`, newPath, len(oldPath)+1, newDepth-root.Depth, oldPath+"%", rootID).Error; err != nil {
			return err
		}

		// 4. Rebuild full_name for the root and all descendants.
		//    Walk each node and compose full_name from parent chain.
		var allAffected []model.Department
		if err := tx.Where("path LIKE ?", newPath+"%").Order("depth ASC, id ASC").Find(&allAffected).Error; err != nil {
			return err
		}
		// Collect all IDs referenced in any affected node's path (ancestors included).
		idSet := make(map[int64]struct{})
		for _, d := range allAffected {
			idSet[d.ID] = struct{}{}
			for _, seg := range strings.Split(strings.Trim(d.Path, "/"), "/") {
				if id, err := parseID(seg); err == nil {
					idSet[id] = struct{}{}
				}
			}
		}
		allIDs := make([]int64, 0, len(idSet))
		for id := range idSet {
			allIDs = append(allIDs, id)
		}
		// Load ALL referenced departments (affected + ancestors) into name map.
		var allRefs []model.Department
		if len(allIDs) > 0 {
			if err := tx.Where("id IN ?", allIDs).Find(&allRefs).Error; err != nil {
				return err
			}
		}
		nameMap := make(map[int64]string, len(allRefs))
		for _, d := range allRefs {
			nameMap[d.ID] = d.Name
		}
		for i := range allAffected {
			d := &allAffected[i]
			fullName := buildFullName(d.Path, nameMap)
			if d.FullName != fullName {
				if err := tx.Model(&model.Department{}).Where("id = ?", d.ID).Update("full_name", fullName).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// MaxDepthInSubtree returns the maximum depth of non-deleted departments whose path starts with prefixPath.
func (r *DepartmentRepo) MaxDepthInSubtree(ctx context.Context, prefixPath string) (int, error) {
	var maxDepth int
	err := r.db.WithContext(ctx).Model(&model.Department{}).
		Where("path LIKE ? AND deleted_at IS NULL", prefixPath+"%").
		Select("COALESCE(MAX(depth), 0)").
		Scan(&maxDepth).Error
	return maxDepth, err
}

// FindDeletedByParentAndName finds a soft-deleted department by parent_id and name.
func (r *DepartmentRepo) FindDeletedByParentAndName(ctx context.Context, parentID int64, name string) (*model.Department, error) {
	var dep model.Department
	err := r.db.WithContext(ctx).Where("parent_id = ? AND name = ? AND deleted_at IS NOT NULL", parentID, name).First(&dep).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &dep, err
}

// Reactivate clears deleted_at/deleted_by and sets is_active=1 for a department.
func (r *DepartmentRepo) Reactivate(ctx context.Context, id int64, adminID int64, name string, sortOrder int) error {
	updates := map[string]any{
		"is_active":  1,
		"deleted_at": nil,
		"deleted_by": nil,
		"updated_by": adminID,
	}
	if name != "" {
		updates["name"] = name
	}
	if sortOrder >= 0 {
		updates["sort_order"] = sortOrder
	}
	return r.db.WithContext(ctx).Model(&model.Department{}).Where("id = ?", id).Updates(updates).Error
}

// SoftDelete marks a department as deleted by setting deleted_at, deleted_by and is_active=0.
func (r *DepartmentRepo) SoftDelete(ctx context.Context, id int64, adminID int64) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.Department{}).Where("id = ?", id).Updates(map[string]any{
		"is_active":  0,
		"deleted_at": &now,
		"deleted_by": adminID,
	}).Error
}

// SyncJobDepartmentText updates jobs.department for all jobs referencing the given department.
func (r *DepartmentRepo) SyncJobDepartmentText(ctx context.Context, deptID int64, fullName string) error {
	return r.db.WithContext(ctx).Model(&model.Job{}).
		Where("department_id = ?", deptID).
		Update("department", fullName).Error
}

// SyncSubtreeJobDepartmentText updates jobs.department for all jobs whose department_id
// is in the given list of descendant IDs (including root). The full_name is looked up from the department.
func (r *DepartmentRepo) SyncSubtreeJobDepartmentText(ctx context.Context, deptIDs []int64) error {
	if len(deptIDs) == 0 {
		return nil
	}
	// Subquery: set department = departments.full_name WHERE department_id IN (...)
	return r.db.WithContext(ctx).Exec(`
		UPDATE jobs j
		JOIN departments d ON d.id = j.department_id
		SET j.department = d.full_name
		WHERE j.department_id IN ?
	`, deptIDs).Error
}

// buildFullName reconstructs the full department name from a path like "/1/8/13/"
// using a map of id→name.
func buildFullName(path string, nameMap map[int64]string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	names := make([]string, 0, len(parts))
	for _, p := range parts {
		id, err := parseID(p)
		if err != nil {
			continue
		}
		if name, ok := nameMap[id]; ok {
			names = append(names, name)
		}
	}
	return strings.Join(names, "/")
}

func parseID(s string) (int64, error) {
	var id int64
	_, err := fmt.Sscanf(s, "%d", &id)
	return id, err
}

// BuildFullNameFromDB reconstructs full_name for a department by walking up to root.
// This is used when setting full_name on initial insert.
func (r *DepartmentRepo) BuildFullNameFromDB(ctx context.Context, deptID int64) (string, error) {
	var dep model.Department
	if err := r.db.WithContext(ctx).Where("id = ?", deptID).First(&dep).Error; err != nil {
		return "", err
	}
	parts := strings.Split(strings.Trim(dep.Path, "/"), "/")
	ids := make([]int64, 0, len(parts))
	for _, p := range parts {
		id, err := parseID(p)
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return dep.Name, nil
	}
	var names []string
	if err := r.db.WithContext(ctx).Model(&model.Department{}).
		Where("id IN ?", ids).Order("depth ASC").Pluck("name", &names).Error; err != nil {
		return "", err
	}
	return strings.Join(names, "/"), nil
}
