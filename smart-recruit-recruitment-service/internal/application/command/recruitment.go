package command

import "time"

type CreateJob struct {
	HRID         int64
	Title        string
	Department   string
	DepartmentID int64
	Location     string
	LocationID   int64
	SalaryRange  string
	Description  string
	Requirements string
}

type SetJobStatus struct {
	HRID   int64
	JobID  int64
	Online bool
}

type UpdateCandidateProfile struct {
	UserID         int64
	RealName       string
	Phone          string
	Education      string
	School         string
	WorkExperience string
	Skills         string
}

type PresignResumeUpload struct {
	UserID   int64
	FileName string
	FileType string
}

type ConfirmResumeUpload struct {
	UserID        int64
	UploadID      string
	OSSKey        string
	FileName      string
	FileType      string
	FileSize      int64
	StrictSession bool
}

type ApplyJob struct {
	UserID int64
	JobID  int64
}

type UpdateApplicationStatus struct {
	HRID          int64
	ApplicationID int64
	Status        int32
	StatusKey     string
	Reason        string
}

type CreateNote struct {
	StaffUserID     uint64
	CandidateUserID uint64
	ApplicationID   uint64
	Content         string
}

type CreateTag struct {
	StaffUserID uint64
	Name        string
	Color       string
}

type AssignTag struct {
	StaffUserID     uint64
	CandidateUserID uint64
	TagID           uint64
}

type CreateFollowUpTask struct {
	StaffUserID     uint64
	CandidateUserID uint64
	ApplicationID   uint64
	AssigneeUserID  uint64
	Title           string
	Description     string
	DueAt           *time.Time
}

type CreateDepartment struct {
	AdminID   int64
	ParentID  int64
	Name      string
	SortOrder int
}

type GetUsageStats struct {
	ActorUserID uint64
	StartTime   string
	EndTime     string
	Dimension   string
}

type GetUsageTrend struct {
	ActorUserID uint64
	StartTime   string
	EndTime     string
	Granularity string
}
