package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"go.uber.org/zap"

	"logic-grpc-service/model"
	"logic-grpc-service/pkg/authz"
	"logic-grpc-service/pkg/errs"
	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/recruitment/pb"
	"logic-grpc-service/repository"
)

var ErrResumeProfileExtractorUnavailable = errors.New("resume profile extractor is not configured")

type unavailableResumeProfileExtractor struct{}

func (unavailableResumeProfileExtractor) Extract(context.Context, string) (string, error) {
	return "", ErrResumeProfileExtractorUnavailable
}

type RecruitingIntelligenceService struct {
	applications  *repository.ApplicationRepo
	jobs          *repository.JobRepo
	resumes       *repository.ResumeRepo
	resumeProfile *repository.ResumeProfileRepo
	matches       *repository.CandidateMatchRepo
	profileSvc    *ResumeProfileService
	matchSvc      *CandidateMatchService
	serviceAuth   *ServiceAuthorizer
}

func NewRecruitingIntelligenceService(
	applications *repository.ApplicationRepo,
	jobs *repository.JobRepo,
	resumes *repository.ResumeRepo,
	resumeProfile *repository.ResumeProfileRepo,
	matches *repository.CandidateMatchRepo,
	profileSvc *ResumeProfileService,
	matchSvc *CandidateMatchService,
	serviceAuth *ServiceAuthorizer,
) *RecruitingIntelligenceService {
	return &RecruitingIntelligenceService{
		applications:  applications,
		jobs:          jobs,
		resumes:       resumes,
		resumeProfile: resumeProfile,
		matches:       matches,
		profileSvc:    profileSvc,
		matchSvc:      matchSvc,
		serviceAuth:   serviceAuth,
	}
}

func (s *RecruitingIntelligenceService) GetResumeProfile(ctx context.Context, req *pb.GetResumeProfileRequest) (*pb.GetResumeProfileResponse, error) {
	started := time.Now()
	log := logger.GetRequestLogger(ctx)
	log.Info("[logic][recruiting_intelligence] GetResumeProfile started",
		zap.Int64("staff_user_id", req.GetStaffUserId()),
		zap.Int64("application_id", req.GetApplicationId()),
		zap.Int64("resume_id", req.GetResumeId()),
		zap.Uint64("profile_id", req.GetProfileId()),
	)
	if req.GetApplicationId() <= 0 && req.GetResumeId() <= 0 && req.GetProfileId() == 0 {
		log.Warn("[logic][recruiting_intelligence] GetResumeProfile validation failed: missing identifier")
		return &pb.GetResumeProfileResponse{Code: errs.ErrBadRequest, Msg: "resume_id, profile_id, or application_id is required"}, nil
	}
	if err := s.validateResumeIdentifierConsistency(ctx, req.GetResumeId(), req.GetProfileId(), req.GetApplicationId()); err != nil {
		log.Warn("[logic][recruiting_intelligence] GetResumeProfile consistency check failed", zap.Error(err))
		return &pb.GetResumeProfileResponse{Code: errs.ErrBadRequest, Msg: err.Error()}, nil
	}
	if err := s.authorizeApplicationRead(ctx, req.GetStaffUserId(), req.GetApplicationId(), req.GetResumeId(), req.GetProfileId()); err != nil {
		log.Warn("[logic][recruiting_intelligence] GetResumeProfile authorization failed", zap.Error(err))
		return &pb.GetResumeProfileResponse{Code: errs.ErrForbidden, Msg: err.Error()}, nil
	}
	snapshot, resp := s.resolveResumeProfile(ctx, req.GetResumeId(), req.GetProfileId(), req.GetApplicationId())
	if resp != nil {
		log.Warn("[logic][recruiting_intelligence] GetResumeProfile resolution failed", zap.String("msg", resp.GetMsg()))
		return resp, nil
	}
	log.Info("[logic][recruiting_intelligence] GetResumeProfile succeeded",
		zap.Int64("duration_ms", time.Since(started).Milliseconds()),
		zap.Uint64("profile_id", snapshot.Profile.ID),
		zap.Int32("version", snapshot.Profile.Version),
	)
	return &pb.GetResumeProfileResponse{Code: errs.OK, Msg: "success", Profile: resumeProfileSnapshotPB(snapshot)}, nil
}

