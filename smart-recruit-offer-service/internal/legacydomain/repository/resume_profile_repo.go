package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"smart-recruit-offer-service/internal/legacydomain/model"
)

type ResumeProfileRepo struct {
	db *gorm.DB
}

func NewResumeProfileRepo(db *gorm.DB) *ResumeProfileRepo {
	return &ResumeProfileRepo{db: db}
}

type ResumeProfileSnapshot struct {
	ParseRun    model.ResumeParseRun
	Profile     model.ResumeProfile
	Educations  []model.ResumeEducation
	Experiences []model.ResumeExperience
	Projects    []model.ResumeProject
	Skills      []model.ResumeSkill
}

func (r *ResumeProfileRepo) SaveProfileVersion(ctx context.Context, snapshot *ResumeProfileSnapshot) error {
	if snapshot == nil {
		return fmt.Errorf("resume profile snapshot is nil")
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		if snapshot.ParseRun.ID == 0 {
			if snapshot.ParseRun.StartedAt.IsZero() {
				snapshot.ParseRun.StartedAt = now
			}
			if snapshot.ParseRun.Status == "" {
				snapshot.ParseRun.Status = "succeeded"
			}
			if snapshot.ParseRun.CompletedAt == nil && snapshot.ParseRun.Status != "running" {
				snapshot.ParseRun.CompletedAt = &now
			}
			if err := tx.Create(&snapshot.ParseRun).Error; err != nil {
				return fmt.Errorf("create resume parse run: %w", err)
			}
		} else {
			if err := tx.Model(&model.ResumeParseRun{}).
				Where("id = ?", snapshot.ParseRun.ID).
				Updates(map[string]any{
					"status":         snapshot.ParseRun.Status,
					"parser_version": snapshot.ParseRun.ParserVersion,
					"input_hash":     snapshot.ParseRun.InputHash,
					"error_message":  snapshot.ParseRun.ErrorMessage,
					"completed_at":   snapshot.ParseRun.CompletedAt,
				}).Error; err != nil {
				return fmt.Errorf("update resume parse run: %w", err)
			}
		}

		if snapshot.ParseRun.Status == "failed" && isEmptyResumeProfileSnapshot(snapshot) {
			return nil
		}

		var existing model.ResumeProfile
		err := tx.Where("parse_run_id = ?", snapshot.ParseRun.ID).First(&existing).Error
		switch {
		case err == nil:
			snapshot.Profile.ID = existing.ID
			snapshot.Profile.ResumeID = existing.ResumeID
			snapshot.Profile.UserID = existing.UserID
			snapshot.Profile.ParseRunID = existing.ParseRunID
			snapshot.Profile.Version = existing.Version
			snapshot.Profile.IsCurrent = existing.IsCurrent
			if err := tx.Model(&model.ResumeProfile{}).
				Where("id = ?", existing.ID).
				Updates(map[string]any{
					"full_name":              snapshot.Profile.FullName,
					"email":                  snapshot.Profile.Email,
					"phone":                  snapshot.Profile.Phone,
					"location":               snapshot.Profile.Location,
					"headline":               snapshot.Profile.Headline,
					"summary":                snapshot.Profile.Summary,
					"total_experience_years": snapshot.Profile.TotalExperience,
					"highest_degree":         snapshot.Profile.HighestDegree,
					"raw_json":               snapshot.Profile.RawJSON,
				}).Error; err != nil {
				return fmt.Errorf("update resume profile: %w", err)
			}
		case errors.Is(err, gorm.ErrRecordNotFound):
			var version int32
			if err := tx.Model(&model.ResumeProfile{}).
				Where("resume_id = ?", snapshot.ParseRun.ResumeID).
				Select("COALESCE(MAX(version), 0)").
				Scan(&version).Error; err != nil {
				return fmt.Errorf("get next resume profile version: %w", err)
			}
			if err := tx.Model(&model.ResumeProfile{}).
				Where("resume_id = ? AND is_current = 1", snapshot.ParseRun.ResumeID).
				Update("is_current", 0).Error; err != nil {
				return fmt.Errorf("clear current resume profile: %w", err)
			}
			snapshot.Profile.ResumeID = snapshot.ParseRun.ResumeID
			snapshot.Profile.UserID = snapshot.ParseRun.UserID
			snapshot.Profile.ParseRunID = snapshot.ParseRun.ID
			snapshot.Profile.Version = version + 1
			snapshot.Profile.IsCurrent = 1
			if err := tx.Create(&snapshot.Profile).Error; err != nil {
				return fmt.Errorf("create resume profile: %w", err)
			}
		default:
			return fmt.Errorf("find resume profile by parse run: %w", err)
		}

		return replaceResumeProfileChildren(tx, snapshot)
	})
}

