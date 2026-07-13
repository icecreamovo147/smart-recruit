package client

import (
	"context"
	"fmt"

	"smart-recruit-analytics-service/internal/application/port"
	"smart-recruit-analytics-service/internal/domain/model"
	"smart-recruit-domain-go/repository"
)

type AuthzAdapter struct {
	repo *repository.AuthzRepo
}

func NewAuthzAdapter(repo *repository.AuthzRepo) *AuthzAdapter {
	return &AuthzAdapter{repo: repo}
}

func (a *AuthzAdapter) AuthorizePermission(ctx context.Context, actorID uint64, permission string) error {
	if a == nil || a.repo == nil {
		return nil
	}
	perms, err := a.repo.GetUserPermissions(ctx, actorID)
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
	if a == nil || a.repo == nil {
		return model.ScopeData{ActorID: actorID}, nil
	}
	scopeKeys, err := a.repo.GetUserScopeKeys(ctx, actorID)
	if err != nil {
		return model.ScopeData{}, err
	}
	deptIDs, err := a.repo.GetUserDepartmentIDs(ctx, actorID)
	if err != nil {
		return model.ScopeData{}, err
	}
	locIDs, err := a.repo.GetUserLocationIDs(ctx, actorID)
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
