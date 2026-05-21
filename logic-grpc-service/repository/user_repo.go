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
