package service

import (
	"context"
	"fmt"

	"smart-recruit-identity-service/internal/domain/model"
	"smart-recruit-identity-service/internal/domain/repository"
)

type memoryUsers struct {
	nextID     int64
	byID       map[int64]*model.User
	byUsername map[string]*model.User
}

func newMemoryUsers() *memoryUsers {
	return &memoryUsers{nextID: 1, byID: map[int64]*model.User{}, byUsername: map[string]*model.User{}}
}

func (m *memoryUsers) seed(user *model.User) {
	m.byID[user.ID] = user
	m.byUsername[user.Username] = user
	if user.ID >= m.nextID {
		m.nextID = user.ID + 1
	}
}

func (m *memoryUsers) GetByUsername(_ context.Context, username string) (*model.User, error) {
	return m.byUsername[username], nil
}

func (m *memoryUsers) GetByID(_ context.Context, id int64) (*model.User, error) {
	return m.byID[id], nil
}

func (m *memoryUsers) Create(_ context.Context, user *model.User) error {
	if user.ID == 0 {
		user.ID = m.nextID
		m.nextID++
	}
	m.byID[user.ID] = user
	m.byUsername[user.Username] = user
	return nil
}

func (m *memoryUsers) UpdateEmail(_ context.Context, userID int64, email string) error {
	user := m.byID[userID]
	if user == nil {
		return fmt.Errorf("user %d not found", userID)
	}
	user.Email = email
	return nil
}

func (m *memoryUsers) ListStaff(context.Context, int32, int32, string) ([]model.User, int64, error) {
	rows := make([]model.User, 0)
	for _, user := range m.byID {
		if user.AccountType == model.AccountTypeStaff {
			rows = append(rows, *user)
		}
	}
	return rows, int64(len(rows)), nil
}

type memoryAuthz struct {
	roles          map[string]*model.Role
	userRoles      map[int64][]string
	userPerms      map[int64][]string
	assignedRoles  map[int64][]string
	assignedScopes map[int64][]string
	restoredRole   bool
	auditRecords   []model.AuthAuditDecision
}

func newMemoryAuthz() *memoryAuthz {
	return &memoryAuthz{
		roles: map[string]*model.Role{
			model.RoleCandidate:   {ID: 1, RoleKey: model.RoleCandidate},
			model.RoleRecruiter:   {ID: 2, RoleKey: model.RoleRecruiter},
			model.RoleSystemAdmin: {ID: 3, RoleKey: model.RoleSystemAdmin},
		},
		userRoles:      map[int64][]string{},
		userPerms:      map[int64][]string{},
		assignedRoles:  map[int64][]string{},
		assignedScopes: map[int64][]string{},
	}
}

func (m *memoryAuthz) GetRoleByKey(_ context.Context, roleKey string) (*model.Role, error) {
	return m.roles[roleKey], nil
}

func (m *memoryAuthz) ListRoles(context.Context) ([]model.Role, error) {
	rows := make([]model.Role, 0, len(m.roles))
	for _, role := range m.roles {
		rows = append(rows, *role)
	}
	return rows, nil
}

func (m *memoryAuthz) ListPermissions(context.Context) ([]model.Permission, error) {
	return []model.Permission{{PermissionKey: model.PermAdminRoleManage}}, nil
}

func (m *memoryAuthz) GetUserRoles(_ context.Context, userID uint64) ([]string, error) {
	return append([]string(nil), m.userRoles[int64(userID)]...), nil
}

func (m *memoryAuthz) GetUserPermissions(_ context.Context, userID uint64) ([]string, error) {
	return append([]string(nil), m.userPerms[int64(userID)]...), nil
}

func (m *memoryAuthz) GetUserDataScopes(context.Context, uint64) ([]model.DataScope, error) {
	return nil, nil
}

func (m *memoryAuthz) LoadPrincipal(_ context.Context, userID uint64) (*model.Principal, error) {
	return &model.Principal{UserID: int64(userID), Roles: m.userRoles[int64(userID)]}, nil
}

func (m *memoryAuthz) AssignRole(_ context.Context, userID, roleID uint64, _ *uint64) error {
	for _, role := range m.roles {
		if role.ID == roleID {
			m.assignedRoles[int64(userID)] = append(m.assignedRoles[int64(userID)], role.RoleKey)
			m.userRoles[int64(userID)] = append(m.userRoles[int64(userID)], role.RoleKey)
			return nil
		}
	}
	return fmt.Errorf("role %d not found", roleID)
}

func (m *memoryAuthz) RevokeRoleWithLastAdminGuard(_ context.Context, userID, roleID uint64, roleKey string, _ *uint64) (bool, error) {
	if roleKey == model.RoleSystemAdmin {
		return false, repository.ErrLastAdmin
	}
	for _, role := range m.roles {
		if role.ID == roleID {
			m.userRoles[int64(userID)] = nil
			return true, nil
		}
	}
	return false, repository.ErrUserRoleNotFound
}

func (m *memoryAuthz) RestoreRole(context.Context, uint64, uint64) error {
	m.restoredRole = true
	return nil
}

func (m *memoryAuthz) CountActiveUsersWithRole(context.Context, string) (int64, error) {
	return 2, nil
}

func (m *memoryAuthz) AssignDataScope(_ context.Context, userID uint64, scopeKey, _ string, _ uint64, _ *uint64) error {
	m.assignedScopes[int64(userID)] = append(m.assignedScopes[int64(userID)], scopeKey)
	return nil
}

func (m *memoryAuthz) RevokeDataScope(context.Context, uint64) error {
	return nil
}

func (m *memoryAuthz) GetScopeOwnerID(context.Context, uint64) (uint64, error) {
	return 7, nil
}

func (m *memoryAuthz) IncrementTokenVersion(context.Context, uint64) (int32, error) {
	return 2, nil
}

func (m *memoryAuthz) RecordAuthDecision(_ context.Context, decision model.AuthAuditDecision) error {
	m.auditRecords = append(m.auditRecords, decision)
	return nil
}

func (m *memoryAuthz) QueryAuthAuditLogs(context.Context, *uint64, string, string, int, int) ([]model.AuthAuditLog, int64, error) {
	return nil, 0, nil
}
