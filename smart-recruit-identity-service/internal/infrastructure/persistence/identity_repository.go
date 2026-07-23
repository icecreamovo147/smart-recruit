package persistence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"smart-recruit-identity-service/internal/domain/model"
	"smart-recruit-identity-service/internal/domain/repository"
)

type Repositories struct {
	Users   *UserRepository
	Tokens  *RefreshTokenRepository
	Authz   *AuthzRepository
	Invites *InviteCodeRepository
	Audit   *AuditRepository
	Tenants *TenantRepository
}

func NewRepositories(db *gorm.DB) Repositories {
	authz := NewAuthzRepository(db)
	return Repositories{
		Users:   NewUserRepository(db),
		Tokens:  NewRefreshTokenRepository(db),
		Authz:   authz,
		Invites: NewInviteCodeRepository(db),
		Audit:   NewAuditRepository(db),
		Tenants: NewTenantRepository(db),
	}
}

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	row := userRow{
		Username:     user.Username,
		Password:     user.PasswordHash,
		Role:         user.Role,
		Email:        user.Email,
		AccountType:  user.AccountType,
		Status:       user.Status,
		TokenVersion: user.TokenVersion,
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return err
	}
	user.ID = row.ID
	user.CreatedAt = row.CreatedAt
	return nil
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	var row userRow
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return row.toDomain(), nil
}

func (r *UserRepository) GetByID(ctx context.Context, userID int64) (*model.User, error) {
	var row userRow
	err := r.db.WithContext(ctx).Where("id = ?", userID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return row.toDomain(), nil
}

func (r *UserRepository) UpdateEmail(ctx context.Context, userID int64, email string) error {
	return r.db.WithContext(ctx).Model(&userRow{}).Where("id = ?", userID).Update("email", email).Error
}

func (r *UserRepository) ListStaff(ctx context.Context, page, pageSize int32, status string) ([]model.User, int64, error) {
	query := r.db.WithContext(ctx).Model(&userRow{}).Where("account_type = ?", model.AccountTypeStaff)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []userRow
	err := query.Order("id DESC").Offset(int((page - 1) * pageSize)).Limit(int(pageSize)).Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	users := make([]model.User, len(rows))
	for i := range rows {
		users[i] = *rows[i].toDomain()
	}
	return users, total, nil
}

func (r *UserRepository) ListPlatformAccounts(ctx context.Context, page, pageSize int32, status string) ([]model.PlatformAccount, int64, error) {
	query := r.db.WithContext(ctx).Table("users user").
		Joins("JOIN platform_user_roles assignment ON assignment.user_id = user.id AND assignment.revoked_at IS NULL").
		Joins("JOIN roles role ON role.id = assignment.role_id AND role.scope_type = ?", model.RoleScopePlatform)
	if status != "" {
		query = query.Where("user.status = ?", status)
	}
	var total int64
	if err := query.Distinct("user.id").Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []platformAccountRow
	if err := query.Select("user.id, user.username, user.email, user.account_type, user.status, user.token_version, user.created_at, GROUP_CONCAT(DISTINCT role.role_key ORDER BY role.role_key) AS role_keys").
		Group("user.id, user.username, user.email, user.account_type, user.status, user.token_version, user.created_at").
		Order("user.id DESC").Offset(int((page - 1) * pageSize)).Limit(int(pageSize)).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	result := make([]model.PlatformAccount, len(rows))
	for i, row := range rows {
		result[i] = row.toDomain()
	}
	return result, total, nil
}

func (r *UserRepository) CreatePlatformAccount(ctx context.Context, user *model.User, roleID uint64, assignedBy *uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var role roleRow
		if err := tx.Where("id = ? AND scope_type = ?", roleID, model.RoleScopePlatform).First(&role).Error; err != nil {
			return err
		}
		row := userRow{Username: user.Username, Password: user.PasswordHash, Role: user.Role, Email: user.Email, AccountType: user.AccountType, Status: user.Status, TokenVersion: user.TokenVersion}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		user.ID = row.ID
		if err := tx.Create(&platformUserRoleRow{UserID: uint64(row.ID), RoleID: roleID, AssignedBy: assignedBy, AssignedAt: time.Now()}).Error; err != nil {
			return err
		}
		return createPlatformResourceAudit(tx, ctx, "platform_user.create", "platform_user", row.ID, 0, nil, map[string]any{"role": role.RoleKey, "username": row.Username})
	})
}