func (s *RecruitingIntelligenceService) ParseResumeProfile(ctx context.Context, req *pb.ParseResumeProfileRequest) (*pb.GetResumeProfileResponse, error) {
	started := time.Now()
	log := logger.GetRequestLogger(ctx)
	log.Info("[logic][recruiting_intelligence] ParseResumeProfile started",
		zap.Int64("staff_user_id", req.GetStaffUserId()),
		zap.Int64("application_id", req.GetApplicationId()),
		zap.Int64("resume_id", req.GetResumeId()),
	)
	if req.GetApplicationId() <= 0 && req.GetResumeId() <= 0 {
		log.Warn("[logic][recruiting_intelligence] ParseResumeProfile validation failed: missing identifier")
		return &pb.GetResumeProfileResponse{Code: errs.ErrBadRequest, Msg: "resume_id or application_id is required"}, nil
	}
	if err := s.validateResumeIdentifierConsistency(ctx, req.GetResumeId(), 0, req.GetApplicationId()); err != nil {
		log.Warn("[logic][recruiting_intelligence] ParseResumeProfile consistency check failed", zap.Error(err))
		return &pb.GetResumeProfileResponse{Code: errs.ErrBadRequest, Msg: err.Error()}, nil
	}
	if err := s.authorizeApplicationRead(ctx, req.GetStaffUserId(), req.GetApplicationId(), req.GetResumeId(), 0); err != nil {
		log.Warn("[logic][recruiting_intelligence] ParseResumeProfile authz failed", zap.Error(err))
		return &pb.GetResumeProfileResponse{Code: errs.ErrForbidden, Msg: err.Error()}, nil
	}
	if err := s.authorizePermission(ctx, req.GetStaffUserId(), authz.PermAIHRUse); err != nil {
		log.Warn("[logic][recruiting_intelligence] ParseResumeProfile permission denied", zap.Error(err))
		return &pb.GetResumeProfileResponse{Code: errs.ErrForbidden, Msg: err.Error()}, nil
	}
	resumeID, err := s.resolveResumeID(ctx, req.GetResumeId(), req.GetApplicationId())
	if err != nil {
		log.Warn("[logic][recruiting_intelligence] ParseResumeProfile resolveResumeID failed", zap.Error(err))
		return &pb.GetResumeProfileResponse{Code: errs.ErrBadRequest, Msg: err.Error()}, nil
	}
	log.Info("[logic][recruiting_intelligence] ParseResumeProfile calling profileSvc.ParseResume",
		zap.Int64("resolved_resume_id", resumeID))
	snapshot, err := s.profileSvc.ParseResume(ctx, resumeID)
	if err != nil {
		log.Warn("[logic][recruiting_intelligence] ParseResumeProfile parse failed",
			zap.Int64("duration_ms", time.Since(started).Milliseconds()),
			zap.Error(err))
		return &pb.GetResumeProfileResponse{Code: businessCodeForResumeProfileError(err), Msg: err.Error()}, nil
	}
	log.Info("[logic][recruiting_intelligence] ParseResumeProfile succeeded",
		zap.Int64("duration_ms", time.Since(started).Milliseconds()),
		zap.Int64("resume_id", resumeID),
		zap.Uint64("profile_id", snapshot.Profile.ID),
	)
	return &pb.GetResumeProfileResponse{Code: errs.OK, Msg: "success", Profile: resumeProfileSnapshotPB(snapshot)}, nil
}

