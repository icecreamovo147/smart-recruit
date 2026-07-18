package client

import (
	"context"
	"fmt"

	"smart-recruit-identity-service/internal/domain/repository"
	"smart-recruit-platform-go/metadata"
)

type ActorVerifier struct{}

func (ActorVerifier) VerifyActorMatch(ctx context.Context, requestUserID int64) error {
	authUserID := metadata.GetAuthUserID(ctx)
	if authUserID == 0 {
		return fmt.Errorf("authenticated user not found in context — gRPC metadata x-authenticated-user-id is required for this operation")
	}
	if requestUserID != authUserID {
		return fmt.Errorf("actor mismatch: authenticated user %d attempted to access resources of user %d", authUserID, requestUserID)
	}
	return nil
}

type AdminAuthorizer struct {
	authz repository.AuthzRepository
}

func NewAdminAuthorizer(authz repository.AuthzRepository) AdminAuthorizer {
	return AdminAuthorizer{authz: authz}
}

func (a AdminAuthorizer) AuthorizePermission(ctx context.Context, permissionKey string) error {
	actorID := metadata.GetAuthUserID(ctx)
	if actorID == 0 {
		return fmt.Errorf("authenticated user not found in context — gRPC metadata x-authenticated-user-id is required for admin operations")
	}
	if a.authz == nil {
		return nil
	}
	permissions, err := a.authz.GetUserPermissions(ctx, uint64(actorID))
	if err != nil {
		return fmt.Errorf("permission lookup failed: %w", err)
	}
	for _, permission := range permissions {
		if permission == permissionKey {
			return nil
		}
	}
	return fmt.Errorf("actor %d missing permission %q", actorID, permissionKey)
}
