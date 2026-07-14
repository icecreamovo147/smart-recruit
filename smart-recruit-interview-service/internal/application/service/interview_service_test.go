package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"smart-recruit-interview-service/internal/application/command"
	"smart-recruit-interview-service/internal/application/port"
	"smart-recruit-interview-service/internal/application/query"
	"smart-recruit-interview-service/internal/domain/model"
	"smart-recruit-interview-service/internal/domain/repository"
)

func TestScheduleInterviewCreatesScheduledInterviewAndPublishesMessages(t *testing.T) {
	fixture := newInterviewFixture(t)
	fixture.repo.maxRound = 2
	interviewID, err := fixture.service.ScheduleInterview(context.Background(), command.ScheduleInterview{
		HRID:          100,
		ApplicationID: 10,
		InterviewerID: 200,
	})
	if err != nil {
		t.Fatalf("ScheduleInterview returned error: %v", err)
	}
	interview := fixture.repo.interviews[interviewID]
	if interview == nil {
		t.Fatalf("created interview %d not persisted", interviewID)
	}
	if interview.RoundNo != 3 || interview.Title != "第 3 轮面试" || interview.Mode != "video" {
		t.Fatalf("unexpected scheduled interview defaults: %+v", interview)
	}
	if got := fixture.lifecycle.last.ToStatus; got != model.ApplicationStatusInterviewPending {
		t.Fatalf("application transition to=%s, want interview_pending", got)
	}
	if len(fixture.outbox.messages) != 4 {
		t.Fatalf("outbox messages=%d, want 4", len(fixture.outbox.messages))
	}
	if fixture.outbox.messages[0].Type != "interview_assigned" || fixture.outbox.messages[1].RoutingKey != "email.send" {
		t.Fatalf("unexpected outbox messages: %+v", fixture.outbox.messages)
	}
	if !fixture.outbox.signaled {
		t.Fatal("expected outbox signal")
	}
}

func TestScheduleInterviewCreatesWithoutLifecycleTransitionForLegacyNoopStates(t *testing.T) {
	fixture := newInterviewFixture(t)
	fixture.applications.snapshots[10].StatusKey = model.ApplicationStatusScreening

	interviewID, err := fixture.service.ScheduleInterview(context.Background(), command.ScheduleInterview{
		HRID:          100,
		ApplicationID: 10,
		InterviewerID: 200,
	})
	if err != nil {
		t.Fatalf("ScheduleInterview returned error: %v", err)
	}
	if fixture.repo.interviews[interviewID] == nil {
		t.Fatalf("created interview %d not persisted", interviewID)
	}
	if fixture.lifecycle.calls != 0 {
		t.Fatalf("lifecycle calls=%d, want 0", fixture.lifecycle.calls)
	}
	if len(fixture.outbox.messages) != 4 {
		t.Fatalf("outbox messages=%d, want 4", len(fixture.outbox.messages))
	}
}

func TestUpdateInterviewAppliesPatchAndPublishesMessages(t *testing.T) {
	fixture := newInterviewFixture(t)
	interview := fixture.seedInterview(model.InterviewStatusScheduled, model.ApplicationStatusInterviewPending)
	scheduledAt := fixture.service.clock.Now().Add(time.Hour)
	if err := fixture.service.UpdateInterview(context.Background(), command.UpdateInterview{
		HRID:        100,
		InterviewID: interview.ID,
		Title:       "Updated Interview",
		MeetingURL:  "https://meet.example.com/updated",
		ScheduledAt: &scheduledAt,
	}); err != nil {
		t.Fatalf("UpdateInterview returned error: %v", err)
	}
	if interview.Title != "Updated Interview" || interview.MeetingURL == "" || interview.ScheduledAt == nil {
		t.Fatalf("patch not applied: %+v", interview)
	}
	if len(fixture.outbox.messages) != 4 {
		t.Fatalf("outbox messages=%d, want 4", len(fixture.outbox.messages))
	}
	if got := fixture.outbox.messages[0].Type; got != "interview_updated" {
		t.Fatalf("message type=%s, want interview_updated", got)
	}
}

