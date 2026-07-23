package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"smart-recruit-gateway/middleware"
	"smart-recruit-gateway/pkg/logger"
	"smart-recruit-gateway/rpc"
	"smart-recruit-proto/recruitment/pb"
)

type BillingHandler struct {
	clients   *rpc.Clients
	ownerType pb.BillingOwnerType
}

type saveBillingPriceRequest struct {
	ProductID               FlexInt64 `json:"product_id" binding:"required"`
	PriceVersionID          FlexInt64 `json:"price_version_id"`
	BillingTerm             string    `json:"billing_term" binding:"required"`
	AmountFen               FlexInt64 `json:"amount_fen" binding:"required"`
	IncludedCredits         FlexInt64 `json:"included_credits" binding:"required"`
	EntitlementSnapshotJSON string    `json:"entitlement_snapshot_json"`
}

type createBillingOrderRequest struct {
	PriceVersionID      FlexInt64 `json:"price_version_id" binding:"required"`
	OrderType           string    `json:"order_type" binding:"required"`
	IdempotencyKey      string    `json:"idempotency_key" binding:"required"`
	ReplacePendingOrder bool      `json:"replace_pending_order"`
}

func NewBillingHandler(clients *rpc.Clients, ownerType pb.BillingOwnerType) *BillingHandler {
	return &BillingHandler{clients: clients, ownerType: ownerType}
}

func (h *BillingHandler) owner(c *gin.Context) *pb.BillingOwner {
	id := middleware.UserID(c)
	if h.ownerType == pb.BillingOwnerType_BILLING_OWNER_TYPE_TENANT {
		id = middleware.TenantID(c)
	}
	return &pb.BillingOwner{Type: h.ownerType, Id: id}
}

func (h *BillingHandler) Catalog(c *gin.Context) {
	response, err := h.clients.Billing.ListBillingCatalog(c.Request.Context(), &pb.ListBillingCatalogRequest{Owner: h.owner(c)})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, response)
}
func (h *BillingHandler) Account(c *gin.Context) {
	response, err := h.clients.Billing.GetBillingAccount(c.Request.Context(), &pb.GetBillingAccountRequest{Owner: h.owner(c)})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, response)
}
func (h *BillingHandler) Orders(c *gin.Context) {
	response, err := h.clients.Billing.ListBillingOrders(c.Request.Context(), &pb.ListBillingOrdersRequest{Owner: h.owner(c), Page: queryInt32(c, "page", 1), PageSize: queryInt32(c, "page_size", 20)})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, response)
}

func (h *BillingHandler) CreateOrder(c *gin.Context) {
	var request createBillingOrderRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		BadRequest(c, "价格版本、订单类型和幂等键不能为空")
		return
	}
	response, err := h.clients.Billing.CreateBillingOrder(c.Request.Context(), &pb.CreateBillingOrderRequest{Owner: h.owner(c), ActorUserId: middleware.UserID(c), PriceVersionId: int64(request.PriceVersionID), OrderType: request.OrderType, IdempotencyKey: request.IdempotencyKey, ReplacePendingOrder: request.ReplacePendingOrder})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, response)
}

