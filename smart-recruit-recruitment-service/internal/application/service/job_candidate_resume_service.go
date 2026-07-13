package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"smart-recruit-recruitment-service/internal/application/command"
	"smart-recruit-recruitment-service/internal/application/dto"
	"smart-recruit-recruitment-service/internal/application/port"
	"smart-recruit-recruitment-service/internal/domain/model"
	"smart-recruit-recruitment-service/internal/domain/policy"
	"smart-recruit-recruitment-service/internal/domain/repository"
)

var (
	ErrJobRepositoryRequired     = errors.New("job repository is required")
	ErrJobScopeCheckerRequired   = errors.New("job scope checker is required")
	ErrJobForbidden              = errors.New("无权限操作该岗位")
	ErrResumeRepositoryRequired  = errors.New("resume repository is required")
	ErrStorageRequired           = errors.New("object storage is required")
	ErrUploadIDGeneratorRequired = errors.New("upload ID generator is required")
)

type JobService struct {
	jobs      repository.JobRepository
	scopes    repository.JobScopeChecker
	validator repository.DepartmentLocationValidator
}

func NewJobService(jobs repository.JobRepository, scopes repository.JobScopeChecker, validator repository.DepartmentLocationValidator) (*JobService, error) {
	if jobs == nil {
		return nil, ErrJobRepositoryRequired
	}
	if scopes == nil {
		return nil, ErrJobScopeCheckerRequired
	}
	return &JobService{jobs: jobs, scopes: scopes, validator: validator}, nil
}

func (s *JobService) CreateJob(ctx context.Context, cmd command.CreateJob) (dto.CreateJobResult, error) {
	job := model.Job{
		HRID:         cmd.HRID,
		Title:        cmd.Title,
		Department:   strings.TrimSpace(cmd.Department),
		Location:     strings.TrimSpace(cmd.Location),
		SalaryRange:  cmd.SalaryRange,
		Description:  cmd.Description,
		Requirements: cmd.Requirements,
	}
	if cmd.DepartmentID > 0 {
		department, err := s.jobs.LookupDepartment(ctx, cmd.DepartmentID)
		if err != nil {
			return dto.CreateJobResult{}, err
		}
		if department != nil {
			job.Department = department.FullName
		}
		job.DepartmentID = &cmd.DepartmentID
	}
	if cmd.LocationID > 0 {
		location, err := s.jobs.LookupLocation(ctx, cmd.LocationID)
		if err != nil {
			return dto.CreateJobResult{}, err
		}
		if location != nil {
			job.Location = location.Name
		}
		job.LocationID = &cmd.LocationID
	}
	if cmd.DepartmentID > 0 && cmd.LocationID > 0 && s.validator != nil {
		if err := s.validator.ValidateDepartmentLocation(ctx, cmd.DepartmentID, cmd.LocationID); err != nil {
			return dto.CreateJobResult{}, err
		}
	}
	built, err := policy.BuildJob(job)
	if err != nil {
		return dto.CreateJobResult{}, err
	}
	if err := s.jobs.Create(ctx, &built); err != nil {
		return dto.CreateJobResult{}, err
	}
	return dto.CreateJobResult{JobID: built.ID}, nil
}

func (s *JobService) SetJobStatus(ctx context.Context, cmd command.SetJobStatus) error {
	status := model.JobStatusOffline
	if cmd.Online {
		status = model.JobStatusOnline
	}
	scope, err := s.scopes.CheckJobScope(ctx, cmd.HRID, cmd.JobID)
	if err != nil {
		return err
	}
	var rows int64
	switch scope.Level {
	case repository.JobScopeFull:
		rows, err = s.jobs.SetStatusAny(ctx, cmd.JobID, status)
	case repository.JobScopeDepartmentOrLocation:
		rows, err = s.jobs.SetStatusInScope(ctx, cmd.JobID, scope.DepartmentIDs, scope.LocationIDs, status)
	case repository.JobScopeOwned:
		rows, err = s.jobs.SetStatusOwned(ctx, cmd.HRID, cmd.JobID, status)
	default:
		return ErrJobForbidden
	}
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrJobForbidden
	}
	return nil
}

