package port

import (
	"context"
	"errors"

	"smart-recruit-analytics-service/internal/domain/model"
)

var ErrPermissionDenied = errors.New("analytics permission denied")

type PermissionAuthorizer interface {
	AuthorizePermission(ctx context.Context, actorID uint64, permission string) error
}

type ScopeProvider interface {
	GetUserScopeData(ctx context.Context, actorID uint64) (model.ScopeData, error)
}
