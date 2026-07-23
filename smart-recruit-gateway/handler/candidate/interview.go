package candidate

import (
	base "smart-recruit-gateway/handler"
	"smart-recruit-gateway/middleware"
	"smart-recruit-gateway/rpc"
	"smart-recruit-proto/recruitment/pb"

	"github.com/gin-gonic/gin"
)

type InterviewHandler struct {
	clients *rpc.Clients
}

func NewInterviewHandler(clients *rpc.Clients) *InterviewHandler {
	return &InterviewHandler{clients: clients}
}

// List returns upcoming and past interviews for the authenticated candidate.
func (h *InterviewHandler) List(c *gin.Context) {
	resp, err := h.clients.Interview.ListCandidateInterviews(c.Request.Context(), &pb.ListCandidateInterviewsRequest{
		UserId: middleware.UserID(c),
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}