func (s *RecruitingIntelligenceService) EvaluateCandidateMatch(ctx context.Context, req *pb.EvaluateCandidateMatchRequest) (*pb.GetCandidateMatchEvaluationResponse, error) {
	started := time.Now()
	log := logger.GetRequestLogger(ctx)
	log.Info("[logic][recruiting_intelligence] EvaluateCandidateMatch started",
		zap.Int64("staff_user_id", req.GetStaffUserId()),
		zap.Int64("application_id", req.GetApplicationId()),
		zap.Uint64("agent_run_id", req.GetAgentRunId()),
	)
	if err := s.authorizeApplication(ctx, req.GetStaffUserId(), req.GetApplicationId(), authz.PermApplicationRead); err != nil {
		log.Warn("[logic][recruiting_intelligence] EvaluateCandidateMatch authz failed", zap.Error(err))
		return &pb.GetCandidateMatchEvaluationResponse{Code: errs.ErrForbidden, Msg: err.Error()}, nil
	}
	if err := s.authorizePermission(ctx, req.GetStaffUserId(), authz.PermAIHRUse); err != nil {
		log.Warn("[logic][recruiting_intelligence] EvaluateCandidateMatch permission denied", zap.Error(err))
		return &pb.GetCandidateMatchEvaluationResponse{Code: errs.ErrForbidden, Msg: err.Error()}, nil
	}
	var agentRunID *uint64
	if req.GetAgentRunId() > 0 {
		id := req.GetAgentRunId()
		agentRunID = &id
	}
	snapshot, err := s.matchSvc.EvaluateApplication(ctx, req.GetApplicationId(), agentRunID)
	if err != nil {
		log.Warn("[logic][recruiting_intelligence] EvaluateCandidateMatch failed",
			zap.Int64("duration_ms", time.Since(started).Milliseconds()),
			zap.Error(err))
		return &pb.GetCandidateMatchEvaluationResponse{Code: businessCodeForCandidateMatchError(err), Msg: err.Error()}, nil
	}
	log.Info("[logic][recruiting_intelligence] EvaluateCandidateMatch succeeded",
		zap.Int64("duration_ms", time.Since(started).Milliseconds()),
		zap.Uint64("evaluation_id", snapshot.Evaluation.ID),
		zap.Float64("overall_score", snapshot.Evaluation.OverallScore),
		zap.String("recommendation", snapshot.Evaluation.Recommendation),
	)
	return &pb.GetCandidateMatchEvaluationResponse{Code: errs.OK, Msg: "success", Evaluation: candidateMatchSnapshotPB(snapshot)}, nil
}

func (s *RecruitingIntelligenceService) GetCandidateMatchEvaluation(ctx context.Context, req *pb.GetCandidateMatchEvaluationRequest) (*pb.GetCandidateMatchEvaluationResponse, error) {
	started := time.Now()
	log := logger.GetRequestLogger(ctx)
	log.Info("[logic][recruiting_intelligence] GetCandidateMatchEvaluation started",
		zap.Int64("staff_user_id", req.GetStaffUserId()),
		zap.Int64("application_id", req.GetApplicationId()),
		zap.Uint64("evaluation_id", req.GetEvaluationId()),
		zap.Int32("evaluation_version", req.GetEvaluationVersion()),
	)
	if err := s.authorizeApplication(ctx, req.GetStaffUserId(), req.GetApplicationId(), authz.PermApplicationRead); err != nil {
		log.Warn("[logic][recruiting_intelligence] GetCandidateMatchEvaluation authz failed", zap.Error(err))
		return &pb.GetCandidateMatchEvaluationResponse{Code: errs.ErrForbidden, Msg: err.Error()}, nil
	}
	evaluation, err := s.resolveCandidateMatchEvaluation(ctx, req)
	if err != nil {
		log.Error("[logic][recruiting_intelligence] GetCandidateMatchEvaluation resolve failed", zap.Error(err))
		return nil, err
	}
	if evaluation == nil {
		log.Warn("[logic][recruiting_intelligence] GetCandidateMatchEvaluation not found",
			zap.Int64("duration_ms", time.Since(started).Milliseconds()))
		return &pb.GetCandidateMatchEvaluationResponse{Code: errs.ErrBadRequest, Msg: "candidate match evaluation not found"}, nil
	}
	snapshot, err := s.matches.GetSnapshot(ctx, evaluation.ID)
	if err != nil {
		log.Error("[logic][recruiting_intelligence] GetCandidateMatchEvaluation GetSnapshot failed", zap.Error(err))
		return nil, err
	}
	log.Info("[logic][recruiting_intelligence] GetCandidateMatchEvaluation succeeded",
		zap.Int64("duration_ms", time.Since(started).Milliseconds()),
		zap.Uint64("evaluation_id", snapshot.Evaluation.ID),
		zap.Int32("evaluation_version", snapshot.Evaluation.EvaluationVersion),
	)
	return &pb.GetCandidateMatchEvaluationResponse{Code: errs.OK, Msg: "success", Evaluation: candidateMatchSnapshotPB(snapshot)}, nil
}

