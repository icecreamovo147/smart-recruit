package service

import "context"

// DepartmentLocationValidator validates department-location combinations
// and provides effective location lookups for a department.
type DepartmentLocationValidator interface {
	// ValidateDepartmentLocation checks whether a department_id + location_id
	// combination is valid (both active, non-deleted, and associated).
	// Returns nil if valid, or an error describing why the combination is invalid.
	ValidateDepartmentLocation(ctx context.Context, departmentID, locationID int64) error

	// EffectiveLocations returns the list of locations available for the given department,
	// after resolving the inherit_locations chain.
	EffectiveLocationIDs(ctx context.Context, departmentID int64) ([]int64, error)
}
