package client

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	"smart-recruit-interview-service/internal/application/port"
)

type StaffDirectory struct {
	db *gorm.DB
}

func NewStaffDirectory(db *gorm.DB) *StaffDirectory {
	return &StaffDirectory{db: db}
}

func (d *StaffDirectory) ListInterviewers(ctx context.Context, page int32, pageSize int32, keyword string) (port.StaffPage, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	query := d.db.WithContext(ctx).Model(&staffUserRecord{}).
		Joins("JOIN user_roles ur ON ur.user_id = users.id AND ur.revoked_at IS NULL").
		Joins("JOIN roles r ON r.id = ur.role_id AND r.role_key = ?", "interviewer").
		Where("users.account_type = ?", "staff").
		Where("users.status = ?", "active")
	keyword = strings.TrimSpace(keyword)
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("(users.username LIKE ? OR users.email LIKE ?)", like, like)
	}

	var total int64
	if err := query.Distinct("users.id").Count(&total).Error; err != nil {
		return port.StaffPage{}, err
	}

	var users []staffUserRecord
	if err := query.Select("users.*").
		Group("users.id").
		Order("users.id DESC").
		Offset(int((page - 1) * pageSize)).
		Limit(int(pageSize)).
		Find(&users).Error; err != nil {
		return port.StaffPage{}, err
	}
	list := make([]port.StaffUser, 0, len(users))
	for _, user := range users {
		list = append(list, port.StaffUser{
			UserID:       user.ID,
			Username:     user.Username,
			Email:        user.Email,
			Status:       user.Status,
			AccountType:  user.AccountType,
			Roles:        []string{"interviewer"},
			TokenVersion: user.TokenVersion,
			CreatedAt:    user.CreatedAt,
		})
	}
	return port.StaffPage{Total: total, List: list}, nil
}

type staffUserRecord struct {
	ID           int64 `gorm:"primaryKey"`
	Username     string
	Password     string
	Role         int32 `gorm:"column:role"`
	Email        string
	AccountType  string `gorm:"column:account_type"`
	Status       string `gorm:"column:status"`
	TokenVersion int32  `gorm:"column:token_version"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (staffUserRecord) TableName() string { return "users" }