func (s *RecruitingIntelligenceService) CompareCandidatesForJob(ctx context.Context, req *pb.CompareCandidatesForJobRequest) (*pb.CompareCandidatesForJobResponse, error) {
	started := time.Now()
	log := logger.GetRequestLogger(ctx)
	log.Info("[logic][recruiting_intelligence] CompareCandidatesForJob started",
		zap.Int64("staff_user_id", req.GetStaffUserId()),
		zap.Int64("job_id", req.GetJobId()),
	)
	if err := s.authorizeJob(ctx, req.GetStaffUserId(), req.GetJobId(), authz.PermApplicationRead); err != nil {
		log.Warn("[logic][recruiting_intelligence] CompareCandidatesForJob authz failed", zap.Error(err))
		return &pb.CompareCandidatesForJobResponse{Code: errs.ErrForbidden, Msg: err.Error(), JobId: req.GetJobId()}, nil
	}
	rows, err := s.applications.ListCurrentByJob(ctx, req.GetJobId())
	if err != nil {
		log.Error("[logic][recruiting_intelligence] CompareCandidatesForJob ListCurrentByJob failed", zap.Error(err))
		return nil, err
	}
	applicationIDs := make([]int64, 0, len(rows))
	for _, row := range rows {
		applicationIDs = append(applicationIDs, row.ApplicationID)
	}
	evaluations, err := s.matches.ListLatestByApplicationIDs(ctx, applicationIDs)
	if err != nil {
		log.Error("[logic][recruiting_intelligence] CompareCandidatesForJob ListLatestByApplicationIDs failed", zap.Error(err))
		return nil, err
	}
	byApplication := make(map[int64]model.CandidateMatchEvaluation, len(evaluations))
	for _, evaluation := range evaluations {
		byApplication[evaluation.ApplicationID] = evaluation
	}
	items := make([]*pb.CandidateComparisonItem, 0, len(rows))
	missing := make([]int64, 0)
	for _, row := range rows {
		item := &pb.CandidateComparisonItem{
			ApplicationId:   row.ApplicationID,
			CandidateUserId: row.UserID,
			CandidateName:   row.RealName,
			ResumeId:        row.ResumeID,
		}
		if evaluation, ok := byApplication[row.ApplicationID]; ok {
			item.EvaluationId = evaluation.ID
			item.EvaluationVersion = evaluation.EvaluationVersion
			item.OverallScore = evaluation.OverallScore
			item.Recommendation = evaluation.Recommendation
			item.Summary = evaluation.Summary
			item.EvaluatedAt = formatTime(evaluation.EvaluatedAt)
			item.HasEvaluation = true
		} else {
			missing = append(missing, row.ApplicationID)
		}
		items = append(items, item)
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].GetHasEvaluation() != items[j].GetHasEvaluation() {
			return items[i].GetHasEvaluation()
		}
		if items[i].GetOverallScore() != items[j].GetOverallScore() {
			return items[i].GetOverallScore() > items[j].GetOverallScore()
		}
		return items[i].GetApplicationId() < items[j].GetApplicationId()
	})
	log.Info("[logic][recruiting_intelligence] CompareCandidatesForJob succeeded",
		zap.Int64("duration_ms", time.Since(started).Milliseconds()),
		zap.Int64("job_id", req.GetJobId()),
		zap.Int("candidate_count", len(items)),
		zap.Int("missing_evaluation_count", len(missing)),
	)
	return &pb.CompareCandidatesForJobResponse{Code: errs.OK, Msg: "success", JobId: req.GetJobId(), Candidates: items, MissingApplicationIds: missing}, nil
}

