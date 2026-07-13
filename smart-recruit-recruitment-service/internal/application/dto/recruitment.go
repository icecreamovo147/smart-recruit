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
