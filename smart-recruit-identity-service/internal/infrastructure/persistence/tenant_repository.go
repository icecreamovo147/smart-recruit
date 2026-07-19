package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"smart-recruit-identity-service/internal/domain/model"
	platformmetadata "smart-recruit-platform-go/metadata"
)

type TenantRepository struct {
	db *gorm.DB
}

func NewTenantRepository(db *gorm.DB) *TenantRepository {
	return &TenantRepository{db: db}
}

func (r *TenantRepository) GetByID(ctx context.Context, tenantID int64) (*model.Tenant, error) {
	if tenantID <= 0 {
		return nil, nil
	}
	var row tenantRow
	err := r.db.WithContext(ctx).Where("id = ?", tenantID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return row.toDomain(), nil
}

func (r *TenantRepository) GetDefault(ctx context.Context) (*model.Tenant, error) {
	var row tenantRow
	err := r.db.WithContext(ctx).Where("is_default = 1").First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return row.toDomain(), nil
}

func (r *TenantRepository) Create(ctx context.Context, tenant *model.Tenant) error {
	row := tenantRow{TenantKey: tenant.TenantKey, Slug: tenant.Slug, Name: tenant.Name, Status: tenant.Status, Timezone: tenant.Timezone, Locale: tenant.Locale}
	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return createPlatformAudit(tx, ctx, "tenant.create", row.ID, nil, tenantAuditSnapshot(row))
	}); err != nil {
		return err
	}
	tenant.ID = row.ID
	return nil
}

func (r *TenantRepository) List(ctx context.Context, offset, limit int, keyword, status string) ([]model.Tenant, int64, error) {
	query := r.db.WithContext(ctx).Model(&tenantRow{})
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("name LIKE ? OR slug LIKE ? OR tenant_key LIKE ?", like, like, like)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []tenantRow
	if err := query.Select("tenants.*, (SELECT COUNT(*) FROM tenant_memberships membership WHERE membership.tenant_id = tenants.id) AS membership_count").Order("is_default DESC, created_at DESC, id DESC").Offset(offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	result := make([]model.Tenant, len(rows))
	for i := range rows {
		result[i] = *rows[i].toDomain()
	}
	return result, total, nil
}

func (r *TenantRepository) UpdateStatus(ctx context.Context, tenantID int64, status string) (*model.Tenant, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var before tenantRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND is_default = 0", tenantID).First(&before).Error; err != nil {
			return err
		}
		if before.Status == status {
			return nil
		}
		if err := tx.Model(&tenantRow{}).Where("id = ? AND is_default = 0", tenantID).Update("status", status).Error; err != nil {
			return err
		}
		after := before
		after.Status = status
		return createPlatformAudit(tx, ctx, "tenant.status.update", tenantID, tenantAuditSnapshot(before), tenantAuditSnapshot(after))
	})
	if err != nil {
		return nil, fmt.Errorf("update tenant status: %w", err)
	}
	return r.GetByID(ctx, tenantID)
}

func createPlatformAudit(tx *gorm.DB, ctx context.Context, action string, tenantID int64, before, after map[string]any) error {
	beforeJSON, err := json.Marshal(before)
	if err != nil {
		return err
	}
	afterJSON, err := json.Marshal(after)
	if err != nil {
		return err
	}
	row := platformAuditRow{
		ActorUserID:    platformmetadata.GetAuthUserID(ctx),
		Action:         action,
		ResourceType:   "tenant",
		ResourceID:     tenantID,
		TargetTenantID: tenantID,
		BeforeJSON:     nullableAuditJSON(before, beforeJSON),
		AfterJSON:      nullableAuditJSON(after, afterJSON),
		RequestID:      platformmetadata.GetRequestID(ctx),
		ClientIP:       platformmetadata.GetClientIP(ctx),
	}
	return tx.Create(&row).Error
}

func nullableAuditJSON(value map[string]any, encoded []byte) *string {
	if value == nil {
		return nil
	}
	result := string(encoded)
	return &result
}

func tenantAuditSnapshot(row tenantRow) map[string]any {
	return map[string]any{"id": row.ID, "tenant_key": row.TenantKey, "slug": row.Slug, "name": row.Name, "status": row.Status, "timezone": row.Timezone, "locale": row.Locale, "is_default": row.IsDefault}
}

