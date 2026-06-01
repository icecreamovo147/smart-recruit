package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"logic-grpc-service/model"
	"logic-grpc-service/oss"
	"logic-grpc-service/pkg/errs"
	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/recruitment/pb"
	"logic-grpc-service/repository"
)

type CandidateService struct {
	profiles        *repository.ProfileRepo
	resumes         *repository.ResumeRepo
	oss             oss.Storage
	outboxPublisher *OutboxPublisher
	usageLogs       *repository.UsageLogRepo
}

func NewCandidateService(profiles *repository.ProfileRepo, resumes *repository.ResumeRepo, ossClient oss.Storage, outboxPublisher *OutboxPublisher, usageLogs *repository.UsageLogRepo) *CandidateService {
	return &CandidateService{profiles: profiles, resumes: resumes, oss: ossClient, outboxPublisher: outboxPublisher, usageLogs: usageLogs}
}

func (s *CandidateService) GetProfile(ctx context.Context, req *pb.GetProfileRequest) (*pb.GetProfileResponse, error) {
	profile, err := s.profiles.GetByUserID(ctx, req.UserId)
	if err != nil {
		logger.L().Error("get profile failed", zap.Int64("user_id", req.UserId), zap.Error(err))
		return nil, err
	}
	if profile == nil {
		return &pb.GetProfileResponse{Code: errs.OK, Msg: "success", Profile: &pb.CandidateProfile{}}, nil
	}
	return &pb.GetProfileResponse{Code: errs.OK, Msg: "success", Profile: toPBProfile(profile)}, nil
}

func (s *CandidateService) UpdateProfile(ctx context.Context, req *pb.UpdateProfileRequest) (*pb.GetProfileResponse, error) {
	complete := allNotEmpty(req.RealName, req.Phone, req.Education, req.School, req.WorkExperience, req.Skills)
	profile := &model.CandidateProfile{UserID: req.UserId, RealName: req.RealName, Phone: req.Phone, Education: req.Education, School: req.School, WorkExperience: req.WorkExperience, Skills: req.Skills}
	if complete {
		profile.IsComplete = 1
	}
	if err := s.profiles.Upsert(ctx, profile); err != nil {
		logger.L().Error("update profile failed", zap.Int64("user_id", req.UserId), zap.Error(err))
		return nil, err
	}
	logger.L().Info("profile updated", zap.Int64("user_id", req.UserId), zap.Bool("complete", complete))
	return &pb.GetProfileResponse{Code: errs.OK, Msg: "保存成功", Profile: toPBProfile(profile)}, nil
}

func (s *CandidateService) GetResume(ctx context.Context, req *pb.GetResumeRequest) (*pb.GetResumeResponse, error) {
	resume, err := s.resumes.GetValidByUserID(ctx, req.UserId)
	if err != nil {
		logger.L().Error("get resume failed", zap.Int64("user_id", req.UserId), zap.Error(err))
		return nil, err
	}
	if resume == nil {
		return &pb.GetResumeResponse{Code: errs.OK, Msg: "success"}, nil
	}
	resumeURL, err := s.oss.GeneratePresignedGetURL(resume.OSSKey)
	if err != nil {
		return nil, wrapOSSError(err)
	}
	return &pb.GetResumeResponse{Code: errs.OK, Msg: "success", Resume: toPBResume(resume, resumeURL)}, nil
}

