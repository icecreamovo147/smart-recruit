// Package service provides domain service implementations for the recruitment system.
package service

import (
	"context"
	"fmt"

	"logic-grpc-service/pkg/metadata"
	"logic-grpc-service/repository"
)

// ServiceAuthorizer provides a unified interface for authorization checks
// across all services. It combines permission verification with data scope
// evaluation, ensuring that:
//   - The actor has the required permission key.
//   - The actor's data scope grants access to the target resource.
//   - Identity context validation (future: passed via gRPC metadata).
//
// All protected service methods should use this helper instead of ad-hoc checks.
type ServiceAuthorizer struct {
	authzRepo  *repository.AuthzRepo
	scopeEval  *scopeEvaluator
}

// NewServiceAuthorizer creates an authorizer with the given dependencies.
// Both parameters are optional; when authzRepo is nil, Authorize returns
// scopeFull without actual checking (for test compatibility).
func NewServiceAuthorizer(authzRepo *repository.AuthzRepo, scopeEval *scopeEvaluator) *ServiceAuthorizer {
	return &ServiceAuthorizer{authzRepo: authzRepo, scopeEval: scopeEval}
}

// AuthorizePermission checks that the actor has the given permission key.
// Returns an error with a human-readable message if the check fails.
func (a *ServiceAuthorizer) AuthorizePermission(ctx context.Context, actorID uint64, permKey string) error {
	if a.authzRepo == nil {
		return nil // nil repo = no enforcement (test mode)
	}
	perms, err := a.authzRepo.GetUserPermissions(ctx, actorID)
	if err != nil {
		return fmt.Errorf("permission lookup failed: %w", err)
	}
	for _, p := range perms {
		if p == permKey {
			return nil
		}
	}
	return fmt.Errorf("actor %d missing permission %q", actorID, permKey)
}

// AuthorizeScope evaluates the actor's data scope for the given job.
// Returns the effective ScopeLevel and nil error on success, or scopeDenied
// with an error on failure.
// Pass jobGetter=nil for create-like operations where no existing job is checked.
func (a *ServiceAuthorizer) AuthorizeScope(ctx context.Context, actorID uint64, jobGetter func() (*jobScopeTarget, error)) (ScopeLevel, error) {
	if a.scopeEval == nil {
		return scopeDenied, fmt.Errorf("scope evaluator not configured")
	}
	return a.scopeEval.evalScope(ctx, actorID, jobGetter)
}

// AuthorizePermissionAndScope checks both permission and scope in one call.
// Returns nil if both checks pass.
func (a *ServiceAuthorizer) AuthorizePermissionAndScope(ctx context.Context, actorID uint64, permKey string, jobGetter func() (*jobScopeTarget, error)) (ScopeLevel, error) {
	if err := a.AuthorizePermission(ctx, actorID, permKey); err != nil {
		return scopeDenied, err
	}
	return a.AuthorizeScope(ctx, actorID, jobGetter)
}

// HasPermission is a convenience wrapper that returns a boolean.
func (a *ServiceAuthorizer) HasPermission(ctx context.Context, actorID uint64, permKey string) bool {
	return a.AuthorizePermission(ctx, actorID, permKey) == nil
}

// VerifyActorMatch checks that the request's user ID matches the authenticated user
// from the gRPC context (set by the server interceptor from x-authenticated-user-id metadata).
//
// This method is FAIL-CLOSED: if the authenticated user is not present in the context
// (e.g. direct gRPC call bypassing web-gin), the request is REJECTED. This closes
// the service-layer authorization gap — every user-facing read/write must carry
// the authenticated actor via gRPC metadata.
func (a *ServiceAuthorizer) VerifyActorMatch(ctx context.Context, requestUserID int64) error {
	authUserID := metadata.GetAuthUserID(ctx)
	if authUserID == 0 {
		return fmt.Errorf("authenticated user not found in context — gRPC metadata x-authenticated-user-id is required for this operation")
	}
	if requestUserID != authUserID {
		return fmt.Errorf("actor mismatch: authenticated user %d attempted to access resources of user %d", authUserID, requestUserID)
	}
	return nil
}