type platformAuditRow struct {
	ID             uint64    `gorm:"primaryKey"`
	ActorUserID    int64     `gorm:"column:actor_user_id"`
	Action         string    `gorm:"column:action"`
	ResourceType   string    `gorm:"column:resource_type"`
	ResourceID     int64     `gorm:"column:resource_id"`
	TargetTenantID int64     `gorm:"column:target_tenant_id"`
	BeforeJSON     *string   `gorm:"column:before_json"`
	AfterJSON      *string   `gorm:"column:after_json"`
	RequestID      string    `gorm:"column:request_id"`
	ClientIP       string    `gorm:"column:client_ip"`
	CreatedAt      time.Time `gorm:"column:created_at"`
}

func (platformAuditRow) TableName() string { return "platform_audit_logs" }

func (r *TenantRepository) ListTenantMemberships(ctx context.Context, tenantID int64, offset, limit int) ([]model.TenantMembership, int64, error) {
	query := r.db.WithContext(ctx).Table("tenant_memberships membership").Where("membership.tenant_id = ?", tenantID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []membershipTenantRow
	if err := query.Select("membership.id, membership.tenant_id, membership.user_id, membership.status, membership.joined_at, user.username, tenant.tenant_key, tenant.slug, tenant.name AS tenant_name, tenant.status AS tenant_status, tenant.timezone, tenant.locale, tenant.is_default").
		Joins("JOIN users user ON user.id = membership.user_id").Joins("JOIN tenants tenant ON tenant.id = membership.tenant_id").
		Order("membership.created_at DESC, membership.id DESC").Offset(offset).Limit(limit).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	result := make([]model.TenantMembership, len(rows))
	for i := range rows {
		result[i] = rows[i].toDomain()
		roles, err := r.membershipRoles(ctx, rows[i].ID)
		if err != nil {
			return nil, 0, err
		}
		result[i].Roles = roles
	}
	return result, total, nil
}

func (r *TenantRepository) ListUserMemberships(ctx context.Context, userID int64) ([]model.TenantMembership, error) {
	var rows []membershipTenantRow
	if err := r.db.WithContext(ctx).
		Table("tenant_memberships membership").
		Select("membership.id, membership.tenant_id, membership.user_id, membership.status, membership.joined_at, tenant.tenant_key, tenant.slug, tenant.name AS tenant_name, tenant.status AS tenant_status, tenant.timezone, tenant.locale, tenant.is_default").
		Joins("JOIN tenants tenant ON tenant.id = membership.tenant_id").
		Where("membership.user_id = ?", userID).
		Order("tenant.is_default DESC, tenant.name ASC, tenant.id ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	memberships := make([]model.TenantMembership, len(rows))
	for i := range rows {
		memberships[i] = rows[i].toDomain()
		roles, err := r.membershipRoles(ctx, rows[i].ID)
		if err != nil {
			return nil, err
		}
		memberships[i].Roles = roles
	}
	return memberships, nil
}

func (r *TenantRepository) GetActiveMembership(ctx context.Context, userID, tenantID int64) (*model.TenantMembership, error) {
	var row membershipTenantRow
	err := r.db.WithContext(ctx).
		Table("tenant_memberships membership").
		Select("membership.id, membership.tenant_id, membership.user_id, membership.status, membership.joined_at, tenant.tenant_key, tenant.slug, tenant.name AS tenant_name, tenant.status AS tenant_status, tenant.timezone, tenant.locale, tenant.is_default").
		Joins("JOIN tenants tenant ON tenant.id = membership.tenant_id").
		Where("membership.user_id = ? AND membership.tenant_id = ? AND membership.status = 'active' AND tenant.status = 'active'", userID, tenantID).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	membership := row.toDomain()
	membership.Roles, err = r.membershipRoles(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	return &membership, nil
}

func (r *TenantRepository) LoadTenantPrincipal(ctx context.Context, userID, tenantID int64, clientApp string) (*model.Principal, error) {
	membership, err := r.GetActiveMembership(ctx, userID, tenantID)
	if err != nil || membership == nil {
		return nil, err
	}
	var user userRow
	if err := r.db.WithContext(ctx).Where("id = ? AND status = ?", userID, model.UserStatusActive).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	permissions, err := r.membershipPermissions(ctx, membership.ID)
	if err != nil {
		return nil, err
	}
	scopes, err := r.membershipScopes(ctx, membership.ID)
	if err != nil {
		return nil, err
	}
	return &model.Principal{
		UserID: user.ID, Username: user.Username, AccountType: user.AccountType,
		Roles: membership.Roles, Permissions: permissions, DataScopes: scopes,
		TokenVersion: user.TokenVersion, Email: user.Email, LegacyRole: user.Role,
		TenantID: tenantID, MembershipID: membership.ID, ClientApp: clientApp,
	}, nil
}

func (r *TenantRepository) EnsureMembership(ctx context.Context, userID, tenantID int64, status string) (*model.TenantMembership, error) {
	if userID <= 0 || tenantID <= 0 {
		return nil, fmt.Errorf("valid user and tenant are required")
	}
	if status == "" {
		status = "active"
	}
	row := tenantMembershipRow{TenantID: tenantID, UserID: userID, Status: status}
	if status == "active" {
		now := time.Now()
		row.JoinedAt = &now
	}
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "user_id"}},
		DoUpdates: clause.Assignments(map[string]any{"status": status, "updated_at": time.Now()}),
	}).Create(&row).Error; err != nil {
		return nil, err
	}
	return r.GetActiveMembership(ctx, userID, tenantID)
}