func TestCancelInterviewTransitionsAndPublishesMessages(t *testing.T) {
	fixture := newInterviewFixture(t)
	interview := fixture.seedInterview(model.InterviewStatusScheduled, model.ApplicationStatusInterviewing)
	if err := fixture.service.CancelInterview(context.Background(), command.CancelInterview{
		HRID:         100,
		InterviewID:  interview.ID,
		CancelReason: "candidate unavailable",
	}); err != nil {
		t.Fatalf("CancelInterview returned error: %v", err)
	}
	if interview.Status != model.InterviewStatusCancelled {
		t.Fatalf("status=%s, want cancelled", interview.Status)
	}
	if got := fixture.lifecycle.last.ToStatus; got != model.ApplicationStatusInterviewCancelled {
		t.Fatalf("application transition to=%s, want interview_cancelled", got)
	}
	if len(fixture.outbox.messages) != 4 {
		t.Fatalf("outbox messages=%d, want 4", len(fixture.outbox.messages))
	}
	if got := fixture.outbox.messages[2].ReceiverAccountType; got != "candidate" {
		t.Fatalf("candidate notification receiver=%s, want candidate", got)
	}
}

func TestBatchCancelInterviewsCancelsOnlyActiveRows(t *testing.T) {
	fixture := newInterviewFixture(t)
	first := fixture.seedInterview(model.InterviewStatusScheduled, model.ApplicationStatusInterviewPending)
	second := fixture.seedInterview(model.InterviewStatusPending, model.ApplicationStatusInterviewPending)
	completed := fixture.seedInterview(model.InterviewStatusCompleted, model.ApplicationStatusInterviewPending)

	affected, err := fixture.service.BatchCancelInterviews(context.Background(), command.BatchCancelInterviews{
		HRID:          100,
		ApplicationID: 10,
		CancelReason:  "role closed",
	})
	if err != nil {
		t.Fatalf("BatchCancelInterviews returned error: %v", err)
	}
	if affected != 2 {
		t.Fatalf("affected=%d, want 2", affected)
	}
	if first.Status != model.InterviewStatusCancelled || second.Status != model.InterviewStatusCancelled {
		t.Fatalf("active interviews not cancelled: first=%s second=%s", first.Status, second.Status)
	}
	if completed.Status != model.InterviewStatusCompleted {
		t.Fatalf("completed status=%s, want completed", completed.Status)
	}
	if len(fixture.outbox.messages) != 4 {
		t.Fatalf("outbox messages=%d, want one notification/email group", len(fixture.outbox.messages))
	}
}

func TestSubmitFeedbackCompletesScheduledInterviewAndAdvancesApplication(t *testing.T) {
	fixture := newInterviewFixture(t)
	interview := fixture.seedInterview(model.InterviewStatusScheduled, model.ApplicationStatusInterviewPending)
	if err := fixture.service.SubmitFeedback(context.Background(), command.SubmitFeedback{
		InterviewerID:  interview.InterviewerID,
		InterviewID:    interview.ID,
		ApplicationID:  interview.ApplicationID,
		Recommendation: "recommend",
		Score:          8,
	}); err != nil {
		t.Fatalf("SubmitFeedback returned error: %v", err)
	}
	if got := fixture.repo.interviews[interview.ID].Status; got != model.InterviewStatusCompleted {
		t.Fatalf("interview status=%s, want completed", got)
	}
	if len(fixture.repo.feedbacks) != 1 {
		t.Fatalf("feedbacks=%d, want 1", len(fixture.repo.feedbacks))
	}
	if got := fixture.lifecycle.last.ToStatus; got != model.ApplicationStatusInterviewing {
		t.Fatalf("application transition to=%s, want interviewing", got)
	}
}

