package memory

import "fmt"

const (
	ScopeHR          = "hr"
	ScopeJob         = "job"
	ScopeApplication = "application"
	ScopeCandidate   = "candidate"
	ScopeUser        = "user"
)

var (
	hrScopeTypes = map[string]struct{}{
		ScopeHR:          {},
		ScopeJob:         {},
		ScopeApplication: {},
		ScopeCandidate:   {},
	}
	candidateScopeTypes = map[string]struct{}{
		ScopeUser:        {},
		ScopeJob:         {},
		ScopeApplication: {},
	}
)

func AllowedScopeTypes(ownerRole OwnerRole) []string {
	switch ownerRole {
	case OwnerRoleHR:
		return []string{ScopeHR, ScopeJob, ScopeApplication, ScopeCandidate}
	case OwnerRoleCandidate:
		return []string{ScopeUser, ScopeJob, ScopeApplication}
	default:
		return nil
	}
}

func ScopeAllowedForOwner(ownerRole OwnerRole, scopeType string) bool {
	switch ownerRole {
	case OwnerRoleHR:
		_, ok := hrScopeTypes[scopeType]
		return ok
	case OwnerRoleCandidate:
		_, ok := candidateScopeTypes[scopeType]
		return ok
	default:
		return false
	}
}

func ValidateScope(ownerRole OwnerRole, scope Scope) error {
	if !ScopeAllowedForOwner(ownerRole, scope.Type) {
		return fmt.Errorf("scope type %q is not allowed for owner role %d", scope.Type, ownerRole)
	}
	switch scope.Type {
	case ScopeHR:
		if scope.ID != 0 {
			return fmt.Errorf("hr scope id must be 0")
		}
	case ScopeUser:
		if scope.ID == 0 {
			return fmt.Errorf("user scope id is required")
		}
	}
	return nil
}
