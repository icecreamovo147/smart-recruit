package service

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"logic-grpc-service/model"
	"logic-grpc-service/pkg/errs"
	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/recruitment/pb"
	"logic-grpc-service/repository"
)

type InterviewService struct {
	authzRepo       *repository.AuthzRepo
	interviews      *repository.InterviewRepo
	applications    *repository.ApplicationRepo
	jobs            *repository.JobRepo
	notifications   *repository.NotificationRepo
	outboxPublisher *OutboxPublisher
	scopeEval       *scopeEvaluator
	serviceAuth     *ServiceAuthorizer
}

func NewInterviewService(
	authzRepo *repository.AuthzRepo,
	interviews *repository.InterviewRepo,
	applications *repository.ApplicationRepo,
	jobs *repository.JobRepo,
	notifications *repository.NotificationRepo,
	outboxPublisher *OutboxPublisher,
	scopeEval *scopeEvaluator,
	serviceAuth *ServiceAuthorizer,
) *InterviewService {
	return &InterviewService{
		authzRepo:       authzRepo,
		interviews:      interviews,
		applications:    applications,
		jobs:            jobs,
		notifications:   notifications,
		outboxPublisher: outboxPublisher,
		scopeEval:       scopeEval,
		serviceAuth:     serviceAuth,
	}
}

// ── Authorization helpers ─────────────────────────────────────────────

// checkInterviewScheduleScope verifies the user has interview.schedule permission
// and scope access to the application's job.
func (s *InterviewService) checkInterviewScheduleScope(ctx context.Context, userID int64, applicationID int64) error {
	if err := s.serviceAuth.AuthorizePermission(ctx, uint64(userID), "interview.schedule"); err != nil {
		return fmt.Errorf("permission denied: interview.schedule required: %w", err)
	}

	// Load the application to get the job ID for scope check.
	detail, err := s.applications.GetDetail(ctx, applicationID)
	if err != nil {
		return err
	}
	if detail == nil {
		return fmt.Errorf("application %d not found", applicationID)
	}

	// Evaluate scope against the application's job.
	_, err = s.scopeEval.evalScope(ctx, uint64(userID), func() (*jobScopeTarget, error) {
		job, err := s.jobs.GetByID(ctx, detail.JobID)
		if err != nil {
			return nil, err
		}
		if job == nil {
			return nil, fmt.Errorf("job %d not found", detail.JobID)
		}
		return &jobScopeTarget{
			ID:           job.ID,
			HrID:         job.HrID,
			DepartmentID: job.DepartmentID,
			LocationID:   job.LocationID,
		}, nil
	})
	if err != nil {
		return fmt.Errorf("scope denied for application %d: %w", applicationID, err)
	}
	return nil
}

// checkInterviewReadScope verifies the user can view a specific interview.
// For interviewers, this checks the assigned_interviews scope.
// For recruiters/admins, this checks job scope.
func (s *InterviewService) checkInterviewReadScope(ctx context.Context, userID int64, interviewID int64) error {
	// First check if user has interview.read permission
	if err := s.serviceAuth.AuthorizePermission(ctx, uint64(userID), "interview.read"); err != nil {
		return fmt.Errorf("permission denied: interview.read required: %w", err)
	}

	// Get the interview with details to check scope
	detail, err := s.interviews.GetByID(ctx, interviewID)
	if err != nil {
		return err
	}
	if detail == nil {
		return fmt.Errorf("interview %d not found", interviewID)
	}

	// Check if the user is the assigned interviewer (assigned_interviews scope)
	if detail.InterviewerID == userID {
		return nil
	}

	// For non-interviewer access, check job-level scope via the application
	scopeKeys, err := s.authzRepo.GetUserScopeKeys(ctx, uint64(userID))
	if err != nil {
		return err
	}

	for _, sk := range scopeKeys {
		if sk == "recruiting_all" || sk == "system_all" {
			return nil
		}
	}

	// Check own_jobs, department, location scopes against the application's job
	// Resolve the job from the application
	appDetail, err := s.applications.GetDetail(ctx, detail.ApplicationID)
	if err != nil {
		return err
	}
	if appDetail == nil {
		return fmt.Errorf("application %d not found", detail.ApplicationID)
	}

	hasOwnJobs := false
	for _, sk := range scopeKeys {
		if sk == "own_jobs" {
			hasOwnJobs = true
			break
		}
	}

	if hasOwnJobs && appDetail.JobID > 0 {
		// Verify the HR actually owns this job (jobs.hr_id == userID)
		belongs, err := s.jobs.BelongsToHR(ctx, userID, appDetail.JobID)
		if err != nil {
			return err
		}
		if belongs {
			return nil
		}
	}

	return fmt.Errorf("access denied to interview %d", interviewID)
}

