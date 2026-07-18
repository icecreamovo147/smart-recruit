package client

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"smart-recruit-analytics-service/internal/application/port"
	"smart-recruit-analytics-service/internal/domain/model"
	sharedauthz "smart-recruit-commons/pkg/authz"
)

type AuthzAdapter struct {
	db *gorm.DB
}

func NewAuthzAdapter(db *gorm.DB) *AuthzAdapter {
	return &AuthzAdapter{db: db}
}

func (a *AuthzAdapter) AuthorizePermission(ctx context.Context, actorID uint64, permission string) error {
	if a == nil || a.db == nil {
		return nil
	}
	perms, err := a.userPermissions(ctx, actorID)
	if err != nil {
		return fmt.Errorf("%w: permission lookup failed: %v", port.ErrPermissionDenied, err)
	}
	for _, perm := range perms {
		if perm == permission {
			return nil
		}
	}
	return fmt.Errorf("%w: actor %d missing permission %q", port.ErrPermissionDenied, actorID, permission)
}

func (a *AuthzAdapter) GetUserScopeData(ctx context.Context, actorID uint64) (model.ScopeData, error) {
	if a == nil || a.db == nil {
		return model.ScopeData{ActorID: actorID}, nil
	}
	scopeKeys, err := a.userScopeKeys(ctx, actorID)
	if err != nil {
		return model.ScopeData{}, err
	}
	deptIDs, err := a.userScopedResourceIDs(ctx, actorID, sharedauthz.ScopeDepartment, "department")
	if err != nil {
		return model.ScopeData{}, err
	}
	locIDs, err := a.userScopedResourceIDs(ctx, actorID, sharedauthz.ScopeLocation, "location")
	if err != nil {
		return model.ScopeData{}, err
	}
	return model.ScopeData{
		ActorID:       actorID,
		ScopeKeys:     scopeKeys,
		DepartmentIDs: deptIDs,
		LocationIDs:   locIDs,
	}, nil
}

func (a *AuthzAdapter) userPermissions(ctx context.Context, actorID uint64) ([]string, error) {
	var permissions []string
	err := a.db.WithContext(ctx).
		Table("user_roles").
		Select("DISTINCT p.permission_key").
		Joins("JOIN role_permissions rp ON rp.role_id = user_roles.role_id").
		Joins("JOIN permissions p ON p.id = rp.permission_id").
		Where("user_roles.user_id = ? AND user_roles.revoked_at IS NULL", actorID).
		Pluck("p.permission_key", &permissions).Error
	return permissions, err
}

func (a *AuthzAdapter) userScopeKeys(ctx context.Context, actorID uint64) ([]string, error) {
	var keys []string
	err := a.db.WithContext(ctx).
		Table("user_data_scopes").
		Where("user_id = ? AND revoked_at IS NULL", actorID).
		Distinct("scope_key").
		Pluck("scope_key", &keys).Error
	return keys, err
}

func (a *AuthzAdapter) userScopedResourceIDs(ctx context.Context, actorID uint64, scopeKey string, resourceType string) ([]uint64, error) {
	var ids []uint64
	err := a.db.WithContext(ctx).
		Table("user_data_scopes").
		Where("user_id = ? AND scope_key = ? AND revoked_at IS NULL AND resource_type = ? AND resource_id > 0",
			actorID, scopeKey, resourceType).
		Distinct("resource_id").
		Pluck("resource_id", &ids).Error
	return ids, err
}
