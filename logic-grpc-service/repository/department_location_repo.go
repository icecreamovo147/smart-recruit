package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"logic-grpc-service/model"
)

type DepartmentLocationRepo struct {
	db *gorm.DB
}

func NewDepartmentLocationRepo(db *gorm.DB) *DepartmentLocationRepo {
	return &DepartmentLocationRepo{db: db}
}

// ListDirectLocationIDs returns the location IDs directly configured for a department
// (active, non-deleted rows only).
func (r *DepartmentLocationRepo) ListDirectLocationIDs(ctx context.Context, departmentID int64) ([]int64, error) {
	var ids []int64
	err := r.db.WithContext(ctx).Model(&model.DepartmentLocation{}).
		Where("department_id = ? AND is_active = 1 AND deleted_at IS NULL", departmentID).
		Pluck("location_id", &ids).Error
	return ids, err
}

// ListDirectLocations returns the full location rows directly configured for a department.
func (r *DepartmentLocationRepo) ListDirectLocations(ctx context.Context, departmentID int64) ([]model.JobLocation, error) {
	var locs []model.JobLocation
	err := r.db.WithContext(ctx).
		Table("job_locations l").
		Joins("JOIN department_locations dl ON dl.location_id = l.id AND dl.department_id = ? AND dl.is_active = 1 AND dl.deleted_at IS NULL", departmentID).
		Where("l.is_active = 1 AND l.deleted_at IS NULL").
		Order("l.sort_order ASC, l.id ASC").
		Find(&locs).Error
	return locs, err
}

// ExistsEffective checks whether an active (non-deleted) department-location pair exists.
func (r *DepartmentLocationRepo) ExistsEffective(ctx context.Context, departmentID, locationID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.DepartmentLocation{}).
		Where("department_id = ? AND location_id = ? AND is_active = 1 AND deleted_at IS NULL", departmentID, locationID).
		Count(&count).Error
	return count > 0, err
}

// FindDeleted looks up a soft-deleted department-location row.
func (r *DepartmentLocationRepo) FindDeleted(ctx context.Context, departmentID, locationID int64) (*model.DepartmentLocation, error) {
	var dl model.DepartmentLocation
	err := r.db.WithContext(ctx).
		Where("department_id = ? AND location_id = ? AND deleted_at IS NOT NULL", departmentID, locationID).
		First(&dl).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &dl, err
}

// Reactivate clears deleted_at/deleted_by and sets is_active=1 for a department-location row.
func (r *DepartmentLocationRepo) Reactivate(ctx context.Context, id, adminID int64) error {
	return r.db.WithContext(ctx).Model(&model.DepartmentLocation{}).Where("id = ?", id).Updates(map[string]any{
		"is_active":  1,
		"deleted_at": nil,
		"deleted_by": nil,
		"updated_by": adminID,
	}).Error
}

// SoftDeleteMissing marks as deleted all active department-location rows for the given
// department whose location_id is NOT in keepLocationIDs. Used during full replace.
func (r *DepartmentLocationRepo) SoftDeleteMissing(ctx context.Context, adminID, departmentID int64, keepLocationIDs []int64) error {
	now := time.Now()
	q := r.db.WithContext(ctx).Model(&model.DepartmentLocation{}).
		Where("department_id = ? AND is_active = 1 AND deleted_at IS NULL", departmentID)
	if len(keepLocationIDs) > 0 {
		q = q.Where("location_id NOT IN ?", keepLocationIDs)
	}
	return q.Updates(map[string]any{
		"is_active":  0,
		"deleted_at": &now,
		"deleted_by": adminID,
	}).Error
}

// BatchInsert inserts new department-location rows (ignoring duplicates).
func (r *DepartmentLocationRepo) BatchInsert(ctx context.Context, adminID, departmentID int64, locationIDs []int64) error {
	if len(locationIDs) == 0 {
		return nil
	}
	rows := make([]model.DepartmentLocation, 0, len(locationIDs))
	for _, lid := range locationIDs {
		rows = append(rows, model.DepartmentLocation{
			DepartmentID: departmentID,
			LocationID:   lid,
			IsActive:     1,
			CreatedBy:    &adminID,
		})
	}
	return r.db.WithContext(ctx).Create(&rows).Error
}

// ReplaceDepartmentLocations updates or replaces the location assignment for a department
// in a transaction. Used by UpdateDepartmentLocationConfig.
// Deletes relations not in the new list, reactivates deleted relations that are re-added,
// and inserts new ones.
func (r *DepartmentLocationRepo) ReplaceDepartmentLocations(ctx context.Context, adminID, departmentID int64, locationIDs []int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Soft-delete existing rows whose location_id is not in the new list.
		now := time.Now()
		delQ := tx.Model(&model.DepartmentLocation{}).
			Where("department_id = ? AND is_active = 1 AND deleted_at IS NULL", departmentID)
		if len(locationIDs) > 0 {
			delQ = delQ.Where("location_id NOT IN ?", locationIDs)
		}
		if err := delQ.Updates(map[string]any{
			"is_active":  0,
			"deleted_at": &now,
			"deleted_by": adminID,
		}).Error; err != nil {
			return err
		}

		// 2. For each location_id in the new list, reactivate a deleted row or insert.
		for _, lid := range locationIDs {
			var existing model.DepartmentLocation
			err := tx.Where("department_id = ? AND location_id = ?", departmentID, lid).First(&existing).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// Insert new
				dl := model.DepartmentLocation{
					DepartmentID: departmentID,
					LocationID:   lid,
					IsActive:     1,
					CreatedBy:    &adminID,
				}
				if err := tx.Create(&dl).Error; err != nil {
					// Ignore duplicate key errors (concurrent insert)
					if !errors.Is(err, gorm.ErrDuplicatedKey) {
						return err
					}
				}
			} else if err != nil {
				return err
			} else {
				// Reactivate if soft-deleted or just ensure active
				if existing.DeletedAt != nil || existing.IsActive != 1 {
					if err := tx.Model(&existing).Updates(map[string]any{
						"is_active":  1,
						"deleted_at": nil,
						"deleted_by": nil,
						"updated_by": adminID,
					}).Error; err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
}