func (s *CandidateService) PresignResumeUpload(ctx context.Context, req *pb.PresignResumeUploadRequest) (*pb.PresignResumeUploadResponse, error) {
	if !allowedFileType(req.FileName, req.FileType) {
		return &pb.PresignResumeUploadResponse{Code: errs.ErrBadRequest, Msg: "仅支持 " + allowedFileTypesText() + " 格式"}, nil
	}
	uploadID, err := oss.GenerateUploadID()
	if err != nil {
		return nil, err
	}
	safeName := sanitizeFileName(req.FileName)
	ossKey := fmt.Sprintf("resumes/tmp/%d/%s/%s", req.UserId, uploadID, safeName)
	contentType := oss.ContentTypeFromFileType(req.FileType)
	uploadURL, expireAt, err := s.oss.GeneratePresignedPutURL(ossKey, contentType)
	if err != nil {
		logger.L().Error("presign upload failed", zap.Int64("user_id", req.UserId), zap.Error(err))
		return nil, err
	}

	// Store upload session in Redis to bind the upload to this user.
	// The session is validated at confirm time to prevent cross-user confirmation.
	session := oss.PresignSession{
		UserID:      req.UserId,
		OssKey:      ossKey,
		FileName:    req.FileName,
		FileType:    req.FileType,
		ContentType: contentType,
		MaxSize:     oss.MaxResumeSizeBytes,
		Status:      "pending",
	}
	if err := s.oss.SavePresignSessionWithID(ctx, uploadID, session); err != nil {
		logger.L().Error("save presign session failed", zap.Int64("user_id", req.UserId), zap.Error(err))
		return nil, err
	}

	writeAuditLog(ctx, s.usageLogs, AuditLogEntry{
		UserID: req.UserId, Role: 1, ServiceType: "oss_presign",
		Endpoint: "/candidate/resume/presign", Provider: ossProviderName(s.oss),
		ObjectKey: ossKey, ObjectSize: oss.MaxResumeSizeBytes,
	})
	logger.L().Info("resume presign generated", zap.Int64("user_id", req.UserId), zap.String("file_name", req.FileName))
	return &pb.PresignResumeUploadResponse{Code: errs.OK, Msg: "success", UploadUrl: uploadURL, OssKey: ossKey, ExpireAt: formatTime(expireAt), UploadId: uploadID}, nil
}

