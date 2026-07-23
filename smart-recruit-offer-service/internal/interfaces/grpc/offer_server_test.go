package grpc

import (
	"context"
	"errors"
	"testing"
	"time"

	"smart-recruit-offer-service/internal/application/command"
	"smart-recruit-offer-service/internal/application/query"
	appservice "smart-recruit-offer-service/internal/application/service"
	domainevent "smart-recruit-offer-service/internal/domain/event"
	"smart-recruit-offer-service/internal/domain/model"
	"smart-recruit-offer-service/internal/domain/repository"
	"smart-recruit-platform-go/errs"
	"smart-recruit-proto/recruitment/pb"
)

func TestCreateOfferRejectsInvalidExpiry(t *testing.T) {
	server := newTestServer(t, &fakeOfferUsecase{})
	resp, err := server.CreateOffer(context.Background(), &pb.CreateOfferRequest{ExpiresAt: "tomorrow"})
	if err != nil {
		t.Fatalf("CreateOffer returned grpc error: %v", err)
	}
	if resp.Code != errs.ErrBadRequest {
		t.Fatalf("code=%d, want bad request", resp.Code)
	}
}

func TestCreateOfferMapsCommandAndSuccessResponse(t *testing.T) {
	fake := &fakeOfferUsecase{createdID: 42}
	server := newTestServer(t, fake)
	expires := "2026-08-01T10:00:00Z"
	resp, err := server.CreateOffer(context.Background(), &pb.CreateOfferRequest{
		HrId:          100,
		ApplicationId: 10,
		Title:         "Offer",
		ExpiresAt:     expires,
	})
	if err != nil {
		t.Fatalf("CreateOffer returned grpc error: %v", err)
	}
	if resp.Code != errs.OK || resp.OfferId != 42 {
		t.Fatalf("response=%+v, want OK offer_id=42", resp)
	}
	if fake.create.HRID != 100 || fake.create.ApplicationID != 10 || fake.create.ExpiresAt == nil {
		t.Fatalf("command not mapped: %+v", fake.create)
	}
}

func TestGetOfferMapsNotFoundToBadRequest(t *testing.T) {
	server := newTestServer(t, &fakeOfferUsecase{getErr: appservice.ErrOfferNotFound})
	resp, err := server.GetOffer(context.Background(), &pb.GetOfferRequest{UserId: 20, OfferId: 404})
	if err != nil {
		t.Fatalf("GetOffer returned grpc error: %v", err)
	}
	if resp.Code != errs.ErrBadRequest || resp.Msg != "Offer 不存在" {
		t.Fatalf("response=%+v, want legacy not found response", resp)
	}
}

func TestGetOfferHidesReadScopeFailure(t *testing.T) {
	server := newTestServer(t, &fakeOfferUsecase{getErr: errors.New("scope denied for user 100")})
	resp, err := server.GetOffer(context.Background(), &pb.GetOfferRequest{UserId: 100, OfferId: 1})
	if err != nil {
		t.Fatalf("GetOffer returned grpc error: %v", err)
	}
	if resp.Code != errs.ErrForbidden || resp.Msg != "无权限查看该 Offer" {
		t.Fatalf("response=%+v, want legacy forbidden response", resp)
	}
}

func TestAcceptOfferMapsCandidateMismatchToForbidden(t *testing.T) {
	server := newTestServer(t, &fakeOfferUsecase{acceptErr: model.ErrCandidateMismatch})
	resp, err := server.AcceptOffer(context.Background(), &pb.AcceptOfferRequest{UserId: 99, OfferId: 1})
	if err != nil {
		t.Fatalf("AcceptOffer returned grpc error: %v", err)
	}
	if resp.Code != errs.ErrForbidden || resp.Msg != "您不是该 Offer 的候选人，无法接受" {
		t.Fatalf("response=%+v, want legacy accept mismatch response", resp)
	}
}

func TestRejectOfferMapsCandidateMismatchToForbidden(t *testing.T) {
	server := newTestServer(t, &fakeOfferUsecase{rejectErr: model.ErrCandidateMismatch})
	resp, err := server.RejectOffer(context.Background(), &pb.RejectOfferRequest{UserId: 99, OfferId: 1})
	if err != nil {
		t.Fatalf("RejectOffer returned grpc error: %v", err)
	}
	if resp.Code != errs.ErrForbidden || resp.Msg != "您不是该 Offer 的候选人，无法拒绝" {
		t.Fatalf("response=%+v, want legacy reject mismatch response", resp)
	}
}

