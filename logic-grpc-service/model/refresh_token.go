package model

import "time"

type RefreshToken struct {
	ID                int64      `gorm:"primaryKey"`
	UserID            int64      `gorm:"column:user_id;not null"`
	TokenHash         string     `gorm:"column:token_hash;type:char(64);uniqueIndex;not null"`
	FamilyID          string     `gorm:"column:family_id;type:varchar(64);not null;index"`
	ExpiresAt         time.Time  `gorm:"column:expires_at;not null;index"`
	RevokedAt         *time.Time `gorm:"column:revoked_at"`
	ReplacedByHash    *string    `gorm:"column:replaced_by_hash;type:char(64)"`
	ReuseDetectedAt   *time.Time `gorm:"column:reuse_detected_at"`
	CreatedIP         *string    `gorm:"column:created_ip;type:varchar(64)"`
	CreatedUserAgent  *string    `gorm:"column:created_user_agent;type:varchar(255)"`
	CreatedAt         time.Time  `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt         time.Time  `gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (RefreshToken) TableName() string { return "refresh_tokens" }
