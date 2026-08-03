package grpc

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"smart-recruit-interview-service/internal/application/command"
	"smart-recruit-interview-service/internal/application/port"
	"smart-recruit-interview-service/internal/application/query"
	appservice "smart-recruit-interview-service/internal/application/service"
	"smart-recruit-interview-service/internal/domain/model"
	"smart-recruit-interview-service/internal/domain/repository"
	"smart-recruit-interview-service/internal/interfaces/mapper"
	"smart-recruit-platform-go/errs"
	"smart-recruit-platform-go/i18n"
	"smart-recruit-proto/recruitment/pb"
)

type interviewUsecase interface {
	ScheduleInterview(context.Context, command.ScheduleInterview) (int64, error)
	UpdateInterview(context.Context, command.UpdateInterview) error
	CancelInterview(context.Context, command.CancelInterview) error
	BatchCancelInterviews(context.Context, command.BatchCancelInterviews) (int32, error)
	SubmitFeedback(context.Context, command.SubmitFeedback) error
	GetFeedback(context.Context, query.GetFeedback) (*model.Feedback, error)
	GetInterview(context.Context, query.GetInterview) (*repository.InterviewDetails, error)
	ListInterviewers(context.Context, query.ListInterviewers) (port.StaffPage, error)
	ListApplicationInterviews(context.Context, query.ListApplicationInterviews) ([]repository.InterviewDetails, error)
	ListMyInterviews(context.Context, query.ListMyInterviews) ([]repository.InterviewDetails, error)
	ListCandidateInterviews(context.Context, query.ListCandidateInterviews) ([]repository.InterviewDetails, error)
}

type Server struct {
	pb.UnimplementedInterviewServiceServer
	interviews interviewUsecase
}

func NewServer(interviews interviewUsecase) (*Server, error) {
	if interviews == nil {
		return nil, fmt.Errorf("interview usecase is required")
	}
	return &Server{interviews: interviews}, nil
}

func (s *Server) ScheduleInterview(ctx context.Context, req *pb.ScheduleInterviewRequest) (*pb.ScheduleInterviewResponse, error) {
	scheduledAt, err := mapper.ParseOptionalRFC3339(req.GetScheduledAt())
	if err != nil {
		return &pb.ScheduleInterviewResponse{Code: errs.ErrBadRequest, Msg: "common.invalid_request"}, nil
	}
	interviewID, err := s.interviews.ScheduleInterview(ctx, command.ScheduleInterview{
		HRID:            req.GetHrId(),
		ApplicationID:   req.GetApplicationId(),
		InterviewerID:   req.GetInterviewerId(),
		RoundNo:         req.GetRoundNo(),
		Title:           req.GetTitle(),
		Mode:            req.GetMode(),
		MeetingURL:      req.GetMeetingUrl(),
		Location:        req.GetLocation(),
		DurationMinutes: req.GetDurationMinutes(),
		CandidateNote:   req.GetCandidateNote(),
		InternalNote:    req.GetInternalNote(),
		ScheduledAt:     scheduledAt,
	})
	if err != nil {
		code, _, grpcErr := classifyError(err, "面试安排失败", errorMessages{applicationNotFound: "投递记录不存在"})
		if grpcErr != nil {
			return nil, grpcErr
		}
		return &pb.ScheduleInterviewResponse{Code: code, Msg: i18n.KeyForCode(code)}, nil
	}
	return &pb.ScheduleInterviewResponse{Code: errs.OK, Msg: "common.success", InterviewId: interviewID}, nil
}

func (s *Server) UpdateInterview(ctx context.Context, req *pb.UpdateInterviewRequest) (*pb.CommonResponse, error) {
	scheduledAt, err := mapper.ParseOptionalRFC3339(req.GetScheduledAt())
	if err != nil {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "common.invalid_request"}, nil
	}
	return commonResponse(s.interviews.UpdateInterview(ctx, command.UpdateInterview{
		HRID:            req.GetHrId(),
		InterviewID:     req.GetInterviewId(),
		Title:           req.GetTitle(),
		Mode:            req.GetMode(),
		MeetingURL:      req.GetMeetingUrl(),
		Location:        req.GetLocation(),
		DurationMinutes: req.GetDurationMinutes(),
		CandidateNote:   req.GetCandidateNote(),
		InternalNote:    req.GetInternalNote(),
		ScheduledAt:     scheduledAt,
	}), "面试信息已更新", "面试更新失败", errorMessages{interviewNotFound: "面试记录不存在"})
}

func (s *Server) CancelInterview(ctx context.Context, req *pb.CancelInterviewRequest) (*pb.CommonResponse, error) {
	return commonResponse(s.interviews.CancelInterview(ctx, command.CancelInterview{
		HRID:         req.GetHrId(),
		InterviewID:  req.GetInterviewId(),
		CancelReason: req.GetCancelReason(),
	}), "面试已取消", "取消面试失败", errorMessages{interviewNotFound: "面试记录不存在"})
}

