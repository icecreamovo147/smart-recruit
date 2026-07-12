package domain

// JobID identifies a recruitment-owned job aggregate.
type JobID int64

// StaffUserID identifies the HR/staff owner responsible for a job.
type StaffUserID int64

// DepartmentID identifies a recruitment department snapshot owner.
type DepartmentID int64

// LocationID identifies a recruitment job-location snapshot owner.
type LocationID int64

// JobStatus is the public recruitment job lifecycle flag used by current storage.
type JobStatus int32

const (
	JobStatusOffline JobStatus = 0
	JobStatusOnline  JobStatus = 1
)

// JobScopeLevel describes the effective recruitment job access level after Identity scope evaluation.
type JobScopeLevel int

const (
	JobScopeDenied JobScopeLevel = iota
	JobScopeOwned
	JobScopeDepartmentOrLocation
	JobScopeFull
)
