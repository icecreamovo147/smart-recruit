package grpc

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"smart-recruit-offer-service/internal/application/command"
	"smart-recruit-offer-service/internal/application/query"
	appservice "smart-recruit-offer-service/internal/application/service"
	domainevent "smart-recruit-offer-service/internal/domain/event"
	"smart-recruit-offer-service/internal/domain/model"
	"smart-recruit-offer-service/internal/domain/repository"
	"smart-recruit-offer-service/internal/interfaces/mapper"
	"smart-recruit-platform-go/errs"
	"smart-recruit-proto/recruitment/pb"
)

type offerUsecase interface {
	CreateOffer(context.Context, command.CreateOffer) (int64, error)
	UpdateOffer(context.Context, command.UpdateOffer) error
	SendOffer(context.Context, command.SendOffer) error
	WithdrawOffer(context.Context, command.WithdrawOffer) error
	AcceptOffer(context.Context, command.AcceptOffer) error
	RejectOffer(context.Context, command.RejectOffer) error
	GetOffer(context.Context, query.GetOffer) (*repository.OfferDetails, error)
	ListOffersByApplication(context.Context, query.ListOffersByApplication) ([]repository.OfferDetails, error)
	ListMyOffers(context.Context, query.ListMyOffers) (repository.CandidateOfferPage, error)
	ListOfferEvents(context.Context, query.ListOfferEvents) ([]domainevent.OfferEvent, error)
}

type Server struct {
	pb.UnimplementedOfferServiceServer
	offers offerUsecase
}

func NewServer(offers offerUsecase) (*Server, error) {
	if offers == nil {
		return nil, fmt.Errorf("offer usecase is required")
	}
	return &Server{offers: offers}, nil
}

func (s *Server) CreateOffer(ctx context.Context, req *pb.CreateOfferRequest) (*pb.CreateOfferResponse, error) {
	expiresAt, err := mapper.ParseOptionalRFC3339(req.GetExpiresAt())
	if err != nil {
		return &pb.CreateOfferResponse{Code: errs.ErrBadRequest, Msg: "过期时间格式错误，请使用 RFC 3339 格式"}, nil
	}
	offerID, err := s.offers.CreateOffer(ctx, command.CreateOffer{
		HRID:          req.GetHrId(),
		ApplicationID: req.GetApplicationId(),
		Title:         req.GetTitle(),
		SalaryRange:   req.GetSalaryRange(),
		Level:         req.GetLevel(),
		WorkLocation:  req.GetWorkLocation(),
		StartDate:     req.GetStartDate(),
		ExpiresAt:     expiresAt,
		TermsJSON:     req.GetTermsJson(),
	})
	if err != nil {
		code, msg, grpcErr := classifyError(err, "Offer 创建失败", errorMessages{})
		if grpcErr != nil {
			return nil, grpcErr
		}
		return &pb.CreateOfferResponse{Code: code, Msg: msg}, nil
	}
	return &pb.CreateOfferResponse{Code: errs.OK, Msg: "Offer 创建成功", OfferId: offerID}, nil
}

func (s *Server) UpdateOffer(ctx context.Context, req *pb.UpdateOfferRequest) (*pb.CommonResponse, error) {
	expiresAt, err := mapper.ParseOptionalRFC3339(req.GetExpiresAt())
	if err != nil {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "过期时间格式错误，请使用 RFC 3339 格式"}, nil
	}
	err = s.offers.UpdateOffer(ctx, command.UpdateOffer{
		HRID:         req.GetHrId(),
		OfferID:      req.GetOfferId(),
		Title:        req.GetTitle(),
		SalaryRange:  req.GetSalaryRange(),
		Level:        req.GetLevel(),
		WorkLocation: req.GetWorkLocation(),
		StartDate:    req.GetStartDate(),
		ExpiresAt:    expiresAt,
		TermsJSON:    req.GetTermsJson(),
	})
	return commonResponse(err, "Offer 已更新", "Offer 更新失败", errorMessages{
		offerNotFound: "Offer 记录不存在",
	})
}

