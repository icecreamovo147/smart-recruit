package candidate

import (
	"errors"
	"html"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	base "smart-recruit-gateway/handler"
	"smart-recruit-gateway/middleware"
	"smart-recruit-gateway/rpc"
	"smart-recruit-proto/recruitment/pb"
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

type profileUpdateBody struct {
	RealName           string                         `json:"real_name"`
	Phone              string                         `json:"phone"`
	Education          string                         `json:"education"`
	School             string                         `json:"school"`
	WorkExperience     string                         `json:"work_experience"`
	Skills             base.FlexSkills               `json:"skills"`
	City               string                         `json:"city"`
	YearsOfExperience  float64                        `json:"years_of_experience"`
	JobStatus          string                         `json:"job_status"`
	ExpectedPosition   string                         `json:"expected_position"`
	ExpectedSalaryMin  int32                          `json:"expected_salary_min"`
	ExpectedSalaryMax  int32                          `json:"expected_salary_max"`
	AvailableFrom      string                         `json:"available_from"`
	Summary            string                         `json:"summary"`
	Educations         []profileEducationBody         `json:"educations"`
	Experiences        []profileExperienceBody        `json:"experiences"`
}

type profileEducationBody struct {
	School      string `json:"school"`
	Degree      string `json:"degree"`
	Major       string `json:"major"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
	Description string `json:"description"`
	SortOrder   int32  `json:"sort_order"`
}

type profileExperienceBody struct {
	Company     string `json:"company"`
	Title       string `json:"title"`
	Location    string `json:"location"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
	IsCurrent   int32  `json:"is_current"`
	Description string `json:"description"`
	SortOrder   int32  `json:"sort_order"`
}

type applyFillBody struct {
	Draft             profileUpdateBody `json:"draft"`
	OverwriteExisting bool              `json:"overwrite_existing"`
}

func (h *ProfileHandler) Update(c *gin.Context) {
	var req profileUpdateBody
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "请求参数错误")
		return
	}
	if err := validateProfileFields(req); err != nil {
		base.BadRequest(c, err.Error())
		return
	}
	resp, err := h.clients.Candidate.UpdateProfile(c.Request.Context(), buildUpdateProfileRequest(middleware.UserID(c), req))
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, resp.Profile)
}

func (h *ProfileHandler) FillFromResume(c *gin.Context) {
	userID := middleware.UserID(c)
	ctx := c.Request.Context()
	forceRefresh := parseBoolQueryOrForm(c, "force_refresh")
	overwriteExisting := parseBoolQueryOrForm(c, "overwrite_existing")
	var body struct {
		ForceRefresh      *bool `json:"force_refresh"`
		OverwriteExisting *bool `json:"overwrite_existing"`
	}
	if err := c.ShouldBindJSON(&body); err == nil {
		if body.ForceRefresh != nil {
			forceRefresh = *body.ForceRefresh
		}
		if body.OverwriteExisting != nil {
			overwriteExisting = *body.OverwriteExisting
		}
	}

	resp, err := h.clients.Candidate.FillProfileFromResume(ctx, &pb.FillProfileFromResumeRequest{
		UserId: userID, ForceRefresh: forceRefresh, OverwriteExisting: overwriteExisting,
	})
	if err != nil {
		base.Internal(c, err)
		return
	}

	refreshed := false
	refreshReason := resp.GetRefreshReason()
	resumeID := resp.GetResumeId()
	if resp.GetNeedsRefresh() || resp.Code == 40402 {
		if resumeID <= 0 {
			resumeResp, resumeErr := h.clients.Candidate.GetResume(ctx, &pb.GetResumeRequest{UserId: userID})
			if resumeErr != nil {
				base.Internal(c, resumeErr)
				return
			}
			if resumeResp.GetResume() == nil || resumeResp.GetResume().GetResumeId() <= 0 {
				base.From(c, resp.Code, resp.Msg, nil)
				return
			}
			resumeID = resumeResp.GetResume().GetResumeId()
		}
		parseResp, parseErr := h.clients.RecruitingIntelligence.ParseResumeProfileForCandidate(ctx, &pb.ParseResumeProfileForCandidateRequest{
			CandidateUserId: userID,
			ResumeId:        resumeID,
		})
		if parseErr != nil {
			base.Internal(c, parseErr)
			return
		}
		if parseResp.GetCode() != 0 {
			base.From(c, parseResp.GetCode(), parseResp.GetMsg(), nil)
			return
		}
		refreshed = true
		resp, err = h.clients.Candidate.FillProfileFromResume(ctx, &pb.FillProfileFromResumeRequest{
			UserId: userID, ForceRefresh: false, OverwriteExisting: overwriteExisting,
		})
		if err != nil {
			base.Internal(c, err)
			return
		}
	}
	if resp.Code != 0 {
		base.From(c, resp.Code, resp.Msg, nil)
		return
	}
	draft := resp.GetDraft()
	if draft == nil {
		base.From(c, resp.Code, resp.Msg, nil)
		return
	}
	draft.Refreshed = refreshed
	if refreshed {
		if refreshReason == "" {
			refreshReason = "missing"
		}
		draft.RefreshReason = refreshReason
	} else if draft.RefreshReason == "" {
		draft.RefreshReason = "reused"
	}
	base.From(c, resp.Code, resp.Msg, draft)
}