// checkInterviewerAssignment checks that the user is the assigned interviewer for an interview.
func (s *InterviewService) checkInterviewerAssignment(ctx context.Context, userID int64, interviewID int64) error {
	detail, err := s.interviews.GetByID(ctx, interviewID)
	if err != nil {
		return err
	}
	if detail == nil {
		return fmt.Errorf("interview %d not found", interviewID)
	}
	if detail.InterviewerID != userID {
		return fmt.Errorf("user %d is not the assigned interviewer for interview %d", userID, interviewID)
	}
	return nil
}

// ── Service methods ───────────────────────────────────────────────────

func (s *InterviewService) ScheduleInterview(ctx context.Context, req *pb.ScheduleInterviewRequest) (*pb.ScheduleInterviewResponse, error) {
	// Verify actor
	if err := s.serviceAuth.VerifyActorMatch(ctx, req.HrId); err != nil {
		return nil, err
	}

	// Permission + scope check
	if err := s.checkInterviewScheduleScope(ctx, req.HrId, req.ApplicationId); err != nil {
		return &pb.ScheduleInterviewResponse{Code: errs.ErrForbidden, Msg: err.Error()}, nil
	}

	// Parse scheduled_at
	var scheduledAt *time.Time
	if req.ScheduledAt != "" {
		t, err := time.Parse(time.RFC3339, req.ScheduledAt)
		if err != nil {
			return &pb.ScheduleInterviewResponse{Code: errs.ErrBadRequest, Msg: "面试时间格式错误，请使用 RFC 3339 格式"}, nil
		}
		scheduledAt = &t
	}

	// Load application detail for job title / candidate name
	appDetail, err := s.applications.GetDetail(ctx, req.ApplicationId)
	if err != nil {
		return nil, err
	}
	if appDetail == nil {
		return &pb.ScheduleInterviewResponse{Code: errs.ErrBadRequest, Msg: "投递记录不存在"}, nil
	}

	// Build defaults
	title := req.Title
	if title == "" {
		if req.RoundNo > 0 {
			title = fmt.Sprintf("第 %d 轮面试", req.RoundNo)
		} else {
			title = "面试"
		}
	}

	mode := req.Mode
	if mode == "" {
		mode = "video"
	}

	interview := &model.InterviewSchedule{
		ApplicationID:   req.ApplicationId,
		InterviewerID:   req.InterviewerId,
		RoundNo:         req.RoundNo,
		Title:           title,
		Mode:            mode,
		MeetingURL:      req.MeetingUrl,
		Location:        req.Location,
		DurationMinutes: req.DurationMinutes,
		CandidateNote:   req.CandidateNote,
		InternalNote:    req.InternalNote,
		ScheduledAt:     scheduledAt,
		Status:          "scheduled",
		CreatedBy:       &req.HrId,
	}

	// Transaction: create interview + notification outbox
	err = s.interviews.Transaction(ctx, func(tx *gorm.DB) error {
		if err := s.interviews.CreateWithTx(ctx, tx, interview); err != nil {
			return err
		}

		// Notify the interviewer
		if err := s.outboxPublisher.WriteEventTx(tx, "notification.create", "interview", uint64(interview.ID), "notification.create", notificationPayload{
			ReceiverID:          req.InterviewerId,
			ReceiverRole:        2,
			ReceiverAccountType: "staff",
			Type:                "interview_assigned",
			Title:               "新的面试安排",
			Content:             fmt.Sprintf("您被安排为「%s」岗位的面试官：%s", appDetail.JobTitle, title),
			Link:                "/hr/interviews",
			BizType:             "interview",
			BizID:               interview.ID,
		}); err != nil {
			return err
		}

		// Notify the candidate
		if err := s.outboxPublisher.WriteEventTx(tx, "notification.create", "interview", uint64(interview.ID), "notification.create", notificationPayload{
			ReceiverID:          appDetail.UserID,
			ReceiverRole:        1,
			ReceiverAccountType: "candidate",
			Type:                "interview_scheduled",
			Title:               "面试安排通知",
			Content:             fmt.Sprintf("您的「%s」岗位面试已安排：%s", appDetail.JobTitle, title),
			Link:                "/applications",
			BizType:             "interview",
			BizID:               interview.ID,
		}); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		logger.L().Error("schedule interview failed",
			zap.Int64("application_id", req.ApplicationId),
			zap.Error(err),
		)
		return nil, err
	}
	s.outboxPublisher.Signal()

	logger.L().Info("interview scheduled",
		zap.Int64("interview_id", interview.ID),
		zap.Int64("application_id", req.ApplicationId),
		zap.Int64("interviewer_id", req.InterviewerId),
	)

	return &pb.ScheduleInterviewResponse{Code: errs.OK, Msg: "面试安排成功", InterviewId: interview.ID}, nil
}

func (s *InterviewService) UpdateInterview(ctx context.Context, req *pb.UpdateInterviewRequest) (*pb.CommonResponse, error) {
	if err := s.serviceAuth.VerifyActorMatch(ctx, req.HrId); err != nil {
		return nil, err
	}

	// Load existing interview
	existing, err := s.interviews.GetModelByID(ctx, req.InterviewId)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "面试记录不存在"}, nil
	}

	// Permission + scope check via the application
	if err := s.checkInterviewScheduleScope(ctx, req.HrId, existing.ApplicationID); err != nil {
		return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: err.Error()}, nil
	}

	// Parse scheduled_at
	if req.ScheduledAt != "" {
		t, err := time.Parse(time.RFC3339, req.ScheduledAt)
		if err != nil {
			return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "面试时间格式错误，请使用 RFC 3339 格式"}, nil
		}
		existing.ScheduledAt = &t
	}

	// Update fields
	if req.Title != "" {
		existing.Title = req.Title
	}
	if req.Mode != "" {
		existing.Mode = req.Mode
	}
	if req.MeetingUrl != "" {
		existing.MeetingURL = req.MeetingUrl
	}
	if req.Location != "" {
		existing.Location = req.Location
	}
	if req.DurationMinutes > 0 {
		existing.DurationMinutes = req.DurationMinutes
	}
	if req.CandidateNote != "" {
		existing.CandidateNote = req.CandidateNote
	}
	if req.InternalNote != "" {
		existing.InternalNote = req.InternalNote
	}

	// Transaction: update + notifications
	err = s.interviews.Transaction(ctx, func(tx *gorm.DB) error {
		if err := s.interviews.UpdateWithTx(ctx, tx, existing); err != nil {
			return err
		}

		// Load application to get candidate info
		appDetail, err := s.applications.GetDetail(ctx, existing.ApplicationID)
		if err != nil {
			return err
		}
		if appDetail == nil {
			return fmt.Errorf("application %d not found", existing.ApplicationID)
		}

		// Notify interviewer
		if err := s.outboxPublisher.WriteEventTx(tx, "notification.create", "interview", uint64(existing.ID), "notification.create", notificationPayload{
			ReceiverID:          existing.InterviewerID,
			ReceiverRole:        2,
			ReceiverAccountType: "staff",
			Type:                "interview_updated",
			Title:               "面试信息已更新",
			Content:             fmt.Sprintf("「%s」岗位的面试安排已更新：%s", appDetail.JobTitle, existing.Title),
			Link:                "/hr/interviews",
			BizType:             "interview",
			BizID:               existing.ID,
		}); err != nil {
			return err
		}

		// Notify candidate
		if err := s.outboxPublisher.WriteEventTx(tx, "notification.create", "interview", uint64(existing.ID), "notification.create", notificationPayload{
			ReceiverID:          appDetail.UserID,
			ReceiverRole:        1,
			ReceiverAccountType: "candidate",
			Type:                "interview_updated",
			Title:               "面试时间变更通知",
			Content:             fmt.Sprintf("您的「%s」岗位面试信息已更新，请查看最新安排", appDetail.JobTitle),
			Link:                "/applications",
			BizType:             "interview",
			BizID:               existing.ID,
		}); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		logger.L().Error("update interview failed",
			zap.Int64("interview_id", req.InterviewId),
			zap.Error(err),
		)
		return nil, err
	}
	s.outboxPublisher.Signal()

	return &pb.CommonResponse{Code: errs.OK, Msg: "面试信息已更新"}, nil
}