func (s *RecruitingIntelligenceService) resolveResumeProfile(ctx context.Context, resumeID int64, profileID uint64, applicationID int64) (*repository.ResumeProfileSnapshot, *pb.GetResumeProfileResponse) {
	if err := s.validateResumeIdentifierConsistency(ctx, resumeID, profileID, applicationID); err != nil {
		return nil, &pb.GetResumeProfileResponse{Code: errs.ErrBadRequest, Msg: err.Error()}
	}
	if profileID > 0 {
		snapshot, err := s.resumeProfile.GetSnapshot(ctx, profileID)
		if err != nil {
			return nil, &pb.GetResumeProfileResponse{Code: errs.ErrBadRequest, Msg: "resume profile not found"}
		}
		return snapshot, nil
	}
	resolvedResumeID, err := s.resolveResumeID(ctx, resumeID, applicationID)
	if err != nil {
		return nil, &pb.GetResumeProfileResponse{Code: errs.ErrBadRequest, Msg: err.Error()}
	}
	profile, err := s.resumeProfile.GetCurrentByResumeID(ctx, resolvedResumeID)
	if err != nil {
		return nil, &pb.GetResumeProfileResponse{Code: errs.ErrInternal, Msg: err.Error()}
	}
	if profile == nil {
		return nil, &pb.GetResumeProfileResponse{Code: errs.ErrBadRequest, Msg: "current resume profile not found"}
	}
	snapshot, err := s.resumeProfile.GetSnapshot(ctx, profile.ID)
	if err != nil {
		return nil, &pb.GetResumeProfileResponse{Code: errs.ErrInternal, Msg: err.Error()}
	}
	return snapshot, nil
}

func (s *RecruitingIntelligenceService) resolveResumeID(ctx context.Context, resumeID int64, applicationID int64) (int64, error) {
	if applicationID > 0 {
		application, err := s.applications.GetByID(ctx, applicationID)
		if err != nil {
			return 0, err
		}
		if application == nil || application.ResumeID <= 0 {
			return 0, fmt.Errorf("application resume not found")
		}
		if resumeID > 0 && resumeID != application.ResumeID {
			return 0, fmt.Errorf("application_id and resume_id refer to different resumes")
		}
		return application.ResumeID, nil
	}
	if resumeID > 0 {
		return resumeID, nil
	}
	return 0, fmt.Errorf("resume_id or application_id is required")
}

func (s *RecruitingIntelligenceService) validateResumeIdentifierConsistency(ctx context.Context, resumeID int64, profileID uint64, applicationID int64) error {
	var application *model.Application
	if applicationID > 0 && (resumeID > 0 || profileID > 0) {
		var err error
		application, err = s.applications.GetByID(ctx, applicationID)
		if err != nil {
			return err
		}
		if application == nil || application.ResumeID <= 0 {
			return fmt.Errorf("application resume not found")
		}
		if resumeID > 0 && resumeID != application.ResumeID {
			return fmt.Errorf("application_id and resume_id refer to different resumes")
		}
	}

	if profileID > 0 && (resumeID > 0 || applicationID > 0) {
		profile, err := s.resumeProfile.GetByID(ctx, profileID)
		if err != nil {
			return err
		}
		if profile == nil {
			return fmt.Errorf("resume profile not found")
		}
		expectedResumeID := resumeID
		if expectedResumeID <= 0 && application != nil {
			expectedResumeID = application.ResumeID
		}
		if expectedResumeID > 0 && profile.ResumeID != expectedResumeID {
			return fmt.Errorf("profile_id does not belong to requested resume/application")
		}
	}

	return nil
}

func (s *RecruitingIntelligenceService) resolveCandidateMatchEvaluation(ctx context.Context, req *pb.GetCandidateMatchEvaluationRequest) (*model.CandidateMatchEvaluation, error) {
	if req.GetEvaluationId() > 0 {
		snapshot, err := s.matches.GetSnapshot(ctx, req.GetEvaluationId())
		if err != nil {
			return nil, nil
		}
		if snapshot.Evaluation.ApplicationID != req.GetApplicationId() {
			return nil, nil
		}
		return &snapshot.Evaluation, nil
	}
	if req.GetEvaluationVersion() > 0 {
		return s.matches.GetByApplicationIDAndVersion(ctx, req.GetApplicationId(), req.GetEvaluationVersion())
	}
	return s.matches.GetLatestByApplicationID(ctx, req.GetApplicationId())
}

