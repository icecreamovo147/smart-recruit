package dto

import "smart-recruit-identity-service/internal/domain/model"

type AuthResult struct {
	UserID           int64
	Username         string
	Role             int32
	Email            string
	AccountType      string
	Roles            []string
	Permissions      []string
	TokenVersion     int32
	RefreshToken     string
	RefreshExpiresAt int64
}

type RegisterResult struct {
	UserID   int64
	Username string
	Role     int32
}

type UserRolesResult struct {
	RoleKeys       []string
	PermissionKeys []string
	DataScopes     []model.DataScope
}

type StaffUser struct {
	UserID       int64
	Username     string
	Email        string
	Status       string
	AccountType  string
	Roles        []string
	TokenVersion int32
	CreatedAt    string
}

type StaffUsersResult struct {
	Total int64
	List  []StaffUser
}

type AuditLogsResult struct {
	Total int64
	List  []model.AuthAuditLog
}
