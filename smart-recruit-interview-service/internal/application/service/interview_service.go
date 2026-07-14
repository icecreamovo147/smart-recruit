package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"smart-recruit-interview-service/internal/application/command"
	"smart-recruit-interview-service/internal/application/port"
	"smart-recruit-interview-service/internal/application/query"
	"smart-recruit-interview-service/internal/domain/model"
	"smart-recruit-interview-service/internal/domain/repository"
	domainservice "smart-recruit-interview-service/internal/domain/service"
)

var (
	ErrApplicationNotFound = errors.New("投递记录不存在")
	ErrInterviewNotFound   = errors.New("面试记录不存在")
)

type InterviewService struct {
	interviews   repository.InterviewRepository
	applications port.ApplicationSnapshotReader
	staff        port.StaffDirectory
	lifecycle    port.ApplicationLifecycle
	outbox       port.OutboxPublisher
	authorizer   port.Authorizer
	resumeURLs   port.ResumeURLSigner
	clock        port.Clock
}

type Deps struct {
	Interviews   repository.InterviewRepository
	Applications port.ApplicationSnapshotReader
	Staff        port.StaffDirectory
	Lifecycle    port.ApplicationLifecycle
	Outbox       port.OutboxPublisher
	Authorizer   port.Authorizer
	ResumeURLs   port.ResumeURLSigner
	Clock        port.Clock
}

func NewInterviewService(deps Deps) (*InterviewService, error) {
	if deps.Interviews == nil {
		return nil, errors.New("interview repository is required")
	}
	if deps.Applications == nil {
		return nil, errors.New("application snapshot reader is required")
	}
	if deps.Staff == nil {
		return nil, errors.New("staff directory is required")
	}
	if deps.Lifecycle == nil {
		return nil, errors.New("application lifecycle port is required")
	}
	if deps.Outbox == nil {
		return nil, errors.New("outbox publisher is required")
	}
	if deps.Authorizer == nil {
		return nil, errors.New("authorizer is required")
	}
	clock := deps.Clock
	if clock == nil {
		clock = port.SystemClock{}
	}
	return &InterviewService{
		interviews:   deps.Interviews,
		applications: deps.Applications,
		staff:        deps.Staff,
		lifecycle:    deps.Lifecycle,
		outbox:       deps.Outbox,
		authorizer:   deps.Authorizer,
		resumeURLs:   deps.ResumeURLs,
		clock:        clock,
	}, nil
}

func (s *InterviewService) ScheduleInterview(ctx context.Context, cmd command.ScheduleInterview) (int64, error) {
	if err := s.authorizer.VerifyActor(ctx, cmd.HRID); err != nil {
		return 0, err
	}
	if err := s.authorizer.Authorize(ctx, cmd.HRID, port.PermissionInterviewSchedule); err != nil {
		return 0, err
	}
	if err := s.authorizer.CanScheduleApplication(ctx, cmd.HRID, cmd.ApplicationID); err != nil {
		return 0, err
	}
	snapshot, err := s.applicationSnapshot(ctx, cmd.ApplicationID)
	if err != nil {
		return 0, err
	}
	roundNo := cmd.RoundNo
	if roundNo <= 0 {
		maxRound, err := s.interviews.MaxRoundNo(ctx, cmd.ApplicationID)
		if err != nil {
			return 0, err
		}
		roundNo = maxRound + 1
	}
	interview, err := model.NewScheduledInterview(model.ScheduleDetails{
		ApplicationID:   cmd.ApplicationID,
		InterviewerID:   cmd.InterviewerID,
		RoundNo:         roundNo,
		Title:           cmd.Title,
		Mode:            cmd.Mode,
		MeetingURL:      cmd.MeetingURL,
		Location:        cmd.Location,
		DurationMinutes: cmd.DurationMinutes,
		CandidateNote:   cmd.CandidateNote,
		InternalNote:    cmd.InternalNote,
		ScheduledAt:     cmd.ScheduledAt,
		CreatedBy:       cmd.HRID,
	})
	if err != nil {
		return 0, err
	}
	transition, needsTransition, err := domainservice.ScheduleTransition(snapshot.StatusKey)
	if err != nil {
		return 0, err
	}
	err = s.interviews.Transaction(ctx, func(txCtx context.Context, writer repository.InterviewWriter) error {
		if err := writer.Create(txCtx, interview); err != nil {
			return err
		}
		if needsTransition {
			if _, err := s.lifecycle.ApplyTransition(txCtx, port.LifecycleTransitionCommand{
				ApplicationID:    cmd.ApplicationID,
				FromStatus:       transition.From,
				ToStatus:         transition.To,
				ActorUserID:      cmd.HRID,
				ActorAccountType: "staff",
				Reason:           fmt.Sprintf("安排第 %d 轮面试自动推进", roundNo),
			}); err != nil {
				return err
			}
		}
		return s.publishScheduleMessages(txCtx, interview, snapshot)
	})
	if err != nil {
		return 0, err
	}
	s.outbox.Signal()
	return interview.ID, nil
}

