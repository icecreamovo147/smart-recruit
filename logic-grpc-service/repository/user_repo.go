package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"logic-grpc-service/model"
)

type UserRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, err
}

func (r *UserRepo) UpdateRole(ctx context.Context, userID int64, role int32) error {
	return r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userID).Update("role", role).Error
}

// GetByID retrieves a user by primary key.
func (r *UserRepo) GetByID(ctx context.Context, userID int64) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Where("id = ?", userID).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, err
}

// UpdateAccountType sets the account_type for a user.
func (r *UserRepo) UpdateAccountType(ctx context.Context, userID int64, accountType string) error {
	return r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userID).Update("account_type", accountType).Error
}

// IncrementTokenVersion bumps the user's token_version field.
func (r *UserRepo) IncrementTokenVersion(ctx context.Context, userID int64) error {
	return r.db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", userID).
		Update("token_version", gorm.Expr("token_version + 1")).Error
}

// UpdateStatus sets the user status (active, disabled, locked, pending).
func (r *UserRepo) UpdateStatus(ctx context.Context, userID int64, status string) error {
	return r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userID).Update("status", status).Error
}

// StaffUserResult holds a user with their role keys for staff listing.
type StaffUserResult struct {
	model.User
	RoleKeys []string `gorm:"-"`
}

// ListStaff returns staff users with optional status filter.
func (r *UserRepo) ListStaff(ctx context.Context, page, pageSize int32, status string) ([]model.User, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.User{}).Where("account_type = ?", "staff")
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var users []model.User
	if err := query.Order("id DESC").Offset(int((page - 1) * pageSize)).Limit(int(pageSize)).Find(&users).Error; err != nil {
		return nil, 0, err
	}
	return users, total, nil
}