func (s *InterviewService) CancelInterview(ctx context.Context, req *pb.CancelInterviewRequest) (*pb.CommonResponse, error) {
	if err := s.serviceAuth.VerifyActorMatch(ctx, req.HrId); err != nil {
		return nil, err
	}

	existing, err := s.interviews.GetModelByID(ctx, req.InterviewId)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "面试记录不存在"}, nil
	}

	if existing.Status == "cancelled" {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "该面试已取消"}, nil
	}

	// Permission + scope check
	if err := s.checkInterviewScheduleScope(ctx, req.HrId, existing.ApplicationID); err != nil {
		return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: err.Error()}, nil
	}

	existing.Status = "cancelled"
	existing.CancelReason = req.CancelReason

	err = s.interviews.Transaction(ctx, func(tx *gorm.DB) error {
		if err := s.interviews.UpdateWithTx(ctx, tx, existing); err != nil {
			return err
		}

		appDetail, err := s.applications.GetDetail(ctx, existing.ApplicationID)
		if err != nil {
			return err
		}
		if appDetail == nil {
			return fmt.Errorf("application %d not found", existing.ApplicationID)
		}

		reasonText := req.CancelReason
		if reasonText == "" {
			reasonText = "暂无说明"
		}

		// Notify interviewer
		if err := s.outboxPublisher.WriteEventTx(tx, "notification.create", "interview", uint64(existing.ID), "notification.create", notificationPayload{
			ReceiverID:          existing.InterviewerID,
			ReceiverRole:        2,
			ReceiverAccountType: "staff",
			Type:                "interview_cancelled",
			Title:               "面试已取消",
			Content:             fmt.Sprintf("「%s」岗位的面试已取消。原因：%s", appDetail.JobTitle, reasonText),
			Link:                "/hr/interviews",
			BizType:             "interview",
			BizID:               existing.ID,
		}); err != nil {
			return err
		}

		// Notify candidate
		if err := s.outboxPublisher.WriteEventTx(tx, "notification.create", "interview", uint64(existing.ID), "notification.create", notificationPayload{
			ReceiverID:          appDetail.UserID,
			ReceiverRole:        1,
			ReceiverAccountType: "candidate",
			Type:                "interview_cancelled",
			Title:               "面试已取消",
			Content:             fmt.Sprintf("您的「%s」岗位面试已取消。原因：%s", appDetail.JobTitle, reasonText),
			Link:                "/applications",
			BizType:             "interview",
			BizID:               existing.ID,
		}); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		logger.L().Error("cancel interview failed",
			zap.Int64("interview_id", req.InterviewId),
			zap.Error(err),
		)
		return nil, err
	}
	s.outboxPublisher.Signal()

	logger.L().Info("interview cancelled",
		zap.Int64("interview_id", existing.ID),
		zap.Int64("application_id", existing.ApplicationID),
	)

	return &pb.CommonResponse{Code: errs.OK, Msg: "面试已取消"}, nil
}