func (s *InterviewService) UpdateInterview(ctx context.Context, cmd command.UpdateInterview) error {
	if err := s.authorizer.VerifyActor(ctx, cmd.HRID); err != nil {
		return err
	}
	interview, err := s.interviewModel(ctx, cmd.InterviewID)
	if err != nil {
		return err
	}
	if err := s.authorizer.Authorize(ctx, cmd.HRID, port.PermissionInterviewSchedule); err != nil {
		return err
	}
	if err := s.authorizer.CanScheduleApplication(ctx, cmd.HRID, interview.ApplicationID); err != nil {
		return err
	}
	interview.ApplyPatch(model.InterviewPatch{
		Title:           cmd.Title,
		Mode:            cmd.Mode,
		MeetingURL:      cmd.MeetingURL,
		Location:        cmd.Location,
		DurationMinutes: cmd.DurationMinutes,
		CandidateNote:   cmd.CandidateNote,
		InternalNote:    cmd.InternalNote,
		ScheduledAt:     cmd.ScheduledAt,
	})
	snapshot, err := s.applicationSnapshot(ctx, interview.ApplicationID)
	if err != nil {
		return err
	}
	err = s.interviews.Transaction(ctx, func(txCtx context.Context, writer repository.InterviewWriter) error {
		if err := writer.Save(txCtx, interview); err != nil {
			return err
		}
		return s.publishUpdateMessages(txCtx, interview, snapshot)
	})
	if err != nil {
		return err
	}
	s.outbox.Signal()
	return nil
}

func (s *InterviewService) CancelInterview(ctx context.Context, cmd command.CancelInterview) error {
	if err := s.authorizer.VerifyActor(ctx, cmd.HRID); err != nil {
		return err
	}
	interview, err := s.interviewModel(ctx, cmd.InterviewID)
	if err != nil {
		return err
	}
	if err := s.authorizer.Authorize(ctx, cmd.HRID, port.PermissionInterviewSchedule); err != nil {
		return err
	}
	if err := s.authorizer.CanScheduleApplication(ctx, cmd.HRID, interview.ApplicationID); err != nil {
		return err
	}
	if err := interview.Cancel(cmd.CancelReason); err != nil {
		return err
	}
	snapshot, err := s.applicationSnapshot(ctx, interview.ApplicationID)
	if err != nil {
		return err
	}
	reasonText := cmd.CancelReason
	if reasonText == "" {
		reasonText = "暂无说明"
	}
	err = s.interviews.Transaction(ctx, func(txCtx context.Context, writer repository.InterviewWriter) error {
		if err := writer.Save(txCtx, interview); err != nil {
			return err
		}
		transition, ok := domainservice.CancelTransition(snapshot.StatusKey)
		if !ok {
			return nil
		}
		transitioned, err := s.lifecycle.ApplyTransition(txCtx, port.LifecycleTransitionCommand{
			ApplicationID:    interview.ApplicationID,
			FromStatus:       transition.From,
			ToStatus:         transition.To,
			ActorUserID:      cmd.HRID,
			ActorAccountType: "staff",
			Reason:           fmt.Sprintf("取消面试（ID=%d）：%s", interview.ID, reasonText),
		})
		if err != nil {
			return err
		}
		if !transitioned {
			return nil
		}
		return s.publishCancelMessages(txCtx, interview, snapshot, reasonText)
	})
	if err != nil {
		return err
	}
	s.outbox.Signal()
	return nil
}

