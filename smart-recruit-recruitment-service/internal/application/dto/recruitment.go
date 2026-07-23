package dto

import "smart-recruit-recruitment-service/internal/domain/model"

type CreateJobResult struct {
	JobID int64
}

type CandidateProfileResult struct {
	Profile model.CandidateProfile
}

type PresignResumeUploadResult struct {
	UploadURL string
	OSSKey    string
	ExpireAt  int64
	UploadID  string
}

type ConfirmResumeUploadResult struct {
	ResumeID int64
}

type ApplyJobResult struct {
	ApplicationID int64
}

type StatusChangeResult struct {
	FromStatus string
	ToStatus   string
	IsRePass   bool
}

type ApplicationSnapshot struct {
	ApplicationID   int64
	CandidateUserID int64
	JobID           int64
	JobTitle        string
	CandidateName   string
	ResumeID        int64
	LegacyStatus    int32
	StatusKey       string
	RoundNo         int32
	IsCurrent       bool
	JobHRID         int64
	DepartmentID    *int64
	LocationID      *int64
}

type ApplicationLifecycleTransitionResult struct {
	Changed          bool
	FromStatusKey    string
	CurrentStatusKey string
}

type CreateNoteResult struct {
	Note model.CandidateNote
}

type CreateTagResult struct {
	Tag model.CandidateTag
}

type CreateFollowUpTaskResult struct {
	Task model.FollowUpTask
}

type CreateDepartmentResult struct {
	Department model.DepartmentNode
}

type UsageStatsResult struct {
	List []model.UsageStatsRow
}

type UsageTrendResult struct {
	List []model.UsageTrendRow
}