func (r *UserRepository) UpdatePlatformAccount(ctx context.Context, userID int64, roleID uint64, status, reason string, actorID int64) (*model.PlatformAccount, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var user userRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", userID).First(&user).Error; err != nil {
			return err
		}
		var targetRole roleRow
		if err := tx.Where("id = ? AND scope_type = ?", roleID, model.RoleScopePlatform).First(&targetRole).Error; err != nil {
			return err
		}
		var hasAdmin int64
		if err := tx.Table("platform_user_roles assignment").Joins("JOIN roles role ON role.id = assignment.role_id").Where("assignment.user_id = ? AND assignment.revoked_at IS NULL AND role.role_key = ?", userID, model.RolePlatformAdmin).Count(&hasAdmin).Error; err != nil {
			return err
		}
		if hasAdmin > 0 && (targetRole.RoleKey != model.RolePlatformAdmin || status != model.UserStatusActive) {
			var activeAdmins int64
			if err := tx.Table("platform_user_roles assignment").Joins("JOIN roles role ON role.id = assignment.role_id").Joins("JOIN users user ON user.id = assignment.user_id AND user.status = ?", model.UserStatusActive).Where("assignment.revoked_at IS NULL AND role.role_key = ?", model.RolePlatformAdmin).Distinct("assignment.user_id").Count(&activeAdmins).Error; err != nil {
				return err
			}
			if activeAdmins <= 1 {
				return repository.ErrLastAdmin
			}
		}
		now := time.Now()
		if err := tx.Model(&platformUserRoleRow{}).Where("user_id = ? AND revoked_at IS NULL", userID).Update("revoked_at", now).Error; err != nil {
			return err
		}
		actor := uint64(actorID)
		if err := tx.Create(&platformUserRoleRow{UserID: uint64(userID), RoleID: roleID, AssignedBy: &actor, AssignedAt: now}).Error; err != nil {
			return err
		}
		if err := tx.Model(&userRow{}).Where("id = ?", userID).Updates(map[string]any{"status": status, "token_version": gorm.Expr("token_version + 1")}).Error; err != nil {
			return err
		}
		return createPlatformResourceAudit(tx, ctx, "platform_user.update", "platform_user", userID, 0, map[string]any{"status": user.Status}, map[string]any{"status": status, "role": targetRole.RoleKey, "reason": reason})
	})
	if err != nil {
		return nil, err
	}
	return r.getPlatformAccount(ctx, userID)
}

func (r *UserRepository) getPlatformAccount(ctx context.Context, userID int64) (*model.PlatformAccount, error) {
	var row platformAccountRow
	err := r.db.WithContext(ctx).Table("users user").
		Select("user.id, user.username, user.email, user.account_type, user.status, user.token_version, user.created_at, GROUP_CONCAT(DISTINCT role.role_key ORDER BY role.role_key) AS role_keys").
		Joins("JOIN platform_user_roles assignment ON assignment.user_id = user.id AND assignment.revoked_at IS NULL").
		Joins("JOIN roles role ON role.id = assignment.role_id AND role.scope_type = ?", model.RoleScopePlatform).
		Where("user.id = ?", userID).
		Group("user.id, user.username, user.email, user.account_type, user.status, user.token_version, user.created_at").Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	result := row.toDomain()
	return &result, nil
}

type platformAccountRow struct {
	ID           int64
	Username     string
	Email        string
	AccountType  string
	Status       string
	TokenVersion int32
	CreatedAt    time.Time
	RoleKeys     string
}