func (s *InterviewService) GetInterview(ctx context.Context, req *pb.GetInterviewRequest) (*pb.GetInterviewResponse, error) {
	if err := s.serviceAuth.VerifyActorMatch(ctx, req.UserId); err != nil {
		return nil, err
	}

	detail, err := s.interviews.GetByID(ctx, req.InterviewId)
	if err != nil {
		return nil, err
	}
	if detail == nil {
		return &pb.GetInterviewResponse{Code: errs.ErrBadRequest, Msg: "面试记录不存在"}, nil
	}

	// Check if user is the candidate (candidate access is direct, no scope check)
	appDetail, _ := s.applications.GetDetail(ctx, detail.ApplicationID)
	if appDetail != nil && appDetail.UserID == req.UserId {
		// Candidate — allowed to view their own interview; filter internal_note
		detail.InternalNote = ""
	} else {
		// For all other users (interviewers not assigned to this interview, staff with
		// interview.read but no scope, etc.), use checkInterviewReadScope which enforces:
		//   - interview.read permission
		//   - interviewer assignment (detail.InterviewerID == userID)
		//   - recruiting_all / system_all scope
		//   - own_jobs scope
		if err := s.checkInterviewReadScope(ctx, req.UserId, req.InterviewId); err != nil {
			return &pb.GetInterviewResponse{Code: errs.ErrForbidden, Msg: "无权限查看该面试"}, nil
		}
	}

	return &pb.GetInterviewResponse{
		Code: errs.OK,
		Msg:  "success",
		Interview: toPBInterview(detail),
	}, nil
}