func (s *RecruitingIntelligenceService) authorizeApplicationRead(ctx context.Context, staffUserID int64, applicationID int64, resumeID int64, profileID uint64) error {
	if applicationID > 0 {
		return s.authorizeApplication(ctx, staffUserID, applicationID, authz.PermApplicationRead)
	}
	var jobID int64
	switch {
	case profileID > 0:
		profile, err := s.resumeProfile.GetByID(ctx, profileID)
		if err != nil {
			return err
		}
		if profile == nil {
			return fmt.Errorf("resume profile not found")
		}
		resumeID = profile.ResumeID
	case resumeID <= 0:
		return fmt.Errorf("resume_id, profile_id, or application_id is required")
	}
	application, err := s.applications.GetLatestByResumeID(ctx, resumeID)
	if err != nil {
		return err
	}
	if application != nil {
		jobID = application.JobID
	}
	if application == nil || jobID <= 0 {
		return fmt.Errorf("application context not found for resume")
	}
	return s.authorizeJob(ctx, staffUserID, jobID, authz.PermApplicationRead)
}

func (s *RecruitingIntelligenceService) authorizeApplication(ctx context.Context, staffUserID int64, applicationID int64, permission string) error {
	if applicationID <= 0 {
		return fmt.Errorf("application_id is required")
	}
	application, err := s.applications.GetByID(ctx, applicationID)
	if err != nil {
		return err
	}
	if application == nil {
		return fmt.Errorf("application not found")
	}
	return s.authorizeJob(ctx, staffUserID, application.JobID, permission)
}

func (s *RecruitingIntelligenceService) authorizeJob(ctx context.Context, staffUserID int64, jobID int64, permission string) error {
	if err := s.authorizePermission(ctx, staffUserID, permission); err != nil {
		return err
	}
	_, err := s.serviceAuth.AuthorizeScope(ctx, uint64(staffUserID), func() (*jobScopeTarget, error) {
		job, err := s.jobs.GetByID(ctx, jobID)
		if err != nil {
			return nil, err
		}
		if job == nil {
			return nil, fmt.Errorf("job not found")
		}
		return &jobScopeTarget{ID: job.ID, HrID: job.HrID, DepartmentID: job.DepartmentID, LocationID: job.LocationID}, nil
	})
	return err
}

func (s *RecruitingIntelligenceService) authorizePermission(ctx context.Context, staffUserID int64, permission string) error {
	if staffUserID <= 0 {
		return fmt.Errorf("staff_user_id is required")
	}
	return s.serviceAuth.AuthorizePermission(ctx, uint64(staffUserID), permission)
}

func businessCodeForResumeProfileError(err error) int32 {
	switch {
	case errors.Is(err, ErrResumeProfileMissingResume), errors.Is(err, ErrResumeProfileMissingText), errors.Is(err, ErrResumeProfileInvalidOutput), errors.Is(err, ErrResumeProfileExtractorUnavailable):
		return errs.ErrBadRequest
	default:
		return errs.ErrInternal
	}
}

func businessCodeForCandidateMatchError(err error) int32 {
	switch {
	case errors.Is(err, ErrCandidateMatchMissingApplication), errors.Is(err, ErrCandidateMatchMissingJob), errors.Is(err, ErrCandidateMatchMissingResume), errors.Is(err, ErrCandidateMatchMissingResumeProfile), errors.Is(err, ErrCandidateMatchIncompleteProfile):
		return errs.ErrBadRequest
	default:
		return errs.ErrInternal
	}
}

