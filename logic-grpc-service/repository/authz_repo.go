package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"logic-grpc-service/model"
	"logic-grpc-service/pkg/authz"
	"logic-grpc-service/pkg/logger"
)

// ── Sentinel errors ────────────────────────────────────────────────────

var (
	ErrRoleNotFound        = errors.New("role not found")
	ErrPermissionNotFound  = errors.New("permission not found")
	ErrUserRoleNotFound    = errors.New("user role assignment not found")
	ErrLastAdmin           = errors.New("cannot remove the last active admin")
)

// interviewSchedulesMissingOnce ensures we only warn once when the
// interview_schedules table is absent, so scope evaluations don't spam logs.
var interviewSchedulesMissingOnce sync.Once

func warnInterviewSchedulesMissing() {
	interviewSchedulesMissingOnce.Do(func() {
		logger.L().Warn("interview_schedules table missing — interviewer scope (assigned_interviews) will never match. Run migration 000016_add_interview_schedules.sql or re-import db.sql.")
	})
}

// ── AuthzRepo ──────────────────────────────────────────────────────────

// AuthzRepo manages roles, permissions, user-role assignments, data scopes, and auth audit logs.
type AuthzRepo struct {
	db *gorm.DB
}

func NewAuthzRepo(db *gorm.DB) *AuthzRepo {
	return &AuthzRepo{db: db}
}

// ── Roles ──────────────────────────────────────────────────────────────

func (r *AuthzRepo) GetRoleByKey(ctx context.Context, roleKey string) (*model.Role, error) {
	var role model.Role
	err := r.db.WithContext(ctx).Where("role_key = ?", roleKey).First(&role).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrRoleNotFound
	}
	return &role, err
}

func (r *AuthzRepo) ListRoles(ctx context.Context) ([]model.Role, error) {
	var roles []model.Role
	err := r.db.WithContext(ctx).Order("id ASC").Find(&roles).Error
	return roles, err
}

// ── Permissions ────────────────────────────────────────────────────────

func (r *AuthzRepo) GetPermissionByKey(ctx context.Context, permKey string) (*model.Permission, error) {
	var perm model.Permission
	err := r.db.WithContext(ctx).Where("permission_key = ?", permKey).First(&perm).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrPermissionNotFound
	}
	return &perm, err
}

func (r *AuthzRepo) ListPermissions(ctx context.Context) ([]model.Permission, error) {
	var perms []model.Permission
	err := r.db.WithContext(ctx).Order("id ASC").Find(&perms).Error
	return perms, err
}

// GetPermissionsByRole loads all permission keys for a given role key.
func (r *AuthzRepo) GetPermissionsByRole(ctx context.Context, roleKey string) ([]string, error) {
	var permKeys []string
	err := r.db.WithContext(ctx).
		Table("role_permissions").
		Select("p.permission_key").
		Joins("JOIN permissions p ON p.id = role_permissions.permission_id").
		Joins("JOIN roles r ON r.id = role_permissions.role_id").
		Where("r.role_key = ?", roleKey).
		Pluck("p.permission_key", &permKeys).Error
	return permKeys, err
}

// GetRolePermissionsByRoleID loads all permission keys for a given role ID.
func (r *AuthzRepo) GetRolePermissionsByRoleID(ctx context.Context, roleID uint64) ([]string, error) {
	var permKeys []string
	err := r.db.WithContext(ctx).
		Table("role_permissions").
		Select("p.permission_key").
		Joins("JOIN permissions p ON p.id = role_permissions.permission_id").
		Where("role_permissions.role_id = ?", roleID).
		Pluck("p.permission_key", &permKeys).Error
	return permKeys, err
}

// ── User-Role assignments ──────────────────────────────────────────────

// AssignRole grants a role to a user. Uses a transaction to prevent TOCTOU race
// between the duplicate check and the insert. Returns an error if the assignment
// already exists and is active.
func (r *AuthzRepo) AssignRole(ctx context.Context, userID, roleID uint64, assignedBy *uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&model.UserRole{}).
			Where("user_id = ? AND role_id = ? AND revoked_at IS NULL", userID, roleID).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("user %d already has role %d", userID, roleID)
		}

		ur := &model.UserRole{
			UserID:     userID,
			RoleID:     roleID,
			AssignedBy: assignedBy,
			AssignedAt: time.Now(),
		}
		return tx.Create(ur).Error
	})
}

