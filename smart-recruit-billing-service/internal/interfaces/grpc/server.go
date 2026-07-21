package grpc

import (
	"context"
	"errors"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"smart-recruit-billing-service/internal/application/service"
	"smart-recruit-billing-service/internal/domain/model"
	"smart-recruit-platform-go/businessclock"
	"smart-recruit-proto/recruitment/pb"
)

type Server struct {
	pb.UnimplementedBillingServiceServer
	billing  *service.Billing
	commerce *service.Commerce
}

func NewServer(billing *service.Billing, commerce *service.Commerce) (*Server, error) {
	if billing == nil {
		return nil, errors.New("billing application service is required")
	}
	if commerce == nil {
		return nil, errors.New("billing commerce service is required")
	}
	return &Server{billing: billing, commerce: commerce}, nil
}

func (s *Server) CheckAIAccess(ctx context.Context, req *pb.CheckAIAccessRequest) (*pb.CheckAIAccessResponse, error) {
	owner, err := ownerFromProto(req.GetOwner())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if req.GetEstimatedCredits() < 0 {
		return nil, status.Error(codes.InvalidArgument, "estimated credits cannot be negative")
	}
	decision, err := s.billing.CheckAccess(ctx, owner, req.GetCapability(), uint64(req.GetEstimatedCredits()))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.CheckAIAccessResponse{Code: 0, Msg: "ok", Allowed: decision.Allowed, Reason: decision.Reason, AvailableCredits: int64(decision.Balance.AvailableCredits), EnforcementMode: modeToProto(decision.Mode), CapabilityVersionId: int64(decision.CapabilityVersionID)}, nil
}