func TestSubmitFeedbackRejectsDuplicateAndTerminalApplication(t *testing.T) {
	fixture := newInterviewFixture(t)
	interview := fixture.seedInterview(model.InterviewStatusScheduled, model.ApplicationStatusRejected)
	err := fixture.service.SubmitFeedback(context.Background(), command.SubmitFeedback{
		InterviewerID:  interview.InterviewerID,
		InterviewID:    interview.ID,
		ApplicationID:  interview.ApplicationID,
		Recommendation: "recommend",
		Score:          8,
	})
	if err != model.ErrFeedbackTerminalApplication {
		t.Fatalf("terminal application error=%v, want ErrFeedbackTerminalApplication", err)
	}

	interview = fixture.seedInterview(model.InterviewStatusScheduled, model.ApplicationStatusInterviewPending)
	fixture.repo.duplicateFeedback = true
	err = fixture.service.SubmitFeedback(context.Background(), command.SubmitFeedback{
		InterviewerID:  interview.InterviewerID,
		InterviewID:    interview.ID,
		ApplicationID:  interview.ApplicationID,
		Recommendation: "recommend",
		Score:          8,
	})
	if err != model.ErrFeedbackAlreadyExists {
		t.Fatalf("duplicate feedback error=%v, want ErrFeedbackAlreadyExists", err)
	}
}

func TestListMyInterviewsMarksFeedbackForRequester(t *testing.T) {
	fixture := newInterviewFixture(t)
	first := fixture.seedInterview(model.InterviewStatusCompleted, model.ApplicationStatusInterviewing)
	second := fixture.seedInterview(model.InterviewStatusScheduled, model.ApplicationStatusInterviewing)
	fixture.repo.feedbacks = append(fixture.repo.feedbacks, model.Feedback{
		InterviewID:   first.ID,
		InterviewerID: first.InterviewerID,
	})
	fixture.repo.feedbacks = append(fixture.repo.feedbacks, model.Feedback{
		InterviewID:   second.ID,
		InterviewerID: 999,
	})

	rows, err := fixture.service.ListMyInterviews(context.Background(), query.ListMyInterviews{InterviewerID: 200})
	if err != nil {
		t.Fatalf("ListMyInterviews returned error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows=%d, want 2", len(rows))
	}
	feedbackByInterview := map[int64]bool{}
	for _, row := range rows {
		feedbackByInterview[row.Interview.ID] = row.HasFeedbackForRequest
	}
	if !feedbackByInterview[first.ID] {
		t.Fatalf("first interview feedback flag=false, want true")
	}
	if feedbackByInterview[second.ID] {
		t.Fatalf("second interview feedback flag=true, want false for another interviewer")
	}
}

func TestListCandidateInterviewsFiltersInternalNotes(t *testing.T) {
	fixture := newInterviewFixture(t)
	interview := fixture.seedInterview(model.InterviewStatusScheduled, model.ApplicationStatusInterviewPending)
	interview.InternalNote = "staff-only"
	detail := fixture.repo.details[interview.ID]
	detail.Interview.InternalNote = "staff-only"
	fixture.repo.details[interview.ID] = detail

	rows, err := fixture.service.ListCandidateInterviews(context.Background(), query.ListCandidateInterviews{UserID: 300})
	if err != nil {
		t.Fatalf("ListCandidateInterviews returned error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows=%d, want 1", len(rows))
	}
	if rows[0].Interview.InternalNote != "" {
		t.Fatalf("internal note=%q, want filtered", rows[0].Interview.InternalNote)
	}
}

