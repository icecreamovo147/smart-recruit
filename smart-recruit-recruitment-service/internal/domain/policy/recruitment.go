package policy

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"smart-recruit-recruitment-service/internal/domain/model"
)

var (
	ErrJobTitleRequired        = errors.New("岗位名称不能为空")
	ErrJobDepartmentRequired   = errors.New("请选择部门")
	ErrJobLocationRequired     = errors.New("请选择地点")
	ErrResumeFileTypeInvalid   = errors.New("仅支持 PDF、DOCX 格式")
	ErrResumeUploadIDRequired  = errors.New("缺少上传凭证，请重新上传")
	ErrResumeSessionInvalid    = errors.New("上传凭证无效或已过期，请重新上传")
	ErrResumeSessionUser       = errors.New("上传凭证与当前用户不匹配")
	ErrResumeSessionFile       = errors.New("文件信息不匹配，请重新上传")
	ErrResumeSessionFileType   = errors.New("文件类型与上传凭证不一致，请重新上传")
	ErrResumeSessionStatus     = errors.New("上传凭证已失效，请重新上传")
	ErrResumeFileTooLarge      = errors.New("简历文件大小超过限制（最大 20MB）")
	ErrResumeLegacyKeyMismatch = errors.New("文件信息与当前用户不匹配")
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
	if allNotEmpty(profile.RealName, profile.Phone, profile.Education, profile.School, profile.WorkExperience, profile.Skills) {
		profile.IsComplete = 1
		return
	}
	profile.IsComplete = 0
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
