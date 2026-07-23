package policy

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"smart-recruit-recruitment-service/internal/domain/model"
	profilepkg "smart-recruit-recruitment-service/internal/domain/profile"
)

var (
	ErrJobTitleRequired         = errors.New("岗位名称不能为空")
	ErrJobDepartmentRequired    = errors.New("请选择部门")
	ErrJobLocationRequired      = errors.New("请选择地点")
	ErrResumeFileTypeInvalid    = errors.New("仅支持 PDF、DOCX 格式")
	ErrResumeUploadIDRequired   = errors.New("缺少上传凭证，请重新上传")
	ErrResumeSessionInvalid     = errors.New("上传凭证无效或已过期，请重新上传")
	ErrResumeSessionUser        = errors.New("上传凭证与当前用户不匹配")
	ErrResumeSessionFile        = errors.New("文件信息不匹配，请重新上传")
	ErrResumeSessionFileType    = errors.New("文件类型与上传凭证不一致，请重新上传")
	ErrResumeSessionStatus      = errors.New("上传凭证已失效，请重新上传")
	ErrResumeFileTooLarge       = errors.New("简历文件大小超过限制（最大 20MB）")
	ErrResumeLegacyKeyMismatch  = errors.New("文件信息与当前用户不匹配")
	ErrProfileIncomplete        = errors.New("请先完善个人资料后再投递")
	ErrResumeMissing            = errors.New("请先上传简历后再投递")
	ErrJobUnavailable           = errors.New("该岗位已下架或不存在，无法投递")
	ErrReasonRequired           = errors.New("该状态变更必须填写原因")
	ErrApplicationNotCurrent    = errors.New("该投递已不是当前有效流程，不能修改状态")
	ErrDepartmentNameRequired   = errors.New("部门名称不能为空")
	ErrDepartmentParentMissing  = errors.New("父部门不存在")
	ErrDepartmentParentInactive = errors.New("父部门已停用，无法在其下新增子部门")
)

var allowedResumeExtensions = map[string]bool{
	"pdf":  true,
	"docx": true,
}

func BuildJob(input model.Job) (model.Job, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Department = strings.TrimSpace(input.Department)
	input.Location = strings.TrimSpace(input.Location)
	if input.HRID == 0 || input.Title == "" {
		return model.Job{}, ErrJobTitleRequired
	}
	if input.Department == "" {
		return model.Job{}, ErrJobDepartmentRequired
	}
	if input.Location == "" {
		return model.Job{}, ErrJobLocationRequired
	}
	input.Status = model.JobStatusOnline
	return input, nil
}

func CompleteCandidateProfile(profile *model.CandidateProfile) {
	if profile == nil {
		return
	}
	bundle := &profilepkg.Bundle{
		Profile: *profile,
	}
	profilepkg.Complete(bundle)
	*profile = bundle.Profile
}

func ValidateResumeFile(fileName, fileType string) error {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(fileName)), ".")
	if fileType != "" && ext != strings.ToLower(strings.TrimSpace(fileType)) {
		return ErrResumeFileTypeInvalid
	}
	if !allowedResumeExtensions[ext] {
		return ErrResumeFileTypeInvalid
	}
	return nil
}

func ContentTypeFromFileType(fileType string) string {
	switch strings.ToLower(strings.TrimSpace(fileType)) {
	case "pdf":
		return "application/pdf"
	case "docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case "doc":
		return "application/msword"
	default:
		return "application/octet-stream"
	}
}

func SanitizeFileName(fileName string) string {
	base := filepath.Base(fileName)
	if base == "." || base == "/" || base == "\\" {
		return "resume"
	}
	return strings.NewReplacer("/", "_", "\\", "_", " ", "_").Replace(base)
}

func ValidateStrictResumeSession(userID int64, uploadID, ossKey, fileType string, fileSize int64, session *model.PresignSession) error {
	if uploadID == "" {
		return ErrResumeUploadIDRequired
	}
	if session == nil {
		return ErrResumeSessionInvalid
	}
	if session.UserID != userID {
		return ErrResumeSessionUser
	}
	if session.OSSKey != ossKey {
		return ErrResumeSessionFile
	}
	if session.FileType != fileType {
		return ErrResumeSessionFileType
	}
	if session.ContentType != "" && fileType != "" && session.ContentType != ContentTypeFromFileType(fileType) {
		return ErrResumeSessionFileType
	}
	if session.Status != "pending" {
		return ErrResumeSessionStatus
	}
	if fileSize > session.MaxSize {
		return ErrResumeFileTooLarge
	}
	return nil
}

