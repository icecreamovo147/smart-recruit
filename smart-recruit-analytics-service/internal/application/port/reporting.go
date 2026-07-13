package port

import (
	"context"

	"smart-recruit-analytics-service/internal/domain/model"
)

type PermissionAuthorizer interface {
	AuthorizePermission(ctx context.Context, actorID uint64, permission string) error
}

type ScopeProvider interface {
	GetUserScopeData(ctx context.Context, actorID uint64) (model.ScopeData, error)
}
