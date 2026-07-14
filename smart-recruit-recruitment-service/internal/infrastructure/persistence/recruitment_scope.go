package persistence

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	sharedauthz "smart-recruit-commons/pkg/authz"
	"smart-recruit-platform-go/metadata"
)

type userDataScopeRecord struct {
	ID           uint64 `gorm:"primaryKey"`
	UserID       uint64
	ScopeKey     string
	ResourceType string
	ResourceID   uint64
	AssignedBy   *uint64
	AssignedAt   time.Time
	RevokedAt    *time.Time
}

func (userDataScopeRecord) TableName() string { return "user_data_scopes" }

type recruitmentScopeLevel int

const (
	recruitmentScopeDenied recruitmentScopeLevel = iota
	recruitmentScopeOwned
	recruitmentScopeDepartmentOrLocation
	recruitmentScopeFull
)

type recruitmentScope struct {
	ActorID       int64
	Level         recruitmentScopeLevel
	HasOwnJobs    bool
	DepartmentIDs []int64
	LocationIDs   []int64
}

func (s recruitmentScope) allowed() bool {
	return s.Level != recruitmentScopeDenied
}

func (s recruitmentScope) full() bool {
	return s.Level == recruitmentScopeFull
}

func (a *nativeStore) evaluateRecruitmentScope(ctx context.Context, actorID int64) (recruitmentScope, error) {
	scope := recruitmentScope{ActorID: actorID, Level: recruitmentScopeDenied}
	if a == nil || a.db == nil || actorID <= 0 {
		return scope, nil
	}
	if !a.db.Migrator().HasTable(&userDataScopeRecord{}) {
		return scope, nil
	}
	var rows []userDataScopeRecord
	if err := a.db.WithContext(ctx).
		Where("user_id = ? AND revoked_at IS NULL", uint64(actorID)).
		Find(&rows).Error; err != nil {
		return scope, err
	}
	departments := map[int64]struct{}{}
	locations := map[int64]struct{}{}
	for _, row := range rows {
		switch row.ScopeKey {
		case sharedauthz.ScopeRecruitingAll, sharedauthz.ScopeSystemAll:
			scope.Level = recruitmentScopeFull
			return scope, nil
		case sharedauthz.ScopeOwnJobs:
			scope.HasOwnJobs = true
		case sharedauthz.ScopeDepartment:
			if row.ResourceID > 0 && (row.ResourceType == "" || row.ResourceType == "department") {
				departments[int64(row.ResourceID)] = struct{}{}
			}
		case sharedauthz.ScopeLocation:
			if row.ResourceID > 0 && (row.ResourceType == "" || row.ResourceType == "location") {
				locations[int64(row.ResourceID)] = struct{}{}
			}
		}
	}
	scope.DepartmentIDs = sortedScopeIDs(departments)
	scope.LocationIDs = sortedScopeIDs(locations)
	switch {
	case len(scope.DepartmentIDs) > 0 || len(scope.LocationIDs) > 0:
		scope.Level = recruitmentScopeDepartmentOrLocation
	case scope.HasOwnJobs:
		scope.Level = recruitmentScopeOwned
	default:
		scope.Level = recruitmentScopeDenied
	}
	return scope, nil
}

func sortedScopeIDs(values map[int64]struct{}) []int64 {
	if len(values) == 0 {
		return nil
	}
	out := make([]int64, 0, len(values))
	for id := range values {
		out = append(out, id)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j-1] > out[j]; j-- {
			out[j-1], out[j] = out[j], out[j-1]
		}
	}
	return out
}

func (a *nativeStore) checkRecruitmentJobScope(ctx context.Context, actorID, jobID int64) (recruitmentScope, error) {
	scope, err := a.evaluateRecruitmentScope(ctx, actorID)
	if err != nil || scope.full() || !scope.allowed() {
		return scope, err
	}
	if jobID <= 0 {
		scope.Level = recruitmentScopeDenied
		return scope, nil
	}
	job, err := a.lookupJobScopeTarget(ctx, jobID)
	if err != nil {
		return recruitmentScope{ActorID: actorID, Level: recruitmentScopeDenied}, err
	}
	if job == nil {
		return recruitmentScope{ActorID: actorID, Level: recruitmentScopeDenied}, nil
	}
	if scope.HasOwnJobs && job.HrID == actorID {
		scope.Level = recruitmentScopeOwned
		return scope, nil
	}
	if ptrInScope(job.DepartmentID, scope.DepartmentIDs) || ptrInScope(job.LocationID, scope.LocationIDs) {
		scope.Level = recruitmentScopeDepartmentOrLocation
		return scope, nil
	}
	scope.Level = recruitmentScopeDenied
	return scope, nil
}