func TestGetInterviewSignsResumeURLAndFiltersCandidateNotes(t *testing.T) {
	fixture := newInterviewFixture(t)
	interview := fixture.seedInterview(model.InterviewStatusScheduled, model.ApplicationStatusInterviewPending)
	detail := fixture.repo.details[interview.ID]
	detail.Interview.InternalNote = "staff-only"
	detail.ResumeOssKey = "resumes/300/cv.pdf"
	fixture.repo.details[interview.ID] = detail
	fixture.resumeURLs.urls[detail.ResumeOssKey] = "https://signed.example/resumes/300/cv.pdf"

	got, err := fixture.service.GetInterview(context.Background(), query.GetInterview{UserID: 300, InterviewID: interview.ID})
	if err != nil {
		t.Fatalf("GetInterview returned error: %v", err)
	}
	if got.Interview.InternalNote != "" {
		t.Fatalf("internal note=%q, want filtered", got.Interview.InternalNote)
	}
	if got.ResumeURL != "https://signed.example/resumes/300/cv.pdf" {
		t.Fatalf("resume url=%q, want signed URL", got.ResumeURL)
	}
	if len(fixture.resumeURLs.calls) != 1 || fixture.resumeURLs.calls[0] != detail.ResumeOssKey {
		t.Fatalf("signer calls=%v, want [%s]", fixture.resumeURLs.calls, detail.ResumeOssKey)
	}
}

func TestListMyInterviewsIgnoresResumeURLSigningFailure(t *testing.T) {
	fixture := newInterviewFixture(t)
	interview := fixture.seedInterview(model.InterviewStatusScheduled, model.ApplicationStatusInterviewPending)
	detail := fixture.repo.details[interview.ID]
	detail.ResumeOssKey = "resumes/300/cv.pdf"
	fixture.repo.details[interview.ID] = detail
	fixture.resumeURLs.err = errors.New("presign failed")

	rows, err := fixture.service.ListMyInterviews(context.Background(), query.ListMyInterviews{InterviewerID: 200})
	if err != nil {
		t.Fatalf("ListMyInterviews returned error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows=%d, want 1", len(rows))
	}
	if rows[0].ResumeURL != "" {
		t.Fatalf("resume url=%q, want empty after signer failure", rows[0].ResumeURL)
	}
	if len(fixture.resumeURLs.calls) != 1 || fixture.resumeURLs.calls[0] != detail.ResumeOssKey {
		t.Fatalf("signer calls=%v, want [%s]", fixture.resumeURLs.calls, detail.ResumeOssKey)
	}
}

func TestGetInterviewWithoutResumeURLSignerLeavesResumeURLEmpty(t *testing.T) {
	fixture := newInterviewFixture(t)
	fixture.service.resumeURLs = nil
	interview := fixture.seedInterview(model.InterviewStatusScheduled, model.ApplicationStatusInterviewPending)
	detail := fixture.repo.details[interview.ID]
	detail.ResumeOssKey = "resumes/300/cv.pdf"
	fixture.repo.details[interview.ID] = detail

	got, err := fixture.service.GetInterview(context.Background(), query.GetInterview{UserID: 300, InterviewID: interview.ID})
	if err != nil {
		t.Fatalf("GetInterview returned error: %v", err)
	}
	if got.ResumeURL != "" {
		t.Fatalf("resume url=%q, want empty without signer", got.ResumeURL)
	}
}