// RevokeRole soft-revokes an active user-role assignment.
func (r *AuthzRepo) RevokeRole(ctx context.Context, userID, roleID uint64, revokedBy *uint64) error {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&model.UserRole{}).
		Where("user_id = ? AND role_id = ? AND revoked_at IS NULL", userID, roleID).
		Updates(map[string]any{"revoked_at": now})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserRoleNotFound
	}
	return nil
}

// RestoreRole undoes a soft-revoke: clears revoked_at on the most-recently
// revoked record. Used to compensate when post-DB side-effects (e.g. Redis
// token-version sync) fail and we need to roll back without an explicit txn.
// Returns ErrUserRoleNotFound if no recently-revoked record matches.
func (r *AuthzRepo) RestoreRole(ctx context.Context, userID, roleID uint64) error {
	var ur model.UserRole
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND role_id = ? AND revoked_at IS NOT NULL", userID, roleID).
		Order("revoked_at DESC").
		First(&ur).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserRoleNotFound
		}
		return err
	}
	return r.db.WithContext(ctx).Model(&model.UserRole{}).
		Where("id = ?", ur.ID).
		Update("revoked_at", nil).Error
}

// RevokeRoleWithLastAdminGuard atomically soft-revokes a user-role assignment
// while preventing removal of the last active system_admin. The check and update
// happen inside one SQL statement, so it is safe against concurrent revoke races.
//
// Returns:
//   - (true, nil)   — revoke succeeded
//   - (false, ErrLastAdmin)  — would have removed the last system_admin; nothing changed
//   - (false, ErrUserRoleNotFound) — the user does not hold this role
//   - (false, other err) — DB error
//
// roleKey must match roleID; pass it explicitly so the SQL can branch without a join.
func (r *AuthzRepo) RevokeRoleWithLastAdminGuard(ctx context.Context, userID, roleID uint64, roleKey string, revokedBy *uint64) (bool, error) {
	now := time.Now()

	// For non-system_admin roles, fall back to a plain revoke.
	if roleKey != authz.RoleSystemAdmin {
		err := r.RevokeRole(ctx, userID, roleID, revokedBy)
		if err != nil {
			return false, err
		}
		return true, nil
	}

	// For system_admin, do the count + revoke in one SQL with a sub-select guard.
	// The sub-select counts active system_admin assignments and we only revoke
	// when count > 1.
	//
	// We wrap the COUNT(*) in an extra SELECT layer so MySQL accepts it: MySQL
	// disallows referencing the UPDATE target table in a direct subquery, but
	// allows it through a derived table (the inner SELECT is materialized first).
	// SQLite tolerates either form. No table alias on the outer UPDATE for
	// SQLite compatibility.
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
	res := r.db.WithContext(ctx).Exec(sql, now, userID, roleID, authz.RoleSystemAdmin)
	if res.Error != nil {
		return false, res.Error
	}
	if res.RowsAffected > 0 {
		return true, nil
	}

	// 0 rows affected — distinguish "user doesn't hold this role" from
	// "would have removed the last system_admin".
	count, err := r.CountActiveUsersWithRole(ctx, authz.RoleSystemAdmin)
	if err != nil {
		return false, err
	}
	if count <= 1 {
		// Confirm the user actually still holds the role; if not, prefer the not-found error.
		var hold int64
		if cerr := r.db.WithContext(ctx).Model(&model.UserRole{}).
			Where("user_id = ? AND role_id = ? AND revoked_at IS NULL", userID, roleID).
			Count(&hold).Error; cerr == nil && hold > 0 {
			return false, ErrLastAdmin
		}
	}
	return false, ErrUserRoleNotFound
}

// GetUserRoles returns all active (non-revoked) role keys for a user.
func (r *AuthzRepo) GetUserRoles(ctx context.Context, userID uint64) ([]string, error) {
	var roleKeys []string
	err := r.db.WithContext(ctx).
		Table("user_roles").
		Select("r.role_key").
		Joins("JOIN roles r ON r.id = user_roles.role_id").
		Where("user_roles.user_id = ? AND user_roles.revoked_at IS NULL", userID).
		Pluck("r.role_key", &roleKeys).Error
	return roleKeys, err
}

