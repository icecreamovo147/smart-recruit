package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	authzRepo       *repository.AuthzRepo
	applications    *repository.ApplicationRepo
	profiles        *repository.ProfileRepo
	resumes         *repository.ResumeRepo
	jobs            *repository.JobRepo
	notifications   *repository.NotificationRepo
	outboxPublisher *OutboxPublisher
	oss             oss.Storage
	jobCache        *cache.JobCache
	scopeEval       *scopeEvaluator
}

func NewApplicationService(authzRepo *repository.AuthzRepo, applications *repository.ApplicationRepo, profiles *repository.ProfileRepo, resumes *repository.ResumeRepo, jobs *repository.JobRepo, notifications *repository.NotificationRepo, outboxPublisher *OutboxPublisher, ossClient oss.Storage, jobCache *cache.JobCache, scopeEval *scopeEvaluator) *ApplicationService {
	return &ApplicationService{authzRepo: authzRepo, applications: applications, profiles: profiles, resumes: resumes, jobs: jobs, notifications: notifications, outboxPublisher: outboxPublisher, oss: ossClient, jobCache: jobCache, scopeEval: scopeEval}
}

// getAppScopeDeptAndLocIDs returns the department and location IDs for a department/location-scoped user.
func (s *ApplicationService) getAppScopeDeptAndLocIDs(ctx context.Context, userID uint64) ([]uint64, []uint64) {
	if s.authzRepo == nil {
		return nil, nil
	}
	deptIDs, _ := s.authzRepo.GetUserDepartmentIDs(ctx, userID)
	locIDs, _ := s.authzRepo.GetUserLocationIDs(ctx, userID)
	return deptIDs, locIDs
}