func (r platformAccountRow) toDomain() model.PlatformAccount {
	roles := []string{}
	if r.RoleKeys != "" {
		roles = strings.Split(r.RoleKeys, ",")
	}
	return model.PlatformAccount{User: model.User{ID: r.ID, Username: r.Username, Email: r.Email, AccountType: r.AccountType, Status: r.Status, TokenVersion: r.TokenVersion, CreatedAt: r.CreatedAt}, Roles: roles}
}

type platformUserRoleRow struct {
	ID         uint64 `gorm:"primaryKey"`
	UserID     uint64
	RoleID     uint64
	AssignedBy *uint64
	AssignedAt time.Time
	RevokedAt  *time.Time
}

func (platformUserRoleRow) TableName() string { return "platform_user_roles" }

type RefreshTokenRepository struct {
	db *gorm.DB
}

func NewRefreshTokenRepository(db *gorm.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

func TokenHash(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

func (r *RefreshTokenRepository) Create(ctx context.Context, userID int64, plainToken, familyID, clientApp string, tenantID, membershipID int64, expiresAt time.Time, ip, userAgent string) error {
	row := refreshTokenRow{
		UserID:           userID,
		TokenHash:        TokenHash(plainToken),
		FamilyID:         familyID,
		ClientApp:        clientApp,
		ExpiresAt:        expiresAt,
		CreatedIP:        &ip,
		CreatedUserAgent: &userAgent,
	}
	if tenantID > 0 {
		row.ActiveTenantID = &tenantID
	}
	if membershipID > 0 {
		row.MembershipID = &membershipID
	}
	return r.db.WithContext(ctx).Create(&row).Error
}

func (r *RefreshTokenRepository) GetActive(ctx context.Context, plainToken string) (*model.RefreshSession, error) {
	var token refreshTokenRow
	if err := r.db.WithContext(ctx).Where("token_hash = ?", TokenHash(plainToken)).First(&token).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repository.ErrTokenNotFound
		}
		return nil, err
	}
	if token.RevokedAt != nil {
		return nil, repository.ErrTokenReuseDetected
	}
	if time.Now().After(token.ExpiresAt) {
		return nil, repository.ErrTokenExpired
	}
	var user userRow
	if err := r.db.WithContext(ctx).Where("id = ?", token.UserID).First(&user).Error; err != nil {
		return nil, err
	}
	return refreshSessionFromRows(user, token), nil
}

func (r *RefreshTokenRepository) Rotate(ctx context.Context, plainToken, newPlainToken string, newExpiresAt time.Time, newIP, newUserAgent string) (*model.RefreshSession, error) {
	return r.rotate(ctx, plainToken, newPlainToken, 0, 0, false, newExpiresAt, newIP, newUserAgent)
}

func (r *RefreshTokenRepository) RotateToTenant(ctx context.Context, plainToken, newPlainToken string, tenantID, membershipID int64, newExpiresAt time.Time, newIP, newUserAgent string) (*model.RefreshSession, error) {
	return r.rotate(ctx, plainToken, newPlainToken, tenantID, membershipID, true, newExpiresAt, newIP, newUserAgent)
}