func (h *BillingHandler) Pay(c *gin.Context) {
	var request struct {
		Scene string `json:"scene" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil || (request.Scene != "desktop" && request.Scene != "wap" && request.Scene != "sync") {
		BadRequest(c, "支付场景必须是 desktop、wap 或 sync")
		return
	}
	response, err := h.clients.Billing.CreateAlipayPayment(c.Request.Context(), &pb.CreateAlipayPaymentRequest{Owner: h.owner(c), ActorUserId: middleware.UserID(c), OrderNo: c.Param("order_no"), Scene: request.Scene})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, response)
}

func (h *BillingHandler) SyncReturn(c *gin.Context) {
	token := strings.TrimSpace(c.Param("return_token"))
	if token == "" {
		BadRequest(c, "支付返回令牌不能为空")
		return
	}
	response, err := h.clients.Billing.SyncAlipayReturn(c.Request.Context(), &pb.SyncAlipayReturnRequest{Owner: h.owner(c), ActorUserId: middleware.UserID(c), ReturnToken: token})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, response)
}

func (h *BillingHandler) Refund(c *gin.Context) {
	var request struct {
		Reason         string `json:"reason" binding:"required"`
		IdempotencyKey string `json:"idempotency_key" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil || strings.TrimSpace(request.Reason) == "" || strings.TrimSpace(request.IdempotencyKey) == "" {
		BadRequest(c, "退款原因和幂等键不能为空")
		return
	}
	response, err := h.clients.Billing.RequestBillingRefund(c.Request.Context(), &pb.RequestBillingRefundRequest{Owner: h.owner(c), ActorUserId: middleware.UserID(c), OrderNo: c.Param("order_no"), Reason: strings.TrimSpace(request.Reason), IdempotencyKey: strings.TrimSpace(request.IdempotencyKey)})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, response)
}

func (h *BillingHandler) AdminCatalog(c *gin.Context) {
	response, err := h.clients.Billing.ListBillingAdminCatalog(c.Request.Context(), &pb.ListBillingAdminCatalogRequest{})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, response)
}

func (h *BillingHandler) AdminRefunds(c *gin.Context) {
	response, err := h.clients.Billing.ListBillingRefunds(c.Request.Context(), &pb.ListBillingRefundsRequest{Status: strings.TrimSpace(c.Query("status")), Page: queryInt32(c, "page", 1), PageSize: queryInt32(c, "page_size", 50)})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, response)
}

func (h *BillingHandler) ReviewRefund(c *gin.Context) {
	var request struct {
		Action string `json:"action" binding:"required"`
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&request); err != nil || (request.Action != "approve" && request.Action != "reject") {
		BadRequest(c, "退款审批动作必须是 approve 或 reject")
		return
	}
	response, err := h.clients.Billing.ReviewBillingRefund(c.Request.Context(), &pb.ReviewBillingRefundRequest{RefundNo: c.Param("refund_no"), ActorUserId: middleware.UserID(c), Action: request.Action, Reason: strings.TrimSpace(request.Reason)})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, response)
}

func (h *BillingHandler) SavePrice(c *gin.Context) {
	var request saveBillingPriceRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		BadRequest(c, "商品、计费周期、金额和额度不能为空")
		return
	}
	response, err := h.clients.Billing.SaveBillingPriceVersion(c.Request.Context(), &pb.SaveBillingPriceVersionRequest{ProductId: int64(request.ProductID), PriceVersionId: int64(request.PriceVersionID), BillingTerm: request.BillingTerm, AmountFen: int64(request.AmountFen), IncludedCredits: int64(request.IncludedCredits), EntitlementSnapshotJson: request.EntitlementSnapshotJSON, Publish: true})
	if err != nil {
		Internal(c, err)
		return
	}
	OK(c, "保存成功", response)
}

func (h *BillingHandler) AdminRateCards(c *gin.Context) {
	response, err := h.clients.Billing.ListAIRateCards(c.Request.Context(), &pb.ListAIRateCardsRequest{})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, response)
}