func (s *CandidateService) ConfirmResumeUpload(ctx context.Context, req *pb.ConfirmResumeUploadRequest) (*pb.ConfirmResumeUploadResponse, error) {
	if !allowedFileType(req.FileName, req.FileType) {
		return &pb.ConfirmResumeUploadResponse{Code: errs.ErrBadRequest, Msg: "仅支持 " + allowedFileTypesText() + " 格式"}, nil
	}

	// STRICT_RESUME_CONFIRM=true enables full session validation (production default).
	// When false (legacy compat), falls back to oss_key prefix check.
	if os.Getenv("STRICT_RESUME_CONFIRM") != "false" {
		if req.UploadId == "" {
			return &pb.ConfirmResumeUploadResponse{Code: errs.ErrBadRequest, Msg: "缺少上传凭证，请重新上传"}, nil
		}
		session, err := s.oss.GetAndDeletePresignSession(ctx, req.UploadId)
		if err != nil {
			return &pb.ConfirmResumeUploadResponse{Code: errs.ErrBadRequest, Msg: "上传凭证无效或已过期，请重新上传"}, nil
		}
		if session.UserID != req.UserId {
			logger.L().Warn("resume confirm user mismatch", zap.Int64("req_user", req.UserId), zap.Int64("session_user", session.UserID))
			return &pb.ConfirmResumeUploadResponse{Code: errs.ErrBadRequest, Msg: "上传凭证与当前用户不匹配"}, nil
		}
		if session.OssKey != req.OssKey {
			return &pb.ConfirmResumeUploadResponse{Code: errs.ErrBadRequest, Msg: "文件信息不匹配，请重新上传"}, nil
		}
		if session.FileType != req.FileType {
			return &pb.ConfirmResumeUploadResponse{Code: errs.ErrBadRequest, Msg: "文件类型与上传凭证不一致，请重新上传"}, nil
		}
		if session.ContentType != "" && req.FileType != "" {
			expectedCT := oss.ContentTypeFromFileType(req.FileType)
			if session.ContentType != expectedCT {
				return &pb.ConfirmResumeUploadResponse{Code: errs.ErrBadRequest, Msg: "文件类型与上传凭证不一致，请重新上传"}, nil
			}
		}
		if session.Status != "pending" {
			return &pb.ConfirmResumeUploadResponse{Code: errs.ErrBadRequest, Msg: "上传凭证已失效，请重新上传"}, nil
		}
		if req.FileSize > session.MaxSize {
			return &pb.ConfirmResumeUploadResponse{Code: errs.ErrBadRequest, Msg: "简历文件大小超过限制（最大 20MB）"}, nil
		}
	} else {
		// Legacy compat: verify oss_key starts with resumes/{user_id}/
		expectedPrefix := fmt.Sprintf("resumes/%d/", req.UserId)
		if !strings.HasPrefix(req.OssKey, expectedPrefix) {
			logger.L().Warn("resume confirm oss_key prefix mismatch", zap.String("key", req.OssKey), zap.Int64("user_id", req.UserId))
			return &pb.ConfirmResumeUploadResponse{Code: errs.ErrBadRequest, Msg: "文件信息与当前用户不匹配"}, nil
		}
	}

	// Verify file exists and check size.
	if err := s.oss.VerifyObject(ctx, req.OssKey); oss.IsNotFound(err) {
		return &pb.ConfirmResumeUploadResponse{Code: errs.ErrBadRequest, Msg: "未在 OSS 中找到已上传的简历文件"}, nil
	} else if err != nil {
		logger.L().Error("verify oss object failed", zap.String("oss_key", req.OssKey), zap.Error(err))
		return nil, err
	}
	if err := s.oss.VerifyObjectSize(ctx, req.OssKey, oss.MaxResumeSizeBytes); err != nil {
		logger.L().Warn("resume file size exceeds limit", zap.String("oss_key", req.OssKey), zap.Error(err))
		return &pb.ConfirmResumeUploadResponse{Code: errs.ErrBadRequest, Msg: "简历文件大小超过限制（最大 20MB）"}, nil
	}

	// Copy from temp to permanent key, then delete the temp object.
	safeName := sanitizeFileName(req.FileName)
	permanentKey := fmt.Sprintf("resumes/%d/%d_%s", req.UserId, time.Now().Unix(), safeName)
	if err := s.oss.CopyObject(ctx, req.OssKey, permanentKey); err != nil {
		logger.L().Error("copy resume to permanent key failed", zap.String("src", req.OssKey), zap.String("dst", permanentKey), zap.Error(err))
		return &pb.ConfirmResumeUploadResponse{Code: 500, Msg: "简历保存失败，请重新上传"}, nil
	}
	if err := s.oss.DeleteObject(ctx, req.OssKey); err != nil {
		logger.L().Warn("delete tmp resume object failed", zap.String("key", req.OssKey), zap.Error(err))
	}

	resume := &model.Resume{UserID: req.UserId, OSSKey: permanentKey, FileName: req.FileName, FileType: strings.ToLower(req.FileType), FileSize: req.FileSize, IsValid: 1, UploadedAt: time.Now()}
	if err := s.resumes.ConfirmUploadWithTx(ctx, resume, func(tx *gorm.DB) error {
		return s.outboxPublisher.WriteEventTx(tx, "resume.parse", "resume", uint64(resume.ID), "resume.parse", resumeParsePayload{
			ResumeID: resume.ID,
			FileType: resume.FileType,
			OSSKey:   resume.OSSKey,
		})
	}); err != nil {
		logger.L().Error("confirm resume failed", zap.Int64("user_id", req.UserId), zap.Error(err))
		if errors.Is(err, errs.ErrDuplicateResumeSentinel) {
			return &pb.ConfirmResumeUploadResponse{
				Code: errs.ErrDuplicateResume,
				Msg:  "系统繁忙，请稍后重试",
			}, nil
		}
		return nil, err
	}
	s.outboxPublisher.Signal()
	writeAuditLog(ctx, s.usageLogs, AuditLogEntry{
		UserID: req.UserId, Role: 1, ServiceType: "oss_confirm",
		Endpoint: "/candidate/resume/confirm", Provider: ossProviderName(s.oss),
		ObjectKey: req.OssKey, ObjectSize: req.FileSize,
	})
	logger.L().Info("resume upload confirmed", zap.Int64("user_id", req.UserId), zap.Int64("resume_id", resume.ID))
	return &pb.ConfirmResumeUploadResponse{Code: errs.OK, Msg: "success", ResumeId: resume.ID}, nil
}


func ossProviderName(s oss.Storage) string {
	return s.ProviderName()
}