type CandidateResumeDeps struct {
	Profiles  repository.CandidateProfileRepository
	Resumes   repository.ResumeRepository
	Storage   repository.ObjectStorage
	Outbox    repository.OutboxPublisher
	UsageLogs repository.UsageLogRepository
	UploadIDs port.UploadIDGenerator
	Clock     port.Clock
}

type CandidateResumeService struct {
	profiles  repository.CandidateProfileRepository
	resumes   repository.ResumeRepository
	storage   repository.ObjectStorage
	outbox    repository.OutboxPublisher
	usageLogs repository.UsageLogRepository
	uploadIDs port.UploadIDGenerator
	clock     port.Clock
}

func NewCandidateResumeService(deps CandidateResumeDeps) (*CandidateResumeService, error) {
	if deps.Resumes == nil {
		return nil, ErrResumeRepositoryRequired
	}
	if deps.Storage == nil {
		return nil, ErrStorageRequired
	}
	if deps.UploadIDs == nil {
		return nil, ErrUploadIDGeneratorRequired
	}
	clock := deps.Clock
	if clock == nil {
		clock = port.SystemClock{}
	}
	return &CandidateResumeService{
		profiles:  deps.Profiles,
		resumes:   deps.Resumes,
		storage:   deps.Storage,
		outbox:    deps.Outbox,
		usageLogs: deps.UsageLogs,
		uploadIDs: deps.UploadIDs,
		clock:     clock,
	}, nil
}

func (s *CandidateResumeService) UpdateProfile(ctx context.Context, cmd command.UpdateCandidateProfile) (dto.CandidateProfileResult, error) {
	profile := model.CandidateProfile{
		UserID:         cmd.UserID,
		RealName:       cmd.RealName,
		Phone:          cmd.Phone,
		Education:      cmd.Education,
		School:         cmd.School,
		WorkExperience: cmd.WorkExperience,
		Skills:         cmd.Skills,
	}
	policy.CompleteCandidateProfile(&profile)
	if s.profiles != nil {
		if err := s.profiles.Upsert(ctx, &profile); err != nil {
			return dto.CandidateProfileResult{}, err
		}
	}
	return dto.CandidateProfileResult{Profile: profile}, nil
}

func (s *CandidateResumeService) PresignResumeUpload(ctx context.Context, cmd command.PresignResumeUpload) (dto.PresignResumeUploadResult, error) {
	if err := policy.ValidateResumeFile(cmd.FileName, cmd.FileType); err != nil {
		return dto.PresignResumeUploadResult{}, err
	}
	uploadID, err := s.uploadIDs.NewUploadID()
	if err != nil {
		return dto.PresignResumeUploadResult{}, err
	}
	safeName := policy.SanitizeFileName(cmd.FileName)
	ossKey := fmt.Sprintf("resumes/tmp/%d/%s/%s", cmd.UserID, uploadID, safeName)
	contentType := policy.ContentTypeFromFileType(cmd.FileType)
	uploadURL, expireAt, err := s.storage.GeneratePresignedPutURL(ossKey, contentType)
	if err != nil {
		return dto.PresignResumeUploadResult{}, err
	}
	session := model.PresignSession{
		UserID:      cmd.UserID,
		OSSKey:      ossKey,
		FileName:    cmd.FileName,
		FileType:    cmd.FileType,
		ContentType: contentType,
		MaxSize:     model.MaxResumeSizeBytes,
		Status:      "pending",
	}
	if err := s.storage.SavePresignSession(ctx, uploadID, session); err != nil {
		return dto.PresignResumeUploadResult{}, err
	}
	s.writeUsageLog(ctx, model.UsageLogEntry{
		UserID:      cmd.UserID,
		Role:        model.CandidateLegacyRole,
		ServiceType: "oss_presign",
		Endpoint:    "/candidate/resume/presign",
		Provider:    s.storage.ProviderName(),
		ObjectKey:   ossKey,
		ObjectSize:  model.MaxResumeSizeBytes,
		Status:      "ok",
	})
	return dto.PresignResumeUploadResult{UploadURL: uploadURL, OSSKey: ossKey, ExpireAt: expireAt, UploadID: uploadID}, nil
}