func (r *TenantRepository) AssignMembershipRole(ctx context.Context, membershipID int64, roleID uint64, assignedBy *uint64) error {
	row := tenantMembershipRoleRow{MembershipID: membershipID, RoleID: roleID, AssignedBy: assignedBy, AssignedAt: time.Now()}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error
}

func (r *TenantRepository) RevokeMembershipRole(ctx context.Context, membershipID int64, roleID uint64) (bool, error) {
	result := r.db.WithContext(ctx).Model(&tenantMembershipRoleRow{}).
		Where("membership_id = ? AND role_id = ? AND revoked_at IS NULL", membershipID, roleID).
		Update("revoked_at", time.Now())
	return result.RowsAffected > 0, result.Error
}

func (r *TenantRepository) CountActiveTenantMembersWithRole(ctx context.Context, tenantID int64, roleID uint64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("tenant_membership_roles assignment").
		Joins("JOIN tenant_memberships membership ON membership.id = assignment.membership_id").
		Where("membership.tenant_id = ? AND membership.status = 'active' AND assignment.role_id = ? AND assignment.revoked_at IS NULL", tenantID, roleID).
		Count(&count).Error
	return count, err
}

func (r *TenantRepository) AssignMembershipDataScope(ctx context.Context, membershipID int64, scopeKey, resourceType string, resourceID uint64, assignedBy *uint64) error {
	row := tenantMembershipScopeWriteRow{
		MembershipID: membershipID, ScopeKey: scopeKey, ResourceType: resourceType,
		ResourceID: resourceID, AssignedBy: assignedBy, AssignedAt: time.Now(),
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error
}

func (r *TenantRepository) RevokeMembershipDataScope(ctx context.Context, membershipID, scopeID int64) (bool, error) {
	result := r.db.WithContext(ctx).Model(&tenantMembershipScopeWriteRow{}).
		Where("id = ? AND membership_id = ? AND revoked_at IS NULL", scopeID, membershipID).
		Update("revoked_at", time.Now())
	return result.RowsAffected > 0, result.Error
}

func (r *TenantRepository) RevokeTenantDataScope(ctx context.Context, tenantID, scopeID int64) (int64, bool, error) {
	var owner struct {
		MembershipID int64
		UserID       int64
	}
	err := r.db.WithContext(ctx).Table("tenant_membership_data_scopes scope").
		Select("scope.membership_id, membership.user_id").
		Joins("JOIN tenant_memberships membership ON membership.id = scope.membership_id").
		Where("scope.id = ? AND scope.revoked_at IS NULL AND membership.tenant_id = ?", scopeID, tenantID).
		Scan(&owner).Error
	if err != nil || owner.MembershipID == 0 {
		return 0, false, err
	}
	revoked, err := r.RevokeMembershipDataScope(ctx, owner.MembershipID, scopeID)
	return owner.UserID, revoked, err
}

func (r *TenantRepository) membershipRoles(ctx context.Context, membershipID int64) ([]string, error) {
	var roles []string
	err := r.db.WithContext(ctx).
		Table("tenant_membership_roles assignment").
		Joins("JOIN roles role ON role.id = assignment.role_id AND role.scope_type = ?", model.RoleScopeTenant).
		Where("assignment.membership_id = ? AND assignment.revoked_at IS NULL", membershipID).
		Order("role.role_key ASC").
		Pluck("role.role_key", &roles).Error
	return roles, err
}

func (r *TenantRepository) membershipPermissions(ctx context.Context, membershipID int64) ([]string, error) {
	var permissions []string
	err := r.db.WithContext(ctx).
		Table("tenant_membership_roles assignment").
		Joins("JOIN roles role ON role.id = assignment.role_id AND role.scope_type = ?", model.RoleScopeTenant).
		Joins("JOIN role_permissions mapping ON mapping.role_id = role.id").
		Joins("JOIN permissions permission ON permission.id = mapping.permission_id").
		Where("assignment.membership_id = ? AND assignment.revoked_at IS NULL", membershipID).
		Distinct().Order("permission.permission_key ASC").
		Pluck("permission.permission_key", &permissions).Error
	return permissions, err
}

func (r *TenantRepository) membershipScopes(ctx context.Context, membershipID int64) ([]model.ScopeAssignment, error) {
	var rows []tenantMembershipScopeRow
	if err := r.db.WithContext(ctx).
		Where("membership_id = ? AND revoked_at IS NULL", membershipID).
		Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	scopes := make([]model.ScopeAssignment, len(rows))
	for i := range rows {
		scopes[i] = model.ScopeAssignment{ID: rows[i].ID, ScopeKey: rows[i].ScopeKey, ResourceType: rows[i].ResourceType, ResourceID: rows[i].ResourceID, AssignedAt: rows[i].AssignedAt}
	}
	return scopes, nil
}

type tenantRow struct {
	ID              int64 `gorm:"primaryKey"`
	TenantKey       string
	Slug            string
	Name            string
	Status          string
	Timezone        string
	Locale          string
	IsDefault       bool
	MembershipCount int64 `gorm:"->"`
	CreatedAt       time.Time
}

func (tenantRow) TableName() string { return "tenants" }

func (r tenantRow) toDomain() *model.Tenant {
	return &model.Tenant{ID: r.ID, TenantKey: r.TenantKey, Slug: r.Slug, Name: r.Name, Status: r.Status, Timezone: r.Timezone, Locale: r.Locale, IsDefault: r.IsDefault, MembershipCount: r.MembershipCount}
}

type membershipTenantRow struct {
	ID           int64
	TenantID     int64
	UserID       int64
	Username     string
	Status       string
	JoinedAt     *time.Time
	TenantKey    string
	Slug         string
	TenantName   string
	TenantStatus string
	Timezone     string
	Locale       string
	IsDefault    bool
}

func (r membershipTenantRow) toDomain() model.TenantMembership {
	return model.TenantMembership{
		ID: r.ID, TenantID: r.TenantID, UserID: r.UserID, Username: r.Username, Status: r.Status, JoinedAt: r.JoinedAt,
		Tenant: model.Tenant{ID: r.TenantID, TenantKey: r.TenantKey, Slug: r.Slug, Name: r.TenantName, Status: r.TenantStatus, Timezone: r.Timezone, Locale: r.Locale, IsDefault: r.IsDefault},
	}
}

type tenantMembershipScopeRow struct {
	ID           int64 `gorm:"primaryKey"`
	MembershipID int64
	ScopeKey     string
	ResourceType string
	ResourceID   int64
	AssignedAt   time.Time
}

func (tenantMembershipScopeRow) TableName() string { return "tenant_membership_data_scopes" }

type tenantMembershipRow struct {
	ID       int64 `gorm:"primaryKey"`
	TenantID int64
	UserID   int64
	Status   string
	JoinedAt *time.Time
}

func (tenantMembershipRow) TableName() string { return "tenant_memberships" }

type tenantMembershipRoleRow struct {
	ID           int64 `gorm:"primaryKey"`
	MembershipID int64
	RoleID       uint64
	AssignedBy   *uint64
	AssignedAt   time.Time
	RevokedAt    *time.Time
}

func (tenantMembershipRoleRow) TableName() string { return "tenant_membership_roles" }

type tenantMembershipScopeWriteRow struct {
	ID           int64 `gorm:"primaryKey"`
	MembershipID int64
	ScopeKey     string
	ResourceType string
	ResourceID   uint64
	AssignedBy   *uint64
	AssignedAt   time.Time
	RevokedAt    *time.Time
}

func (tenantMembershipScopeWriteRow) TableName() string { return "tenant_membership_data_scopes" }