func (s *InterviewService) ListApplicationInterviews(ctx context.Context, req *pb.ListApplicationInterviewsRequest) (*pb.ListApplicationInterviewsResponse, error) {
	if err := s.serviceAuth.VerifyActorMatch(ctx, req.HrId); err != nil {
		return nil, err
	}

	// Check permission + scope
	if err := s.checkInterviewScheduleScope(ctx, req.HrId, req.ApplicationId); err != nil {
		return &pb.ListApplicationInterviewsResponse{Code: errs.ErrForbidden, Msg: err.Error()}, nil
	}

	rows, err := s.interviews.ListByApplication(ctx, req.ApplicationId)
	if err != nil {
		return nil, err
	}

	list := make([]*pb.InterviewSchedule, 0, len(rows))
	for _, row := range rows {
		list = append(list, toPBInterview(&row))
	}

	return &pb.ListApplicationInterviewsResponse{Code: errs.OK, Msg: "success", List: list}, nil
}

func (s *InterviewService) ListMyInterviews(ctx context.Context, req *pb.ListMyInterviewsRequest) (*pb.ListMyInterviewsResponse, error) {
	if err := s.serviceAuth.VerifyActorMatch(ctx, req.InterviewerId); err != nil {
		return nil, err
	}

	// Check interview.read permission
	if err := s.serviceAuth.AuthorizePermission(ctx, uint64(req.InterviewerId), "interview.read"); err != nil {
		return &pb.ListMyInterviewsResponse{Code: errs.ErrForbidden, Msg: "无权限查看面试列表"}, nil
	}

	rows, err := s.interviews.ListByInterviewer(ctx, req.InterviewerId, req.Status)
	if err != nil {
		return nil, err
	}

	list := make([]*pb.InterviewSchedule, 0, len(rows))
	for _, row := range rows {
		list = append(list, toPBInterview(&row))
	}

	return &pb.ListMyInterviewsResponse{Code: errs.OK, Msg: "success", List: list}, nil
}

func (s *InterviewService) ListCandidateInterviews(ctx context.Context, req *pb.ListCandidateInterviewsRequest) (*pb.ListCandidateInterviewsResponse, error) {
	if err := s.serviceAuth.VerifyActorMatch(ctx, req.UserId); err != nil {
		return nil, err
	}

	rows, err := s.interviews.ListByCandidate(ctx, req.UserId)
	if err != nil {
		return nil, err
	}

	list := make([]*pb.InterviewSchedule, 0, len(rows))
	for _, row := range rows {
		// Remove internal notes for candidate-facing display
		row.InternalNote = ""
		list = append(list, toPBInterview(&row))
	}

	return &pb.ListCandidateInterviewsResponse{Code: errs.OK, Msg: "success", List: list}, nil
}

func (s *InterviewService) SubmitFeedback(ctx context.Context, req *pb.SubmitFeedbackRequest) (*pb.CommonResponse, error) {
	if err := s.serviceAuth.VerifyActorMatch(ctx, req.InterviewerId); err != nil {
		return nil, err
	}

	// Check feedback.submit permission
	if err := s.serviceAuth.AuthorizePermission(ctx, uint64(req.InterviewerId), "interview.feedback.submit"); err != nil {
		return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: err.Error()}, nil
	}

	// Check interviewer assignment
	if err := s.checkInterviewerAssignment(ctx, req.InterviewerId, req.InterviewId); err != nil {
		return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: "您不是该面试的面试官，无法提交反馈"}, nil
	}

	// Validate that the ApplicationId matches the interview's actual ApplicationID
	interviewDetail, err := s.interviews.GetByID(ctx, req.InterviewId)
	if err != nil {
		return nil, err
	}
	if interviewDetail == nil {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "面试记录不存在"}, nil
	}
	if req.ApplicationId != interviewDetail.ApplicationID {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "ApplicationId 与面试记录不匹配"}, nil
	}

	// Check if feedback already exists (immutability)
	exists, err := s.interviews.FeedbackExistsByInterviewer(ctx, req.InterviewId, req.InterviewerId)
	if err != nil {
		return nil, err
	}
	if exists {
		return &pb.CommonResponse{Code: errs.ErrConflict, Msg: "您已提交过面试反馈，不可重复提交（如有更正需求请联系 HR）"}, nil
	}

	// Validate
	if req.Recommendation == "" {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "请选择面试推荐结论"}, nil
	}
	if req.Score < 0 || req.Score > 10 {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "评分范围为 0-10"}, nil
	}
	validRecs := map[string]bool{"positive": true, "negative": true, "pending": true}
	if !validRecs[req.Recommendation] {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "推荐结论 must be positive/negative/pending"}, nil
	}

	feedback := &model.InterviewFeedback{
		InterviewID:         req.InterviewId,
		ApplicationID:       req.ApplicationId,
		InterviewerID:       req.InterviewerId,
		Recommendation:      req.Recommendation,
		Score:               req.Score,
		DimensionScoresJSON: req.DimensionScoresJson,
		Comments:            req.Comments,
		SubmittedAt:         time.Now(),
	}

	if err := s.interviews.CreateFeedback(ctx, feedback); err != nil {
		logger.L().Error("submit feedback failed",
			zap.Int64("interview_id", req.InterviewId),
			zap.Error(err),
		)
		return nil, err
	}

	logger.L().Info("interview feedback submitted",
		zap.Int64("feedback_id", feedback.ID),
		zap.Int64("interview_id", req.InterviewId),
		zap.Int64("interviewer_id", req.InterviewerId),
	)

	// After feedback submission, mark interview as completed
	interviewModel, err := s.interviews.GetModelByID(ctx, req.InterviewId)
	if err != nil {
		return nil, err
	}
	if interviewModel != nil && interviewModel.Status == "scheduled" {
		interviewModel.Status = "completed"
		if err := s.interviews.Update(ctx, interviewModel); err != nil {
			logger.L().Error("update interview status to completed failed", zap.Error(err))
			// Non-fatal: feedback was still saved
		}
	}

	return &pb.CommonResponse{Code: errs.OK, Msg: "面试反馈已提交"}, nil
}