func (h *BillingHandler) SaveRateCard(c *gin.Context) {
	var request struct {
		ProviderKey                  string `json:"provider_key" binding:"required"`
		ModelKey                     string `json:"model_key" binding:"required"`
		InputMicrosPer1KTokens       int64  `json:"input_micros_per_1k_tokens"`
		OutputMicrosPer1KTokens      int64  `json:"output_micros_per_1k_tokens"`
		CachedInputMicrosPer1KTokens int64  `json:"cached_input_micros_per_1k_tokens"`
		CreditMicros                 int64  `json:"credit_micros" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil || request.InputMicrosPer1KTokens < 0 || request.OutputMicrosPer1KTokens < 0 || request.CachedInputMicrosPer1KTokens < 0 || request.CreditMicros <= 0 {
		BadRequest(c, "模型、供应商成本和额度换算不能为空")
		return
	}
	response, err := h.clients.Billing.SaveAIRateCard(c.Request.Context(), &pb.SaveAIRateCardRequest{
		ProviderKey: request.ProviderKey, ModelKey: request.ModelKey,
		InputMicrosPer_1KTokens:       request.InputMicrosPer1KTokens,
		OutputMicrosPer_1KTokens:      request.OutputMicrosPer1KTokens,
		CachedInputMicrosPer_1KTokens: request.CachedInputMicrosPer1KTokens,
		CreditMicros:                  request.CreditMicros, Publish: true,
	})
	if err != nil {
		Internal(c, err)
		return
	}
	OK(c, "保存成功", response)
}

type AlipayWebhookHandler struct {
	clients    *rpc.Clients
	returnURLs map[string]string
}

func NewAlipayWebhookHandler(clients *rpc.Clients, hrReturnURL, candidateReturnURL string) *AlipayWebhookHandler {
	return &AlipayWebhookHandler{clients: clients, returnURLs: map[string]string{"hr": hrReturnURL, "candidate": candidateReturnURL}}
}
func (h *AlipayWebhookHandler) Notify(c *gin.Context) {
	if err := c.Request.ParseForm(); err != nil {
		c.String(http.StatusBadRequest, "failure")
		return
	}
	fields := make([]*pb.AlipayNotificationField, 0, len(c.Request.PostForm))
	for key, values := range c.Request.PostForm {
		if len(values) > 0 {
			fields = append(fields, &pb.AlipayNotificationField{Key: key, Value: values[0]})
		}
	}
	response, err := h.clients.Billing.ProcessAlipayNotification(c.Request.Context(), &pb.ProcessAlipayNotificationRequest{Fields: fields})
	if err != nil || response == nil || !response.GetAccepted() {
		logger.L().Error("alipay sandbox notify rejected",
			zap.Error(err),
			zap.String("out_trade_no_hash", billingLogID(c.Request.PostForm.Get("out_trade_no"))),
			zap.String("trade_no_hash", billingLogID(c.Request.PostForm.Get("trade_no"))),
			zap.String("trade_status", c.Request.PostForm.Get("trade_status")),
			zap.Bool("accepted", response != nil && response.GetAccepted()),
		)
		statusCode := http.StatusServiceUnavailable
		if status.Code(err) == codes.InvalidArgument || (err == nil && response != nil) {
			statusCode = http.StatusBadRequest
		}
		c.String(statusCode, "failure")
		return
	}
	c.String(http.StatusOK, "success")
}

func billingLogID(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:6])
}

func (h *AlipayWebhookHandler) Return(c *gin.Context) {
	fields := make([]*pb.AlipayNotificationField, 0, len(c.Request.URL.Query()))
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			fields = append(fields, &pb.AlipayNotificationField{Key: key, Value: values[0]})
		}
	}
	response, err := h.clients.Billing.ResolveAlipayReturn(c.Request.Context(), &pb.ResolveAlipayReturnRequest{Fields: fields})
	if err != nil || response == nil {
		logger.L().Warn("alipay browser return rejected", zap.Error(err))
		c.String(http.StatusBadRequest, "invalid Alipay return")
		return
	}
	target, ok := h.returnURLs[response.GetSourceApp()]
	parsed, parseErr := url.Parse(target)
	if !ok || parseErr != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil {
		logger.L().Error("billing return target is invalid", zap.String("source_app", response.GetSourceApp()))
		c.String(http.StatusServiceUnavailable, "billing return is unavailable")
		return
	}
	query := parsed.Query()
	query.Set("payment_return", response.GetReturnToken())
	parsed.RawQuery = query.Encode()
	c.Redirect(http.StatusSeeOther, parsed.String())
}

func queryInt32(c *gin.Context, key string, fallback int32) int32 {
	value := c.Query(key)
	if value == "" {
		return fallback
	}
	var parsed int32
	if _, err := fmt.Sscan(value, &parsed); err != nil || parsed < 1 {
		return fallback
	}
	return parsed
}