func parseBoolQueryOrForm(c *gin.Context, key string) bool {
	value := strings.TrimSpace(c.Query(key))
	if value == "" {
		value = strings.TrimSpace(c.PostForm(key))
	}
	switch strings.ToLower(value) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func (h *ProfileHandler) ApplyFill(c *gin.Context) {
	var req applyFillBody
	if err := c.ShouldBindJSON(&req); err != nil {
		base.BadRequest(c, "请求参数错误："+err.Error())
		return
	}
	if err := validateProfileFields(req.Draft); err != nil {
		base.BadRequest(c, err.Error())
		return
	}
	updateReq := buildUpdateProfileRequest(middleware.UserID(c), req.Draft)
	resp, err := h.clients.Candidate.ApplyProfileFill(c.Request.Context(), &pb.ApplyProfileFillRequest{
		UserId:            middleware.UserID(c),
		OverwriteExisting: req.OverwriteExisting,
		Draft: &pb.ProfileFillDraft{Draft: &pb.CandidateProfile{
			RealName: updateReq.RealName, Phone: updateReq.Phone, Education: updateReq.Education, School: updateReq.School,
			WorkExperience: updateReq.WorkExperience, Skills: splitSkills(updateReq.Skills), City: updateReq.City,
			YearsOfExperience: updateReq.YearsOfExperience, JobStatus: updateReq.JobStatus, ExpectedPosition: updateReq.ExpectedPosition,
			ExpectedSalaryMin: updateReq.ExpectedSalaryMin, ExpectedSalaryMax: updateReq.ExpectedSalaryMax,
			AvailableFrom: updateReq.AvailableFrom, Summary: updateReq.Summary,
			Educations: updateReq.Educations, Experiences: updateReq.Experiences,
		}},
	})
	if err != nil {
		base.Internal(c, err)
		return
	}
	base.From(c, resp.Code, resp.Msg, resp.Profile)
}

func buildUpdateProfileRequest(userID int64, req profileUpdateBody) *pb.UpdateProfileRequest {
	educations := make([]*pb.CandidateEducationInfo, 0, len(req.Educations))
	for _, item := range req.Educations {
		educations = append(educations, &pb.CandidateEducationInfo{
			School: html.EscapeString(strings.TrimSpace(item.School)),
			Degree: html.EscapeString(strings.TrimSpace(item.Degree)),
			Major:  html.EscapeString(strings.TrimSpace(item.Major)),
			StartDate: strings.TrimSpace(item.StartDate), EndDate: strings.TrimSpace(item.EndDate),
			Description: strings.TrimSpace(item.Description), SortOrder: item.SortOrder,
		})
	}
	experiences := make([]*pb.CandidateExperienceInfo, 0, len(req.Experiences))
	for _, item := range req.Experiences {
		experiences = append(experiences, &pb.CandidateExperienceInfo{
			Company: html.EscapeString(strings.TrimSpace(item.Company)),
			Title:   html.EscapeString(strings.TrimSpace(item.Title)),
			Location: html.EscapeString(strings.TrimSpace(item.Location)),
			StartDate: strings.TrimSpace(item.StartDate), EndDate: strings.TrimSpace(item.EndDate),
			IsCurrent: item.IsCurrent, Description: strings.TrimSpace(item.Description), SortOrder: item.SortOrder,
		})
	}
	return &pb.UpdateProfileRequest{
		UserId: userID,
		RealName:       html.EscapeString(strings.TrimSpace(req.RealName)),
		Phone:          strings.TrimSpace(req.Phone),
		Education:      strings.TrimSpace(req.Education),
		School:         html.EscapeString(strings.TrimSpace(req.School)),
		WorkExperience: strings.TrimSpace(req.WorkExperience),
		Skills:         strings.TrimSpace(req.Skills.String()),
		City:           html.EscapeString(strings.TrimSpace(req.City)),
		YearsOfExperience: req.YearsOfExperience,
		JobStatus:      strings.TrimSpace(req.JobStatus),
		ExpectedPosition: html.EscapeString(strings.TrimSpace(req.ExpectedPosition)),
		ExpectedSalaryMin: req.ExpectedSalaryMin,
		ExpectedSalaryMax: req.ExpectedSalaryMax,
		AvailableFrom:  strings.TrimSpace(req.AvailableFrom),
		Summary:        html.EscapeString(strings.TrimSpace(req.Summary)),
		Educations: educations,
		Experiences: experiences,
	}
}

func splitSkills(value string) []string {
	parts := strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == '，' || r == ';' || r == '；' || r == '\n' })
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