func (r *RefreshTokenRepository) rotate(ctx context.Context, plainToken, newPlainToken string, tenantID, membershipID int64, rebind bool, newExpiresAt time.Time, newIP, newUserAgent string) (*model.RefreshSession, error) {
	var result *model.RefreshSession
	reuseDetected := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var token refreshTokenRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("token_hash = ?", TokenHash(plainToken)).
			First(&token).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return repository.ErrTokenNotFound
			}
			return err
		}
		if token.RevokedAt != nil {
			now := time.Now()
			if err := tx.Model(&refreshTokenRow{}).Where("id = ?", token.ID).Update("reuse_detected_at", now).Error; err != nil {
				return err
			}
			if err := tx.Model(&refreshTokenRow{}).
				Where("family_id = ? AND revoked_at IS NULL", token.FamilyID).
				Update("revoked_at", now).Error; err != nil {
				return err
			}
			reuseDetected = true
			return nil
		}
		if time.Now().After(token.ExpiresAt) {
			return repository.ErrTokenExpired
		}
		var user userRow
		if err := tx.Where("id = ?", token.UserID).First(&user).Error; err != nil {
			return err
		}
		newHash := TokenHash(newPlainToken)
		activeTenantID := token.ActiveTenantID
		activeMembershipID := token.MembershipID
		if rebind {
			activeTenantID = &tenantID
			activeMembershipID = &membershipID
		}
		newToken := refreshTokenRow{
			UserID:           token.UserID,
			TokenHash:        newHash,
			FamilyID:         token.FamilyID,
			ClientApp:        token.ClientApp,
			ActiveTenantID:   activeTenantID,
			MembershipID:     activeMembershipID,
			ExpiresAt:        newExpiresAt,
			CreatedIP:        &newIP,
			CreatedUserAgent: &newUserAgent,
		}
		if err := tx.Create(&newToken).Error; err != nil {
			return err
		}
		now := time.Now()
		if err := tx.Model(&refreshTokenRow{}).
			Where("id = ?", token.ID).
			Updates(map[string]any{"revoked_at": now, "replaced_by_hash": newHash}).Error; err != nil {
			return err
		}
		newToken.ActiveTenantID = activeTenantID
		newToken.MembershipID = activeMembershipID
		result = refreshSessionFromRows(user, newToken)
		return nil
	})
	if err == nil && reuseDetected {
		return nil, repository.ErrTokenReuseDetected
	}
	return result, err
}

func refreshSessionFromRows(user userRow, token refreshTokenRow) *model.RefreshSession {
	result := &model.RefreshSession{
		UserID: user.ID, Username: user.Username, Role: user.Role, AccountType: user.AccountType,
		TokenVersion: user.TokenVersion, FamilyID: token.FamilyID, ClientApp: token.ClientApp,
	}
	if token.ActiveTenantID != nil {
		result.TenantID = *token.ActiveTenantID
	}
	if token.MembershipID != nil {
		result.MembershipID = *token.MembershipID
	}
	return result
}

func (r *RefreshTokenRepository) Revoke(ctx context.Context, plainToken string) error {
	return r.db.WithContext(ctx).Model(&refreshTokenRow{}).
		Where("token_hash = ?", TokenHash(plainToken)).
		Update("revoked_at", time.Now()).Error
}

type InviteCodeRepository struct {
	db *gorm.DB
}

func NewInviteCodeRepository(db *gorm.DB) *InviteCodeRepository {
	return &InviteCodeRepository{db: db}
}

func (r *InviteCodeRepository) GetByCode(ctx context.Context, code string) (*model.InviteCode, error) {
	var row inviteCodeRow
	err := r.db.WithContext(ctx).
		Where("code = ? AND is_active = 1 AND (expires_at IS NULL OR expires_at > ?)", code, time.Now()).
		First(&row).Error
	if err != nil {
		return nil, err
	}
	return row.toDomain(), nil
}

type AuthzRepository struct {
	db *gorm.DB
}

func NewAuthzRepository(db *gorm.DB) *AuthzRepository {
	return &AuthzRepository{db: db}
}

func (r *AuthzRepository) GetRoleByKey(ctx context.Context, roleKey string) (*model.Role, error) {
	var row roleRow
	err := r.db.WithContext(ctx).Where("role_key = ?", roleKey).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return row.toDomain(), nil
}

func (r *AuthzRepository) ListRoles(ctx context.Context) ([]model.Role, error) {
	var rows []roleRow
	if err := r.db.WithContext(ctx).Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	roles := make([]model.Role, len(rows))
	for i := range rows {
		roles[i] = *rows[i].toDomain()
	}
	return roles, nil
}

func (r *AuthzRepository) ListPermissions(ctx context.Context) ([]model.Permission, error) {
	var rows []permissionRow
	if err := r.db.WithContext(ctx).Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	permissions := make([]model.Permission, len(rows))
	for i := range rows {
		permissions[i] = *rows[i].toDomain()
	}
	return permissions, nil
}