func (r *ResumeProfileRepo) GetCurrentByResumeID(ctx context.Context, resumeID int64) (*model.ResumeProfile, error) {
	var profile model.ResumeProfile
	err := r.db.WithContext(ctx).
		Where("resume_id = ? AND is_current = 1", resumeID).
		Order("version DESC").
		First(&profile).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &profile, err
}

func (r *ResumeProfileRepo) GetByID(ctx context.Context, profileID uint64) (*model.ResumeProfile, error) {
	var profile model.ResumeProfile
	err := r.db.WithContext(ctx).First(&profile, profileID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &profile, err
}

func (r *ResumeProfileRepo) GetSnapshot(ctx context.Context, profileID uint64) (*ResumeProfileSnapshot, error) {
	var profile model.ResumeProfile
	if err := r.db.WithContext(ctx).First(&profile, profileID).Error; err != nil {
		return nil, err
	}

	snapshot := &ResumeProfileSnapshot{Profile: profile}
	if err := r.db.WithContext(ctx).First(&snapshot.ParseRun, profile.ParseRunID).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Where("resume_profile_id = ?", profileID).Order("sort_order ASC, id ASC").Find(&snapshot.Educations).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Where("resume_profile_id = ?", profileID).Order("sort_order ASC, id ASC").Find(&snapshot.Experiences).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Where("resume_profile_id = ?", profileID).Order("sort_order ASC, id ASC").Find(&snapshot.Projects).Error; err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Where("resume_profile_id = ?", profileID).Order("sort_order ASC, id ASC").Find(&snapshot.Skills).Error; err != nil {
		return nil, err
	}
	return snapshot, nil
}

func replaceResumeProfileChildren(tx *gorm.DB, snapshot *ResumeProfileSnapshot) error {
	profileID := snapshot.Profile.ID
	childDeletes := []any{
		&model.ResumeEducation{},
		&model.ResumeExperience{},
		&model.ResumeProject{},
		&model.ResumeSkill{},
	}
	for _, child := range childDeletes {
		if err := tx.Where("resume_profile_id = ?", profileID).Delete(child).Error; err != nil {
			return fmt.Errorf("delete resume profile child rows: %w", err)
		}
	}

	for i := range snapshot.Educations {
		snapshot.Educations[i].ResumeProfileID = profileID
		if err := tx.Create(&snapshot.Educations[i]).Error; err != nil {
			return fmt.Errorf("create resume education: %w", err)
		}
	}
	for i := range snapshot.Experiences {
		snapshot.Experiences[i].ResumeProfileID = profileID
		if err := tx.Create(&snapshot.Experiences[i]).Error; err != nil {
			return fmt.Errorf("create resume experience: %w", err)
		}
	}
	for i := range snapshot.Projects {
		snapshot.Projects[i].ResumeProfileID = profileID
		if err := tx.Create(&snapshot.Projects[i]).Error; err != nil {
			return fmt.Errorf("create resume project: %w", err)
		}
	}
	for i := range snapshot.Skills {
		snapshot.Skills[i].ResumeProfileID = profileID
		if err := tx.Create(&snapshot.Skills[i]).Error; err != nil {
			return fmt.Errorf("create resume skill: %w", err)
		}
	}
	return nil
}

func isEmptyResumeProfileSnapshot(snapshot *ResumeProfileSnapshot) bool {
	return snapshot.Profile.FullName == "" &&
		snapshot.Profile.Email == "" &&
		snapshot.Profile.Phone == "" &&
		snapshot.Profile.Location == "" &&
		snapshot.Profile.Headline == "" &&
		snapshot.Profile.Summary == "" &&
		snapshot.Profile.TotalExperience == 0 &&
		snapshot.Profile.HighestDegree == "" &&
		snapshot.Profile.RawJSON == "" &&
		len(snapshot.Educations) == 0 &&
		len(snapshot.Experiences) == 0 &&
		len(snapshot.Projects) == 0 &&
		len(snapshot.Skills) == 0
}
