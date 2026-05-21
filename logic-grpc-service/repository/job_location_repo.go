package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"logic-grpc-service/model"
)

type JobLocationRepo struct {
	db *gorm.DB
}

func NewJobLocationRepo(db *gorm.DB) *JobLocationRepo {
	return &JobLocationRepo{db: db}
}

// ListAll returns all non-deleted locations ordered by sort_order, then id.
func (r *JobLocationRepo) ListAll(ctx context.Context) ([]model.JobLocation, error) {
	var locs []model.JobLocation
	err := r.db.WithContext(ctx).Where("deleted_at IS NULL").Order("sort_order ASC, id ASC").Find(&locs).Error
	return locs, err
}

// ListActive returns only active, non-deleted locations.
func (r *JobLocationRepo) ListActive(ctx context.Context) ([]model.JobLocation, error) {
	var locs []model.JobLocation
	err := r.db.WithContext(ctx).Where("is_active = 1 AND deleted_at IS NULL").Order("sort_order ASC, id ASC").Find(&locs).Error
	return locs, err
}

// GetByID returns a single location by id, or nil if not found.
func (r *JobLocationRepo) GetByID(ctx context.Context, id int64) (*model.JobLocation, error) {
	var loc model.JobLocation
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&loc).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &loc, err
}

// CountJobReferences returns the number of jobs referencing the given location id.
func (r *JobLocationRepo) CountJobReferences(ctx context.Context, locID int64) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.Job{}).Where("location_id = ?", locID).Count(&n).Error
	return n, err
}

// Create inserts a new location.
func (r *JobLocationRepo) Create(ctx context.Context, loc *model.JobLocation) error {
	return r.db.WithContext(ctx).Create(loc).Error
}

// UpdateFields updates specific columns of a location.
func (r *JobLocationRepo) UpdateFields(ctx context.Context, id int64, fields map[string]any) error {
	return r.db.WithContext(ctx).Model(&model.JobLocation{}).Where("id = ?", id).Updates(fields).Error
}

// FindDeletedByName finds a soft-deleted location by name.
func (r *JobLocationRepo) FindDeletedByName(ctx context.Context, name string) (*model.JobLocation, error) {
	var loc model.JobLocation
	err := r.db.WithContext(ctx).Where("name = ? AND deleted_at IS NOT NULL", name).First(&loc).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &loc, err
}

// Reactivate clears deleted_at/deleted_by and sets is_active=1 for a location.
func (r *JobLocationRepo) Reactivate(ctx context.Context, id int64, adminID int64, name string, sortOrder int) error {
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
	return r.db.WithContext(ctx).Model(&model.JobLocation{}).Where("id = ?", id).Updates(updates).Error
}

// SoftDelete marks a location as deleted by setting deleted_at, deleted_by and is_active=0.
func (r *JobLocationRepo) SoftDelete(ctx context.Context, id int64, adminID int64) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.JobLocation{}).Where("id = ?", id).Updates(map[string]any{
		"is_active":  0,
		"deleted_at": &now,
		"deleted_by": adminID,
	}).Error
}

// SyncJobLocationText updates jobs.location for all jobs referencing the given location.
func (r *JobLocationRepo) SyncJobLocationText(ctx context.Context, locID int64, name string) error {
	return r.db.WithContext(ctx).Model(&model.Job{}).
		Where("location_id = ?", locID).
		Update("location", name).Error
}