func ValidateLegacyResumeKey(userID int64, ossKey string) error {
	expectedPrefix := fmt.Sprintf("resumes/%d/", userID)
	if !strings.HasPrefix(ossKey, expectedPrefix) {
		return ErrResumeLegacyKeyMismatch
	}
	return nil
}

func allNotEmpty(values ...string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return false
		}
	}
	return true
}

var allowedTransitions = map[string]map[string]bool{
	model.StatusKeyApplied: {
		model.StatusKeyViewed:       true,
		model.StatusKeyScreenPassed: true,
		model.StatusKeyRejected:     true,
		model.StatusKeyWithdrawn:    true,
	},
	model.StatusKeyViewed: {
		model.StatusKeyScreening:        true,
		model.StatusKeyScreenPassed:     true,
		model.StatusKeyInterviewPending: true,
		model.StatusKeyInterviewPassed:  true,
		model.StatusKeyOfferPending:     true,
		model.StatusKeyRejected:         true,
		model.StatusKeyWithdrawn:        true,
	},
	model.StatusKeyScreening: {
		model.StatusKeyScreenPassed: true,
		model.StatusKeyRejected:     true,
		model.StatusKeyWithdrawn:    true,
	},
	model.StatusKeyScreenPassed: {
		model.StatusKeyInterviewPending: true,
		model.StatusKeyRejected:         true,
		model.StatusKeyWithdrawn:        true,
	},
	model.StatusKeyInterviewPending: {
		model.StatusKeyInterviewing:       true,
		model.StatusKeyInterviewCancelled: true,
		model.StatusKeyInterviewPassed:    true,
		model.StatusKeyRejected:           true,
		model.StatusKeyWithdrawn:          true,
	},
	model.StatusKeyInterviewing: {
		model.StatusKeyInterviewPending:   true,
		model.StatusKeyInterviewCancelled: true,
		model.StatusKeyInterviewPassed:    true,
		model.StatusKeyRejected:           true,
		model.StatusKeyWithdrawn:          true,
	},
	model.StatusKeyInterviewCancelled: {
		model.StatusKeyInterviewPending: true,
		model.StatusKeyRejected:         true,
		model.StatusKeyWithdrawn:        true,
	},
	model.StatusKeyInterviewPassed: {
		model.StatusKeyInterviewPending: true,
		model.StatusKeyOfferPending:     true,
		model.StatusKeyRejected:         true,
		model.StatusKeyWithdrawn:        true,
	},
	model.StatusKeyOfferPending: {
		model.StatusKeyOfferSent: true,
		model.StatusKeyRejected:  true,
		model.StatusKeyWithdrawn: true,
	},
	model.StatusKeyOfferSent: {
		model.StatusKeyOfferAccepted: true,
		model.StatusKeyOfferRejected: true,
		model.StatusKeyOfferPending:  true,
		model.StatusKeyWithdrawn:     true,
	},
	model.StatusKeyOfferAccepted: {
		model.StatusKeyHired: true,
	},
	model.StatusKeyHired:         {},
	model.StatusKeyRejected:      {model.StatusKeyScreenPassed: true},
	model.StatusKeyOfferRejected: {},
	model.StatusKeyWithdrawn:     {},
}

type TransitionError struct {
	From string
	To   string
	Msg  string
}

func (e *TransitionError) Error() string { return e.Msg }

func DefaultApplicationStatusKey() string { return model.StatusKeyApplied }

func IsTerminalApplicationStatus(key string) bool { return model.TerminalStatusKeys[key] }

func ValidateStatusKey(key string) error {
	if _, ok := model.HRStatusLabels[key]; !ok {
		return &TransitionError{From: key, Msg: fmt.Sprintf("未知的投递状态：%s", key)}
	}
	return nil
}

func ValidateTransition(from, to string) error {
	if from == to {
		return &TransitionError{From: from, To: to, Msg: fmt.Sprintf("状态未变更：%s", model.HRStatusLabels[from])}
	}
	if targets, ok := allowedTransitions[from]; ok && targets[to] {
		return nil
	}
	return &TransitionError{
		From: from,
		To:   to,
		Msg:  fmt.Sprintf("不允许从「%s」变更为「%s」", model.HRStatusLabels[from], model.HRStatusLabels[to]),
	}
}

func TargetStatusKey(statusKey string, legacyStatus int32) (string, error) {
	if statusKey == "" {
		statusKey = model.LegacyStatusToKey[legacyStatus]
	}
	if statusKey == "" {
		return "", &TransitionError{Msg: "投递状态不合法"}
	}
	return statusKey, ValidateStatusKey(statusKey)
}

