package client

import (
	"context"
	"fmt"

	sharedauthz "smart-recruit-commons/pkg/authz"
	"smart-recruit-interview-service/internal/application/port"
	sharedrepo "smart-recruit-interview-service/internal/legacydomain/repository"
	"smart-recruit-platform-go/metadata"
)

type Authorizer struct {
	authz        *sharedrepo.AuthzRepo
	applications *sharedrepo.ApplicationRepo
	jobs         *sharedrepo.JobRepo
	interviews   *sharedrepo.InterviewRepo
}

func NewAuthorizer(authz *sharedrepo.AuthzRepo, applications *sharedrepo.ApplicationRepo, jobs *sharedrepo.JobRepo, interviews *sharedrepo.InterviewRepo) *Authorizer {
	return &Authorizer{authz: authz, applications: applications, jobs: jobs, interviews: interviews}
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
	if a.authz == nil {
		return nil
	}
	perms, err := a.authz.GetUserPermissions(ctx, uint64(actorID))
	if err != nil {
		return fmt.Errorf("permission lookup failed: %w", err)
	}
	for _, perm := range perms {
		if perm == string(permission) {
			return nil
		}
	}
	return fmt.Errorf("actor %d missing permission %q", actorID, permission)
}

func (a *Authorizer) CanScheduleApplication(ctx context.Context, actorID int64, applicationID int64) error {
	detail, err := a.applications.GetDetail(ctx, applicationID)
	if err != nil {
		return err
	}
	if detail == nil {
		return fmt.Errorf("application %d not found", applicationID)
	}
	job, err := a.jobs.GetByID(ctx, detail.JobID)
	if err != nil {
		return err
	}
	if job == nil {
		return fmt.Errorf("job %d not found", detail.JobID)
	}
	return a.canAccessJob(ctx, actorID, job.ID, job.HrID, job.DepartmentID, job.LocationID)
}

func (a *Authorizer) CanReadInterview(ctx context.Context, actorID int64, interviewID int64) error {
	detail, err := a.interviews.GetByID(ctx, interviewID)
	if err != nil {
		return err
	}
	if detail == nil {
		return fmt.Errorf("interview %d not found", interviewID)
	}
	if detail.InterviewerID == actorID {
		return nil
	}
	scopeKeys, err := a.authz.GetUserScopeKeys(ctx, uint64(actorID))
	if err != nil {
		return err
	}
	for _, scopeKey := range scopeKeys {
		if scopeKey == sharedauthz.ScopeRecruitingAll || scopeKey == sharedauthz.ScopeSystemAll {
			return nil
		}
	}
	appDetail, err := a.applications.GetDetail(ctx, detail.ApplicationID)
	if err != nil {
		return err
	}
	if appDetail == nil {
		return fmt.Errorf("application %d not found", detail.ApplicationID)
	}
	for _, scopeKey := range scopeKeys {
		if scopeKey == sharedauthz.ScopeOwnJobs {
			belongs, err := a.jobs.BelongsToHR(ctx, actorID, appDetail.JobID)
			if err != nil {
				return err
			}
			if belongs {
				return nil
			}
		}
	}
	return fmt.Errorf("access denied to interview %d", interviewID)
}

func (a *Authorizer) canAccessJob(ctx context.Context, actorID int64, jobID int64, hrID int64, departmentID *int64, locationID *int64) error {
	scopeKeys, err := a.authz.GetUserScopeKeys(ctx, uint64(actorID))
	if err != nil {
		return fmt.Errorf("scope lookup failed: %w", err)
	}
	hasOwnJobs, hasDept, hasLoc, hasInterview := false, false, false, false
	for _, scopeKey := range scopeKeys {
		switch scopeKey {
		case sharedauthz.ScopeRecruitingAll, sharedauthz.ScopeSystemAll:
			return nil
		case sharedauthz.ScopeOwnJobs:
			hasOwnJobs = true
		case sharedauthz.ScopeDepartment:
			hasDept = true
		case sharedauthz.ScopeLocation:
			hasLoc = true
		case sharedauthz.ScopeAssignedInterviews:
			hasInterview = true
		}
	}
	if hasOwnJobs && hrID == actorID {
		return nil
	}
	if hasDept && departmentID != nil {
		departmentIDs, err := a.authz.GetUserDepartmentIDs(ctx, uint64(actorID))
		if err != nil {
			return fmt.Errorf("department scope lookup: %w", err)
		}
		for _, id := range departmentIDs {
			if uint64(*departmentID) == id {
				return nil
			}
		}
	}
	if hasLoc && locationID != nil {
		locationIDs, err := a.authz.GetUserLocationIDs(ctx, uint64(actorID))
		if err != nil {
			return fmt.Errorf("location scope lookup: %w", err)
		}
		for _, id := range locationIDs {
			if uint64(*locationID) == id {
				return nil
			}
		}
	}
	if hasInterview {
		ok, err := a.authz.IsInterviewerForJob(ctx, uint64(actorID), uint64(jobID))
		if err != nil {
			return fmt.Errorf("interviewer scope lookup: %w", err)
		}
		if ok {
			return nil
		}
	}
	return fmt.Errorf("scope denied for user %d", actorID)
}
