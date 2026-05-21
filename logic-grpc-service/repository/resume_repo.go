package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"logic-grpc-service/model"
	"logic-grpc-service/pkg/errs"
)

type ResumeRepo struct {
	db *gorm.DB
}

func NewResumeRepo(db *gorm.DB) *ResumeRepo {
	return &ResumeRepo{db: db}
}

func (r *ResumeRepo) ConfirmUpload(ctx context.Context, resume *model.Resume) error {
	return r.ConfirmUploadWithTx(ctx, resume, nil)
}

func (r *ResumeRepo) ConfirmUploadWithTx(ctx context.Context, resume *model.Resume, afterCreate func(tx *gorm.DB) error) error {
	const maxRetries = 1
	resumeID := resume.ID // save original (usually 0 for new records)
	for i := 0; i <= maxRetries; i++ {
		resume.ID = resumeID // reset after a rolled-back transaction may have set it
		err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&model.Resume{}).
				Where("user_id = ? AND is_valid = ?", resume.UserID, 1).
				Update("is_valid", 0).Error; err != nil {
				return err
			}
			if err := tx.Create(resume).Error; err != nil {
				return err
			}
			if afterCreate != nil {
				return afterCreate(tx)
			}
			return nil
		})
		if err == nil {
			return nil
		}
		// If another concurrent upload committed first, the unique constraint
		// uk_user_valid_resume fires. Retry once: the other transaction already
		// invalidated the old resume, so the update is a no-op and the insert
		// will succeed.
		if errors.Is(err, gorm.ErrDuplicatedKey) && i < maxRetries {
			continue
		}
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return errs.ErrDuplicateResumeSentinel
		}
		return err
	}
	return errs.ErrDuplicateResumeSentinel
}

func (r *ResumeRepo) GetValidByUserID(ctx context.Context, userID int64) (*model.Resume, error) {
	var resume model.Resume
	err := r.db.WithContext(ctx).Where("user_id = ? AND is_valid = ?", userID, 1).Order("uploaded_at DESC").First(&resume).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &resume, err
}

func (r *ResumeRepo) UpdateParsedText(ctx context.Context, resumeID int64, text string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&model.Resume{}).
		Where("id = ?", resumeID).
		Updates(map[string]any{"parsed_text": text, "parsed_at": now}).Error
}
