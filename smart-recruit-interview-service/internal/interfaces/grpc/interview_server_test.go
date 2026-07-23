package grpc

import (
	"context"
	"errors"
	"testing"
	"time"

	"smart-recruit-interview-service/internal/application/command"
	"smart-recruit-interview-service/internal/application/port"
	"smart-recruit-interview-service/internal/application/query"
	appservice "smart-recruit-interview-service/internal/application/service"
	"smart-recruit-interview-service/internal/domain/model"
	"smart-recruit-interview-service/internal/domain/repository"
	"smart-recruit-platform-go/businessclock"
	"smart-recruit-platform-go/errs"
	"smart-recruit-proto/recruitment/pb"
)

func TestScheduleInterviewPreservesLegacyTimeParseMessage(t *testing.T) {
	server := newTestServer(t, &fakeInterviewUsecase{})
	resp, err := server.ScheduleInterview(context.Background(), &pb.ScheduleInterviewRequest{ScheduledAt: "not-time"})
	if err != nil {
		t.Fatalf("ScheduleInterview error=%v", err)
	}
	if resp.Code != errs.ErrBadRequest || resp.Msg != "面试时间格式错误，请使用 RFC 3339 格式" {
		t.Fatalf("response=%+v, want legacy time parse bad request", resp)
	}
}

func TestScheduleInterviewSuccessMessage(t *testing.T) {
	uc := &fakeInterviewUsecase{scheduleID: 42}
	server := newTestServer(t, uc)
	resp, err := server.ScheduleInterview(context.Background(), &pb.ScheduleInterviewRequest{HrId: 100, ApplicationId: 10, InterviewerId: 200})
	if err != nil {
		t.Fatalf("ScheduleInterview error=%v", err)
	}
	if resp.Code != errs.OK || resp.Msg != "面试安排成功" || resp.InterviewId != 42 {
		t.Fatalf("response=%+v, want success with interview id", resp)
	}
}

func TestBatchCancelZeroUsesLegacyMessage(t *testing.T) {
	server := newTestServer(t, &fakeInterviewUsecase{})
	resp, err := server.BatchCancelInterviews(context.Background(), &pb.BatchCancelInterviewsRequest{HrId: 100, ApplicationId: 10})
	if err != nil {
		t.Fatalf("BatchCancelInterviews error=%v", err)
	}
	if resp.Code != errs.OK || resp.Msg != "没有需要取消的面试" || resp.Affected != 0 {
		t.Fatalf("response=%+v, want no active interviews message", resp)
	}
}

func TestSubmitFeedbackMapsLegacyErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code int32
		msg  string
	}{
		{name: "assignment", err: model.ErrInterviewerMismatch, code: errs.ErrForbidden, msg: "您不是该面试的面试官，无法提交反馈"},
		{name: "terminal", err: model.ErrFeedbackTerminalApplication, code: errs.ErrForbidden, msg: "该候选人已结束投递流程（已被淘汰或撤回），无法提交面试反馈"},
		{name: "duplicate", err: model.ErrFeedbackAlreadyExists, code: errs.ErrConflict, msg: "您已提交过面试反馈，不可重复提交（如有更正需求请联系 HR）"},
		{name: "recommendation", err: model.ErrFeedbackInvalidRecommendation, code: errs.ErrBadRequest, msg: "推荐结论值不合法"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newTestServer(t, &fakeInterviewUsecase{submitErr: tt.err})
			resp, err := server.SubmitFeedback(context.Background(), &pb.SubmitFeedbackRequest{InterviewerId: 200, InterviewId: 1, ApplicationId: 10})
			if err != nil {
				t.Fatalf("SubmitFeedback error=%v", err)
			}
			if resp.Code != tt.code || resp.Msg != tt.msg {
				t.Fatalf("response=%+v, want code=%d msg=%q", resp, tt.code, tt.msg)
			}
		})
	}
}

func TestGetInterviewMapsCandidateFilteredDetails(t *testing.T) {
	server := newTestServer(t, &fakeInterviewUsecase{details: &repository.InterviewDetails{
		Interview: model.Interview{
			ID:            1,
			ApplicationID: 10,
			InterviewerID: 200,
			Title:         "Interview",
			InternalNote:  "",
			Status:        model.InterviewStatusScheduled,
			ScheduledAt:   ptrTime(time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)),
		},
		ApplicationStatusKey: model.ApplicationStatusInterviewPending,
		JobTitle:             "Backend Engineer",
		CandidateName:        "Candidate A",
		ResumeURL:            "https://signed.example/resume.pdf",
	}})
	resp, err := server.GetInterview(context.Background(), &pb.GetInterviewRequest{UserId: 300, InterviewId: 1})
	if err != nil {
		t.Fatalf("GetInterview error=%v", err)
	}
	if resp.Code != errs.OK || resp.Interview.InternalNote != "" || resp.Interview.JobTitle != "Backend Engineer" || resp.Interview.ResumeUrl != "https://signed.example/resume.pdf" {
		t.Fatalf("response=%+v, want candidate-safe interview", resp)
	}
}