func (s *InterviewService) GetFeedback(ctx context.Context, req *pb.GetFeedbackRequest) (*pb.GetFeedbackResponse, error) {
	if err := s.serviceAuth.VerifyActorMatch(ctx, req.InterviewerId); err != nil {
		return nil, err
	}

	feedback, err := s.interviews.GetFeedbackByInterviewAndInterviewer(ctx, req.InterviewId, req.InterviewerId)
	if err != nil {
		return nil, err
	}
	if feedback == nil {
		return &pb.GetFeedbackResponse{Code: errs.OK, Msg: "success", Feedback: nil}, nil
	}

	return &pb.GetFeedbackResponse{
		Code: errs.OK,
		Msg:  "success",
		Feedback: &pb.InterviewFeedback{
			FeedbackId:          feedback.ID,
			InterviewId:         feedback.InterviewID,
			ApplicationId:       feedback.ApplicationID,
			InterviewerId:       feedback.InterviewerID,
			Recommendation:      feedback.Recommendation,
			Score:               feedback.Score,
			DimensionScoresJson: feedback.DimensionScoresJSON,
			Comments:            feedback.Comments,
			SubmittedAt:         formatTime(feedback.SubmittedAt),
			UpdatedAt:           formatTime(feedback.UpdatedAt),
		},
	}, nil
}

// ── PB Conversion helpers ─────────────────────────────────────────────

func toPBInterview(row *repository.InterviewWithDetailsRow) *pb.InterviewSchedule {
	var scheduledAt string
	if row.ScheduledAt != nil {
		scheduledAt = row.ScheduledAt.Format(time.RFC3339)
	}
	return &pb.InterviewSchedule{
		InterviewId:         row.ID,
		ApplicationId:       row.ApplicationID,
		InterviewerId:       row.InterviewerID,
		RoundNo:             row.RoundNo,
		Title:               row.Title,
		Mode:                row.Mode,
		MeetingUrl:          row.MeetingURL,
		Location:            row.Location,
		DurationMinutes:     row.DurationMinutes,
		CandidateNote:       row.CandidateNote,
		InternalNote:        row.InternalNote,
		CancelReason:        row.CancelReason,
		ScheduledAt:         scheduledAt,
		Status:              row.Status,
		CreatedBy:           int64PtrToInt64(row.CreatedBy),
		CreatedAt:           formatTime(row.CreatedAt),
		UpdatedAt:           formatTime(row.UpdatedAt),
		InterviewerName:     row.InterviewerName,
		ApplicationStatusKey: row.ApplicationStatusKey,
		JobTitle:            row.JobTitle,
		CandidateName:       row.CandidateName,
		CandidatePhone:      row.CandidatePhone,
	}
}

func int64PtrToInt64(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}