func resumeProfileSnapshotPB(snapshot *repository.ResumeProfileSnapshot) *pb.ResumeProfileSnapshotInfo {
	if snapshot == nil {
		return nil
	}
	out := &pb.ResumeProfileSnapshotInfo{
		ParseRun:    resumeParseRunPB(snapshot.ParseRun),
		Profile:     resumeProfilePB(snapshot.Profile),
		Educations:  make([]*pb.ResumeEducationInfo, 0, len(snapshot.Educations)),
		Experiences: make([]*pb.ResumeExperienceInfo, 0, len(snapshot.Experiences)),
		Projects:    make([]*pb.ResumeProjectInfo, 0, len(snapshot.Projects)),
		Skills:      make([]*pb.ResumeSkillInfo, 0, len(snapshot.Skills)),
	}
	for _, row := range snapshot.Educations {
		out.Educations = append(out.Educations, resumeEducationPB(row))
	}
	for _, row := range snapshot.Experiences {
		out.Experiences = append(out.Experiences, resumeExperiencePB(row))
	}
	for _, row := range snapshot.Projects {
		out.Projects = append(out.Projects, resumeProjectPB(row))
	}
	for _, row := range snapshot.Skills {
		out.Skills = append(out.Skills, resumeSkillPB(row))
	}
	return out
}

func resumeParseRunPB(row model.ResumeParseRun) *pb.ResumeParseRunInfo {
	out := &pb.ResumeParseRunInfo{
		Id:            row.ID,
		ResumeId:      row.ResumeID,
		UserId:        row.UserID,
		Status:        row.Status,
		ParserVersion: row.ParserVersion,
		InputHash:     row.InputHash,
		ErrorMessage:  row.ErrorMessage,
		StartedAt:     formatTime(row.StartedAt),
		CreatedAt:     formatTime(row.CreatedAt),
		UpdatedAt:     formatTime(row.UpdatedAt),
	}
	if row.AgentRunID != nil {
		out.AgentRunId = *row.AgentRunID
	}
	if row.CompletedAt != nil {
		out.CompletedAt = formatTime(*row.CompletedAt)
	}
	return out
}

func resumeProfilePB(row model.ResumeProfile) *pb.ResumeProfileInfo {
	return &pb.ResumeProfileInfo{
		Id:                   row.ID,
		ResumeId:             row.ResumeID,
		UserId:               row.UserID,
		ParseRunId:           row.ParseRunID,
		Version:              row.Version,
		IsCurrent:            row.IsCurrent,
		FullName:             row.FullName,
		Email:                row.Email,
		Phone:                row.Phone,
		Location:             row.Location,
		Headline:             row.Headline,
		Summary:              row.Summary,
		TotalExperienceYears: row.TotalExperience,
		HighestDegree:        row.HighestDegree,
		RawJson:              row.RawJSON,
		CreatedAt:            formatTime(row.CreatedAt),
		UpdatedAt:            formatTime(row.UpdatedAt),
	}
}

func resumeEducationPB(row model.ResumeEducation) *pb.ResumeEducationInfo {
	out := &pb.ResumeEducationInfo{Id: row.ID, School: row.School, Degree: row.Degree, Major: row.Major, Description: row.Description, SortOrder: row.SortOrder}
	if row.StartDate != nil {
		out.StartDate = formatTime(*row.StartDate)
	}
	if row.EndDate != nil {
		out.EndDate = formatTime(*row.EndDate)
	}
	return out
}

func resumeExperiencePB(row model.ResumeExperience) *pb.ResumeExperienceInfo {
	out := &pb.ResumeExperienceInfo{Id: row.ID, Company: row.Company, Title: row.Title, Location: row.Location, IsCurrent: row.IsCurrent, Description: row.Description, AchievementsJson: row.AchievementsJSON, SortOrder: row.SortOrder}
	if row.StartDate != nil {
		out.StartDate = formatTime(*row.StartDate)
	}
	if row.EndDate != nil {
		out.EndDate = formatTime(*row.EndDate)
	}
	return out
}

func resumeProjectPB(row model.ResumeProject) *pb.ResumeProjectInfo {
	out := &pb.ResumeProjectInfo{Id: row.ID, Name: row.Name, Role: row.Role, Description: row.Description, TechnologiesJson: row.TechnologiesJSON, HighlightsJson: row.HighlightsJSON, SortOrder: row.SortOrder}
	if row.StartDate != nil {
		out.StartDate = formatTime(*row.StartDate)
	}
	if row.EndDate != nil {
		out.EndDate = formatTime(*row.EndDate)
	}
	return out
}