func ValidateApplyPreconditions(profile *model.CandidateProfile, resume *model.Resume, job *model.Job) error {
	if profile == nil || profile.IsComplete != 1 {
		return ErrProfileIncomplete
	}
	if resume == nil {
		return ErrResumeMissing
	}
	if job == nil || job.Status != model.JobStatusOnline {
		return ErrJobUnavailable
	}
	return nil
}

func ValidateStatusChange(detail model.ApplicationDetail, targetKey, reason string) (currentKey string, isRePass bool, legacy int32, err error) {
	currentKey = detail.StatusKey
	if currentKey == "" {
		currentKey = model.LegacyStatusToKey[detail.Status]
	}
	if err := ValidateTransition(currentKey, targetKey); err != nil {
		return "", false, 0, err
	}
	if requiresReason(targetKey) && strings.TrimSpace(reason) == "" {
		return "", false, 0, ErrReasonRequired
	}
	isRePass = currentKey == model.StatusKeyRejected && targetKey == model.StatusKeyScreenPassed
	if detail.IsCurrent != 1 && !isRePass {
		return "", false, 0, ErrApplicationNotCurrent
	}
	return currentKey, isRePass, model.StatusKeyToLegacy[targetKey], nil
}

func BuildApplicationNotification(targetKey string, isRePass bool, detail model.ApplicationDetail) (notifyType, content string) {
	switch targetKey {
	case model.StatusKeyScreenPassed:
		if isRePass {
			return "application_approved", fmt.Sprintf("你投递的「%s」岗位已重新通过筛选（第%d轮），请留意后续安排。", detail.JobTitle, detail.RoundNo+1)
		}
		return "application_approved", fmt.Sprintf("你投递的「%s」岗位已通过筛选，请留意后续安排。", detail.JobTitle)
	case model.StatusKeyRejected:
		return "application_rejected", fmt.Sprintf("你投递的「%s」岗位当前未通过筛选，感谢你的投递。", detail.JobTitle)
	case model.StatusKeyWithdrawn:
		return "application_withdrawn", fmt.Sprintf("你已撤回对「%s」岗位的投递。", detail.JobTitle)
	case model.StatusKeyHired:
		return "application_hired", fmt.Sprintf("恭喜！你投递的「%s」岗位已确认入职。", detail.JobTitle)
	default:
		return "", ""
	}
}

func requiresReason(targetKey string) bool {
	return targetKey == model.StatusKeyRejected || targetKey == model.StatusKeyWithdrawn || targetKey == model.StatusKeyOfferRejected
}

func CandidateDisplayName(realName string, userID int64) string {
	if strings.TrimSpace(realName) != "" {
		return strings.TrimSpace(realName)
	}
	return fmt.Sprintf("候选人%d", userID)
}

func BuildDepartmentForCreate(parent *model.DepartmentNode, parentID int64, name string, sortOrder int, adminID int64, activeLocations []int64) (model.DepartmentNode, []int64, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return model.DepartmentNode{}, nil, ErrDepartmentNameRequired
	}
	if parentID > 0 {
		if parent == nil {
			return model.DepartmentNode{}, nil, ErrDepartmentParentMissing
		}
		if parent.IsActive == 0 {
			return model.DepartmentNode{}, nil, ErrDepartmentParentInactive
		}
		if parent.Depth >= 2 {
			return model.DepartmentNode{}, nil, fmt.Errorf("部门层级不能超过 %d 层", 2)
		}
	}
	inherit := int32(1)
	defaultLocationIDs := []int64(nil)
	if parentID == 0 {
		inherit = 0
		defaultLocationIDs = append(defaultLocationIDs, activeLocations...)
	}
	return model.DepartmentNode{
		ParentID:         parentID,
		Name:             name,
		SortOrder:        sortOrder,
		IsActive:         1,
		InheritLocations: inherit,
		CreatedBy:        &adminID,
	}, defaultLocationIDs, nil
}

func DefaultTagColor(color string) string {
	if strings.TrimSpace(color) == "" {
		return "#409eff"
	}
	return color
}

func ParseUsageTimeRange(now time.Time, startStr, endStr string, defaultDays int) (time.Time, time.Time, error) {
	endTime := now
	var err error
	if endStr != "" {
		endTime, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}
	startTime := endTime.AddDate(0, 0, -defaultDays)
	if startStr != "" {
		startTime, err = time.Parse(time.RFC3339, startStr)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}
	return startTime, endTime, nil
}
