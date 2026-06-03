package service

import (
	"context"
	"fmt"

	"logic-grpc-service/pkg/authz"
	"logic-grpc-service/repository"
)

// scopeEvaluator extracts the common data-scope authorization logic shared by
// JobService and ApplicationService. It answers: given a user's scope keys and
// an optional job, what level of access does the user have?
type scopeEvaluator struct {
	authzRepo *repository.AuthzRepo
}

// jobScopeTarget carries the fields needed for scope evaluation against a job.
type jobScopeTarget struct {
	ID           int64
	HrID         int64
	DepartmentID *int64
	LocationID   *int64
}

// evalScope returns the effective access level for a user on a given job.
// Pass jobGetter=nil for create-like operations where no existing job is checked.
// Pass a non-nil jobGetter to fetch job details on demand (avoids a DB call
// when scope is already full-access).
func (e *scopeEvaluator) evalScope(ctx context.Context, userID uint64, jobGetter func() (*jobScopeTarget, error)) (ScopeLevel, error) {
	if e.authzRepo == nil {
		return scopeFull, nil
	}

	scopeKeys, err := e.authzRepo.GetUserScopeKeys(ctx, userID)
	if err != nil {
		return scopeDenied, fmt.Errorf("scope lookup failed: %w", err)
	}

	// Full access scopes — bypass all checks.
	for _, sk := range scopeKeys {
		if sk == authz.ScopeRecruitingAll || sk == authz.ScopeSystemAll {
			return scopeFull, nil
		}
	}

	// Detect scope categories present.
	hasOwnJobs, hasDept, hasLoc, hasInterview := false, false, false, false
	for _, sk := range scopeKeys {
		switch sk {
		case authz.ScopeOwnJobs:
			hasOwnJobs = true
		case authz.ScopeDepartment:
			hasDept = true
		case authz.ScopeLocation:
			hasLoc = true
		case authz.ScopeAssignedInterviews:
			hasInterview = true
		}
	}

	// If no job to check against (create-like operations), any scope is enough.
	if jobGetter == nil {
		if hasOwnJobs || hasDept || hasLoc {
			return scopeOwned, nil
		}
		return scopeDenied, fmt.Errorf("no valid data scope assigned")
	}

	// Fetch the target job to validate ownership/dept/loc/interview scope.
	job, err := jobGetter()
	if err != nil {
		return scopeDenied, err
	}
	if job == nil {
		return scopeDenied, fmt.Errorf("target job not found")
	}

	if hasOwnJobs && job.HrID == int64(userID) {
		return scopeOwned, nil
	}
	if hasDept {
		deptIDs, err := e.authzRepo.GetUserDepartmentIDs(ctx, userID)
		if err != nil {
			return scopeDenied, fmt.Errorf("department scope lookup: %w", err)
		}
		if job.DepartmentID != nil {
			for _, dID := range deptIDs {
				if uint64(*job.DepartmentID) == dID {
					return scopeDepartmentOrLocation, nil
				}
			}
		}
	}
	if hasLoc {
		locIDs, err := e.authzRepo.GetUserLocationIDs(ctx, userID)
		if err != nil {
			return scopeDenied, fmt.Errorf("location scope lookup: %w", err)
		}
		if job.LocationID != nil {
			for _, lID := range locIDs {
				if uint64(*job.LocationID) == lID {
					return scopeDepartmentOrLocation, nil
				}
			}
		}
	}
	if hasInterview {
		isInterviewer, err := e.authzRepo.IsInterviewerForJob(ctx, userID, uint64(job.ID))
		if err != nil {
			return scopeDenied, fmt.Errorf("interviewer scope lookup: %w", err)
		}
		if isInterviewer {
			return scopeOwned, nil
		}
	}

	return scopeDenied, fmt.Errorf("scope denied for user %d", userID)
}