func (s *Server) ReserveAIUsage(ctx context.Context, req *pb.ReserveAIUsageRequest) (*pb.ReserveAIUsageResponse, error) {
	owner, err := ownerFromProto(req.GetOwner())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if req.GetUserId() <= 0 || req.GetEstimatedCredits() < 0 || req.GetTtlSeconds() < 0 {
		return nil, status.Error(codes.InvalidArgument, "user, credits or TTL is invalid")
	}
	result, err := s.billing.Reserve(ctx, service.ReserveCommand{
		Owner: owner, UserID: uint64(req.GetUserId()), Capability: req.GetCapability(), Operation: req.GetOperation(),
		ProviderKey: req.GetProviderKey(), ModelKey: req.GetModelKey(), EstimatedCredits: uint64(req.GetEstimatedCredits()),
		IdempotencyKey: req.GetIdempotencyKey(), TTL: time.Duration(req.GetTtlSeconds()) * time.Second,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	response := &pb.ReserveAIUsageResponse{Code: 0, Msg: "ok", Allowed: result.Allowed, Reason: result.Reason, AvailableCredits: int64(result.Balance.AvailableCredits), EnforcementMode: modeToProto(s.billing.Mode()), CapabilityVersionId: int64(result.CapabilityVersionID)}
	if result.Allowed {
		response.ReservationNo = result.Reservation.No
		response.ReservedCredits = int64(result.Reservation.ReservedCredits)
		response.ExpiresAtUnixMs = result.Reservation.ExpiresAt.UnixMilli()
	}
	return response, nil
}

func (s *Server) SettleAIUsage(ctx context.Context, req *pb.SettleAIUsageRequest) (*pb.SettleAIUsageResponse, error) {
	usages := make([]model.ProviderUsage, 0, len(req.GetProviderUsages()))
	for _, item := range req.GetProviderUsages() {
		if item.GetProviderCallSeq() <= 0 || item.GetInputTokens() < 0 || item.GetOutputTokens() < 0 || item.GetCachedInputTokens() < 0 {
			return nil, status.Error(codes.InvalidArgument, "provider usage contains invalid values")
		}
		occurredAt := time.Time{}
		if item.GetOccurredAtUnixMs() > 0 {
			occurredAt = time.UnixMilli(item.GetOccurredAtUnixMs()).In(businessclock.Location)
		}
		usages = append(usages, model.ProviderUsage{
			CallSequence: uint32(item.GetProviderCallSeq()), ProviderKey: item.GetProviderKey(), ModelKey: item.GetModelKey(),
			InputTokens: uint64(item.GetInputTokens()), OutputTokens: uint64(item.GetOutputTokens()), CachedInputTokens: uint64(item.GetCachedInputTokens()),
			ProviderRequestID: item.GetProviderRequestId(), Estimated: item.GetEstimated(), OccurredAt: occurredAt, MetadataJSON: item.GetMetadataJson(),
		})
	}
	result, err := s.billing.Settle(ctx, req.GetReservationNo(), usages, req.GetIdempotencyKey())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.SettleAIUsageResponse{Code: 0, Msg: "ok", ChargedCredits: int64(result.ChargedCredits), SupplierCostMicros: int64(result.SupplierCostMicros), AvailableCredits: int64(result.AvailableCredits), AlreadySettled: result.AlreadySettled}, nil
}

func (s *Server) CancelAIUsage(ctx context.Context, req *pb.CancelAIUsageRequest) (*pb.CancelAIUsageResponse, error) {
	result, err := s.billing.Cancel(ctx, req.GetReservationNo(), req.GetReason(), req.GetIdempotencyKey())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.CancelAIUsageResponse{Code: 0, Msg: "ok", ReleasedCredits: int64(result.ReleasedCredits), AvailableCredits: int64(result.AvailableCredits), AlreadyCancelled: result.AlreadyCancelled}, nil
}

func (s *Server) GetAICreditBalance(ctx context.Context, req *pb.GetAICreditBalanceRequest) (*pb.GetAICreditBalanceResponse, error) {
	owner, err := ownerFromProto(req.GetOwner())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	balance, err := s.billing.Balance(ctx, owner)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	response := &pb.GetAICreditBalanceResponse{Code: 0, Msg: "ok", AvailableCredits: int64(balance.AvailableCredits), ReservedCredits: int64(balance.ReservedCredits), EnforcementMode: modeToProto(s.billing.Mode())}
	if balance.NextExpiryAt != nil {
		response.NextExpiryAtUnixMs = balance.NextExpiryAt.UnixMilli()
	}
	return response, nil
}

func (s *Server) ListBillingCatalog(ctx context.Context, req *pb.ListBillingCatalogRequest) (*pb.ListBillingCatalogResponse, error) {
	owner, err := ownerFromProto(req.GetOwner())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	products, err := s.commerce.ListCatalog(ctx, owner)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.ListBillingCatalogResponse{Code: 0, Msg: "ok", Products: products, PaymentEnvironment: s.commerce.Environment()}, nil
}

func (s *Server) GetBillingAccount(ctx context.Context, req *pb.GetBillingAccountRequest) (*pb.GetBillingAccountResponse, error) {
	owner, err := ownerFromProto(req.GetOwner())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	// Balance provisions the owner's recurring grant before the aggregate read,
	// so the account response is internally consistent on first visit.
	balance, err := s.billing.Balance(ctx, owner)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	subscription, err := s.commerce.CurrentSubscription(ctx, owner)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	scheduledSubscription, err := s.commerce.ScheduledSubscription(ctx, owner)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	totalCredits, usedCredits, err := s.commerce.CurrentCreditSummary(ctx, owner)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	response := &pb.GetBillingAccountResponse{
		Code: 0, Msg: "ok", Subscription: subscription, ScheduledSubscription: scheduledSubscription,
		AvailableCredits: int64(balance.AvailableCredits), ReservedCredits: int64(balance.ReservedCredits),
		PaymentEnvironment: s.commerce.Environment(), TotalCredits: totalCredits, UsedCredits: usedCredits,
		NextRefreshAtUnixMs: s.commerce.NextRefreshAt(subscription),
	}
	if balance.NextExpiryAt != nil {
		response.NextExpiryAtUnixMs = balance.NextExpiryAt.UnixMilli()
	}
	return response, nil
}

func (s *Server) ListBillingOrders(ctx context.Context, req *pb.ListBillingOrdersRequest) (*pb.ListBillingOrdersResponse, error) {
	owner, err := ownerFromProto(req.GetOwner())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	orders, total, err := s.commerce.ListOrders(ctx, owner, req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.ListBillingOrdersResponse{Code: 0, Msg: "ok", Orders: orders, Total: total}, nil
}

func (s *Server) CreateBillingOrder(ctx context.Context, req *pb.CreateBillingOrderRequest) (*pb.BillingOrderResponse, error) {
	owner, err := ownerFromProto(req.GetOwner())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if req.GetActorUserId() <= 0 || req.GetPriceVersionId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "actor and price are required")
	}
	order, err := s.commerce.CreateOrder(ctx, owner, uint64(req.GetActorUserId()), uint64(req.GetPriceVersionId()), req.GetOrderType(), req.GetIdempotencyKey(), req.GetReplacePendingOrder())
	if err != nil {
		return nil, createBillingOrderError(err)
	}
	return &pb.BillingOrderResponse{Code: 0, Msg: "ok", Order: order}, nil
}

func createBillingOrderError(err error) error {
	switch {
	case errors.Is(err, service.ErrPendingBillingOrder):
		return status.Error(codes.AlreadyExists, "存在尚未支付的订单，请选择继续支付或创建新订单")
	case errors.Is(err, service.ErrSubscriptionRenewalScheduled):
		return status.Error(codes.FailedPrecondition, "当前套餐已完成续费，下一周期套餐将在生效日自动启用，无需重复购买")
	case errors.Is(err, service.ErrBillingPaymentCompleted):
		return status.Error(codes.FailedPrecondition, "原订单已经支付成功，请刷新套餐与订单状态")
	case errors.Is(err, service.ErrBillingOrderNotPayable):
		return status.Error(codes.FailedPrecondition, "原订单状态已经变化，请刷新后重试")
	default:
		return status.Error(codes.InvalidArgument, err.Error())
	}
}

func (s *Server) CreateAlipayPayment(ctx context.Context, req *pb.CreateAlipayPaymentRequest) (*pb.CreateAlipayPaymentResponse, error) {
	owner, err := ownerFromProto(req.GetOwner())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if req.GetScene() == "sync" {
		paymentNo, syncErr := s.commerce.SyncPayment(ctx, owner, req.GetOrderNo())
		if syncErr != nil {
			switch {
			case errors.Is(syncErr, service.ErrBillingPaymentPending):
				return nil, status.Error(codes.FailedPrecondition, "支付结果确认中，请稍候")
			case errors.Is(syncErr, service.ErrBillingOrderNotPayable):
				return nil, status.Error(codes.FailedPrecondition, "当前订单无法继续支付，请刷新订单状态")
			default:
				return nil, status.Error(codes.FailedPrecondition, "支付宝沙箱支付暂不可用，请检查沙箱配置或稍后重试")
			}
		}
		return &pb.CreateAlipayPaymentResponse{Code: 0, Msg: "ok", PaymentNo: paymentNo, RedirectUrl: "", PaymentEnvironment: s.commerce.Environment()}, nil
	}
	paymentNo, redirectURL, reused, expiresAt, err := s.commerce.CreatePayment(ctx, owner, req.GetOrderNo(), req.GetScene())
	if err != nil {
		switch {
		case errors.Is(err, service.ErrBillingOrderExpired):
			return nil, status.Error(codes.FailedPrecondition, "订单已超过支付时限并关闭，请重新创建订单")
		case errors.Is(err, service.ErrBillingOrderNotPayable):
			return nil, status.Error(codes.FailedPrecondition, "当前订单无法继续支付，请刷新订单状态")
		case errors.Is(err, service.ErrBillingPaymentCompleted):
			return nil, status.Error(codes.FailedPrecondition, "订单已经支付成功，请刷新套餐与订单状态")
		default:
			return nil, status.Error(codes.FailedPrecondition, "支付宝沙箱支付暂不可用，请检查沙箱配置或稍后重试")
		}
	}
	return &pb.CreateAlipayPaymentResponse{Code: 0, Msg: "ok", PaymentNo: paymentNo, RedirectUrl: redirectURL, PaymentEnvironment: s.commerce.Environment(), Reused: reused, Status: "pending", ExpiresAtUnixMs: expiresAt.UnixMilli()}, nil
}

func (s *Server) RequestBillingRefund(ctx context.Context, req *pb.RequestBillingRefundRequest) (*pb.BillingRefundResponse, error) {
	owner, err := ownerFromProto(req.GetOwner())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if req.GetActorUserId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "refund actor is required")
	}
	refundNo, statusValue, reviewMode, err := s.commerce.RequestRefund(ctx, owner, uint64(req.GetActorUserId()), req.GetOrderNo(), req.GetReason(), req.GetIdempotencyKey())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.BillingRefundResponse{Code: 0, Msg: "ok", RefundNo: refundNo, Status: statusValue, ReviewMode: reviewMode}, nil
}

