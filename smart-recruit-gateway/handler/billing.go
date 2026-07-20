package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

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
	Publish                 bool      `json:"publish"`
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

func (h *BillingHandler) Refund(c *gin.Context) {
	var request struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil || strings.TrimSpace(request.Reason) == "" {
		BadRequest(c, "退款原因不能为空")
		return
	}
	response, err := h.clients.Billing.RequestBillingRefund(c.Request.Context(), &pb.RequestBillingRefundRequest{Owner: h.owner(c), ActorUserId: middleware.UserID(c), OrderNo: c.Param("order_no"), Reason: strings.TrimSpace(request.Reason)})
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

func (h *BillingHandler) SavePrice(c *gin.Context) {
	var request saveBillingPriceRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		BadRequest(c, "商品、计费周期、金额和额度不能为空")
		return
	}
	response, err := h.clients.Billing.SaveBillingPriceVersion(c.Request.Context(), &pb.SaveBillingPriceVersionRequest{ProductId: int64(request.ProductID), PriceVersionId: int64(request.PriceVersionID), BillingTerm: request.BillingTerm, AmountFen: int64(request.AmountFen), IncludedCredits: int64(request.IncludedCredits), EntitlementSnapshotJson: request.EntitlementSnapshotJSON, Publish: request.Publish})
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
		Publish                      bool   `json:"publish"`
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
		CreditMicros:                  request.CreditMicros, Publish: request.Publish,
	})
	if err != nil {
		Internal(c, err)
		return
	}
	OK(c, "保存成功", response)
}

type AlipayWebhookHandler struct{ clients *rpc.Clients }

func NewAlipayWebhookHandler(clients *rpc.Clients) *AlipayWebhookHandler {
	return &AlipayWebhookHandler{clients: clients}
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
			zap.String("out_trade_no", c.Request.PostForm.Get("out_trade_no")),
			zap.String("trade_no", c.Request.PostForm.Get("trade_no")),
			zap.String("trade_status", c.Request.PostForm.Get("trade_status")),
			zap.Bool("accepted", response != nil && response.GetAccepted()),
		)
		c.String(http.StatusOK, "failure")
		return
	}
	c.String(http.StatusOK, "success")
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
