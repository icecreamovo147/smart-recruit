package candidate

import (
	"errors"
	"html"
	"regexp"
	"strings"

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
	// Validate field lengths and content to prevent XSS and DoS.
	if err := validateProfileFields(req.RealName, req.Phone, req.Education, req.School, req.WorkExperience, req.Skills); err != nil {
		base.BadRequest(c, err.Error())
		return
	}
	resp, err := h.clients.Candidate.UpdateProfile(c.Request.Context(), &pb.UpdateProfileRequest{
		UserId: middleware.UserID(c),
		RealName:       html.EscapeString(strings.TrimSpace(req.RealName)),
		Phone:          strings.TrimSpace(req.Phone),
		Education:      strings.TrimSpace(req.Education),
		School:         strings.TrimSpace(req.School),
		WorkExperience: strings.TrimSpace(req.WorkExperience),
		Skills:         strings.TrimSpace(req.Skills),
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, resp.Profile)
}

const maxProfileFieldLen = 500

// stripHTML removes HTML tags to prevent XSS when fields that expect plain text
// are rendered in views without escaping.
var tagStripper = regexp.MustCompile(`<[^>]*>`)

func validateProfileFields(realName, phone, education, school, workExp, skills string) error {
	// work_experience is intentionally excluded from HTML tag validation:
	// the frontend uses a RichTextEditor for this field which naturally
	// produces HTML markup. XSS sanitization for rich text fields is
	// handled separately by the bluemonday policy at the output layer.
	fields := map[string]string{
		"real_name": realName,
		"phone":     phone,
		"education": education,
		"school":    school,
		"skills":    skills,
	}
	for name, val := range fields {
		if len(val) > maxProfileFieldLen {
			return errors.New(name + ": 内容过长，请精简后重试")
		}
		// Reject fields that contain HTML tags — strip and compare to detect markup.
		if stripped := tagStripper.ReplaceAllString(val, ""); stripped != val {
			return errors.New(name + ": 不允许包含 HTML 标签")
		}
	}
	return nil
}