func (s *CandidateResumeService) ConfirmResumeUpload(ctx context.Context, cmd command.ConfirmResumeUpload) (dto.ConfirmResumeUploadResult, error) {
	if err := policy.ValidateResumeFile(cmd.FileName, cmd.FileType); err != nil {
		return dto.ConfirmResumeUploadResult{}, err
	}
	if cmd.StrictSession {
		session, err := s.loadPresignSession(ctx, cmd.UploadID)
		if err != nil {
			return dto.ConfirmResumeUploadResult{}, err
		}
		if err := policy.ValidateStrictResumeSession(cmd.UserID, cmd.UploadID, cmd.OSSKey, cmd.FileType, cmd.FileSize, session); err != nil {
			return dto.ConfirmResumeUploadResult{}, err
		}
	} else if err := policy.ValidateLegacyResumeKey(cmd.UserID, cmd.OSSKey); err != nil {
		return dto.ConfirmResumeUploadResult{}, err
	}
	if err := s.storage.VerifyObject(ctx, cmd.OSSKey); err != nil {
		return dto.ConfirmResumeUploadResult{}, err
	}
	if err := s.storage.VerifyObjectSize(ctx, cmd.OSSKey, model.MaxResumeSizeBytes); err != nil {
		return dto.ConfirmResumeUploadResult{}, policy.ErrResumeFileTooLarge
	}
	permanentKey := fmt.Sprintf("resumes/%d/%d_%s", cmd.UserID, s.clock.Now().Unix(), policy.SanitizeFileName(cmd.FileName))
	if err := s.storage.CopyObject(ctx, cmd.OSSKey, permanentKey); err != nil {
		return dto.ConfirmResumeUploadResult{}, err
	}
	if err := s.storage.DeleteObject(ctx, cmd.OSSKey); err != nil {
		return dto.ConfirmResumeUploadResult{}, err
	}
	resume := &model.Resume{
		UserID:     cmd.UserID,
		OSSKey:     permanentKey,
		FileName:   cmd.FileName,
		FileType:   strings.ToLower(cmd.FileType),
		FileSize:   cmd.FileSize,
		IsValid:    1,
		UploadedAt: s.clock.Now(),
	}
	if err := s.resumes.ConfirmUpload(ctx, resume, func(resumeID int64) error {
		if s.outbox == nil {
			return nil
		}
		return s.outbox.WriteEvent(ctx, model.OutboxEvent{
			Topic:         "resume.parse",
			AggregateType: "resume",
			AggregateID:   uint64(resumeID),
			EventType:     "resume.parse",
			Payload: model.ResumeParsePayload{
				ResumeID: resumeID,
				FileType: resume.FileType,
				OSSKey:   resume.OSSKey,
			},
		})
	}); err != nil {
		return dto.ConfirmResumeUploadResult{}, err
	}
	if s.outbox != nil {
		s.outbox.Signal()
	}
	s.writeUsageLog(ctx, model.UsageLogEntry{
		UserID:      cmd.UserID,
		Role:        model.CandidateLegacyRole,
		ServiceType: "oss_confirm",
		Endpoint:    "/candidate/resume/confirm",
		Provider:    s.storage.ProviderName(),
		ObjectKey:   cmd.OSSKey,
		ObjectSize:  cmd.FileSize,
		Status:      "ok",
	})
	return dto.ConfirmResumeUploadResult{ResumeID: resume.ID}, nil
}

func (s *CandidateResumeService) loadPresignSession(ctx context.Context, uploadID string) (*model.PresignSession, error) {
	if uploadID == "" {
		return nil, nil
	}
	session, err := s.storage.GetAndDeletePresignSession(ctx, uploadID)
	if err != nil {
		return nil, policy.ErrResumeSessionInvalid
	}
	return session, nil
}

func (s *CandidateResumeService) writeUsageLog(ctx context.Context, entry model.UsageLogEntry) {
	if s.usageLogs == nil {
		return
	}
	if entry.Status == "" {
		entry.Status = "ok"
	}
	_ = s.usageLogs.CreateUsageLog(ctx, entry)
}