const (
	maxProfileFieldLen          = 500
	maxRegionFieldLen           = 128
	maxExperienceDescriptionLen = 8000
)

var tagStripper = regexp.MustCompile(`<[^>]*>`)

func validateProfileFields(req profileUpdateBody) error {
	fields := map[string]string{
		"real_name": req.RealName,
		"phone":     req.Phone,
		"education": req.Education,
		"school":    req.School,
		"skills":    req.Skills.String(),
		"city":      req.City,
		"job_status": req.JobStatus,
		"expected_position": req.ExpectedPosition,
		"summary": req.Summary,
	}
	for name, val := range fields {
		limit := maxProfileFieldLen
		if name == "city" {
			limit = maxRegionFieldLen
		}
		if len(val) > limit {
			return errors.New(name + ": 内容过长，请精简后重试")
		}
		if stripped := tagStripper.ReplaceAllString(val, ""); stripped != val {
			return errors.New(name + ": 不允许包含 HTML 标签")
		}
	}
	for i, edu := range req.Educations {
		if len(edu.School) > maxProfileFieldLen {
			return errors.New("educations[" + strconv.Itoa(i) + "].school: 内容过长，请精简后重试")
		}
		if stripped := tagStripper.ReplaceAllString(edu.School, ""); stripped != edu.School {
			return errors.New("educations[" + strconv.Itoa(i) + "].school: 不允许包含 HTML 标签")
		}
	}
	for i, exp := range req.Experiences {
		if len(exp.Location) > maxRegionFieldLen {
			return errors.New("experiences[" + strconv.Itoa(i) + "].location: 内容过长，请精简后重试")
		}
		if len(exp.Description) > maxExperienceDescriptionLen {
			return errors.New("experiences[" + strconv.Itoa(i) + "].description: 内容过长，请精简后重试")
		}
	}
	return nil
}