func (s *InterviewService) BatchCancelInterviews(ctx context.Context, cmd command.BatchCancelInterviews) (int32, error) {
	if err := s.authorizer.VerifyActor(ctx, cmd.HRID); err != nil {
		return 0, err
	}
	if err := s.authorizer.Authorize(ctx, cmd.HRID, port.PermissionInterviewSchedule); err != nil {
		return 0, err
	}
	if err := s.authorizer.CanScheduleApplication(ctx, cmd.HRID, cmd.ApplicationID); err != nil {
		return 0, err
	}
	snapshot, err := s.applicationSnapshot(ctx, cmd.ApplicationID)
	if err != nil {
		return 0, err
	}
	rows, err := s.interviews.ListByApplication(ctx, cmd.ApplicationID)
	if err != nil {
		return 0, err
	}
	active := activeInterviews(rows)
	if len(active) == 0 {
		return 0, nil
	}
	reasonText := cmd.CancelReason
	if reasonText == "" {
		reasonText = "批量取消"
	}
	err = s.interviews.Transaction(ctx, func(txCtx context.Context, writer repository.InterviewWriter) error {
		if err := writer.CancelActiveByApplication(txCtx, cmd.ApplicationID, reasonText); err != nil {
			return err
		}
		if transition, ok := domainservice.CancelTransition(snapshot.StatusKey); ok {
			if _, err := s.lifecycle.ApplyTransition(txCtx, port.LifecycleTransitionCommand{
				ApplicationID:    cmd.ApplicationID,
				FromStatus:       transition.From,
				ToStatus:         transition.To,
				ActorUserID:      cmd.HRID,
				ActorAccountType: "staff",
				Reason:           fmt.Sprintf("批量取消面试：%s", reasonText),
			}); err != nil {
				return err
			}
		}
		return s.publishCancelMessages(txCtx, &active[0].Interview, snapshot, reasonText)
	})
	if err != nil {
		return 0, err
	}
	s.outbox.Signal()
	return int32(len(active)), nil
}

func (s *InterviewService) SubmitFeedback(ctx context.Context, cmd command.SubmitFeedback) error {
	if err := s.authorizer.VerifyActor(ctx, cmd.InterviewerID); err != nil {
		return err
	}
	if err := s.authorizer.Authorize(ctx, cmd.InterviewerID, port.PermissionFeedbackSubmit); err != nil {
		return err
	}
	details, err := s.interviewDetails(ctx, cmd.InterviewID)
	if err != nil {
		return err
	}
	if details.Interview.InterviewerID != cmd.InterviewerID {
		return model.ErrInterviewerMismatch
	}
	if details.Interview.ApplicationID != cmd.ApplicationID {
		return model.ErrFeedbackApplicationMismatch
	}
	if model.IsTerminalApplicationStatus(details.ApplicationStatusKey) {
		return model.ErrFeedbackTerminalApplication
	}
	exists, err := s.interviews.FeedbackExistsByInterviewer(ctx, cmd.InterviewID, cmd.InterviewerID)
	if err != nil {
		return err
	}
	if exists {
		return model.ErrFeedbackAlreadyExists
	}
	feedback, err := model.NewFeedback(model.FeedbackDetails{
		InterviewID:         cmd.InterviewID,
		ApplicationID:       cmd.ApplicationID,
		InterviewerID:       cmd.InterviewerID,
		Recommendation:      cmd.Recommendation,
		Score:               cmd.Score,
		DimensionScoresJSON: cmd.DimensionScoresJSON,
		Comments:            cmd.Comments,
		SubmittedAt:         s.clock.Now(),
	})
	if err != nil {
		return err
	}
	interview := details.Interview
	return s.interviews.Transaction(ctx, func(txCtx context.Context, writer repository.InterviewWriter) error {
		if err := writer.CreateFeedback(txCtx, feedback); err != nil {
			return err
		}
		if interview.CompleteIfScheduled() {
			if err := writer.Save(txCtx, &interview); err != nil {
				return err
			}
		}
		transition, ok := domainservice.FeedbackTransition(details.ApplicationStatusKey)
		if !ok {
			return nil
		}
		_, err := s.lifecycle.ApplyTransition(txCtx, port.LifecycleTransitionCommand{
			ApplicationID:    cmd.ApplicationID,
			FromStatus:       transition.From,
			ToStatus:         transition.To,
			ActorUserID:      cmd.InterviewerID,
			ActorAccountType: "staff",
			Reason:           "面试官提交反馈，自动推进至面试中",
		})
		return err
	})
}

