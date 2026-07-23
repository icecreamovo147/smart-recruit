package client

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	sharedauthz "smart-recruit-commons/pkg/authz"
	"smart-recruit-interview-service/internal/application/port"
	"smart-recruit-interview-service/internal/domain/repository"
	"smart-recruit-platform-go/errs"
	"smart-recruit-platform-go/metadata"
	"smart-recruit-proto/recruitment/pb"
)

type InterviewAssignmentReader interface {
	IsInterviewerForApplication(ctx context.Context, userID, applicationID int64) (bool, error)
	IsInterviewerForJob(ctx context.Context, userID, jobID int64) (bool, error)
}

type GormInterviewAssignmentReader struct {
	db *gorm.DB
}

func NewGormInterviewAssignmentReader(db *gorm.DB) *GormInterviewAssignmentReader {
	return &GormInterviewAssignmentReader{db: db}
}

func (r *GormInterviewAssignmentReader) IsInterviewerForApplication(ctx context.Context, userID, applicationID int64) (bool, error) {
	if !r.db.Migrator().HasTable("interview_schedules") {
		return false, nil
	}
	var count int64
	err := r.db.WithContext(ctx).Table("interview_schedules").
		Where("interviewer_id = ? AND application_id = ? AND deleted_at IS NULL", userID, applicationID).
		Count(&count).Error
	return count > 0, err
}

func (r *GormInterviewAssignmentReader) IsInterviewerForJob(ctx context.Context, userID, jobID int64) (bool, error) {
	if !r.db.Migrator().HasTable("interview_schedules") {
		return false, nil
	}
	var count int64
	err := r.db.WithContext(ctx).Table("interview_schedules").
		Joins("JOIN applications a ON a.id = interview_schedules.application_id").
		Where("interview_schedules.interviewer_id = ? AND a.job_id = ? AND interview_schedules.deleted_at IS NULL", userID, jobID).
		Count(&count).Error
	return count > 0, err
}

type Authorizer struct {
	auth         pb.AuthServiceClient
	applications port.ApplicationSnapshotReader
	interviews   repository.InterviewRepository
	assignments  InterviewAssignmentReader
}

func NewAuthorizer(auth pb.AuthServiceClient, applications port.ApplicationSnapshotReader, interviews repository.InterviewRepository, assignments InterviewAssignmentReader) *Authorizer {
	return &Authorizer{auth: auth, applications: applications, interviews: interviews, assignments: assignments}
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

func (a *Authorizer) CanScheduleApplication(ctx context.Context, actorID int64, applicationID int64) error {
	snapshot, err := a.applications.GetApplicationSnapshot(ctx, applicationID)
	if err != nil {
		return err
	}
	if snapshot == nil {
		return fmt.Errorf("application %d not found", applicationID)
	}
	return a.canAccessApplication(ctx, actorID, snapshot)
}

func (a *Authorizer) CanReadInterview(ctx context.Context, actorID int64, interviewID int64) error {
	detail, err := a.interviews.FindDetailsByID(ctx, interviewID)
	if err != nil {
		return err
	}
	if detail == nil {
		return fmt.Errorf("interview %d not found", interviewID)
	}
	if detail.Interview.InterviewerID == actorID {
		return nil
	}
	snapshot, err := a.applications.GetApplicationSnapshot(ctx, detail.Interview.ApplicationID)
	if err != nil {
		return err
	}
	if snapshot == nil {
		return fmt.Errorf("application %d not found", detail.Interview.ApplicationID)
	}
	return a.canAccessApplication(ctx, actorID, snapshot)
}

func (a *Authorizer) canAccessApplication(ctx context.Context, actorID int64, snapshot *port.ApplicationSnapshot) error {
	principal, err := a.auth.GetPrincipal(ctx, &pb.GetPrincipalRequest{UserId: actorID})
	if err != nil {
		return err
	}
	if principal.Code != errs.OK {
		return fmt.Errorf("principal lookup failed: %s", principal.Msg)
	}
	if ok, err := a.canAccessJob(ctx, actorID, snapshot, principal.DataScopes); err != nil {
		return err
	} else if ok {
		return nil
	}
	return fmt.Errorf("scope denied for user %d", actorID)
}

func (a *Authorizer) canAccessJob(ctx context.Context, actorID int64, snapshot *port.ApplicationSnapshot, scopes []*pb.ScopeAssignment) (bool, error) {
	for _, scope := range scopes {
		switch scope.ScopeKey {
		case sharedauthz.ScopeRecruitingAll, sharedauthz.ScopeSystemAll:
			return true, nil
		case sharedauthz.ScopeOwnJobs:
			if snapshot.JobHRID == actorID {
				return true, nil
			}
		case sharedauthz.ScopeDepartment:
			if snapshot.DepartmentID != nil && scopeMatches(scope, "department", *snapshot.DepartmentID) {
				return true, nil
			}
		case sharedauthz.ScopeLocation:
			if snapshot.LocationID != nil && scopeMatches(scope, "location", *snapshot.LocationID) {
				return true, nil
			}
		case sharedauthz.ScopeAssignedInterviews:
			if a.assignments == nil {
				continue
			}
			ok, err := a.assignments.IsInterviewerForJob(ctx, actorID, snapshot.JobID)
			if err != nil {
				return false, fmt.Errorf("interviewer scope lookup: %w", err)
			}
			if ok {
				return true, nil
			}
		}
	}
	return false, nil
}

func scopeMatches(scope *pb.ScopeAssignment, resourceType string, resourceID int64) bool {
	if scope.ResourceType != "" && scope.ResourceType != resourceType {
		return false
	}
	return scope.ResourceId == 0 || scope.ResourceId == resourceID
}