func (a *nativeStore) lookupJobScopeTarget(ctx context.Context, jobID int64) (*jobRecord, error) {
	var job jobRecord
	err := a.db.WithContext(ctx).
		Select("id", "hr_id", "department_id", "location_id").
		Where("id = ?", jobID).
		First(&job).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func ptrInScope(value *int64, allowed []int64) bool {
	if value == nil {
		return false
	}
	return int64InScope(*value, allowed)
}

func int64InScope(value int64, allowed []int64) bool {
	for _, id := range allowed {
		if id == value {
			return true
		}
	}
	return false
}

func applyRecruitmentScopeToJobsQuery(query *gorm.DB, scope recruitmentScope) *gorm.DB {
	if scope.full() {
		return query
	}
	return applyRecruitmentScopeToJoinedJobsQuery(query, scope, "")
}

func applyRecruitmentScopeToJoinedJobsQuery(query *gorm.DB, scope recruitmentScope, jobAlias string) *gorm.DB {
	if scope.full() {
		return query
	}
	conditions, args := scopeJobConditions(scope, jobAlias, true)
	if len(conditions) == 0 {
		return query.Where("1 = 0")
	}
	return query.Where("("+strings.Join(conditions, " OR ")+")", args...)
}

func applyRecruitmentScopeToJobMutationQuery(query *gorm.DB, scope recruitmentScope) *gorm.DB {
	if scope.full() {
		return query
	}
	conditions, args := scopeJobConditions(scope, "", false)
	if len(conditions) == 0 {
		return query.Where("1 = 0")
	}
	return query.Where("("+strings.Join(conditions, " OR ")+")", args...)
}

func applyRecruitmentScopeToApplicationMutationQuery(query *gorm.DB, scope recruitmentScope) *gorm.DB {
	if scope.full() {
		return query
	}
	conditions, args := scopeJobConditions(scope, "j", false)
	if len(conditions) == 0 {
		return query.Where("1 = 0")
	}
	return query.Where(
		"EXISTS (SELECT 1 FROM jobs j WHERE j.id = applications.job_id AND ("+strings.Join(conditions, " OR ")+"))",
		args...,
	)
}

func scopeJobConditions(scope recruitmentScope, jobAlias string, includeAllEffective bool) ([]string, []any) {
	if !scope.allowed() {
		return nil, nil
	}
	prefix := ""
	if jobAlias != "" {
		prefix = jobAlias + "."
	}
	conditions := make([]string, 0, 3)
	args := make([]any, 0, 3)
	if (includeAllEffective || scope.Level == recruitmentScopeOwned) && scope.HasOwnJobs {
		conditions = append(conditions, prefix+"hr_id = ?")
		args = append(args, scope.ActorID)
	}
	if (includeAllEffective || scope.Level == recruitmentScopeDepartmentOrLocation) && len(scope.DepartmentIDs) > 0 {
		conditions = append(conditions, prefix+"department_id IN ?")
		args = append(args, scope.DepartmentIDs)
	}
	if (includeAllEffective || scope.Level == recruitmentScopeDepartmentOrLocation) && len(scope.LocationIDs) > 0 {
		conditions = append(conditions, prefix+"location_id IN ?")
		args = append(args, scope.LocationIDs)
	}
	return conditions, args
}

func (a *nativeStore) contextStaffJobScope(ctx context.Context, jobID int64) (bool, error) {
	if metadata.GetAuthAccountType(ctx) != "staff" {
		return true, nil
	}
	actorID := metadata.GetAuthUserID(ctx)
	scope, err := a.checkRecruitmentJobScope(ctx, actorID, jobID)
	if err != nil {
		return false, err
	}
	return scope.allowed(), nil
}