func (s *InterviewService) GetFeedback(ctx context.Context, qry query.GetFeedback) (*model.Feedback, error) {
	if err := s.authorizer.VerifyActor(ctx, qry.InterviewerID); err != nil {
		return nil, err
	}
	return s.interviews.FindFeedbackByInterviewAndInterviewer(ctx, qry.InterviewID, qry.InterviewerID)
}

func (s *InterviewService) GetInterview(ctx context.Context, qry query.GetInterview) (*repository.InterviewDetails, error) {
	if err := s.authorizer.VerifyActor(ctx, qry.UserID); err != nil {
		return nil, err
	}
	details, err := s.interviewDetails(ctx, qry.InterviewID)
	if err != nil {
		return nil, err
	}
	candidateUserID := details.CandidateUserID
	if candidateUserID == 0 {
		if snapshot, err := s.applicationSnapshot(ctx, details.Interview.ApplicationID); err == nil && snapshot != nil {
			candidateUserID = snapshot.CandidateUserID
		} else if err != nil {
			return nil, err
		}
	}
	if candidateUserID == qry.UserID {
		details.Interview.InternalNote = ""
		s.attachResumeURL(details)
		return details, nil
	}
	if err := s.authorizer.Authorize(ctx, qry.UserID, port.PermissionInterviewRead); err != nil {
		return nil, err
	}
	if err := s.authorizer.CanReadInterview(ctx, qry.UserID, qry.InterviewID); err != nil {
		return nil, err
	}
	s.attachResumeURL(details)
	return details, nil
}

func (s *InterviewService) ListInterviewers(ctx context.Context, qry query.ListInterviewers) (port.StaffPage, error) {
	if err := s.authorizer.VerifyActor(ctx, qry.HRID); err != nil {
		return port.StaffPage{}, err
	}
	if err := s.authorizer.Authorize(ctx, qry.HRID, port.PermissionInterviewSchedule); err != nil {
		return port.StaffPage{}, err
	}
	page := qry.Page
	if page <= 0 {
		page = 1
	}
	pageSize := qry.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
	}
	return s.staff.ListInterviewers(ctx, page, pageSize, qry.Keyword)
}

func (s *InterviewService) ListApplicationInterviews(ctx context.Context, qry query.ListApplicationInterviews) ([]repository.InterviewDetails, error) {
	if err := s.authorizer.VerifyActor(ctx, qry.HRID); err != nil {
		return nil, err
	}
	if err := s.authorizer.Authorize(ctx, qry.HRID, port.PermissionInterviewSchedule); err != nil {
		return nil, err
	}
	if err := s.authorizer.CanScheduleApplication(ctx, qry.HRID, qry.ApplicationID); err != nil {
		return nil, err
	}
	rows, err := s.interviews.ListByApplication(ctx, qry.ApplicationID)
	if err != nil {
		return nil, err
	}
	s.attachResumeURLs(rows)
	return rows, nil
}

func (s *InterviewService) ListMyInterviews(ctx context.Context, qry query.ListMyInterviews) ([]repository.InterviewDetails, error) {
	if err := s.authorizer.VerifyActor(ctx, qry.InterviewerID); err != nil {
		return nil, err
	}
	if err := s.authorizer.Authorize(ctx, qry.InterviewerID, port.PermissionInterviewRead); err != nil {
		return nil, err
	}
	rows, err := s.interviews.ListByInterviewer(ctx, qry.InterviewerID, qry.Status)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.Interview.ID)
	}
	feedbacks, err := s.interviews.ListFeedbackByInterviews(ctx, ids)
	if err != nil {
		return nil, err
	}
	hasFeedback := make(map[int64]bool, len(feedbacks))
	for _, feedback := range feedbacks {
		if feedback.InterviewerID == qry.InterviewerID {
			hasFeedback[feedback.InterviewID] = true
		}
	}
	for i := range rows {
		rows[i].HasFeedbackForRequest = hasFeedback[rows[i].Interview.ID]
	}
	s.attachResumeURLs(rows)
	return rows, nil
}