func TestGetInterviewNotFoundMessage(t *testing.T) {
	server := newTestServer(t, &fakeInterviewUsecase{getErr: appservice.ErrInterviewNotFound})
	resp, err := server.GetInterview(context.Background(), &pb.GetInterviewRequest{UserId: 100, InterviewId: 99})
	if err != nil {
		t.Fatalf("GetInterview error=%v", err)
	}
	if resp.Code != errs.ErrBadRequest || resp.Msg != "面试记录不存在" {
		t.Fatalf("response=%+v, want legacy not found", resp)
	}
}

func TestListInterviewersMapsStaffPage(t *testing.T) {
	createdAt := time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)
	uc := &fakeInterviewUsecase{staffPage: port.StaffPage{
		Total: 1,
		List: []port.StaffUser{{
			UserID:       200,
			Username:     "interviewer",
			Email:        "interviewer@example.com",
			Status:       "active",
			AccountType:  "staff",
			Roles:        []string{"interviewer"},
			TokenVersion: 3,
			CreatedAt:    createdAt,
		}},
	}}
	server := newTestServer(t, uc)
	resp, err := server.ListInterviewers(context.Background(), &pb.ListInterviewersRequest{HrId: 100, Page: 2, PageSize: 20, Keyword: "int"})
	if err != nil {
		t.Fatalf("ListInterviewers error=%v", err)
	}
	if uc.listInterviewersQuery.HRID != 100 || uc.listInterviewersQuery.Page != 2 || uc.listInterviewersQuery.PageSize != 20 || uc.listInterviewersQuery.Keyword != "int" {
		t.Fatalf("query=%+v, want request values forwarded", uc.listInterviewersQuery)
	}
	if resp.Code != errs.OK || resp.Total != 1 || len(resp.List) != 1 {
		t.Fatalf("response=%+v, want one interviewer", resp)
	}
	if got := resp.List[0]; got.UserId != 200 || got.TokenVersion != 3 || got.CreatedAt != businessclock.FormatRFC3339(createdAt) {
		t.Fatalf("staff=%+v, want mapped staff user", got)
	}
}

func TestListApplicationInterviewsMapsDetails(t *testing.T) {
	uc := &fakeInterviewUsecase{applicationRows: []repository.InterviewDetails{{
		Interview: model.Interview{
			ID:            1,
			ApplicationID: 10,
			InterviewerID: 200,
			Title:         "Round 1",
			Status:        model.InterviewStatusScheduled,
		},
		ApplicationStatusKey: model.ApplicationStatusInterviewPending,
		JobTitle:             "Backend Engineer",
		CandidateName:        "Candidate A",
	}}}
	server := newTestServer(t, uc)
	resp, err := server.ListApplicationInterviews(context.Background(), &pb.ListApplicationInterviewsRequest{HrId: 100, ApplicationId: 10})
	if err != nil {
		t.Fatalf("ListApplicationInterviews error=%v", err)
	}
	if uc.listApplicationQuery.HRID != 100 || uc.listApplicationQuery.ApplicationID != 10 {
		t.Fatalf("query=%+v, want request values forwarded", uc.listApplicationQuery)
	}
	if resp.Code != errs.OK || len(resp.List) != 1 || resp.List[0].JobTitle != "Backend Engineer" || resp.List[0].ApplicationStatusKey != string(model.ApplicationStatusInterviewPending) {
		t.Fatalf("response=%+v, want mapped application interviews", resp)
	}
}

func TestListMyInterviewsMapsHasFeedbackAndForbiddenMessage(t *testing.T) {
	uc := &fakeInterviewUsecase{myRows: []repository.InterviewDetails{{
		Interview: model.Interview{
			ID:            1,
			ApplicationID: 10,
			InterviewerID: 200,
			Title:         "Round 1",
			Status:        model.InterviewStatusCompleted,
		},
		HasFeedbackForRequest: true,
	}}}
	server := newTestServer(t, uc)
	resp, err := server.ListMyInterviews(context.Background(), &pb.ListMyInterviewsRequest{InterviewerId: 200, Status: string(model.InterviewStatusCompleted)})
	if err != nil {
		t.Fatalf("ListMyInterviews error=%v", err)
	}
	if uc.listMyQuery.InterviewerID != 200 || uc.listMyQuery.Status != model.InterviewStatusCompleted {
		t.Fatalf("query=%+v, want request values forwarded", uc.listMyQuery)
	}
	if resp.Code != errs.OK || len(resp.List) != 1 || !resp.List[0].HasFeedback {
		t.Fatalf("response=%+v, want has_feedback mapped", resp)
	}

	forbiddenServer := newTestServer(t, &fakeInterviewUsecase{listMyErr: errors.New("actor 200 missing permission \"interview.read\"")})
	forbiddenResp, err := forbiddenServer.ListMyInterviews(context.Background(), &pb.ListMyInterviewsRequest{InterviewerId: 200})
	if err != nil {
		t.Fatalf("ListMyInterviews forbidden error=%v", err)
	}
	if forbiddenResp.Code != errs.ErrForbidden || forbiddenResp.Msg != "无权限查看面试列表" {
		t.Fatalf("response=%+v, want legacy forbidden list message", forbiddenResp)
	}
}

