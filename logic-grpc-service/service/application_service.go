package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"logic-grpc-service/model"
	"logic-grpc-service/oss"
	"logic-grpc-service/pkg/cache"
	"logic-grpc-service/pkg/errs"
	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/recruitment/pb"
	"logic-grpc-service/repository"
)

type ApplicationService struct {
	applications    *repository.ApplicationRepo
	profiles        *repository.ProfileRepo
	resumes         *repository.ResumeRepo
	jobs            *repository.JobRepo
	notifications   *repository.NotificationRepo
	outboxPublisher *OutboxPublisher
	oss             *oss.Client
	jobCache        *cache.JobCache
}

func NewApplicationService(applications *repository.ApplicationRepo, profiles *repository.ProfileRepo, resumes *repository.ResumeRepo, jobs *repository.JobRepo, notifications *repository.NotificationRepo, outboxPublisher *OutboxPublisher, ossClient *oss.Client, jobCache *cache.JobCache) *ApplicationService {
	return &ApplicationService{applications: applications, profiles: profiles, resumes: resumes, jobs: jobs, notifications: notifications, outboxPublisher: outboxPublisher, oss: ossClient, jobCache: jobCache}
}

func (s *ApplicationService) ApplyJob(ctx context.Context, req *pb.ApplyJobRequest) (*pb.CommonResponse, error) {
	profile, err := s.profiles.GetByUserID(ctx, req.UserId)
	if err != nil {
		return nil, err
	}
	if profile == nil || profile.IsComplete != 1 {
		return &pb.CommonResponse{Code: errs.ErrProfileIncomplete, Msg: "请先完善个人资料后再投递"}, nil
	}
	resume, err := s.resumes.GetValidByUserID(ctx, req.UserId)
	if err != nil {
		return nil, err
	}
	if resume == nil {
		return &pb.CommonResponse{Code: errs.ErrResumeNotFound, Msg: "请先上传简历后再投递"}, nil
	}
	job, err := s.jobs.GetByID(ctx, req.JobId)
	if err != nil {
		return nil, err
	}
	if job == nil || job.Status != 1 {
		return &pb.CommonResponse{Code: errs.ErrJobNotAvailable, Msg: "该岗位已下架或不存在，无法投递"}, nil
	}
	app := &model.Application{UserID: req.UserId, JobID: req.JobId, ResumeID: resume.ID, Status: 0, AppliedAt: time.Now()}
	err = s.applications.Transaction(ctx, func(tx *gorm.DB) error {
		if err := s.applications.CreateNewRoundWithTx(ctx, tx, app); err != nil {
			return err
		}
		return s.outboxPublisher.WriteEventTx(tx, "notification.create", "application", uint64(app.ID), "notification.create", notificationPayload{
			ReceiverID:   job.HrID,
			ReceiverRole: 2,
			Type:         "new_application",
			Title:        "新的岗位投递",
			Content:      fmt.Sprintf("%s 投递了「%s」岗位，请及时查看简历。", candidateDisplayName(profile.RealName, req.UserId), job.Title),
			Link:         fmt.Sprintf("/hr/jobs/%d/applications", job.ID),
			BizType:      "application",
			BizID:        app.ID,
		})
	})
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return &pb.CommonResponse{Code: errs.ErrDuplicateApply, Msg: "您已投递过该岗位，当前流程结束前不能重复投递"}, nil
	}
	if err != nil {
		logger.L().Error("apply job failed", zap.Int64("user_id", req.UserId), zap.Int64("job_id", req.JobId), zap.Error(err))
		return nil, err
	}
	s.outboxPublisher.Signal()
	logger.L().Info("job applied", zap.Int64("user_id", req.UserId), zap.Int64("job_id", req.JobId))
	s.invalidateJobCache(ctx)

	return &pb.CommonResponse{Code: errs.OK, Msg: "投递成功"}, nil
}

func (s *ApplicationService) invalidateJobCache(ctx context.Context) {
	if s.jobCache != nil {
		s.jobCache.InvalidatePublicFirstPage(ctx)
	}
}

func (s *ApplicationService) ListMyApplications(ctx context.Context, req *pb.ListMyApplicationsRequest) (*pb.ListMyApplicationsResponse, error) {
	if req.Cursor != "" || req.Page <= 0 {
		rows, cursor, hasMore, err := s.applications.ListMyCursor(ctx, req.UserId, req.Cursor, pageSize(req.PageSize))
		if err != nil {
			logger.L().Error("list my applications cursor failed", zap.Int64("user_id", req.UserId), zap.Error(err))
			return nil, err
		}
		list := make([]*pb.MyApplication, 0, len(rows))
		for _, row := range rows {
			list = append(list, &pb.MyApplication{ApplicationId: row.ApplicationID, JobId: row.JobID, JobTitle: row.JobTitle, Status: row.Status, AppliedAt: formatTime(row.AppliedAt), RoundNo: row.RoundNo, IsCurrent: row.IsCurrent})
		}
		return &pb.ListMyApplicationsResponse{Code: errs.OK, Msg: "success", List: list, NextCursor: cursor, HasMore: hasMore}, nil
	}
	rows, total, err := s.applications.ListMy(ctx, req.UserId, page(req.Page), pageSize(req.PageSize))
	if err != nil {
		logger.L().Error("list my applications failed", zap.Int64("user_id", req.UserId), zap.Error(err))
		return nil, err
	}
	list := make([]*pb.MyApplication, 0, len(rows))
	for _, row := range rows {
		list = append(list, &pb.MyApplication{ApplicationId: row.ApplicationID, JobId: row.JobID, JobTitle: row.JobTitle, Status: row.Status, AppliedAt: formatTime(row.AppliedAt), RoundNo: row.RoundNo, IsCurrent: row.IsCurrent})
	}
	return &pb.ListMyApplicationsResponse{Code: errs.OK, Msg: "success", Total: total, List: list}, nil
}