func (s *Server) GetOffer(ctx context.Context, req *pb.GetOfferRequest) (*pb.GetOfferResponse, error) {
	details, err := s.offers.GetOffer(ctx, query.GetOffer{UserID: req.GetUserId(), OfferID: req.GetOfferId()})
	if err != nil {
		code, msg, grpcErr := classifyError(err, "Offer 不存在", errorMessages{
			offerNotFound: "Offer 不存在",
			forbidden:     "无权限查看该 Offer",
		})
		if grpcErr != nil {
			return nil, grpcErr
		}
		return &pb.GetOfferResponse{Code: code, Msg: msg}, nil
	}
	return &pb.GetOfferResponse{Code: errs.OK, Msg: "success", Offer: mapper.ToPBOffer(*details)}, nil
}

func (s *Server) ListOffersByApplication(ctx context.Context, req *pb.ListOffersByApplicationRequest) (*pb.ListOffersByApplicationResponse, error) {
	rows, err := s.offers.ListOffersByApplication(ctx, query.ListOffersByApplication{HRID: req.GetHrId(), ApplicationID: req.GetApplicationId()})
	if err != nil {
		code, msg, grpcErr := classifyError(err, "查询 Offer 失败", errorMessages{})
		if grpcErr != nil {
			return nil, grpcErr
		}
		return &pb.ListOffersByApplicationResponse{Code: code, Msg: msg}, nil
	}
	return &pb.ListOffersByApplicationResponse{Code: errs.OK, Msg: "success", List: mapper.ToPBOffers(rows)}, nil
}

func (s *Server) SendOffer(ctx context.Context, req *pb.SendOfferRequest) (*pb.CommonResponse, error) {
	return commonResponse(
		s.offers.SendOffer(ctx, command.SendOffer{HRID: req.GetHrId(), OfferID: req.GetOfferId()}),
		"Offer 已发送",
		"Offer 发送失败",
		errorMessages{
			offerNotFound:     "Offer 不存在",
			sendPermission:    "无权限发送 Offer",
			offerNotSendable:  "仅可发送草稿状态的 Offer",
			offerNotDecidable: "仅可发送草稿状态的 Offer",
		},
	)
}

func (s *Server) WithdrawOffer(ctx context.Context, req *pb.WithdrawOfferRequest) (*pb.CommonResponse, error) {
	return commonResponse(
		s.offers.WithdrawOffer(ctx, command.WithdrawOffer{HRID: req.GetHrId(), OfferID: req.GetOfferId(), Reason: req.GetReason()}),
		"Offer 已撤回",
		"Offer 撤回失败",
		errorMessages{offerNotFound: "Offer 不存在"},
	)
}

func (s *Server) AcceptOffer(ctx context.Context, req *pb.AcceptOfferRequest) (*pb.CommonResponse, error) {
	return commonResponse(
		s.offers.AcceptOffer(ctx, command.AcceptOffer{UserID: req.GetUserId(), OfferID: req.GetOfferId()}),
		"Offer 已接受",
		"Offer 接受失败",
		errorMessages{
			offerNotFound:     "Offer 不存在",
			candidateMismatch: "您不是该 Offer 的候选人，无法接受",
			offerNotDecidable: "仅可接受已发送状态的 Offer",
		},
	)
}

func (s *Server) RejectOffer(ctx context.Context, req *pb.RejectOfferRequest) (*pb.CommonResponse, error) {
	return commonResponse(
		s.offers.RejectOffer(ctx, command.RejectOffer{UserID: req.GetUserId(), OfferID: req.GetOfferId(), Reason: req.GetReason()}),
		"Offer 已拒绝",
		"Offer 拒绝失败",
		errorMessages{
			offerNotFound:     "Offer 不存在",
			candidateMismatch: "您不是该 Offer 的候选人，无法拒绝",
			offerNotDecidable: "仅可拒绝已发送状态的 Offer",
		},
	)
}