// checkApplicationScope verifies the user can access applications for a job based on their data scope.
// checkApplicationJobScope verifies the user can access applications for a job based on their data scope.
// Delegates to the shared scopeEvaluator to avoid duplication with JobService.
func (s *ApplicationService) checkApplicationJobScope(ctx context.Context, userID int64, jobID int64) (ScopeLevel, error) {
	// Defensive nil guard: scopeEval is wired in production via services.go, but
	// missing in some tests. Fail-closed rather than panicking.
	if s.scopeEval == nil {
		return scopeDenied, fmt.Errorf("scope evaluator not configured")
	}
	var jobGetter func() (*jobScopeTarget, error)
	if jobID > 0 {
		jobGetter = func() (*jobScopeTarget, error) {
			job, err := s.jobs.GetByID(ctx, jobID)
			if err != nil {
				return nil, err
			}
			if job == nil {
				return nil, fmt.Errorf("job %d not found", jobID)
			}
			return &jobScopeTarget{
				ID:           job.ID,
				HrID:         job.HrID,
				DepartmentID: job.DepartmentID,
				LocationID:   job.LocationID,
			}, nil
		}
	}
	return s.scopeEval.evalScope(ctx, uint64(userID), jobGetter)
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
	app := &model.Application{UserID: req.UserId, JobID: req.JobId, ResumeID: resume.ID, Status: 0, StatusKey: model.DefaultStatusKey(), AppliedAt: time.Now()}
	err = s.applications.Transaction(ctx, func(tx *gorm.DB) error {
		if err := s.applications.CreateNewRoundWithTx(ctx, tx, app); err != nil {
			return err
		}
		return s.outboxPublisher.WriteEventTx(tx, "notification.create", "application", uint64(app.ID), "notification.create", notificationPayload{
			ReceiverID:   job.HrID,
			ReceiverRole: 2, ReceiverAccountType: "staff",
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
			list = append(list, &pb.MyApplication{ApplicationId: row.ApplicationID, JobId: row.JobID, JobTitle: row.JobTitle, Status: row.Status, StatusKey: row.StatusKey, AppliedAt: formatTime(row.AppliedAt), RoundNo: row.RoundNo, IsCurrent: row.IsCurrent})
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
		list = append(list, &pb.MyApplication{ApplicationId: row.ApplicationID, JobId: row.JobID, JobTitle: row.JobTitle, Status: row.Status, StatusKey: row.StatusKey, AppliedAt: formatTime(row.AppliedAt), RoundNo: row.RoundNo, IsCurrent: row.IsCurrent})
	}
	return &pb.ListMyApplicationsResponse{Code: errs.OK, Msg: "success", Total: total, List: list}, nil
}

func (s *ApplicationService) ListJobApplications(ctx context.Context, req *pb.ListJobApplicationsRequest) (*pb.ListJobApplicationsResponse, error) {
	if _, err := s.checkApplicationJobScope(ctx, req.HrId, req.JobId); err != nil {
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
		list = append(list, &pb.JobApplication{ApplicationId: row.ApplicationID, UserId: row.UserID, RealName: row.RealName, Phone: row.Phone, Education: row.Education, School: row.School, Skills: splitSkills(row.Skills), AppliedAt: formatTime(row.AppliedAt), ResumeUrl: resumeURL, FileName: row.FileName, FileType: row.FileType, Status: row.Status, StatusKey: row.StatusKey, RoundNo: row.RoundNo, IsCurrent: row.IsCurrent})
	}
	return &pb.ListJobApplicationsResponse{Code: errs.OK, Msg: "success", Total: total, List: list}, nil
}

func (s *ApplicationService) UpdateApplicationStatus(ctx context.Context, req *pb.UpdateApplicationStatusRequest) (*pb.CommonResponse, error) {
	// Determine the target status key.
	targetKey := req.StatusKey
	if targetKey == "" {
		// Fallback to legacy numeric status for migration compatibility.
		if req.Status >= 0 && int(req.Status) < len(model.LegacyStatusToKey) {
			targetKey = model.LegacyStatusToKey[req.Status]
		}
	}
	if targetKey == "" {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "投递状态不合法"}, nil
	}
	if err := ValidateStatusKey(targetKey); err != nil {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: err.Error()}, nil
	}

	// Load application first to determine the job for scope check.
	detail, err := s.applications.GetDetail(ctx, req.ApplicationId)
	if err != nil {
		return nil, err
	}
	if detail == nil {
		return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: "该投递记录不存在或无权限访问"}, nil
	}

	// Derive current status key from the detail.
	currentKey := detail.StatusKey
	if currentKey == "" {
		currentKey = model.LegacyStatusToKey[detail.Status]
	}

	// Validate transition.
	if err := ValidateTransition(currentKey, targetKey); err != nil {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: err.Error()}, nil
	}

	// FR-4: Reason is required for rejection, withdrawal, and offer rejection.
	reasonRequired := map[string]bool{
		model.StatusKeyRejected:      true,
		model.StatusKeyWithdrawn:     true,
		model.StatusKeyOfferRejected: true,
	}
	if reasonRequired[targetKey] && strings.TrimSpace(req.Reason) == "" {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "该状态变更必须填写原因"}, nil
	}

	// Scope check against the application's job.
	scopeLevel, scopeErr := s.checkApplicationJobScope(ctx, req.HrId, detail.JobID)
	if scopeErr != nil {
		return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: "无权限操作投递状态"}, nil
	}

	// Detect HR re-pass: rejected → screen_passed creates a new application round.
	isRePass := currentKey == model.StatusKeyRejected && targetKey == model.StatusKeyScreenPassed

	if detail.IsCurrent != 1 && !isRePass {
		return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: "该投递已不是当前有效流程，不能修改状态"}, nil
	}

	legacyStatus := model.StatusKeyToLegacy[targetKey]

	// Determine notification based on target key.
	var notifyType, notifyContent string
	switch targetKey {
	case model.StatusKeyViewed:
		// Viewed is informational only — no notification to candidate.
	case model.StatusKeyScreenPassed:
		notifyType = "application_approved"
		if isRePass {
			notifyContent = fmt.Sprintf("你投递的「%s」岗位已重新通过筛选（第%d轮），请留意后续安排。", detail.JobTitle, detail.RoundNo+1)
		} else {
			notifyContent = fmt.Sprintf("你投递的「%s」岗位已通过筛选，请留意后续安排。", detail.JobTitle)
		}
	case model.StatusKeyRejected:
		notifyType = "application_rejected"
		notifyContent = fmt.Sprintf("你投递的「%s」岗位当前未通过筛选，感谢你的投递。", detail.JobTitle)
	case model.StatusKeyWithdrawn:
		notifyType = "application_withdrawn"
		notifyContent = fmt.Sprintf("你已撤回对「%s」岗位的投递。", detail.JobTitle)
	case model.StatusKeyHired:
		notifyType = "application_hired"
		notifyContent = fmt.Sprintf("恭喜！你投递的「%s」岗位已确认入职。", detail.JobTitle)
	}

	var rows int64
	err = s.applications.Transaction(ctx, func(tx *gorm.DB) error {
		if isRePass {
			// Re-pass: increment round_no, reset is_current, update status — all in one atomic update.
			var deptIDs, locIDs []uint64
			if scopeLevel == scopeDepartmentOrLocation {
				deptIDs, locIDs = s.getAppScopeDeptAndLocIDs(ctx, uint64(req.HrId))
			}
			rows, err = s.applications.RePassWithTx(ctx, tx, req.ApplicationId, currentKey, targetKey, legacyStatus, req.HrId, deptIDs, locIDs, int(scopeLevel))
		} else {
			if scopeLevel >= scopeFull {
				rows, err = s.applications.UpdateStatusAnyWithTx(ctx, tx, req.ApplicationId, currentKey, targetKey, legacyStatus)
			} else if scopeLevel == scopeDepartmentOrLocation {
				deptIDs, locIDs := s.getAppScopeDeptAndLocIDs(ctx, uint64(req.HrId))
				rows, err = s.applications.UpdateStatusInScopeWithTx(ctx, tx, deptIDs, locIDs, req.ApplicationId, currentKey, targetKey, legacyStatus)
			} else {
				rows, err = s.applications.UpdateStatusOwnedWithTx(ctx, tx, req.HrId, req.ApplicationId, currentKey, targetKey, legacyStatus)
			}
		}
		if err != nil {
			return err
		}
		if rows > 0 && model.IsTerminalStatusKey(targetKey) && !isRePass {
			if err := tx.Model(&model.Application{}).Where("id = ?", req.ApplicationId).Update("is_current", 0).Error; err != nil {
				return err
			}
		}

		// Write transition audit row.
		if rows > 0 {
			actorAccountType := "staff"
			transition := &model.ApplicationStatusTransition{
				ApplicationID:    req.ApplicationId,
				FromStatus:       currentKey,
				ToStatus:         targetKey,
				ActorUserID:      req.HrId,
				ActorAccountType: actorAccountType,
				Reason:           req.Reason,
			}
			if err := s.applications.CreateTransition(ctx, tx, transition); err != nil {
				return err
			}
		}

		if rows == 0 || notifyType == "" {
			return nil
		}
		return s.outboxPublisher.WriteEventTx(tx, "notification.create", "application", uint64(req.ApplicationId), "notification.create", notificationPayload{
			ReceiverID:   detail.UserID,
			ReceiverRole: 1, ReceiverAccountType: "candidate",
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
	if rows == 0 {
		return &pb.CommonResponse{Code: errs.ErrConflict, Msg: "投递状态已变化，请刷新后重试"}, nil
	}
	logger.L().Info("application status updated",
		zap.Int64("application_id", req.ApplicationId),
		zap.String("from_status", currentKey),
		zap.String("to_status", targetKey),
	)

	return &pb.CommonResponse{Code: errs.OK, Msg: "投递状态已更新"}, nil
}

func (s *ApplicationService) ListApplicationStatusTransitions(ctx context.Context, req *pb.ListApplicationStatusTransitionsRequest) (*pb.ListApplicationStatusTransitionsResponse, error) {
	// Load application first to determine the job for scope check.
	detail, err := s.applications.GetDetail(ctx, req.ApplicationId)
	if err != nil {
		return nil, err
	}
	if detail == nil {
		return &pb.ListApplicationStatusTransitionsResponse{Code: errs.ErrForbidden, Msg: "该投递记录不存在"}, nil
	}

	// Scope check against the application's job.
	if _, scopeErr := s.checkApplicationJobScope(ctx, req.HrId, detail.JobID); scopeErr != nil {
		return &pb.ListApplicationStatusTransitionsResponse{Code: errs.ErrForbidden, Msg: "无权限查看投递状态变更记录"}, nil
	}

	transitions, err := s.applications.ListTransitions(ctx, req.ApplicationId)
	if err != nil {
		return nil, err
	}

	list := make([]*pb.ApplicationStatusTransition, 0, len(transitions))
	for _, t := range transitions {
		list = append(list, &pb.ApplicationStatusTransition{
			Id:               int64(t.ID),
			ApplicationId:    t.ApplicationID,
			FromStatus:       t.FromStatus,
			ToStatus:         t.ToStatus,
			ActorUserId:      t.ActorUserID,
			ActorAccountType: t.ActorAccountType,
			Reason:           t.Reason,
			CreatedAt:        t.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &pb.ListApplicationStatusTransitionsResponse{Code: errs.OK, Msg: "success", List: list}, nil
}