func (s *InterviewService) ListCandidateInterviews(ctx context.Context, qry query.ListCandidateInterviews) ([]repository.InterviewDetails, error) {
	if err := s.authorizer.VerifyActor(ctx, qry.UserID); err != nil {
		return nil, err
	}
	rows, err := s.interviews.ListByCandidate(ctx, qry.UserID)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		rows[i].Interview.InternalNote = ""
	}
	s.attachResumeURLs(rows)
	return rows, nil
}

func (s *InterviewService) attachResumeURLs(rows []repository.InterviewDetails) {
	for i := range rows {
		s.attachResumeURL(&rows[i])
	}
}

func (s *InterviewService) attachResumeURL(details *repository.InterviewDetails) {
	if details == nil || details.ResumeOssKey == "" || s.resumeURLs == nil {
		return
	}
	url, err := s.resumeURLs.GeneratePresignedGetURL(details.ResumeOssKey)
	if err != nil {
		return
	}
	details.ResumeURL = url
}

func (s *InterviewService) applicationSnapshot(ctx context.Context, applicationID int64) (*port.ApplicationSnapshot, error) {
	snapshot, err := s.applications.GetApplicationSnapshot(ctx, applicationID)
	if err != nil {
		return nil, err
	}
	if snapshot == nil {
		return nil, ErrApplicationNotFound
	}
	return snapshot, nil
}

func (s *InterviewService) interviewModel(ctx context.Context, interviewID int64) (*model.Interview, error) {
	interview, err := s.interviews.FindByID(ctx, interviewID)
	if err != nil {
		return nil, err
	}
	if interview == nil {
		return nil, ErrInterviewNotFound
	}
	return interview, nil
}

func (s *InterviewService) interviewDetails(ctx context.Context, interviewID int64) (*repository.InterviewDetails, error) {
	details, err := s.interviews.FindDetailsByID(ctx, interviewID)
	if err != nil {
		return nil, err
	}
	if details == nil {
		return nil, ErrInterviewNotFound
	}
	return details, nil
}

func activeInterviews(rows []repository.InterviewDetails) []repository.InterviewDetails {
	active := make([]repository.InterviewDetails, 0, len(rows))
	for _, row := range rows {
		if row.Interview.IsActive() {
			active = append(active, row)
		}
	}
	return active
}

func (s *InterviewService) publishScheduleMessages(ctx context.Context, interview *model.Interview, snapshot *port.ApplicationSnapshot) error {
	if err := s.outbox.Publish(ctx, staffNotification(interview, snapshot, "interview_assigned", "新的面试安排", fmt.Sprintf("您被安排为「%s」岗位的面试官：%s", snapshot.JobTitle, interview.Title))); err != nil {
		return err
	}
	if err := s.outbox.Publish(ctx, staffEmail(interview, snapshot, "interview_assigned", "新的面试安排", fmt.Sprintf("您被安排为「%s」岗位的面试官：%s", snapshot.JobTitle, interview.Title))); err != nil {
		return err
	}
	if err := s.outbox.Publish(ctx, candidateNotification(interview, snapshot, "interview_scheduled", "面试安排通知", fmt.Sprintf("您的「%s」岗位面试已安排：%s", snapshot.JobTitle, interview.Title))); err != nil {
		return err
	}
	return s.outbox.Publish(ctx, candidateEmail(interview, snapshot, "interview_scheduled", "面试安排通知", fmt.Sprintf("您的「%s」岗位面试已安排：%s", snapshot.JobTitle, interview.Title)))
}

func (s *InterviewService) publishUpdateMessages(ctx context.Context, interview *model.Interview, snapshot *port.ApplicationSnapshot) error {
	if err := s.outbox.Publish(ctx, staffNotification(interview, snapshot, "interview_updated", "面试信息已更新", fmt.Sprintf("「%s」岗位的面试安排已更新：%s", snapshot.JobTitle, interview.Title))); err != nil {
		return err
	}
	if err := s.outbox.Publish(ctx, staffEmail(interview, snapshot, "interview_updated", "面试信息已更新", fmt.Sprintf("「%s」岗位的面试安排已更新：%s", snapshot.JobTitle, interview.Title))); err != nil {
		return err
	}
	if err := s.outbox.Publish(ctx, candidateNotification(interview, snapshot, "interview_updated", "面试时间变更通知", fmt.Sprintf("您的「%s」岗位面试信息已更新，请查看最新安排", snapshot.JobTitle))); err != nil {
		return err
	}
	return s.outbox.Publish(ctx, candidateEmail(interview, snapshot, "interview_updated", "面试时间变更通知", fmt.Sprintf("您的「%s」岗位面试信息已更新，请查看最新安排", snapshot.JobTitle)))
}