func newInterviewFixture(t *testing.T) *interviewFixture {
	t.Helper()
	repo := &fakeInterviewRepository{
		interviews: map[int64]*model.Interview{},
		details:    map[int64]repository.InterviewDetails{},
		nextID:     1,
	}
	apps := &fakeApplicationSnapshots{snapshots: map[int64]*port.ApplicationSnapshot{
		10: {
			ApplicationID:   10,
			CandidateUserID: 300,
			JobID:           400,
			JobTitle:        "Backend Engineer",
			CandidateName:   "Candidate A",
			StatusKey:       model.ApplicationStatusViewed,
		},
	}}
	lifecycle := &fakeLifecycle{}
	outbox := &fakeOutbox{}
	authorizer := &fakeAuthorizer{}
	resumeURLs := &fakeResumeURLSigner{urls: map[string]string{}}
	service, err := NewInterviewService(Deps{
		Interviews:   repo,
		Applications: apps,
		Staff:        fakeStaffDirectory{},
		Lifecycle:    lifecycle,
		Outbox:       outbox,
		Authorizer:   authorizer,
		ResumeURLs:   resumeURLs,
		Clock:        fixedClock{now: time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewInterviewService returned error: %v", err)
	}
	return &interviewFixture{
		service:      service,
		repo:         repo,
		applications: apps,
		lifecycle:    lifecycle,
		outbox:       outbox,
		authorizer:   authorizer,
		resumeURLs:   resumeURLs,
	}
}

type interviewFixture struct {
	service      *InterviewService
	repo         *fakeInterviewRepository
	applications *fakeApplicationSnapshots
	lifecycle    *fakeLifecycle
	outbox       *fakeOutbox
	authorizer   *fakeAuthorizer
	resumeURLs   *fakeResumeURLSigner
}

func (f *interviewFixture) seedInterview(status model.InterviewStatus, appStatus model.ApplicationStatus) *model.Interview {
	interview := &model.Interview{
		ID:            f.repo.nextID,
		ApplicationID: 10,
		InterviewerID: 200,
		RoundNo:       int32(f.repo.nextID),
		Title:         "Interview",
		Mode:          "video",
		Status:        status,
	}
	f.repo.nextID++
	f.repo.interviews[interview.ID] = interview
	f.repo.details[interview.ID] = repository.InterviewDetails{
		Interview:            *interview,
		ApplicationStatusKey: appStatus,
		JobTitle:             "Backend Engineer",
		CandidateUserID:      300,
		CandidateName:        "Candidate A",
		InterviewerName:      "Interviewer A",
	}
	f.applications.snapshots[interview.ApplicationID].StatusKey = appStatus
	return interview
}

type fakeInterviewRepository struct {
	interviews        map[int64]*model.Interview
	details           map[int64]repository.InterviewDetails
	feedbacks         []model.Feedback
	nextID            int64
	maxRound          int32
	duplicateFeedback bool
}

func (r *fakeInterviewRepository) Create(_ context.Context, interview *model.Interview) error {
	if interview.ID == 0 {
		interview.ID = r.nextID
		r.nextID++
	}
	r.interviews[interview.ID] = interview
	r.details[interview.ID] = repository.InterviewDetails{
		Interview:            *interview,
		ApplicationStatusKey: model.ApplicationStatusInterviewPending,
		JobTitle:             "Backend Engineer",
		CandidateUserID:      300,
		CandidateName:        "Candidate A",
		InterviewerName:      "Interviewer A",
	}
	return nil
}

func (r *fakeInterviewRepository) Save(_ context.Context, interview *model.Interview) error {
	if interview.ID == 0 {
		return errors.New("interview id is required")
	}
	r.interviews[interview.ID] = interview
	detail := r.details[interview.ID]
	detail.Interview = *interview
	r.details[interview.ID] = detail
	return nil
}

func (r *fakeInterviewRepository) CreateFeedback(_ context.Context, feedback *model.Feedback) error {
	r.feedbacks = append(r.feedbacks, *feedback)
	return nil
}

func (r *fakeInterviewRepository) CancelActiveByApplication(_ context.Context, applicationID int64, reason string) error {
	for _, interview := range r.interviews {
		if interview.ApplicationID == applicationID && interview.IsActive() {
			interview.Status = model.InterviewStatusCancelled
			interview.CancelReason = reason
			detail := r.details[interview.ID]
			detail.Interview = *interview
			r.details[interview.ID] = detail
		}
	}
	return nil
}

func (r *fakeInterviewRepository) FindByID(_ context.Context, interviewID int64) (*model.Interview, error) {
	return r.interviews[interviewID], nil
}

func (r *fakeInterviewRepository) FindDetailsByID(_ context.Context, interviewID int64) (*repository.InterviewDetails, error) {
	detail, ok := r.details[interviewID]
	if !ok {
		return nil, nil
	}
	return &detail, nil
}

func (r *fakeInterviewRepository) MaxRoundNo(context.Context, int64) (int32, error) {
	return r.maxRound, nil
}

func (r *fakeInterviewRepository) ListByApplication(_ context.Context, applicationID int64) ([]repository.InterviewDetails, error) {
	rows := make([]repository.InterviewDetails, 0, len(r.details))
	for _, detail := range r.details {
		if detail.Interview.ApplicationID == applicationID {
			rows = append(rows, detail)
		}
	}
	return rows, nil
}

func (r *fakeInterviewRepository) ListByInterviewer(_ context.Context, interviewerID int64, status model.InterviewStatus) ([]repository.InterviewDetails, error) {
	rows := make([]repository.InterviewDetails, 0, len(r.details))
	for _, detail := range r.details {
		if detail.Interview.InterviewerID != interviewerID {
			continue
		}
		if status != "" && detail.Interview.Status != status {
			continue
		}
		rows = append(rows, detail)
	}
	return rows, nil
}

func (r *fakeInterviewRepository) ListByCandidate(_ context.Context, candidateUserID int64) ([]repository.InterviewDetails, error) {
	rows := make([]repository.InterviewDetails, 0, len(r.details))
	for _, detail := range r.details {
		if detail.CandidateUserID == candidateUserID {
			rows = append(rows, detail)
		}
	}
	return rows, nil
}

func (r *fakeInterviewRepository) ListFeedbackByInterviews(_ context.Context, interviewIDs []int64) ([]model.Feedback, error) {
	allowed := make(map[int64]bool, len(interviewIDs))
	for _, id := range interviewIDs {
		allowed[id] = true
	}
	feedbacks := make([]model.Feedback, 0, len(r.feedbacks))
	for _, feedback := range r.feedbacks {
		if allowed[feedback.InterviewID] {
			feedbacks = append(feedbacks, feedback)
		}
	}
	return feedbacks, nil
}

func (r *fakeInterviewRepository) FeedbackExistsByInterviewer(context.Context, int64, int64) (bool, error) {
	return r.duplicateFeedback, nil
}

func (r *fakeInterviewRepository) FindFeedbackByInterviewAndInterviewer(context.Context, int64, int64) (*model.Feedback, error) {
	return nil, nil
}

func (r *fakeInterviewRepository) Transaction(ctx context.Context, fn func(context.Context, repository.InterviewWriter) error) error {
	return fn(ctx, r)
}

type fakeApplicationSnapshots struct {
	snapshots map[int64]*port.ApplicationSnapshot
}

func (a *fakeApplicationSnapshots) GetApplicationSnapshot(_ context.Context, applicationID int64) (*port.ApplicationSnapshot, error) {
	return a.snapshots[applicationID], nil
}

type fakeStaffDirectory struct{}

func (fakeStaffDirectory) ListInterviewers(context.Context, int32, int32, string) (port.StaffPage, error) {
	return port.StaffPage{}, nil
}

type fakeLifecycle struct {
	last  port.LifecycleTransitionCommand
	calls int
}

func (l *fakeLifecycle) ApplyTransition(_ context.Context, command port.LifecycleTransitionCommand) (bool, error) {
	l.last = command
	l.calls++
	return true, nil
}

type fakeOutbox struct {
	messages []port.OutboxMessage
	signaled bool
}

func (o *fakeOutbox) Publish(_ context.Context, message port.OutboxMessage) error {
	o.messages = append(o.messages, message)
	return nil
}

func (o *fakeOutbox) Signal() {
	o.signaled = true
}

type fakeAuthorizer struct{}

func (a *fakeAuthorizer) VerifyActor(_ context.Context, actorID int64) error {
	if actorID == 0 {
		return errors.New("actor id is required")
	}
	return nil
}

func (a *fakeAuthorizer) Authorize(context.Context, int64, port.Permission) error {
	return nil
}

func (a *fakeAuthorizer) CanScheduleApplication(context.Context, int64, int64) error {
	return nil
}

func (a *fakeAuthorizer) CanReadInterview(context.Context, int64, int64) error {
	return nil
}

type fakeResumeURLSigner struct {
	urls  map[string]string
	calls []string
	err   error
}

func (s *fakeResumeURLSigner) GeneratePresignedGetURL(ossKey string) (string, error) {
	s.calls = append(s.calls, ossKey)
	if s.err != nil {
		return "", s.err
	}
	return s.urls[ossKey], nil
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}
