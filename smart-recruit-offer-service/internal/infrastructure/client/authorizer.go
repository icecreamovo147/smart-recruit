package client

import (
	"context"
	"fmt"

	sharedauthz "smart-recruit-commons/pkg/authz"
	"smart-recruit-offer-service/internal/application/port"
	"smart-recruit-platform-go/errs"
	"smart-recruit-platform-go/metadata"
	"smart-recruit-proto/recruitment/pb"
)

type Authorizer struct {
	auth         pb.AuthServiceClient
	applications port.ApplicationSnapshotReader
}

func NewAuthorizer(auth pb.AuthServiceClient, applications port.ApplicationSnapshotReader) *Authorizer {
	return &Authorizer{auth: auth, applications: applications}
}

func (a *Authorizer) VerifyActor(ctx context.Context, actorID int64) error {
	authUserID := metadata.GetAuthUserID(ctx)
	if authUserID == 0 {
		return fmt.Errorf("authenticated user not found in context — gRPC metadata x-authenticated-user-id is required for this operation")
	}
	if actorID != authUserID {
		return fmt.Errorf("actor mismatch: authenticated user %d attempted to access resources of user %d", authUserID, actorID)
	}
	return nil
}

func (a *Authorizer) Authorize(ctx context.Context, actorID int64, permission port.Permission) error {
	resp, err := a.auth.AuthorizeInternal(ctx, &pb.AuthorizeInternalRequest{
		ActorUserId:   actorID,
		PermissionKey: string(permission),
	})
	if err != nil {
		return err
	}
	if resp.Code != errs.OK {
		return fmt.Errorf("permission lookup failed: %s", resp.Msg)
	}
	if !resp.Allowed {
		return fmt.Errorf("actor %d missing permission %q", actorID, permission)
	}
	return nil
}

func (a *Authorizer) CanManageApplication(ctx context.Context, actorID int64, applicationID int64) error {
	return a.canAccessApplication(ctx, actorID, applicationID)
}

func (a *Authorizer) CanReadApplication(ctx context.Context, actorID int64, applicationID int64) error {
	return a.canAccessApplication(ctx, actorID, applicationID)
}

func (a *Authorizer) canAccessApplication(ctx context.Context, actorID int64, applicationID int64) error {
	snapshot, err := a.applications.GetApplicationSnapshot(ctx, applicationID)
	if err != nil {
		return err
	}
	if snapshot == nil {
		return fmt.Errorf("application %d not found", applicationID)
	}
	principal, err := a.auth.GetPrincipal(ctx, &pb.GetPrincipalRequest{UserId: actorID})
	if err != nil {
		return err
	}
	if principal.Code != errs.OK {
		return fmt.Errorf("principal lookup failed: %s", principal.Msg)
	}
	if canAccessJob(actorID, snapshot, principal.DataScopes) {
		return nil
	}
	return fmt.Errorf("scope denied for user %d", actorID)
}

func canAccessJob(actorID int64, snapshot *port.ApplicationSnapshot, scopes []*pb.ScopeAssignment) bool {
	for _, scope := range scopes {
		switch scope.ScopeKey {
		case sharedauthz.ScopeRecruitingAll, sharedauthz.ScopeSystemAll:
			return true
		case sharedauthz.ScopeOwnJobs:
			if snapshot.JobHRID == actorID {
				return true
			}
		case sharedauthz.ScopeDepartment:
			if snapshot.DepartmentID != nil && scopeMatches(scope, "department", *snapshot.DepartmentID) {
				return true
			}
		case sharedauthz.ScopeLocation:
			if snapshot.LocationID != nil && scopeMatches(scope, "location", *snapshot.LocationID) {
				return true
			}
		}
	}
	return false
}

func scopeMatches(scope *pb.ScopeAssignment, resourceType string, resourceID int64) bool {
	if scope.ResourceType != "" && scope.ResourceType != resourceType {
		return false
	}
	return scope.ResourceId == 0 || scope.ResourceId == resourceID
}