func (s *Server) ListMyOffers(ctx context.Context, req *pb.ListMyOffersRequest) (*pb.ListMyOffersResponse, error) {
	page, err := s.offers.ListMyOffers(ctx, query.ListMyOffers{UserID: req.GetUserId(), Cursor: req.GetCursor(), PageSize: req.GetPageSize()})
	if err != nil {
		code, msg, grpcErr := classifyError(err, "查询 Offer 失败", errorMessages{})
		if grpcErr != nil {
			return nil, grpcErr
		}
		return &pb.ListMyOffersResponse{Code: code, Msg: msg}, nil
	}
	return &pb.ListMyOffersResponse{
		Code:       errs.OK,
		Msg:        "success",
		Total:      page.Total,
		List:       mapper.ToPBOffers(page.Offers),
		NextCursor: page.NextCursor,
		HasMore:    page.HasMore,
	}, nil
}

func (s *Server) ListOfferEvents(ctx context.Context, req *pb.ListOfferEventsRequest) (*pb.ListOfferEventsResponse, error) {
	events, err := s.offers.ListOfferEvents(ctx, query.ListOfferEvents{HRID: req.GetHrId(), OfferID: req.GetOfferId()})
	if err != nil {
		code, msg, grpcErr := classifyError(err, "查询 Offer 事件失败", errorMessages{
			offerNotFound: "Offer 不存在",
			forbidden:     "无权限查看该 Offer 事件",
		})
		if grpcErr != nil {
			return nil, grpcErr
		}
		return &pb.ListOfferEventsResponse{Code: code, Msg: msg}, nil
	}
	return &pb.ListOfferEventsResponse{Code: errs.OK, Msg: "success", List: mapper.ToPBOfferEvents(events)}, nil
}

func commonResponse(err error, successMsg string, fallback string, messages errorMessages) (*pb.CommonResponse, error) {
	if err == nil {
		return &pb.CommonResponse{Code: errs.OK, Msg: successMsg}, nil
	}
	code, msg, grpcErr := classifyError(err, fallback, messages)
	if grpcErr != nil {
		return nil, grpcErr
	}
	return &pb.CommonResponse{Code: code, Msg: msg}, nil
}

type errorMessages struct {
	candidateMismatch string
	forbidden         string
	offerNotFound     string
	offerNotDecidable string
	offerNotSendable  string
	sendPermission    string
}

func classifyError(err error, fallback string, messages errorMessages) (int32, string, error) {
	if isActorVerificationError(err) || errors.Is(err, appservice.ErrConcurrentStatus) {
		return 0, "", err
	}
	if errors.Is(err, model.ErrCandidateMismatch) {
		return errs.ErrForbidden, messageOrDefault(messages.candidateMismatch, err.Error()), nil
	}
	if errors.Is(err, appservice.ErrOfferNotFound) {
		return errs.ErrBadRequest, messageOrDefault(messages.offerNotFound, err.Error()), nil
	}
	if errors.Is(err, model.ErrOfferNotDecidable) {
		return errs.ErrBadRequest, messageOrDefault(messages.offerNotDecidable, err.Error()), nil
	}
	if errors.Is(err, model.ErrOfferNotSendable) {
		return errs.ErrBadRequest, messageOrDefault(messages.offerNotSendable, err.Error()), nil
	}
	if isBusinessError(err) {
		return errs.ErrBadRequest, err.Error(), nil
	}
	if messages.sendPermission != "" && strings.Contains(err.Error(), `missing permission "offer.send"`) {
		return errs.ErrForbidden, messages.sendPermission, nil
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
		strings.Contains(msg, "no valid data scope") ||
		strings.Contains(msg, "scope lookup") ||
		strings.Contains(msg, "not found")
}

func isBusinessError(err error) bool {
	var transitionErr *model.TransitionError
	return errors.Is(err, appservice.ErrApplicationNotFound) ||
		errors.Is(err, model.ErrOfferNotDraft) ||
		errors.Is(err, model.ErrOfferNotWithdrawable) ||
		errors.Is(err, model.ErrOfferExpired) ||
		errors.As(err, &transitionErr)
}
