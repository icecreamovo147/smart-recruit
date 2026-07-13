package command

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