func (s *InterviewService) publishCancelMessages(ctx context.Context, interview *model.Interview, snapshot *port.ApplicationSnapshot, reason string) error {
	if err := s.outbox.Publish(ctx, staffNotification(interview, snapshot, "interview_cancelled", "面试已取消", fmt.Sprintf("「%s」岗位的面试已取消。原因：%s", snapshot.JobTitle, reason))); err != nil {
		return err
	}
	if err := s.outbox.Publish(ctx, staffEmail(interview, snapshot, "interview_cancelled", "面试已取消", fmt.Sprintf("「%s」岗位的面试已取消。原因：%s", snapshot.JobTitle, reason))); err != nil {
		return err
	}
	if err := s.outbox.Publish(ctx, candidateNotification(interview, snapshot, "interview_cancelled", "面试已取消", fmt.Sprintf("您的「%s」岗位面试已取消。原因：%s", snapshot.JobTitle, reason))); err != nil {
		return err
	}
	return s.outbox.Publish(ctx, candidateEmail(interview, snapshot, "interview_cancelled", "面试已取消", fmt.Sprintf("您的「%s」岗位面试已取消。原因：%s", snapshot.JobTitle, reason)))
}

func staffNotification(interview *model.Interview, snapshot *port.ApplicationSnapshot, typ string, title string, content string) port.OutboxMessage {
	return baseMessage(interview, snapshot, "notification.create", interview.InterviewerID, 2, "staff", typ, title, content, fmt.Sprintf("/hr/interviews/%d", interview.ID))
}

func staffEmail(interview *model.Interview, snapshot *port.ApplicationSnapshot, typ string, title string, content string) port.OutboxMessage {
	return baseMessage(interview, snapshot, "email.send", interview.InterviewerID, 0, "staff", typ, title, content, fmt.Sprintf("/hr/interviews/%d", interview.ID))
}

func candidateNotification(interview *model.Interview, snapshot *port.ApplicationSnapshot, typ string, title string, content string) port.OutboxMessage {
	return baseMessage(interview, snapshot, "notification.create", snapshot.CandidateUserID, 1, "candidate", typ, title, content, "/applications")
}

func candidateEmail(interview *model.Interview, snapshot *port.ApplicationSnapshot, typ string, title string, content string) port.OutboxMessage {
	message := baseMessage(interview, snapshot, "email.send", snapshot.CandidateUserID, 0, "candidate", typ, title, content, "/applications")
	message.RecipientName = snapshot.CandidateName
	return message
}

func baseMessage(interview *model.Interview, snapshot *port.ApplicationSnapshot, routingKey string, receiverID int64, receiverRole int32, accountType string, typ string, title string, content string, link string) port.OutboxMessage {
	return port.OutboxMessage{
		EventType:           eventTypeForRouting(routingKey),
		AggregateType:       "interview",
		AggregateID:         interview.ID,
		RoutingKey:          routingKey,
		ReceiverID:          receiverID,
		ReceiverRole:        receiverRole,
		ReceiverAccountType: accountType,
		Type:                typ,
		Title:               title,
		Content:             content,
		Link:                link,
		BizType:             "interview",
		BizID:               interview.ID,
		JobTitle:            snapshot.JobTitle,
		InterviewDate:       formatInterviewDate(interview.ScheduledAt),
		InterviewMode:       formatInterviewMode(interview.Mode),
		InterviewLink:       interview.MeetingURL,
		InterviewLocation:   interview.Location,
	}
}

func eventTypeForRouting(routingKey string) string {
	if routingKey == "email.send" {
		return "interview.email_requested"
	}
	return "interview.notification_requested"
}

func formatInterviewDate(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}

func formatInterviewMode(mode string) string {
	switch mode {
	case "phone":
		return "电话面试"
	case "onsite":
		return "现场面试"
	case "video":
		return "视频面试"
	default:
		return mode
	}
}
