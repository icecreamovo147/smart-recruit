package candidate

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

func (h *OfferHandler) ListMyOffers(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "10")
	cursor := c.Query("cursor")

	page, err := strconv.Atoi(pageStr)
	if err != nil {
		page = 1
	}
	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil {
		pageSize = 10
	}

	resp, err := h.clients.Offer.ListMyOffers(c.Request.Context(), &pb.ListMyOffersRequest{
		UserId:   middleware.UserID(c),
		Page:     int32(page),
		PageSize: int32(pageSize),
		Cursor:   cursor,
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

func (h *OfferHandler) Accept(c *gin.Context) {
	offerID, err := strconv.ParseInt(c.Param("offer_id"), 10, 64)
	if err != nil {
		base.BadRequest(c, "Offer ID 不合法")
		return
	}
	resp, err := h.clients.Offer.AcceptOffer(c.Request.Context(), &pb.AcceptOfferRequest{
		UserId:  middleware.UserID(c),
		OfferId: offerID,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

func (h *OfferHandler) Reject(c *gin.Context) {
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
	resp, err := h.clients.Offer.RejectOffer(c.Request.Context(), &pb.RejectOfferRequest{
		UserId:  middleware.UserID(c),
		OfferId: offerID,
		Reason:  req.Reason,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}