func (s *Server) BatchCancelInterviews(ctx context.Context, req *pb.BatchCancelInterviewsRequest) (*pb.BatchCancelInterviewsResponse, error) {
	affected, err := s.interviews.BatchCancelInterviews(ctx, command.BatchCancelInterviews{
		HRID:          req.GetHrId(),
		ApplicationID: req.GetApplicationId(),
		CancelReason:  req.GetCancelReason(),
	})
	if err != nil {
		code, _, grpcErr := classifyError(err, "批量取消面试失败", errorMessages{applicationNotFound: "投递记录不存在"})
		if grpcErr != nil {
			return nil, grpcErr
		}
		return &pb.BatchCancelInterviewsResponse{Code: code, Msg: i18n.KeyForCode(code)}, nil
	}
	if affected == 0 {
		return &pb.BatchCancelInterviewsResponse{Code: errs.OK, Msg: "common.success", Affected: 0}, nil
	}
	return &pb.BatchCancelInterviewsResponse{Code: errs.OK, Msg: "common.success", Affected: affected}, nil
}

func (s *Server) GetInterview(ctx context.Context, req *pb.GetInterviewRequest) (*pb.GetInterviewResponse, error) {
	details, err := s.interviews.GetInterview(ctx, query.GetInterview{UserID: req.GetUserId(), InterviewID: req.GetInterviewId()})
	if err != nil {
		code, _, grpcErr := classifyError(err, "查询面试失败", errorMessages{
			interviewNotFound: "面试记录不存在",
			forbidden:         "无权限查看该面试",
		})
		if grpcErr != nil {
			return nil, grpcErr
		}
		return &pb.GetInterviewResponse{Code: code, Msg: i18n.KeyForCode(code)}, nil
	}
	return &pb.GetInterviewResponse{Code: errs.OK, Msg: "common.success", Interview: mapper.ToPBInterview(*details)}, nil
}

func (s *Server) ListInterviewers(ctx context.Context, req *pb.ListInterviewersRequest) (*pb.ListInterviewersResponse, error) {
	page, err := s.interviews.ListInterviewers(ctx, query.ListInterviewers{HRID: req.GetHrId(), Page: req.GetPage(), PageSize: req.GetPageSize(), Keyword: req.GetKeyword()})
	if err != nil {
		code, _, grpcErr := classifyError(err, "查询面试官失败", errorMessages{})
		if grpcErr != nil {
			return nil, grpcErr
		}
		return &pb.ListInterviewersResponse{Code: code, Msg: i18n.KeyForCode(code)}, nil
	}
	return &pb.ListInterviewersResponse{Code: errs.OK, Msg: "common.success", Total: page.Total, List: mapper.ToPBStaffUsers(page)}, nil
}

func (s *Server) ListApplicationInterviews(ctx context.Context, req *pb.ListApplicationInterviewsRequest) (*pb.ListApplicationInterviewsResponse, error) {
	rows, err := s.interviews.ListApplicationInterviews(ctx, query.ListApplicationInterviews{HRID: req.GetHrId(), ApplicationID: req.GetApplicationId()})
	if err != nil {
		code, _, grpcErr := classifyError(err, "查询面试失败", errorMessages{})
		if grpcErr != nil {
			return nil, grpcErr
		}
		return &pb.ListApplicationInterviewsResponse{Code: code, Msg: i18n.KeyForCode(code)}, nil
	}
	return &pb.ListApplicationInterviewsResponse{Code: errs.OK, Msg: "common.success", List: mapper.ToPBInterviews(rows)}, nil
}

func (s *Server) ListMyInterviews(ctx context.Context, req *pb.ListMyInterviewsRequest) (*pb.ListMyInterviewsResponse, error) {
	rows, err := s.interviews.ListMyInterviews(ctx, query.ListMyInterviews{InterviewerID: req.GetInterviewerId(), Status: model.InterviewStatus(req.GetStatus())})
	if err != nil {
		code, _, grpcErr := classifyError(err, "查询面试列表失败", errorMessages{forbidden: "无权限查看面试列表"})
		if grpcErr != nil {
			return nil, grpcErr
		}
		return &pb.ListMyInterviewsResponse{Code: code, Msg: i18n.KeyForCode(code)}, nil
	}
	return &pb.ListMyInterviewsResponse{Code: errs.OK, Msg: "common.success", List: mapper.ToPBInterviews(rows)}, nil
}

func (s *Server) ListCandidateInterviews(ctx context.Context, req *pb.ListCandidateInterviewsRequest) (*pb.ListCandidateInterviewsResponse, error) {
	rows, err := s.interviews.ListCandidateInterviews(ctx, query.ListCandidateInterviews{UserID: req.GetUserId()})
	if err != nil {
		code, _, grpcErr := classifyError(err, "查询候选人面试失败", errorMessages{})
		if grpcErr != nil {
			return nil, grpcErr
		}
		return &pb.ListCandidateInterviewsResponse{Code: code, Msg: i18n.KeyForCode(code)}, nil
	}
	return &pb.ListCandidateInterviewsResponse{Code: errs.OK, Msg: "common.success", List: mapper.ToPBInterviews(rows)}, nil
}