func resumeSkillPB(row model.ResumeSkill) *pb.ResumeSkillInfo {
	return &pb.ResumeSkillInfo{Id: row.ID, Name: row.Name, Category: row.Category, Level: row.Level, Years: row.Years, Evidence: row.Evidence, SortOrder: row.SortOrder}
}

func candidateMatchSnapshotPB(snapshot *repository.CandidateMatchSnapshot) *pb.CandidateMatchEvaluationSnapshotInfo {
	if snapshot == nil {
		return nil
	}
	out := &pb.CandidateMatchEvaluationSnapshotInfo{
		Evaluation: candidateMatchEvaluationPB(snapshot.Evaluation),
		Evidence:   make([]*pb.CandidateMatchEvidenceInfo, 0, len(snapshot.Evidence)),
	}
	for _, row := range snapshot.Evidence {
		out.Evidence = append(out.Evidence, candidateMatchEvidencePB(row))
	}
	return out
}

func candidateMatchEvaluationPB(row model.CandidateMatchEvaluation) *pb.CandidateMatchEvaluationInfo {
	out := &pb.CandidateMatchEvaluationInfo{
		Id:                      row.ID,
		ApplicationId:           row.ApplicationID,
		JobId:                   row.JobID,
		CandidateUserId:         row.CandidateUserID,
		ResumeProfileId:         row.ResumeProfileID,
		EvaluationVersion:       row.EvaluationVersion,
		IsLatest:                row.IsLatest,
		OverallScore:            row.OverallScore,
		Recommendation:          row.Recommendation,
		Summary:                 row.Summary,
		StrengthsJson:           row.StrengthsJSON,
		RisksJson:               row.RisksJSON,
		MissingRequirementsJson: missingRequirementsJSON(row.ScoreBreakdownJSON),
		DimensionsJson:          dimensionsJSON(row.ScoreBreakdownJSON),
		ScoreBreakdownJson:      row.ScoreBreakdownJSON,
		ModelName:               row.ModelName,
		EvaluatedAt:             formatTime(row.EvaluatedAt),
		CreatedAt:               formatTime(row.CreatedAt),
		UpdatedAt:               formatTime(row.UpdatedAt),
	}
	if row.AgentRunID != nil {
		out.AgentRunId = *row.AgentRunID
	}
	return out
}

func candidateMatchEvidencePB(row model.CandidateMatchEvidence) *pb.CandidateMatchEvidenceInfo {
	out := &pb.CandidateMatchEvidenceInfo{
		Id:           row.ID,
		EvidenceType: row.EvidenceType,
		Dimension:    row.Dimension,
		SourceTable:  row.SourceTable,
		Snippet:      row.Snippet,
		Weight:       row.Weight,
		ScoreImpact:  row.ScoreImpact,
		MetadataJson: row.MetadataJSON,
		CreatedAt:    formatTime(row.CreatedAt),
	}
	if row.SourceID != nil {
		out.SourceId = *row.SourceID
	}
	return out
}

func missingRequirementsJSON(scoreBreakdown string) string {
	var payload struct {
		MissingRequirements []string `json:"missing_requirements"`
		RequirementResults  []struct {
			RequirementID string `json:"requirement_id"`
			Status        string `json:"status"`
		} `json:"requirement_results"`
	}
	if err := json.Unmarshal([]byte(scoreBreakdown), &payload); err != nil {
		return "[]"
	}
	missing := payload.MissingRequirements
	if len(missing) == 0 {
		for _, result := range payload.RequirementResults {
			if result.Status == MatchStatusMissing || result.Status == MatchStatusConflict {
				missing = append(missing, result.RequirementID)
			}
		}
	}
	raw, err := json.Marshal(missing)
	if err != nil {
		return "[]"
	}
	return string(raw)
}

func dimensionsJSON(scoreBreakdown string) string {
	var payload struct {
		Dimensions []any `json:"dimensions"`
	}
	if err := json.Unmarshal([]byte(scoreBreakdown), &payload); err != nil || payload.Dimensions == nil {
		return "[]"
	}
	raw, err := json.Marshal(payload.Dimensions)
	if err != nil {
		return "[]"
	}
	return string(raw)
}