func TestListCandidateInterviewsMapsCandidateSafeRows(t *testing.T) {
	uc := &fakeInterviewUsecase{candidateRows: []repository.InterviewDetails{{
		Interview: model.Interview{
			ID:            1,
			ApplicationID: 10,
			InterviewerID: 200,
			Title:         "Candidate Round",
			InternalNote:  "",
			Status:        model.InterviewStatusScheduled,
		},
		JobTitle:      "Backend Engineer",
		CandidateName: "Candidate A",
	}}}
	server := newTestServer(t, uc)
	resp, err := server.ListCandidateInterviews(context.Background(), &pb.ListCandidateInterviewsRequest{UserId: 300})
	if err != nil {
		t.Fatalf("ListCandidateInterviews error=%v", err)
	}
	if uc.listCandidateQuery.UserID != 300 {
		t.Fatalf("query=%+v, want request values forwarded", uc.listCandidateQuery)
	}
	if resp.Code != errs.OK || len(resp.List) != 1 || resp.List[0].InternalNote != "" || resp.List[0].JobTitle != "Backend Engineer" {
		t.Fatalf("response=%+v, want candidate-safe interviews", resp)
	}
}

func newTestServer(t *testing.T, uc *fakeInterviewUsecase) *Server {
	t.Helper()
	server, err := NewServer(uc)
	if err != nil {
		t.Fatalf("NewServer error=%v", err)
	}
	return server
}

type fakeInterviewUsecase struct {
	scheduleID            int64
	submitErr             error
	getErr                error
	details               *repository.InterviewDetails
	staffPage             port.StaffPage
	applicationRows       []repository.InterviewDetails
	myRows                []repository.InterviewDetails
	candidateRows         []repository.InterviewDetails
	listMyErr             error
	listInterviewersQuery query.ListInterviewers
	listApplicationQuery  query.ListApplicationInterviews
	listMyQuery           query.ListMyInterviews
	listCandidateQuery    query.ListCandidateInterviews
}

func (f *fakeInterviewUsecase) ScheduleInterview(context.Context, command.ScheduleInterview) (int64, error) {
	return f.scheduleID, nil
}

func (f *fakeInterviewUsecase) UpdateInterview(context.Context, command.UpdateInterview) error {
	return nil
}

func (f *fakeInterviewUsecase) CancelInterview(context.Context, command.CancelInterview) error {
	return nil
}

func (f *fakeInterviewUsecase) BatchCancelInterviews(context.Context, command.BatchCancelInterviews) (int32, error) {
	return 0, nil
}

func (f *fakeInterviewUsecase) SubmitFeedback(context.Context, command.SubmitFeedback) error {
	return f.submitErr
}

func (f *fakeInterviewUsecase) GetFeedback(context.Context, query.GetFeedback) (*model.Feedback, error) {
	return nil, nil
}

func (f *fakeInterviewUsecase) GetInterview(context.Context, query.GetInterview) (*repository.InterviewDetails, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	if f.details == nil {
		return nil, errors.New("missing details")
	}
	return f.details, nil
}

func (f *fakeInterviewUsecase) ListInterviewers(_ context.Context, qry query.ListInterviewers) (port.StaffPage, error) {
	f.listInterviewersQuery = qry
	return f.staffPage, nil
}

func (f *fakeInterviewUsecase) ListApplicationInterviews(_ context.Context, qry query.ListApplicationInterviews) ([]repository.InterviewDetails, error) {
	f.listApplicationQuery = qry
	return f.applicationRows, nil
}

func (f *fakeInterviewUsecase) ListMyInterviews(_ context.Context, qry query.ListMyInterviews) ([]repository.InterviewDetails, error) {
	f.listMyQuery = qry
	return f.myRows, f.listMyErr
}

func (f *fakeInterviewUsecase) ListCandidateInterviews(_ context.Context, qry query.ListCandidateInterviews) ([]repository.InterviewDetails, error) {
	f.listCandidateQuery = qry
	return f.candidateRows, nil
}

func ptrTime(value time.Time) *time.Time {
	return &value
}