// GetUserPermissions collects all permissions for a user's active roles.
func (r *AuthzRepo) GetUserPermissions(ctx context.Context, userID uint64) ([]string, error) {
	var permKeys []string
	err := r.db.WithContext(ctx).
		Table("user_roles").
		Select("DISTINCT p.permission_key").
		Joins("JOIN role_permissions rp ON rp.role_id = user_roles.role_id").
		Joins("JOIN permissions p ON p.id = rp.permission_id").
		Where("user_roles.user_id = ? AND user_roles.revoked_at IS NULL", userID).
		Pluck("p.permission_key", &permKeys).Error
	return permKeys, err
}

// GetUserRoleIDs returns active role IDs for a user (for use with scope queries).
func (r *AuthzRepo) GetUserRoleIDs(ctx context.Context, userID uint64) ([]uint64, error) {
	var roleIDs []uint64
	err := r.db.WithContext(ctx).
		Table("user_roles").
		Select("role_id").
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Pluck("role_id", &roleIDs).Error
	return roleIDs, err
}

// HasActiveAdminRole checks if the user has any active admin role assignment.
func (r *AuthzRepo) HasActiveAdminRole(ctx context.Context, userID uint64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("user_roles").
		Joins("JOIN roles r ON r.id = user_roles.role_id").
		Where("user_roles.user_id = ? AND r.role_key IN ? AND user_roles.revoked_at IS NULL",
			userID, authz.AdminRoles()).
		Count(&count).Error
	return count > 0, err
}

// CountActiveAdmins counts users with non-revoked admin role assignments.
func (r *AuthzRepo) CountActiveAdmins(ctx context.Context, adminRoleKeys []string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("user_roles").
		Joins("JOIN roles r ON r.id = user_roles.role_id").
		Where("r.role_key IN ? AND user_roles.revoked_at IS NULL", adminRoleKeys).
		Count(&count).Error
	return count, err
}

// CountActiveUsersWithRole returns the number of users who currently hold a given role
// (revoked_at IS NULL). Used for last-admin safety checks.
func (r *AuthzRepo) CountActiveUsersWithRole(ctx context.Context, roleKey string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("user_roles").
		Joins("JOIN roles r ON r.id = user_roles.role_id").
		Where("r.role_key = ? AND user_roles.revoked_at IS NULL", roleKey).
		Count(&count).Error
	return count, err
}

// ── Data scopes ────────────────────────────────────────────────────────

// AssignDataScope grants a data scope to a user.
func (r *AuthzRepo) AssignDataScope(ctx context.Context, userID uint64, scopeKey, resourceType string, resourceID uint64, assignedBy *uint64) error {
	ds := &model.UserDataScope{
		UserID:       userID,
		ScopeKey:     scopeKey,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		AssignedBy:   assignedBy,
		AssignedAt:   time.Now(),
	}
	return r.db.WithContext(ctx).Create(ds).Error
}

