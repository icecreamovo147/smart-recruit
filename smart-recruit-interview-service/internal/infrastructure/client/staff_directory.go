package client

import (
	"context"

	"smart-recruit-interview-service/internal/application/port"
	sharedrepo "smart-recruit-interview-service/internal/legacydomain/repository"
)

type StaffDirectory struct {
	users *sharedrepo.UserRepo
}

func NewStaffDirectory(users *sharedrepo.UserRepo) *StaffDirectory {
	return &StaffDirectory{users: users}
}

func (d *StaffDirectory) ListInterviewers(ctx context.Context, page int32, pageSize int32, keyword string) (port.StaffPage, error) {
	users, total, err := d.users.ListStaffByRole(ctx, "interviewer", "active", keyword, page, pageSize)
	if err != nil {
		return port.StaffPage{}, err
	}
	list := make([]port.StaffUser, 0, len(users))
	for _, user := range users {
		list = append(list, port.StaffUser{
			UserID:       int64(user.ID),
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
