package candidate

import (
	"github.com/gin-gonic/gin"

	base "web-gin-service/handler"
	"web-gin-service/middleware"
	"web-gin-service/recruitment/pb"
	"web-gin-service/rpc"
)

type ResumeHandler struct {
	clients *rpc.Clients
}

func NewResumeHandler(clients *rpc.Clients) *ResumeHandler {
	return &ResumeHandler{clients: clients}
}

func (h *ResumeHandler) Get(c *gin.Context) {
	resp, err := h.clients.Candidate.GetResume(c.Request.Context(), &pb.GetResumeRequest{UserId: middleware.UserID(c)})
	if err != nil {
		base.Internal(c, err)
		return
	}
	if resp.Resume == nil {
		base.From(c, resp.Code, resp.Msg, gin.H{"resume": nil})
		return
	}
	base.ProtoResponse(c, resp)
}

func (h *ResumeHandler) Presign(c *gin.Context) {
	var req struct {
		FileName string `json:"file_name" binding:"required"`
		FileType string `json:"file_type" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "请求参数错误")
		return
	}
	resp, err := h.clients.Candidate.PresignResumeUpload(c.Request.Context(), &pb.PresignResumeUploadRequest{UserId: middleware.UserID(c), FileName: req.FileName, FileType: req.FileType})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}

func (h *ResumeHandler) Confirm(c *gin.Context) {
	var req struct {
		OSSKey   string `json:"oss_key" binding:"required"`
		FileName string `json:"file_name" binding:"required"`
		FileType string `json:"file_type" binding:"required"`
		FileSize int64  `json:"file_size"`
		UploadId string `json:"upload_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "请求参数错误")
		return
	}
	resp, err := h.clients.Candidate.ConfirmResumeUpload(c.Request.Context(), &pb.ConfirmResumeUploadRequest{UserId: middleware.UserID(c), OssKey: req.OSSKey, FileName: req.FileName, FileType: req.FileType, FileSize: req.FileSize, UploadId: req.UploadId})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.ProtoResponse(c, resp)
}