// RevokeDataScope soft-revokes an active data scope assignment.
func (r *AuthzRepo) RevokeDataScope(ctx context.Context, scopeID uint64) error {
	result := r.db.WithContext(ctx).Model(&model.UserDataScope{}).
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

// GetUserDataScopes returns all active (non-revoked) data scopes for a user.
func (r *AuthzRepo) GetUserDataScopes(ctx context.Context, userID uint64) ([]model.UserDataScope, error) {
	var scopes []model.UserDataScope
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Find(&scopes).Error
	return scopes, err
}

// GetScopeOwnerID returns the user_id for a given data scope assignment.
func (r *AuthzRepo) GetScopeOwnerID(ctx context.Context, scopeID uint64) (uint64, error) {
	var scope model.UserDataScope
	err := r.db.WithContext(ctx).Where("id = ?", scopeID).First(&scope).Error
	if err != nil {
		return 0, err
	}
	return scope.UserID, nil
}

// GetUserScopeKeys returns distinct scope keys for a user.
func (r *AuthzRepo) GetUserScopeKeys(ctx context.Context, userID uint64) ([]string, error) {
	var keys []string
	err := r.db.WithContext(ctx).
		Model(&model.UserDataScope{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Distinct("scope_key").
		Pluck("scope_key", &keys).Error
	return keys, err
}

// ── Authorization audit logs ───────────────────────────────────────────

// RecordAuthDecision writes an authorization audit log entry.
func (r *AuthzRepo) RecordAuthDecision(ctx context.Context, actorUserID uint64, actorRoles, permissionKey, resourceType string, resourceID uint64, decision, reason, requestID, clientIP string) error {
	log := &model.AuthorizationAuditLog{
		ActorUserID:   actorUserID,
		ActorRoles:    actorRoles,
		PermissionKey: permissionKey,
		ResourceType:  resourceType,
		ResourceID:    resourceID,
		Decision:      decision,
		Reason:        reason,
		RequestID:     requestID,
		ClientIP:      clientIP,
	}
	return r.db.WithContext(ctx).Create(log).Error
}

// QueryAuthAuditLogs returns paginated authorization audit logs.
func (r *AuthzRepo) QueryAuthAuditLogs(ctx context.Context, actorUserID *uint64, permissionKey, decision string, offset, limit int) ([]model.AuthorizationAuditLog, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.AuthorizationAuditLog{})
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

	var logs []model.AuthorizationAuditLog
	err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&logs).Error
	return logs, total, err
}

// ── Migration helpers ──────────────────────────────────────────────────

// MigrateLegacyUserRoles maps a user's legacy numeric role to the new RBAC assignments.
// role=1 -> candidate + account_type=candidate
// role=2 -> recruiter + scope=own_jobs + account_type=staff
// role=3 -> recruiting_admin + recruiter + account_type=staff
func (r *AuthzRepo) MigrateLegacyUserRoles(ctx context.Context, userID uint64, legacyRole int32) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Look up role IDs
		roleIDs := make(map[string]uint64)
		var roles []model.Role
		if err := tx.Where("role_key IN ?", []string{authz.RoleCandidate, authz.RoleRecruiter, authz.RoleRecruitingAdmin}).Find(&roles).Error; err != nil {
			return err
		}
		for _, role := range roles {
			roleIDs[role.RoleKey] = role.ID
		}

		switch legacyRole {
		case authz.LegacyRoleCandidate:
			if roleID, ok := roleIDs[authz.RoleCandidate]; ok {
				if err := tx.Create(&model.UserRole{UserID: userID, RoleID: roleID, AssignedAt: time.Now()}).Error; err != nil {
					return err
				}
			}
			if err := tx.Model(&model.User{}).Where("id = ?", userID).Update("account_type", "candidate").Error; err != nil {
				return err
			}

		case authz.LegacyRoleHR:
			// Grant recruiter role
			if roleID, ok := roleIDs[authz.RoleRecruiter]; ok {
				if err := tx.Create(&model.UserRole{UserID: userID, RoleID: roleID, AssignedAt: time.Now()}).Error; err != nil {
					return err
				}
			}
			// Default scope: own_jobs
			if err := tx.Create(&model.UserDataScope{UserID: userID, ScopeKey: authz.ScopeOwnJobs, AssignedAt: time.Now()}).Error; err != nil {
				return err
			}
			if err := tx.Model(&model.User{}).Where("id = ?", userID).Update("account_type", "staff").Error; err != nil {
				return err
			}

		case authz.LegacyRoleAdmin:
			// Grant recruiting_admin + recruiter explicitly
			for _, key := range []string{authz.RoleRecruitingAdmin, authz.RoleRecruiter} {
				if roleID, ok := roleIDs[key]; ok {
					if err := tx.Create(&model.UserRole{UserID: userID, RoleID: roleID, AssignedAt: time.Now()}).Error; err != nil {
						return err
					}
				}
			}
			// Default scope: recruiting_all
			if err := tx.Create(&model.UserDataScope{UserID: userID, ScopeKey: authz.ScopeRecruitingAll, AssignedAt: time.Now()}).Error; err != nil {
				return err
			}
			if err := tx.Model(&model.User{}).Where("id = ?", userID).Update("account_type", "staff").Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// MigrateAllLegacyUsers migrates all users who have not yet been migrated
// (no entry in user_roles) from their legacy users.role to RBAC assignments.
func (r *AuthzRepo) MigrateAllLegacyUsers(ctx context.Context) (totalMigrated int64, err error) {
	type legacyUser struct {
		ID   uint64
		Role int32
	}
	var users []legacyUser
	if err := r.db.WithContext(ctx).
		Table("users").
		Select("id, role").
		Where("id NOT IN (SELECT DISTINCT user_id FROM user_roles)").
		Find(&users).Error; err != nil {
		return 0, err
	}

	for _, u := range users {
		if err := r.MigrateLegacyUserRoles(ctx, u.ID, u.Role); err != nil {
			return totalMigrated, fmt.Errorf("migrate user %d (legacy role %d): %w", u.ID, u.Role, err)
		}
		totalMigrated++
	}

	return totalMigrated, nil
}

// ── Scope evaluation helpers ───────────────────────────────────────────

// ScopeJobIDs returns the list of job IDs a user can access based on their scope.
// This is a convenience wrapper that queries the relevant tables.
func (r *AuthzRepo) ScopeJobIDs(ctx context.Context, userID uint64, scopeKeys []string, departmentIDs, locationIDs []uint64) ([]uint64, error) {
	jobIDSet := make(map[uint64]bool)

	for _, scope := range scopeKeys {
		switch scope {
		case authz.ScopeOwnJobs:
			var ids []uint64
			if err := r.db.WithContext(ctx).Table("jobs").
				Where("hr_id = ?", userID).
				Pluck("id", &ids).Error; err != nil {
				return nil, err
			}
			for _, id := range ids {
				jobIDSet[id] = true
			}

		case authz.ScopeDepartment:
			if len(departmentIDs) > 0 {
				var ids []uint64
				if err := r.db.WithContext(ctx).Table("jobs").
					Where("department_id IN ?", departmentIDs).
					Pluck("id", &ids).Error; err != nil {
					return nil, err
				}
				for _, id := range ids {
					jobIDSet[id] = true
				}
			}

		case authz.ScopeLocation:
			if len(locationIDs) > 0 {
				var ids []uint64
				if err := r.db.WithContext(ctx).Table("jobs").
					Where("location_id IN ?", locationIDs).
					Pluck("id", &ids).Error; err != nil {
					return nil, err
				}
				for _, id := range ids {
					jobIDSet[id] = true
				}
			}

		case authz.ScopeAssignedInterviews:
			// Resolve job IDs from applications where the user is assigned as interviewer.
			// If interview_schedules table is missing, warn once and skip.
			var ids []uint64
			if r.db.Migrator().HasTable("interview_schedules") {
				if err := r.db.WithContext(ctx).Table("interview_schedules").
					Select("DISTINCT a.job_id").
					Joins("JOIN applications a ON a.id = interview_schedules.application_id").
					Where("interview_schedules.interviewer_id = ? AND interview_schedules.deleted_at IS NULL", userID).
					Pluck("a.job_id", &ids).Error; err != nil {
					return nil, err
				}
			} else {
				warnInterviewSchedulesMissing()
			}
			for _, id := range ids {
				jobIDSet[id] = true
			}

		case authz.ScopeRecruitingAll, authz.ScopeSystemAll:
			// All access — return nil to signal no filtering needed
			return nil, nil
		}
	}

	result := make([]uint64, 0, len(jobIDSet))
	for id := range jobIDSet {
		result = append(result, id)
	}
	return result, nil
}

// IsInterviewerForJob checks whether the user is assigned as an interviewer for any
// application under the given job via the interview_schedules table.
// Returns false if the table doesn't exist (with a one-time warning) or the user
// has no active assignments for this job.
func (r *AuthzRepo) IsInterviewerForJob(ctx context.Context, userID, jobID uint64) (bool, error) {
	if !r.db.Migrator().HasTable("interview_schedules") {
		warnInterviewSchedulesMissing()
		return false, nil
	}
	var count int64
	err := r.db.WithContext(ctx).Table("interview_schedules").
		Joins("JOIN applications a ON a.id = interview_schedules.application_id").
		Where("interview_schedules.interviewer_id = ? AND a.job_id = ? AND interview_schedules.deleted_at IS NULL", userID, jobID).
		Count(&count).Error
	return count > 0, err
}

// GetUserDepartmentIDs returns department IDs from a user's department data scopes.
func (r *AuthzRepo) GetUserDepartmentIDs(ctx context.Context, userID uint64) ([]uint64, error) {
	var ids []uint64
	err := r.db.WithContext(ctx).
		Model(&model.UserDataScope{}).
		Where("user_id = ? AND scope_key = ? AND revoked_at IS NULL AND resource_type = ? AND resource_id > 0",
			userID, authz.ScopeDepartment, "department").
		Distinct("resource_id").
		Pluck("resource_id", &ids).Error
	return ids, err
}

// GetUserLocationIDs returns location IDs from a user's location data scopes.
func (r *AuthzRepo) GetUserLocationIDs(ctx context.Context, userID uint64) ([]uint64, error) {
	var ids []uint64
	err := r.db.WithContext(ctx).
		Model(&model.UserDataScope{}).
		Where("user_id = ? AND scope_key = ? AND revoked_at IS NULL AND resource_type = ? AND resource_id > 0",
			userID, authz.ScopeLocation, "location").
		Distinct("resource_id").
		Pluck("resource_id", &ids).Error
	return ids, err
}

// ── Principal loading ──────────────────────────────────────────────────

// LoadPrincipal builds a full Principal for a user from the database.
func (r *AuthzRepo) LoadPrincipal(ctx context.Context, userID uint64) (*authz.Principal, error) {
	var user model.User
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

	perms, err := r.GetUserPermissions(ctx, userID)
	if err != nil {
		return nil, err
	}

	scopes, err := r.GetUserDataScopes(ctx, userID)
	if err != nil {
		return nil, err
	}

	scopeAssignments := make([]authz.ScopeAssignment, 0, len(scopes))
	for _, s := range scopes {
		scopeAssignments = append(scopeAssignments, authz.ScopeAssignment{
			ScopeKey:     s.ScopeKey,
			ResourceType: s.ResourceType,
			ResourceID:   int64(s.ResourceID),
		})
	}

	return &authz.Principal{
		UserID:       user.ID,
		Username:     user.Username,
		AccountType:  user.AccountType,
		Roles:        roles,
		Permissions:  perms,
		DataScopes:   scopeAssignments,
		TokenVersion: user.TokenVersion,
		LegacyRole:   user.Role,
	}, nil
}

// IncrementTokenVersion bumps the user's token_version to invalidate stale access tokens.
// Returns the new token_version value so callers can synchronize caches (e.g. Redis).
// Uses a transaction so the UPDATE and SELECT are atomic.
func (r *AuthzRepo) IncrementTokenVersion(ctx context.Context, userID uint64) (int32, error) {
	var newVersion int32
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.User{}).
			Where("id = ?", userID).
			Update("token_version", gorm.Expr("token_version + 1")).Error; err != nil {
			return err
		}
		var user model.User
		if err := tx.Select("token_version").First(&user, userID).Error; err != nil {
			return err
		}
		newVersion = user.TokenVersion
		return nil
	})
	return newVersion, err
}

