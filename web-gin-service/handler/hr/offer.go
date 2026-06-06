package hr

import (
	"strconv"

	"github.com/gin-gonic/gin"

	base "web-gin-service/handler"
	"web-gin-service/middleware"
	"web-gin-service/recruitment/pb"
	"web-gin-service/rpc"
)

type OfferHandler struct {
	clients *rpc.Clients
}

func NewOfferHandler(clients *rpc.Clients) *OfferHandler {
	return &OfferHandler{clients: clients}
}

func (h *OfferHandler) Create(c *gin.Context) {
	var req struct {
		ApplicationID base.FlexInt64 `json:"application_id" binding:"required"`
		Title         string         `json:"title" binding:"required"`
		SalaryRange   string `json:"salary_range"`
		Level         string `json:"level"`
		WorkLocation  string `json:"work_location"`
		StartDate     string `json:"start_date"`
		ExpiresAt     string `json:"expires_at"`
		TermsJSON     string `json:"terms_json"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "请求参数错误："+err.Error())
		return
	}
	resp, err := h.clients.Offer.CreateOffer(c.Request.Context(), &pb.CreateOfferRequest{
		HrId:          middleware.UserID(c),
		ApplicationId: int64(req.ApplicationID),
		Title:         req.Title,
		SalaryRange:   req.SalaryRange,
		Level:         req.Level,
		WorkLocation:  req.WorkLocation,
		StartDate:     req.StartDate,
		ExpiresAt:     req.ExpiresAt,
		TermsJson:     req.TermsJSON,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

func (h *OfferHandler) Update(c *gin.Context) {
	offerID, err := strconv.ParseInt(c.Param("offer_id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "Offer ID 不合法")
		return
	}
	var req struct {
		Title        string `json:"title"`
		SalaryRange  string `json:"salary_range"`
		Level        string `json:"level"`
		WorkLocation string `json:"work_location"`
		StartDate    string `json:"start_date"`
		ExpiresAt    string `json:"expires_at"`
		TermsJSON    string `json:"terms_json"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "请求参数错误")
		return
	}
	resp, err := h.clients.Offer.UpdateOffer(c.Request.Context(), &pb.UpdateOfferRequest{
		HrId:         middleware.UserID(c),
		OfferId:      offerID,
		Title:        req.Title,
		SalaryRange:  req.SalaryRange,
		Level:        req.Level,
		WorkLocation: req.WorkLocation,
		StartDate:    req.StartDate,
		ExpiresAt:    req.ExpiresAt,
		TermsJson:    req.TermsJSON,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

func (h *OfferHandler) Get(c *gin.Context) {
	offerID, err := strconv.ParseInt(c.Param("offer_id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "Offer ID 不合法")
		return
	}
	resp, err := h.clients.Offer.GetOffer(c.Request.Context(), &pb.GetOfferRequest{
		UserId:  middleware.UserID(c),
		OfferId: offerID,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

func (h *OfferHandler) ListByApplication(c *gin.Context) {
	applicationID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "投递记录 ID 不合法")
		return
	}
	resp, err := h.clients.Offer.ListOffersByApplication(c.Request.Context(), &pb.ListOffersByApplicationRequest{
		HrId:          middleware.UserID(c),
		ApplicationId: applicationID,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

func (h *OfferHandler) Send(c *gin.Context) {
	offerID, err := strconv.ParseInt(c.Param("offer_id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "Offer ID 不合法")
		return
	}
	resp, err := h.clients.Offer.SendOffer(c.Request.Context(), &pb.SendOfferRequest{
		HrId:    middleware.UserID(c),
		OfferId: offerID,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

func (h *OfferHandler) Withdraw(c *gin.Context) {
	offerID, err := strconv.ParseInt(c.Param("offer_id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "Offer ID 不合法")
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "请求参数错误")
		return
	}
	resp, err := h.clients.Offer.WithdrawOffer(c.Request.Context(), &pb.WithdrawOfferRequest{
		HrId:    middleware.UserID(c),
		OfferId: offerID,
		Reason:  req.Reason,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

func (h *OfferHandler) ListEvents(c *gin.Context) {
	offerID, err := strconv.ParseInt(c.Param("offer_id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "Offer ID 不合法")
		return
	}
	resp, err := h.clients.Offer.ListOfferEvents(c.Request.Context(), &pb.ListOfferEventsRequest{
		HrId:    middleware.UserID(c),
		OfferId: offerID,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}