func (r *AuthzRepository) GetUserRoles(ctx context.Context, userID uint64) ([]string, error) {
	var roleKeys []string
	err := r.db.WithContext(ctx).
		Table("user_roles").
		Select("r.role_key").
		Joins("JOIN roles r ON r.id = user_roles.role_id").
		Where("user_roles.user_id = ? AND user_roles.revoked_at IS NULL", userID).
		Pluck("r.role_key", &roleKeys).Error
	return roleKeys, err
}

func (r *AuthzRepository) GetUserPermissions(ctx context.Context, userID uint64) ([]string, error) {
	var permissionKeys []string
	err := r.db.WithContext(ctx).
		Table("user_roles").
		Select("DISTINCT p.permission_key").
		Joins("JOIN role_permissions rp ON rp.role_id = user_roles.role_id").
		Joins("JOIN permissions p ON p.id = rp.permission_id").
		Where("user_roles.user_id = ? AND user_roles.revoked_at IS NULL", userID).
		Pluck("p.permission_key", &permissionKeys).Error
	return permissionKeys, err
}

func (r *AuthzRepository) GetUserDataScopes(ctx context.Context, userID uint64) ([]model.DataScope, error) {
	var rows []dataScopeRow
	if err := r.db.WithContext(ctx).Where("user_id = ? AND revoked_at IS NULL", userID).Find(&rows).Error; err != nil {
		return nil, err
	}
	scopes := make([]model.DataScope, len(rows))
	for i := range rows {
		scopes[i] = *rows[i].toDomain()
	}
	return scopes, nil
}

func (r *AuthzRepository) LoadPrincipal(ctx context.Context, userID uint64) (*model.Principal, error) {
	var user userRow
	if err := r.db.WithContext(ctx).Where("id = ?", userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user %d not found", userID)
		}
		return nil, err
	}
	roles, err := r.GetUserRoles(ctx, userID)
	if err != nil {
		return nil, err
	}
	permissions, err := r.GetUserPermissions(ctx, userID)
	if err != nil {
		return nil, err
	}
	scopes, err := r.GetUserDataScopes(ctx, userID)
	if err != nil {
		return nil, err
	}
	assignments := make([]model.ScopeAssignment, len(scopes))
	for i, scope := range scopes {
		assignments[i] = model.ScopeAssignment{
			ScopeKey:     scope.ScopeKey,
			ResourceType: scope.ResourceType,
			ResourceID:   int64(scope.ResourceID),
		}
	}
	return &model.Principal{
		UserID:       user.ID,
		Username:     user.Username,
		AccountType:  user.AccountType,
		Roles:        roles,
		Permissions:  permissions,
		DataScopes:   assignments,
		TokenVersion: user.TokenVersion,
		Email:        user.Email,
		LegacyRole:   user.Role,
	}, nil
}

func (r *AuthzRepository) LoadPlatformPrincipal(ctx context.Context, userID uint64) (*model.Principal, error) {
	var user userRow
	if err := r.db.WithContext(ctx).Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, err
	}
	var roles []string
	if err := r.db.WithContext(ctx).Table("platform_user_roles pur").
		Select("r.role_key").Joins("JOIN roles r ON r.id = pur.role_id").
		Where("pur.user_id = ? AND pur.revoked_at IS NULL AND r.scope_type = ?", userID, model.RoleScopePlatform).
		Pluck("r.role_key", &roles).Error; err != nil {
		return nil, err
	}
	if len(roles) == 0 {
		return nil, fmt.Errorf("user %d has no platform application admission", userID)
	}
	var permissions []string
	if err := r.db.WithContext(ctx).Table("platform_user_roles pur").Distinct("p.permission_key").
		Joins("JOIN roles r ON r.id = pur.role_id").Joins("JOIN role_permissions rp ON rp.role_id = r.id").
		Joins("JOIN permissions p ON p.id = rp.permission_id").
		Where("pur.user_id = ? AND pur.revoked_at IS NULL AND r.scope_type = ?", userID, model.RoleScopePlatform).
		Pluck("p.permission_key", &permissions).Error; err != nil {
		return nil, err
	}
	return &model.Principal{
		UserID: user.ID, Username: user.Username, AccountType: model.AccountTypePlatform,
		Roles: roles, Permissions: permissions, TokenVersion: user.TokenVersion,
		Email: user.Email, LegacyRole: user.Role, ClientApp: "platform", AvailableApps: []string{"platform"},
	}, nil
}