func (s *Server) ProcessAlipayNotification(ctx context.Context, req *pb.ProcessAlipayNotificationRequest) (*pb.ProcessAlipayNotificationResponse, error) {
	fields := make(map[string]string, len(req.GetFields()))
	for _, field := range req.GetFields() {
		if field.GetKey() != "" {
			fields[field.GetKey()] = field.GetValue()
		}
	}
	if err := s.commerce.ProcessAlipayNotification(ctx, fields); err != nil {
		if errors.Is(err, service.ErrInvalidAlipayNotification) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.ProcessAlipayNotificationResponse{Code: 0, Msg: "ok", Accepted: true}, nil
}

func (s *Server) ResolveAlipayReturn(ctx context.Context, req *pb.ResolveAlipayReturnRequest) (*pb.ResolveAlipayReturnResponse, error) {
	fields := make(map[string]string, len(req.GetFields()))
	for _, field := range req.GetFields() {
		if field.GetKey() != "" {
			fields[field.GetKey()] = field.GetValue()
		}
	}
	sourceApp, returnToken, err := s.commerce.ResolveAlipayReturn(ctx, fields)
	if err != nil {
		if errors.Is(err, service.ErrInvalidAlipayNotification) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.ResolveAlipayReturnResponse{Code: 0, Msg: "ok", SourceApp: sourceApp, ReturnToken: returnToken}, nil
}

func (s *Server) SyncAlipayReturn(ctx context.Context, req *pb.SyncAlipayReturnRequest) (*pb.CreateAlipayPaymentResponse, error) {
	owner, err := ownerFromProto(req.GetOwner())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	paymentNo, err := s.commerce.SyncPaymentReturn(ctx, owner, req.GetReturnToken())
	if err != nil {
		if errors.Is(err, service.ErrBillingPaymentPending) {
			return nil, status.Error(codes.FailedPrecondition, "支付结果确认中，请稍候")
		}
		return nil, status.Error(codes.FailedPrecondition, "当前支付返回无法确认，请刷新订单状态")
	}
	return &pb.CreateAlipayPaymentResponse{Code: 0, Msg: "ok", PaymentNo: paymentNo, PaymentEnvironment: s.commerce.Environment(), Status: "succeeded"}, nil
}

func (s *Server) ListBillingRefunds(ctx context.Context, req *pb.ListBillingRefundsRequest) (*pb.ListBillingRefundsResponse, error) {
	refunds, total, err := s.commerce.ListRefunds(ctx, req.GetStatus(), req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.ListBillingRefundsResponse{Code: 0, Msg: "ok", Refunds: refunds, Total: total}, nil
}

func (s *Server) ReviewBillingRefund(ctx context.Context, req *pb.ReviewBillingRefundRequest) (*pb.BillingRefundResponse, error) {
	refundNo, statusValue, err := s.commerce.ReviewRefund(ctx, req.GetRefundNo(), uint64(req.GetActorUserId()), req.GetAction(), req.GetReason())
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}
	return &pb.BillingRefundResponse{Code: 0, Msg: "ok", RefundNo: refundNo, Status: statusValue, ReviewMode: "manual"}, nil
}

func (s *Server) ListBillingAdminCatalog(ctx context.Context, _ *pb.ListBillingAdminCatalogRequest) (*pb.ListBillingCatalogResponse, error) {
	products, err := s.commerce.ListAdminCatalog(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.ListBillingCatalogResponse{Code: 0, Msg: "ok", Products: products, PaymentEnvironment: s.commerce.Environment()}, nil
}

func (s *Server) SaveBillingPriceVersion(ctx context.Context, req *pb.SaveBillingPriceVersionRequest) (*pb.BillingPriceInfo, error) {
	if req.GetProductId() <= 0 || req.GetPriceVersionId() < 0 || req.GetAmountFen() <= 0 || req.GetIncludedCredits() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "product, amount and credits are required")
	}
	price, err := s.commerce.SavePriceVersion(ctx, uint64(req.GetProductId()), uint64(req.GetPriceVersionId()), req.GetBillingTerm(), uint64(req.GetAmountFen()), uint64(req.GetIncludedCredits()), req.GetEntitlementSnapshotJson(), req.GetPublish())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return price, nil
}

func (s *Server) ListAIRateCards(ctx context.Context, _ *pb.ListAIRateCardsRequest) (*pb.ListAIRateCardsResponse, error) {
	rates, err := s.commerce.ListRateCards(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.ListAIRateCardsResponse{Code: 0, Msg: "ok", Rates: rates}, nil
}

func (s *Server) SaveAIRateCard(ctx context.Context, req *pb.SaveAIRateCardRequest) (*pb.AIRateCardInfo, error) {
	if req.GetInputMicrosPer_1KTokens() < 0 || req.GetOutputMicrosPer_1KTokens() < 0 || req.GetCachedInputMicrosPer_1KTokens() < 0 || req.GetCreditMicros() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "AI rate values are invalid")
	}
	rate, err := s.commerce.SaveRateCard(ctx, req.GetProviderKey(), req.GetModelKey(), uint64(req.GetInputMicrosPer_1KTokens()), uint64(req.GetOutputMicrosPer_1KTokens()), uint64(req.GetCachedInputMicrosPer_1KTokens()), uint64(req.GetCreditMicros()), req.GetPublish())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return rate, nil
}

func ownerFromProto(value *pb.BillingOwner) (model.Owner, error) {
	if value == nil || value.GetId() <= 0 {
		return model.Owner{}, errors.New("billing owner is required")
	}
	owner := model.Owner{ID: uint64(value.GetId())}
	switch value.GetType() {
	case pb.BillingOwnerType_BILLING_OWNER_TYPE_TENANT:
		owner.Type = model.OwnerTenant
	case pb.BillingOwnerType_BILLING_OWNER_TYPE_USER:
		owner.Type = model.OwnerUser
	default:
		return model.Owner{}, errors.New("billing owner type is invalid")
	}
	return owner, owner.Validate()
}

func modeToProto(value model.EnforcementMode) pb.BillingEnforcementMode {
	if value == model.ModeEnforce {
		return pb.BillingEnforcementMode_BILLING_ENFORCEMENT_MODE_ENFORCE
	}
	return pb.BillingEnforcementMode_BILLING_ENFORCEMENT_MODE_SHADOW
}
