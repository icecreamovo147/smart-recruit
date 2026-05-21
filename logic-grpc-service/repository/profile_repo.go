package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"logic-grpc-service/model"
)

type ProfileRepo struct {
	db *gorm.DB
}

func NewProfileRepo(db *gorm.DB) *ProfileRepo {
	return &ProfileRepo{db: db}
}

func (r *ProfileRepo) GetByUserID(ctx context.Context, userID int64) (*model.CandidateProfile, error) {
	var profile model.CandidateProfile
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&profile).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &profile, err
}

func (r *ProfileRepo) Upsert(ctx context.Context, profile *model.CandidateProfile) error {
	fields := map[string]any{
		"real_name":       profile.RealName,
		"phone":           profile.Phone,
		"education":       profile.Education,
		"school":          profile.School,
		"work_experience": profile.WorkExperience,
		"skills":          profile.Skills,
		"is_complete":     profile.IsComplete,
	}
	result := r.db.WithContext(ctx).Model(&model.CandidateProfile{}).Where("user_id = ?", profile.UserID).Updates(fields)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(profile).Error
}
