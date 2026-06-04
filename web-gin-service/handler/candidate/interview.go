package candidate

import (
	base "web-gin-service/handler"
	"web-gin-service/middleware"
	"web-gin-service/recruitment/pb"
	"web-gin-service/rpc"

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
