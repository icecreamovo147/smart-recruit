package candidate

import (
	"github.com/gin-gonic/gin"

	base "web-gin-service/handler"
	"web-gin-service/middleware"
	"web-gin-service/recruitment/pb"
	"web-gin-service/rpc"
)

type ProfileHandler struct {
	clients *rpc.Clients
}

func NewProfileHandler(clients *rpc.Clients) *ProfileHandler {
	return &ProfileHandler{clients: clients}
}

func (h *ProfileHandler) Get(c *gin.Context) {
	resp, err := h.clients.Candidate.GetProfile(c.Request.Context(), &pb.GetProfileRequest{UserId: middleware.UserID(c)})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, resp.Profile)
}

func (h *ProfileHandler) Update(c *gin.Context) {
	var req struct {
		RealName       string `json:"real_name"`
		Phone          string `json:"phone"`
		Education      string `json:"education"`
		School         string `json:"school"`
		WorkExperience string `json:"work_experience"`
		Skills         string `json:"skills"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "请求参数错误")
		return
	}
	resp, err := h.clients.Candidate.UpdateProfile(c.Request.Context(), &pb.UpdateProfileRequest{UserId: middleware.UserID(c), RealName: req.RealName, Phone: req.Phone, Education: req.Education, School: req.School, WorkExperience: req.WorkExperience, Skills: req.Skills})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, resp.Profile)
}