func (s *Server) SubmitFeedback(ctx context.Context, req *pb.SubmitFeedbackRequest) (*pb.CommonResponse, error) {
	return commonResponse(s.interviews.SubmitFeedback(ctx, command.SubmitFeedback{
		InterviewerID:       req.GetInterviewerId(),
		InterviewID:         req.GetInterviewId(),
		ApplicationID:       req.GetApplicationId(),
		Recommendation:      req.GetRecommendation(),
		Score:               req.GetScore(),
		DimensionScoresJSON: req.GetDimensionScoresJson(),
		Comments:            req.GetComments(),
	}), "面试反馈已提交", "提交面试反馈失败", errorMessages{})
}

func (s *Server) GetFeedback(ctx context.Context, req *pb.GetFeedbackRequest) (*pb.GetFeedbackResponse, error) {
	feedback, err := s.interviews.GetFeedback(ctx, query.GetFeedback{InterviewerID: req.GetInterviewerId(), InterviewID: req.GetInterviewId()})
	if err != nil {
		code, _, grpcErr := classifyError(err, "查询面试反馈失败", errorMessages{})
		if grpcErr != nil {
			return nil, grpcErr
		}
		return &pb.GetFeedbackResponse{Code: code, Msg: i18n.KeyForCode(code)}, nil
	}
	return &pb.GetFeedbackResponse{Code: errs.OK, Msg: "common.success", Feedback: mapper.ToPBFeedback(feedback)}, nil
}

func commonResponse(err error, successMsg string, fallback string, messages errorMessages) (*pb.CommonResponse, error) {
	if err == nil {
		return &pb.CommonResponse{Code: errs.OK, Msg: "common.success"}, nil
	}
	code, _, grpcErr := classifyError(err, fallback, messages)
	if grpcErr != nil {
		return nil, grpcErr
	}
	return &pb.CommonResponse{Code: code, Msg: i18n.KeyForCode(code)}, nil
}

type errorMessages struct {
	applicationNotFound string
	interviewNotFound   string
	forbidden           string
}

func classifyError(err error, fallback string, messages errorMessages) (int32, string, error) {
	if isActorVerificationError(err) {
		return 0, "", err
	}
	if errors.Is(err, appservice.ErrApplicationNotFound) {
		return errs.ErrBadRequest, messageOrDefault(messages.applicationNotFound, err.Error()), nil
	}
	if errors.Is(err, appservice.ErrInterviewNotFound) {
		return errs.ErrBadRequest, messageOrDefault(messages.interviewNotFound, err.Error()), nil
	}
	if errors.Is(err, model.ErrInterviewAlreadyCancelled) {
		return errs.ErrBadRequest, "该面试已取消", nil
	}
	if errors.Is(err, model.ErrInterviewerMismatch) {
		return errs.ErrForbidden, "您不是该面试的面试官，无法提交反馈", nil
	}
	if errors.Is(err, model.ErrFeedbackApplicationMismatch) {
		return errs.ErrBadRequest, "ApplicationId 与面试记录不匹配", nil
	}
	if errors.Is(err, model.ErrFeedbackTerminalApplication) {
		return errs.ErrForbidden, "该候选人已结束投递流程（已被淘汰或撤回），无法提交面试反馈", nil
	}
	if errors.Is(err, model.ErrFeedbackAlreadyExists) {
		return errs.ErrConflict, "您已提交过面试反馈，不可重复提交（如有更正需求请联系 HR）", nil
	}
	if errors.Is(err, model.ErrFeedbackRecommendationRequired) {
		return errs.ErrBadRequest, "请选择面试推荐结论", nil
	}
	if errors.Is(err, model.ErrFeedbackScoreOutOfRange) {
		return errs.ErrBadRequest, "评分范围为 0-100", nil
	}
	if errors.Is(err, model.ErrFeedbackInvalidRecommendation) {
		return errs.ErrBadRequest, "推荐结论值不合法", nil
	}
	if isForbiddenError(err) {
		return errs.ErrForbidden, messageOrDefault(messages.forbidden, err.Error()), nil
	}
	return 0, "", fmt.Errorf("%s: %w", fallback, err)
}

func messageOrDefault(message string, fallback string) string {
	if message != "" {
		return message
	}
	return fallback
}

func isActorVerificationError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "authenticated user not found") || strings.Contains(msg, "actor mismatch")
}

func isForbiddenError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "missing permission") ||
		strings.Contains(msg, "permission denied") ||
		strings.Contains(msg, "scope denied") ||
		strings.Contains(msg, "access denied") ||
		strings.Contains(msg, "scope lookup")
}