func (r *AuthzRepository) AssignRole(ctx context.Context, userID, roleID uint64, assignedBy *uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&userRoleRow{}).
			Where("user_id = ? AND role_id = ? AND revoked_at IS NULL", userID, roleID).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("user %d already has role %d", userID, roleID)
		}
		return tx.Create(&userRoleRow{UserID: userID, RoleID: roleID, AssignedBy: assignedBy, AssignedAt: time.Now()}).Error
	})
}

func (r *AuthzRepository) RevokeRoleWithLastAdminGuard(ctx context.Context, userID, roleID uint64, roleKey string, revokedBy *uint64) (bool, error) {
	now := time.Now()
	if roleKey != model.RoleSystemAdmin {
		result := r.db.WithContext(ctx).Model(&userRoleRow{}).
			Where("user_id = ? AND role_id = ? AND revoked_at IS NULL", userID, roleID).
			Update("revoked_at", now)
		if result.Error != nil {
			return false, result.Error
		}
		if result.RowsAffected == 0 {
			return false, repository.ErrUserRoleNotFound
		}
		return true, nil
	}
	sql := `UPDATE user_roles
		SET revoked_at = ?
		WHERE user_id = ?
		  AND role_id = ?
		  AND revoked_at IS NULL
		  AND (
		    SELECT cnt FROM (
		      SELECT COUNT(*) AS cnt
		      FROM user_roles ur2
		      JOIN roles r2 ON r2.id = ur2.role_id
		      WHERE r2.role_key = ? AND ur2.revoked_at IS NULL
		    ) sub
		  ) > 1`
	result := r.db.WithContext(ctx).Exec(sql, now, userID, roleID, model.RoleSystemAdmin)
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected > 0 {
		return true, nil
	}
	count, err := r.CountActiveUsersWithRole(ctx, model.RoleSystemAdmin)
	if err != nil {
		return false, err
	}
	if count <= 1 {
		var hold int64
		if err := r.db.WithContext(ctx).Model(&userRoleRow{}).
			Where("user_id = ? AND role_id = ? AND revoked_at IS NULL", userID, roleID).
			Count(&hold).Error; err == nil && hold > 0 {
			return false, repository.ErrLastAdmin
		}
	}
	return false, repository.ErrUserRoleNotFound
}

func (r *AuthzRepository) RestoreRole(ctx context.Context, userID, roleID uint64) error {
	var row userRoleRow
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND role_id = ? AND revoked_at IS NOT NULL", userID, roleID).
		Order("revoked_at DESC").
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return repository.ErrUserRoleNotFound
		}
		return err
	}
	return r.db.WithContext(ctx).Model(&userRoleRow{}).Where("id = ?", row.ID).Update("revoked_at", nil).Error
}

func (r *AuthzRepository) CountActiveUsersWithRole(ctx context.Context, roleKey string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("user_roles").
		Joins("JOIN roles r ON r.id = user_roles.role_id").
		Where("r.role_key = ? AND user_roles.revoked_at IS NULL", roleKey).
		Count(&count).Error
	return count, err
}

func (r *AuthzRepository) AssignDataScope(ctx context.Context, userID uint64, scopeKey, resourceType string, resourceID uint64, assignedBy *uint64) error {
	return r.db.WithContext(ctx).Create(&dataScopeRow{
		UserID:       userID,
		ScopeKey:     scopeKey,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		AssignedBy:   assignedBy,
		AssignedAt:   time.Now(),
	}).Error
}