func (s *ApplicationService) ListJobApplications(ctx context.Context, req *pb.ListJobApplicationsRequest) (*pb.ListJobApplicationsResponse, error) {
	ok, err := s.jobs.BelongsToHR(ctx, req.HrId, req.JobId)
	if err != nil {
		return nil, err
	}
	if !ok {
		return &pb.ListJobApplicationsResponse{Code: errs.ErrForbidden, Msg: "无权限查看该岗位"}, nil
	}
	rows, total, err := s.applications.ListByJob(ctx, req.JobId, page(req.Page), pageSize(req.PageSize))
	if err != nil {
		logger.L().Error("list job applications failed", zap.Int64("job_id", req.JobId), zap.Error(err))
		return nil, err
	}
	list := make([]*pb.JobApplication, 0, len(rows))
	for _, row := range rows {
		resumeURL, err := s.oss.GeneratePresignedGetURL(row.OSSKey)
		if err != nil {
			return nil, err
		}
		list = append(list, &pb.JobApplication{ApplicationId: row.ApplicationID, UserId: row.UserID, RealName: row.RealName, Phone: row.Phone, Education: row.Education, School: row.School, Skills: splitSkills(row.Skills), AppliedAt: formatTime(row.AppliedAt), ResumeUrl: resumeURL, FileName: row.FileName, FileType: row.FileType, Status: row.Status, RoundNo: row.RoundNo, IsCurrent: row.IsCurrent})
	}
	return &pb.ListJobApplicationsResponse{Code: errs.OK, Msg: "success", Total: total, List: list}, nil
}

func (s *ApplicationService) UpdateApplicationStatus(ctx context.Context, req *pb.UpdateApplicationStatusRequest) (*pb.CommonResponse, error) {
	if req.Status < 0 || req.Status > 3 {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "投递状态不合法"}, nil
	}
	detail, err := s.applications.GetDetailOwned(ctx, req.HrId, req.ApplicationId)
	if err != nil {
		return nil, err
	}
	if detail == nil {
		return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: "该投递记录不存在或无权限访问"}, nil
	}
	if detail.IsCurrent != 1 && !(detail.Status == 3 && req.Status == 2) {
		return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: "该投递已不是当前有效流程，不能修改状态"}, nil
	}
	var notifyType, notifyContent string
	switch req.Status {
	case 2:
		notifyType = "application_approved"
		notifyContent = fmt.Sprintf("你投递的「%s」岗位已通过筛选，请留意后续安排。", detail.JobTitle)
	case 3:
		notifyType = "application_rejected"
		notifyContent = fmt.Sprintf("你投递的「%s」岗位当前未通过筛选，感谢你的投递。", detail.JobTitle)
	}
	var rows int64
	err = s.applications.Transaction(ctx, func(tx *gorm.DB) error {
		var err error
		rows, err = s.applications.UpdateStatusOwnedWithTx(ctx, tx, req.HrId, req.ApplicationId, req.Status)
		if err != nil {
			return err
		}
		if rows > 0 && req.Status == 3 {
			tx.Model(&model.Application{}).Where("id = ?", req.ApplicationId).Update("is_current", 0)
		}
		if rows > 0 && req.Status == 2 && detail.Status == 3 {
			tx.Model(&model.Application{}).Where("user_id = ? AND job_id = ? AND is_current = 1", detail.UserID, detail.JobID).Update("is_current", 0)
			tx.Model(&model.Application{}).Where("id = ?", req.ApplicationId).Update("is_current", 1)
		}
		if rows == 0 || notifyType == "" {
			return nil
		}
		return s.outboxPublisher.WriteEventTx(tx, "notification.create", "application", uint64(req.ApplicationId), "notification.create", notificationPayload{
			ReceiverID:   detail.UserID,
			ReceiverRole: 1,
			Type:         notifyType,
			Title:        "投递进展更新",
			Content:      notifyContent,
			Link:         "/applications",
			BizType:      "application",
			BizID:        req.ApplicationId,
		})
	})
	if err != nil {
		return nil, err
	}
	s.outboxPublisher.Signal()
	if rows == 0 && detail.Status != req.Status {
		return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: "该投递已不是当前有效流程，不能修改状态"}, nil
	}
	logger.L().Info("application status updated", zap.Int64("application_id", req.ApplicationId), zap.Int32("status", req.Status))

	return &pb.CommonResponse{Code: errs.OK, Msg: "投递状态已更新"}, nil
}