func TestAcceptRejectUseEndpointSpecificNotSentMessages(t *testing.T) {
	acceptServer := newTestServer(t, &fakeOfferUsecase{acceptErr: model.ErrOfferNotDecidable})
	acceptResp, err := acceptServer.AcceptOffer(context.Background(), &pb.AcceptOfferRequest{UserId: 20, OfferId: 1})
	if err != nil {
		t.Fatalf("AcceptOffer returned grpc error: %v", err)
	}
	if acceptResp.Code != errs.ErrBadRequest || acceptResp.Msg != "仅可接受已发送状态的 Offer" {
		t.Fatalf("accept response=%+v, want legacy not-sent response", acceptResp)
	}

	rejectServer := newTestServer(t, &fakeOfferUsecase{rejectErr: model.ErrOfferNotDecidable})
	rejectResp, err := rejectServer.RejectOffer(context.Background(), &pb.RejectOfferRequest{UserId: 20, OfferId: 1})
	if err != nil {
		t.Fatalf("RejectOffer returned grpc error: %v", err)
	}
	if rejectResp.Code != errs.ErrBadRequest || rejectResp.Msg != "仅可拒绝已发送状态的 Offer" {
		t.Fatalf("reject response=%+v, want legacy not-sent response", rejectResp)
	}
}

func TestSendOfferMapsSendPermissionMessage(t *testing.T) {
	server := newTestServer(t, &fakeOfferUsecase{sendErr: errors.New(`actor 100 missing permission "offer.send"`)})
	resp, err := server.SendOffer(context.Background(), &pb.SendOfferRequest{HrId: 100, OfferId: 1})
	if err != nil {
		t.Fatalf("SendOffer returned grpc error: %v", err)
	}
	if resp.Code != errs.ErrForbidden || resp.Msg != "无权限发送 Offer" {
		t.Fatalf("response=%+v, want legacy send permission response", resp)
	}
}

func TestListOfferEventsMapsDomainEvents(t *testing.T) {
	createdAt := time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)
	server := newTestServer(t, &fakeOfferUsecase{
		events: []domainevent.OfferEvent{{
			ID:               7,
			OfferID:          3,
			EventType:        domainevent.TypeSent,
			ActorUserID:      100,
			ActorAccountType: "staff",
			CreatedAt:        createdAt,
		}},
	})
	resp, err := server.ListOfferEvents(context.Background(), &pb.ListOfferEventsRequest{HrId: 100, OfferId: 3})
	if err != nil {
		t.Fatalf("ListOfferEvents returned grpc error: %v", err)
	}
	if resp.Code != errs.OK || len(resp.List) != 1 || resp.List[0].CreatedAt != "2026-07-13T18:00:00+08:00" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestListOfferEventsHidesReadScopeFailure(t *testing.T) {
	server := newTestServer(t, &fakeOfferUsecase{eventsErr: errors.New("scope denied for user 100")})
	resp, err := server.ListOfferEvents(context.Background(), &pb.ListOfferEventsRequest{HrId: 100, OfferId: 3})
	if err != nil {
		t.Fatalf("ListOfferEvents returned grpc error: %v", err)
	}
	if resp.Code != errs.ErrForbidden || resp.Msg != "无权限查看该 Offer 事件" {
		t.Fatalf("response=%+v, want legacy event forbidden response", resp)
	}
}

func newTestServer(t *testing.T, fake *fakeOfferUsecase) *Server {
	t.Helper()
	server, err := NewServer(fake)
	if err != nil {
		t.Fatalf("NewServer returned error: %v", err)
	}
	return server
}

type fakeOfferUsecase struct {
	createdID int64
	create    command.CreateOffer
	getErr    error
	sendErr   error
	acceptErr error
	rejectErr error
	events    []domainevent.OfferEvent
	eventsErr error
}

func (f *fakeOfferUsecase) CreateOffer(_ context.Context, cmd command.CreateOffer) (int64, error) {
	f.create = cmd
	return f.createdID, nil
}

func (f *fakeOfferUsecase) UpdateOffer(context.Context, command.UpdateOffer) error {
	return nil
}

func (f *fakeOfferUsecase) SendOffer(context.Context, command.SendOffer) error {
	return f.sendErr
}

func (f *fakeOfferUsecase) WithdrawOffer(context.Context, command.WithdrawOffer) error {
	return nil
}

func (f *fakeOfferUsecase) AcceptOffer(context.Context, command.AcceptOffer) error {
	return f.acceptErr
}

func (f *fakeOfferUsecase) RejectOffer(context.Context, command.RejectOffer) error {
	return f.rejectErr
}

func (f *fakeOfferUsecase) GetOffer(context.Context, query.GetOffer) (*repository.OfferDetails, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return &repository.OfferDetails{Offer: model.Offer{ID: 1}}, nil
}

func (f *fakeOfferUsecase) ListOffersByApplication(context.Context, query.ListOffersByApplication) ([]repository.OfferDetails, error) {
	return nil, nil
}

func (f *fakeOfferUsecase) ListMyOffers(context.Context, query.ListMyOffers) (repository.CandidateOfferPage, error) {
	return repository.CandidateOfferPage{}, nil
}

func (f *fakeOfferUsecase) ListOfferEvents(context.Context, query.ListOfferEvents) ([]domainevent.OfferEvent, error) {
	if f.eventsErr != nil {
		return nil, f.eventsErr
	}
	return f.events, nil
}