func (r *AuthzRepository) RevokeDataScope(ctx context.Context, scopeID uint64) error {
	result := r.db.WithContext(ctx).Model(&dataScopeRow{}).
		Where("id = ? AND revoked_at IS NULL", scopeID).
		Update("revoked_at", time.Now())
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("data scope %d not found or already revoked", scopeID)
	}
	return nil
}

func (r *AuthzRepository) GetScopeOwnerID(ctx context.Context, scopeID uint64) (uint64, error) {
	var row dataScopeRow
	if err := r.db.WithContext(ctx).Where("id = ?", scopeID).First(&row).Error; err != nil {
		return 0, err
	}
	return row.UserID, nil
}

func (r *AuthzRepository) IncrementTokenVersion(ctx context.Context, userID uint64) (int32, error) {
	var newVersion int32
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&userRow{}).Where("id = ?", userID).
			Update("token_version", gorm.Expr("token_version + 1")).Error; err != nil {
			return err
		}
		var user userRow
		if err := tx.Select("token_version").First(&user, userID).Error; err != nil {
			return err
		}
		newVersion = user.TokenVersion
		return nil
	})
	return newVersion, err
}

type AuditRepository struct {
	db *gorm.DB
}

func NewAuditRepository(db *gorm.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

func (r *AuditRepository) RecordAuthDecision(ctx context.Context, decision model.AuthAuditDecision) error {
	row := authAuditRow{
		ActorUserID:   decision.ActorUserID,
		ActorRoles:    decision.ActorRoles,
		PermissionKey: decision.PermissionKey,
		ResourceType:  decision.ResourceType,
		ResourceID:    decision.ResourceID,
		Decision:      decision.Decision,
		Reason:        decision.Reason,
		RequestID:     decision.RequestID,
		ClientIP:      decision.ClientIP,
	}
	if decision.TenantID > 0 {
		row.TenantID = &decision.TenantID
	}
	if decision.MembershipID > 0 {
		row.MembershipID = &decision.MembershipID
	}
	return r.db.WithContext(ctx).Create(&row).Error
}

func (r *AuditRepository) QueryAuthAuditLogs(ctx context.Context, actorUserID *uint64, permissionKey, decision string, offset, limit int) ([]model.AuthAuditLog, int64, error) {
	query := r.db.WithContext(ctx).Model(&authAuditRow{})
	if actorUserID != nil {
		query = query.Where("actor_user_id = ?", *actorUserID)
	}
	if permissionKey != "" {
		query = query.Where("permission_key = ?", permissionKey)
	}
	if decision != "" {
		query = query.Where("decision = ?", decision)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []authAuditRow
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	logs := make([]model.AuthAuditLog, len(rows))
	for i := range rows {
		logs[i] = *rows[i].toDomain()
	}
	return logs, total, nil
}

type userRow struct {
	ID           int64 `gorm:"primaryKey"`
	Username     string
	Password     string
	Role         int32
	Email        string
	AccountType  string
	Status       string
	TokenVersion int32
	CreatedAt    time.Time
}

func (userRow) TableName() string { return "users" }

func (r userRow) toDomain() *model.User {
	return &model.User{
		ID:           r.ID,
		Username:     r.Username,
		PasswordHash: r.Password,
		Role:         r.Role,
		Email:        r.Email,
		AccountType:  r.AccountType,
		Status:       r.Status,
		TokenVersion: r.TokenVersion,
		CreatedAt:    r.CreatedAt,
	}
}

type refreshTokenRow struct {
	ID               int64 `gorm:"primaryKey"`
	UserID           int64
	TokenHash        string
	FamilyID         string
	ClientApp        string
	ActiveTenantID   *int64
	MembershipID     *int64
	ExpiresAt        time.Time
	CreatedIP        *string
	CreatedUserAgent *string
	RevokedAt        *time.Time
	ReplacedByHash   string
	ReuseDetectedAt  *time.Time
	CreatedAt        time.Time
}

func (refreshTokenRow) TableName() string { return "refresh_tokens" }

type inviteCodeRow struct {
	ID        int64 `gorm:"primaryKey"`
	TenantID  int64
	Code      string
	CreatedBy int64
	IsActive  int32
	ExpiresAt *time.Time
	CreatedAt time.Time
}

func (inviteCodeRow) TableName() string { return "invite_codes" }

func (r inviteCodeRow) toDomain() *model.InviteCode {
	return &model.InviteCode{ID: r.ID, TenantID: r.TenantID, Code: r.Code, CreatedBy: r.CreatedBy, IsActive: r.IsActive == 1, ExpiresAt: r.ExpiresAt}
}

type roleRow struct {
	ID          uint64 `gorm:"primaryKey"`
	RoleKey     string
	Name        string
	Description string
	ScopeType   string
	IsSystem    int32
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (roleRow) TableName() string { return "roles" }

func (r roleRow) toDomain() *model.Role {
	return &model.Role{ID: r.ID, RoleKey: r.RoleKey, Name: r.Name, Description: r.Description, ScopeType: r.ScopeType, IsSystem: r.IsSystem == 1, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

type permissionRow struct {
	ID            uint64 `gorm:"primaryKey"`
	PermissionKey string
	Resource      string
	Action        string
	Description   string
}

func (permissionRow) TableName() string { return "permissions" }

func (r permissionRow) toDomain() *model.Permission {
	return &model.Permission{ID: r.ID, PermissionKey: r.PermissionKey, Resource: r.Resource, Action: r.Action, Description: r.Description}
}

type userRoleRow struct {
	ID         uint64 `gorm:"primaryKey"`
	UserID     uint64
	RoleID     uint64
	AssignedBy *uint64
	AssignedAt time.Time
	RevokedAt  *time.Time
}

func (userRoleRow) TableName() string { return "user_roles" }

type dataScopeRow struct {
	ID           uint64 `gorm:"primaryKey"`
	UserID       uint64
	ScopeKey     string
	ResourceType string
	ResourceID   uint64
	AssignedBy   *uint64
	AssignedAt   time.Time
	RevokedAt    *time.Time
}

func (dataScopeRow) TableName() string { return "user_data_scopes" }

func (r dataScopeRow) toDomain() *model.DataScope {
	return &model.DataScope{
		ID:           r.ID,
		UserID:       r.UserID,
		ScopeKey:     r.ScopeKey,
		ResourceType: r.ResourceType,
		ResourceID:   r.ResourceID,
		AssignedBy:   r.AssignedBy,
		AssignedAt:   r.AssignedAt,
		RevokedAt:    r.RevokedAt,
	}
}

type authAuditRow struct {
	ID            uint64 `gorm:"primaryKey"`
	TenantID      *int64
	MembershipID  *int64
	ActorUserID   uint64
	ActorRoles    string
	PermissionKey string
	ResourceType  string
	ResourceID    uint64
	Decision      string
	Reason        string
	RequestID     string
	ClientIP      string
	CreatedAt     time.Time
}

func (authAuditRow) TableName() string { return "authorization_audit_logs" }

func (r authAuditRow) toDomain() *model.AuthAuditLog {
	result := &model.AuthAuditLog{
		ID:            r.ID,
		ActorUserID:   r.ActorUserID,
		ActorRoles:    r.ActorRoles,
		PermissionKey: r.PermissionKey,
		ResourceType:  r.ResourceType,
		ResourceID:    r.ResourceID,
		Decision:      r.Decision,
		Reason:        r.Reason,
		RequestID:     r.RequestID,
		ClientIP:      r.ClientIP,
		CreatedAt:     r.CreatedAt,
	}
	if r.TenantID != nil {
		result.TenantID = *r.TenantID
	}
	if r.MembershipID != nil {
		result.MembershipID = *r.MembershipID
	}
	return result
}