// ── Seed helpers ───────────────────────────────────────────────────────

// EnsureRBACSeeded runs the seed migrations idempotently.
func (r *AuthzRepo) EnsureRBACSeeded(ctx context.Context) error {
	// Execute the seed SQL directly via GORM raw SQL for idempotency.
	// The migrations are expected to be run by the migration tool;
	// this is a programmatic fallback for tests or manual seeding.
	seedSQL := strings.TrimSpace(`
INSERT INTO roles (role_key, name, description, is_system, created_at, updated_at) VALUES
('candidate','求职者','外部求职者，管理个人资料、简历、投递和AI会话',1,NOW(),NOW()),
('recruiter','招聘专员','负责岗位发布、候选人流程、面试安排和HR AI使用',1,NOW(),NOW()),
('recruiting_admin','招聘管理员','管理招聘配置、邀请码、部门、地点、用户角色分配',1,NOW(),NOW()),
('system_admin','系统管理员','管理平台安全配置、角色目录、权限目录、审计日志',1,NOW(),NOW()),
('interviewer','面试官','查看被分配的面试并提交反馈',1,NOW(),NOW())
ON DUPLICATE KEY UPDATE name=VALUES(name), description=VALUES(description), updated_at=NOW()
	`)
	return r.db.WithContext(ctx).Exec(seedSQL).Error
}
