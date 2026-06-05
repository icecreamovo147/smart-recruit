package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"logic-grpc-service/model"
	"logic-grpc-service/pkg/authz"
	"logic-grpc-service/pkg/errs"
	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/recruitment/pb"
	"logic-grpc-service/repository"
)

type OfferService struct {
	authzRepo       *repository.AuthzRepo
	offers          *repository.OfferRepo
	applications    *repository.ApplicationRepo
	jobs            *repository.JobRepo
	notifications   *repository.NotificationRepo
	outboxPublisher *OutboxPublisher
	scopeEval       *scopeEvaluator
	serviceAuth     *ServiceAuthorizer
}

func NewOfferService(
	authzRepo *repository.AuthzRepo,
	offers *repository.OfferRepo,
	applications *repository.ApplicationRepo,
	jobs *repository.JobRepo,
	notifications *repository.NotificationRepo,
	outboxPublisher *OutboxPublisher,
	scopeEval *scopeEvaluator,
	serviceAuth *ServiceAuthorizer,
) *OfferService {
	return &OfferService{
		authzRepo:       authzRepo,
		offers:          offers,
		applications:    applications,
		jobs:            jobs,
		notifications:   notifications,
		outboxPublisher: outboxPublisher,
		scopeEval:       scopeEval,
		serviceAuth:     serviceAuth,
	}
}

// ── Authorization helpers ─────────────────────────────────────────────

// checkOfferManageScope verifies the user has offer.manage permission
// and scope access to the application's job.
func (s *OfferService) checkOfferManageScope(ctx context.Context, userID int64, applicationID int64) error {
	if err := s.serviceAuth.AuthorizePermission(ctx, uint64(userID), authz.PermOfferManage); err != nil {
		return fmt.Errorf("permission denied: offer.manage required: %w", err)
	}

	// Load the application to get the job ID for scope check.
	detail, err := s.applications.GetDetail(ctx, applicationID)
	if err != nil {
		return err
	}
	if detail == nil {
		return fmt.Errorf("application %d not found", applicationID)
	}

	// Evaluate scope against the application's job.
	_, err = s.scopeEval.evalScope(ctx, uint64(userID), func() (*jobScopeTarget, error) {
		job, err := s.jobs.GetByID(ctx, detail.JobID)
		if err != nil {
			return nil, err
		}
		if job == nil {
			return nil, fmt.Errorf("job %d not found", detail.JobID)
		}
		return &jobScopeTarget{
			ID:           job.ID,
			HrID:         job.HrID,
			DepartmentID: job.DepartmentID,
			LocationID:   job.LocationID,
		}, nil
	})
	if err != nil {
		return fmt.Errorf("scope denied for application %d: %w", applicationID, err)
	}
	return nil
}

// checkOfferReadScope verifies the user can view offers for an application.
func (s *OfferService) checkOfferReadScope(ctx context.Context, userID int64, applicationID int64) error {
	if err := s.serviceAuth.AuthorizePermission(ctx, uint64(userID), authz.PermOfferRead); err != nil {
		return fmt.Errorf("permission denied: offer.read required: %w", err)
	}

	detail, err := s.applications.GetDetail(ctx, applicationID)
	if err != nil {
		return err
	}
	if detail == nil {
		return fmt.Errorf("application %d not found", applicationID)
	}

	// Evaluate scope against the application's job.
	_, err = s.scopeEval.evalScope(ctx, uint64(userID), func() (*jobScopeTarget, error) {
		job, err := s.jobs.GetByID(ctx, detail.JobID)
		if err != nil {
			return nil, err
		}
		if job == nil {
			return nil, fmt.Errorf("job %d not found", detail.JobID)
		}
		return &jobScopeTarget{
			ID:           job.ID,
			HrID:         job.HrID,
			DepartmentID: job.DepartmentID,
			LocationID:   job.LocationID,
		}, nil
	})
	if err != nil {
		return fmt.Errorf("scope denied for application %d: %w", applicationID, err)
	}
	return nil
}

// ── Service methods ────────────────────────────────────────────────────

func (s *OfferService) CreateOffer(ctx context.Context, req *pb.CreateOfferRequest) (*pb.CreateOfferResponse, error) {
	if err := s.serviceAuth.VerifyActorMatch(ctx, req.HrId); err != nil {
		return nil, err
	}

	// Permission + scope check
	if err := s.checkOfferManageScope(ctx, req.HrId, req.ApplicationId); err != nil {
		return &pb.CreateOfferResponse{Code: errs.ErrForbidden, Msg: err.Error()}, nil
	}

	// Load application detail
	appDetail, err := s.applications.GetDetail(ctx, req.ApplicationId)
	if err != nil {
		return nil, err
	}
	if appDetail == nil {
		return &pb.CreateOfferResponse{Code: errs.ErrBadRequest, Msg: "投递记录不存在"}, nil
	}

	// Parse expires_at if provided
	var expiresAt *time.Time
	if req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			return &pb.CreateOfferResponse{Code: errs.ErrBadRequest, Msg: "过期时间格式错误，请使用 RFC 3339 格式"}, nil
		}
		expiresAt = &t
	}

	// Build offer model
	offer := &model.Offer{
		ApplicationID:   req.ApplicationId,
		CandidateUserID: appDetail.UserID,
		JobID:           appDetail.JobID,
		Status:          "draft",
		Title:           req.Title,
		SalaryRange:     req.SalaryRange,
		Level:           req.Level,
		WorkLocation:    req.WorkLocation,
		StartDate:       req.StartDate,
		ExpiresAt:       expiresAt,
		TermsJSON:       req.TermsJson,
		CreatedBy:       req.HrId,
	}

	// Validate transition before attempting update
	if err := ValidateTransition(appDetail.StatusKey, model.StatusKeyOfferPending); err != nil {
		return &pb.CreateOfferResponse{Code: errs.ErrBadRequest, Msg: err.Error()}, nil
	}

	// Transaction: create offer + update application status to offer_pending + event + notification
	err = s.offers.Transaction(ctx, func(tx *gorm.DB) error {
		if err := s.offers.CreateWithTx(ctx, tx, offer); err != nil {
			return err
		}

		// Update application status to offer_pending if not already
		rows, err := s.applications.UpdateStatusAnyWithTx(ctx, tx, req.ApplicationId, appDetail.StatusKey, model.StatusKeyOfferPending, 0)
		if err != nil {
			return err
		}
		if rows == 0 {
			return fmt.Errorf("application status changed concurrently, expected %s", appDetail.StatusKey)
		}

		// Write application status transition audit record
		if err := s.applications.CreateTransition(ctx, tx, &model.ApplicationStatusTransition{
			ApplicationID:    req.ApplicationId,
			FromStatus:       appDetail.StatusKey,
			ToStatus:         model.StatusKeyOfferPending,
			ActorUserID:      req.HrId,
			ActorAccountType: "staff",
		}); err != nil {
			return err
		}

		// Write offer event
		event := &model.OfferEvent{
			OfferID:          offer.ID,
			EventType:        "created",
			ActorUserID:      req.HrId,
			ActorAccountType: "staff",
			Reason:           "",
			CreatedAt:        time.Now(),
		}
		if err := s.offers.CreateEventWithTx(ctx, tx, event); err != nil {
			return err
		}

		// Notify candidate
		notifyContent := fmt.Sprintf("您投递的「%s」岗位已生成 Offer，请留意查看。", appDetail.JobTitle)
		if err := s.outboxPublisher.WriteEventTx(tx, "notification.create", "offer", uint64(offer.ID), "notification.create", notificationPayload{
			ReceiverID:          appDetail.UserID,
			ReceiverRole:        1,
			ReceiverAccountType: "candidate",
			Type:                "offer_created",
			Title:               "Offer 已生成",
			Content:             notifyContent,
			Link:                "/applications",
			BizType:             "offer",
			BizID:               offer.ID,
		}); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		logger.L().Error("create offer failed",
			zap.Int64("application_id", req.ApplicationId),
			zap.Error(err),
		)
		return nil, err
	}
	s.outboxPublisher.Signal()

	logger.L().Info("offer created",
		zap.Int64("offer_id", offer.ID),
		zap.Int64("application_id", req.ApplicationId),
	)

	return &pb.CreateOfferResponse{Code: errs.OK, Msg: "Offer 创建成功", OfferId: offer.ID}, nil
}

func (s *OfferService) UpdateOffer(ctx context.Context, req *pb.UpdateOfferRequest) (*pb.CommonResponse, error) {
	if err := s.serviceAuth.VerifyActorMatch(ctx, req.HrId); err != nil {
		return nil, err
	}

	// Load existing offer
	existing, err := s.offers.GetModelByID(ctx, req.OfferId)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "Offer 记录不存在"}, nil
	}

	// Only draft offers can be updated
	if existing.Status != "draft" {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "仅可编辑草稿状态的 Offer"}, nil
	}

	// Permission + scope check
	if err := s.checkOfferManageScope(ctx, req.HrId, existing.ApplicationID); err != nil {
		return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: err.Error()}, nil
	}

	// Update fields
	if req.Title != "" {
		existing.Title = req.Title
	}
	if req.SalaryRange != "" {
		existing.SalaryRange = req.SalaryRange
	}
	if req.Level != "" {
		existing.Level = req.Level
	}
	if req.WorkLocation != "" {
		existing.WorkLocation = req.WorkLocation
	}
	if req.StartDate != "" {
		existing.StartDate = req.StartDate
	}
	if req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "过期时间格式错误，请使用 RFC 3339 格式"}, nil
		}
		existing.ExpiresAt = &t
	}
	if req.TermsJson != "" {
		existing.TermsJSON = req.TermsJson
	}

	// Transaction: update offer + event
	err = s.offers.Transaction(ctx, func(tx *gorm.DB) error {
		if err := s.offers.UpdateWithTx(ctx, tx, existing); err != nil {
			return err
		}

		event := &model.OfferEvent{
			OfferID:          existing.ID,
			EventType:        "updated",
			ActorUserID:      req.HrId,
			ActorAccountType: "staff",
			Reason:           "",
			CreatedAt:        time.Now(),
		}
		if err := s.offers.CreateEventWithTx(ctx, tx, event); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		logger.L().Error("update offer failed",
			zap.Int64("offer_id", req.OfferId),
			zap.Error(err),
		)
		return nil, err
	}

	logger.L().Info("offer updated",
		zap.Int64("offer_id", existing.ID),
	)

	return &pb.CommonResponse{Code: errs.OK, Msg: "Offer 已更新"}, nil
}

func (s *OfferService) GetOffer(ctx context.Context, req *pb.GetOfferRequest) (*pb.GetOfferResponse, error) {
	if err := s.serviceAuth.VerifyActorMatch(ctx, req.UserId); err != nil {
		return nil, err
	}

	offer, err := s.offers.GetByID(ctx, req.OfferId)
	if err != nil {
		return nil, err
	}
	if offer == nil {
		return &pb.GetOfferResponse{Code: errs.ErrBadRequest, Msg: "Offer 不存在"}, nil
	}

	// Check access: candidate can view own offer; staff need offer.read scope.
	if offer.CandidateUserID != req.UserId {
		// Staff access — verify permission + scope
		if err := s.checkOfferReadScope(ctx, req.UserId, offer.ApplicationID); err != nil {
			return &pb.GetOfferResponse{Code: errs.ErrForbidden, Msg: "无权限查看该 Offer"}, nil
		}
	}

	return &pb.GetOfferResponse{
		Code:  errs.OK,
		Msg:   "success",
		Offer: toPBOffer(offer),
	}, nil
}

func (s *OfferService) ListOffersByApplication(ctx context.Context, req *pb.ListOffersByApplicationRequest) (*pb.ListOffersByApplicationResponse, error) {
	if err := s.serviceAuth.VerifyActorMatch(ctx, req.HrId); err != nil {
		return nil, err
	}

	// Check permission + scope
	if err := s.checkOfferReadScope(ctx, req.HrId, req.ApplicationId); err != nil {
		return &pb.ListOffersByApplicationResponse{Code: errs.ErrForbidden, Msg: err.Error()}, nil
	}

	rows, err := s.offers.ListByApplication(ctx, req.ApplicationId)
	if err != nil {
		return nil, err
	}

	list := make([]*pb.Offer, 0, len(rows))
	for _, row := range rows {
		list = append(list, toPBOffer(&row))
	}

	return &pb.ListOffersByApplicationResponse{Code: errs.OK, Msg: "success", List: list}, nil
}

func (s *OfferService) SendOffer(ctx context.Context, req *pb.SendOfferRequest) (*pb.CommonResponse, error) {
	if err := s.serviceAuth.VerifyActorMatch(ctx, req.HrId); err != nil {
		return nil, err
	}

	// Load offer
	offer, err := s.offers.GetModelByID(ctx, req.OfferId)
	if err != nil {
		return nil, err
	}
	if offer == nil {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "Offer 不存在"}, nil
	}

	// Verify offer.send permission
	if err := s.serviceAuth.AuthorizePermission(ctx, uint64(req.HrId), authz.PermOfferSend); err != nil {
		return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: "无权限发送 Offer"}, nil
	}

	// Only draft offers can be sent
	if offer.Status != "draft" {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "仅可发送草稿状态的 Offer"}, nil
	}

	// Permission + scope check
	if err := s.checkOfferManageScope(ctx, req.HrId, offer.ApplicationID); err != nil {
		return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: err.Error()}, nil
	}

	// Load application detail
	appDetail, err := s.applications.GetDetail(ctx, offer.ApplicationID)
	if err != nil {
		return nil, err
	}
	if appDetail == nil {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "投递记录不存在"}, nil
	}

	// Snapshot the current terms as sent_snapshot_json
	snapshot := map[string]string{
		"title":         offer.Title,
		"salary_range":  offer.SalaryRange,
		"level":         offer.Level,
		"work_location": offer.WorkLocation,
		"start_date":    offer.StartDate,
		"terms_json":    offer.TermsJSON,
	}
	snapshotJSON, _ := json.Marshal(snapshot)
	now := time.Now()

	// Validate transition before attempting update
	if err := ValidateTransition(appDetail.StatusKey, model.StatusKeyOfferSent); err != nil {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: err.Error()}, nil
	}

	// Transaction: update offer + application status + event + notification
	err = s.offers.Transaction(ctx, func(tx *gorm.DB) error {
		// Update offer status to sent with snapshot
		if err := s.offers.UpdateStatusWithTx(ctx, tx, offer.ID, map[string]any{
			"status":              "sent",
			"sent_snapshot_json":  string(snapshotJSON),
			"sent_by":             req.HrId,
		}); err != nil {
			return err
		}

		// Update application status to offer_sent
		rows, err := s.applications.UpdateStatusAnyWithTx(ctx, tx, offer.ApplicationID, appDetail.StatusKey, model.StatusKeyOfferSent, 0)
		if err != nil {
			return err
		}
		if rows == 0 {
			return fmt.Errorf("application status changed concurrently, expected %s", appDetail.StatusKey)
		}

		// Write application status transition audit record
		if err := s.applications.CreateTransition(ctx, tx, &model.ApplicationStatusTransition{
			ApplicationID:    offer.ApplicationID,
			FromStatus:       appDetail.StatusKey,
			ToStatus:         model.StatusKeyOfferSent,
			ActorUserID:      req.HrId,
			ActorAccountType: "staff",
		}); err != nil {
			return err
		}

		// Write offer event
		event := &model.OfferEvent{
			OfferID:          offer.ID,
			EventType:        "sent",
			ActorUserID:      req.HrId,
			ActorAccountType: "staff",
			Reason:           "",
			CreatedAt:        now,
		}
		if err := s.offers.CreateEventWithTx(ctx, tx, event); err != nil {
			return err
		}

		// Notify candidate
		notifyContent := fmt.Sprintf("您投递的「%s」岗位的 Offer 已发送，请及时查看并做出决定。", appDetail.JobTitle)
		if err := s.outboxPublisher.WriteEventTx(tx, "notification.create", "offer", uint64(offer.ID), "notification.create", notificationPayload{
			ReceiverID:          offer.CandidateUserID,
			ReceiverRole:        1,
			ReceiverAccountType: "candidate",
			Type:                "offer_sent",
			Title:               "Offer 已发送",
			Content:             notifyContent,
			Link:                "/applications",
			BizType:             "offer",
			BizID:               offer.ID,
		}); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		logger.L().Error("send offer failed",
			zap.Int64("offer_id", req.OfferId),
			zap.Error(err),
		)
		return nil, err
	}
	s.outboxPublisher.Signal()

	logger.L().Info("offer sent",
		zap.Int64("offer_id", offer.ID),
		zap.Int64("application_id", offer.ApplicationID),
	)

	return &pb.CommonResponse{Code: errs.OK, Msg: "Offer 已发送"}, nil
}

func (s *OfferService) WithdrawOffer(ctx context.Context, req *pb.WithdrawOfferRequest) (*pb.CommonResponse, error) {
	if err := s.serviceAuth.VerifyActorMatch(ctx, req.HrId); err != nil {
		return nil, err
	}

	// Load offer
	offer, err := s.offers.GetModelByID(ctx, req.OfferId)
	if err != nil {
		return nil, err
	}
	if offer == nil {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "Offer 不存在"}, nil
	}

	// Verify offer.manage permission
	if err := s.serviceAuth.AuthorizePermission(ctx, uint64(req.HrId), authz.PermOfferManage); err != nil {
		return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: err.Error()}, nil
	}

	// Only sent/draft offers can be withdrawn
	if offer.Status != "draft" && offer.Status != "sent" {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "该状态下的 Offer 无法撤回"}, nil
	}

	// Permission + scope check
	if err := s.checkOfferManageScope(ctx, req.HrId, offer.ApplicationID); err != nil {
		return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: err.Error()}, nil
	}

	// Load application detail
	appDetail, err := s.applications.GetDetail(ctx, offer.ApplicationID)
	if err != nil {
		return nil, err
	}
	if appDetail == nil {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "投递记录不存在"}, nil
	}

	now := time.Now()

	// If the offer was sent, revert the application status back to offer_pending
	// so HR can re-issue. For draft offers the application status stays as-is.
	needsAppRevert := offer.Status == "sent"
	if needsAppRevert {
		if err := ValidateTransition(appDetail.StatusKey, model.StatusKeyOfferPending); err != nil {
			return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: err.Error()}, nil
		}
	}

	// Transaction: withdraw offer + revert application (if sent) + event + notification
	err = s.offers.Transaction(ctx, func(tx *gorm.DB) error {
		if err := s.offers.UpdateStatusWithTx(ctx, tx, offer.ID, map[string]any{
			"status": "withdrawn",
		}); err != nil {
			return err
		}

		// Revert application status when withdrawing a sent offer
		if needsAppRevert {
			rows, err := s.applications.UpdateStatusAnyWithTx(ctx, tx, offer.ApplicationID, appDetail.StatusKey, model.StatusKeyOfferPending, 0)
			if err != nil {
				return err
			}
			if rows == 0 {
				return fmt.Errorf("application status changed concurrently, expected %s", appDetail.StatusKey)
			}

			// Write application status transition audit record
			if err := s.applications.CreateTransition(ctx, tx, &model.ApplicationStatusTransition{
				ApplicationID:    offer.ApplicationID,
				FromStatus:       appDetail.StatusKey,
				ToStatus:         model.StatusKeyOfferPending,
				ActorUserID:      req.HrId,
				ActorAccountType: "staff",
				Reason:           req.Reason,
			}); err != nil {
				return err
			}
		}

		event := &model.OfferEvent{
			OfferID:          offer.ID,
			EventType:        "withdrawn",
			ActorUserID:      req.HrId,
			ActorAccountType: "staff",
			Reason:           req.Reason,
			CreatedAt:        now,
		}
		if err := s.offers.CreateEventWithTx(ctx, tx, event); err != nil {
			return err
		}

		// Notify candidate
		reasonText := req.Reason
		if reasonText == "" {
			reasonText = "暂无说明"
		}
		notifyContent := fmt.Sprintf("您投递的「%s」岗位的 Offer 已被撤回。原因：%s", appDetail.JobTitle, reasonText)
		if err := s.outboxPublisher.WriteEventTx(tx, "notification.create", "offer", uint64(offer.ID), "notification.create", notificationPayload{
			ReceiverID:          offer.CandidateUserID,
			ReceiverRole:        1,
			ReceiverAccountType: "candidate",
			Type:                "offer_withdrawn",
			Title:               "Offer 已撤回",
			Content:             notifyContent,
			Link:                "/applications",
			BizType:             "offer",
			BizID:               offer.ID,
		}); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		logger.L().Error("withdraw offer failed",
			zap.Int64("offer_id", req.OfferId),
			zap.Error(err),
		)
		return nil, err
	}
	s.outboxPublisher.Signal()

	logger.L().Info("offer withdrawn",
		zap.Int64("offer_id", offer.ID),
		zap.Int64("application_id", offer.ApplicationID),
	)

	return &pb.CommonResponse{Code: errs.OK, Msg: "Offer 已撤回"}, nil
}

func (s *OfferService) AcceptOffer(ctx context.Context, req *pb.AcceptOfferRequest) (*pb.CommonResponse, error) {
	if err := s.serviceAuth.VerifyActorMatch(ctx, req.UserId); err != nil {
		return nil, err
	}

	// Load offer
	offer, err := s.offers.GetModelByID(ctx, req.OfferId)
	if err != nil {
		return nil, err
	}
	if offer == nil {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "Offer 不存在"}, nil
	}

	// Verify the user is the candidate for this offer
	if offer.CandidateUserID != req.UserId {
		return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: "您不是该 Offer 的候选人，无法接受"}, nil
	}

	// Verify offer.decision.manage permission (candidate self-service)
	if err := s.serviceAuth.AuthorizePermission(ctx, uint64(req.UserId), authz.PermOfferDecisionManage); err != nil {
		return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: err.Error()}, nil
	}

	// Only sent offers can be accepted
	if offer.Status != "sent" {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "仅可接受已发送状态的 Offer"}, nil
	}

	// Check expiration
	if offer.ExpiresAt != nil && time.Now().After(*offer.ExpiresAt) {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "Offer 已过期，无法接受"}, nil
	}

	// Load application detail
	appDetail, err := s.applications.GetDetail(ctx, offer.ApplicationID)
	if err != nil {
		return nil, err
	}
	if appDetail == nil {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "投递记录不存在"}, nil
	}

	now := time.Now()

	// Validate transition before attempting update
	if err := ValidateTransition(appDetail.StatusKey, model.StatusKeyOfferAccepted); err != nil {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: err.Error()}, nil
	}

	// Transaction: accept offer + update application + event + notification
	err = s.offers.Transaction(ctx, func(tx *gorm.DB) error {
		if err := s.offers.UpdateStatusWithTx(ctx, tx, offer.ID, map[string]any{
			"status":     "accepted",
			"decided_at": now,
		}); err != nil {
			return err
		}

		// Update application status to offer_accepted
		rows, err := s.applications.UpdateStatusAnyWithTx(ctx, tx, offer.ApplicationID, appDetail.StatusKey, model.StatusKeyOfferAccepted, 0)
		if err != nil {
			return err
		}
		if rows == 0 {
			return fmt.Errorf("application status changed concurrently, expected %s", appDetail.StatusKey)
		}

		// Write application status transition audit record
		if err := s.applications.CreateTransition(ctx, tx, &model.ApplicationStatusTransition{
			ApplicationID:    offer.ApplicationID,
			FromStatus:       appDetail.StatusKey,
			ToStatus:         model.StatusKeyOfferAccepted,
			ActorUserID:      req.UserId,
			ActorAccountType: "candidate",
		}); err != nil {
			return err
		}

		event := &model.OfferEvent{
			OfferID:          offer.ID,
			EventType:        "accepted",
			ActorUserID:      req.UserId,
			ActorAccountType: "candidate",
			Reason:           "",
			CreatedAt:        now,
		}
		if err := s.offers.CreateEventWithTx(ctx, tx, event); err != nil {
			return err
		}

		// Notify HR who sent the offer
		if offer.SentBy != nil {
			notifyContent := fmt.Sprintf("候选人已接受「%s」岗位的 Offer。", appDetail.JobTitle)
			if err := s.outboxPublisher.WriteEventTx(tx, "notification.create", "offer", uint64(offer.ID), "notification.create", notificationPayload{
				ReceiverID:          *offer.SentBy,
				ReceiverRole:        2,
				ReceiverAccountType: "staff",
				Type:                "offer_accepted",
				Title:               "Offer 已被接受",
				Content:             notifyContent,
				Link:                fmt.Sprintf("/hr/jobs/%d/applications", offer.JobID),
				BizType:             "offer",
				BizID:               offer.ID,
			}); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		logger.L().Error("accept offer failed",
			zap.Int64("offer_id", req.OfferId),
			zap.Error(err),
		)
		return nil, err
	}
	s.outboxPublisher.Signal()

	logger.L().Info("offer accepted",
		zap.Int64("offer_id", offer.ID),
		zap.Int64("candidate_user_id", req.UserId),
	)

	return &pb.CommonResponse{Code: errs.OK, Msg: "Offer 已接受"}, nil
}

func (s *OfferService) RejectOffer(ctx context.Context, req *pb.RejectOfferRequest) (*pb.CommonResponse, error) {
	if err := s.serviceAuth.VerifyActorMatch(ctx, req.UserId); err != nil {
		return nil, err
	}

	// Load offer
	offer, err := s.offers.GetModelByID(ctx, req.OfferId)
	if err != nil {
		return nil, err
	}
	if offer == nil {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "Offer 不存在"}, nil
	}

	// Verify the user is the candidate for this offer
	if offer.CandidateUserID != req.UserId {
		return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: "您不是该 Offer 的候选人，无法拒绝"}, nil
	}

	// Verify offer.decision.manage permission (candidate self-service)
	if err := s.serviceAuth.AuthorizePermission(ctx, uint64(req.UserId), authz.PermOfferDecisionManage); err != nil {
		return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: err.Error()}, nil
	}

	// Only sent offers can be rejected
	if offer.Status != "sent" {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "仅可拒绝已发送状态的 Offer"}, nil
	}

	// Load application detail
	appDetail, err := s.applications.GetDetail(ctx, offer.ApplicationID)
	if err != nil {
		return nil, err
	}
	if appDetail == nil {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "投递记录不存在"}, nil
	}

	now := time.Now()

	// Validate transition before attempting update
	if err := ValidateTransition(appDetail.StatusKey, model.StatusKeyOfferRejected); err != nil {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: err.Error()}, nil
	}

	// Transaction: reject offer + update application + event + notification
	err = s.offers.Transaction(ctx, func(tx *gorm.DB) error {
		if err := s.offers.UpdateStatusWithTx(ctx, tx, offer.ID, map[string]any{
			"status":     "rejected",
			"decided_at": now,
		}); err != nil {
			return err
		}

		// Update application status to offer_rejected
		rows, err := s.applications.UpdateStatusAnyWithTx(ctx, tx, offer.ApplicationID, appDetail.StatusKey, model.StatusKeyOfferRejected, 0)
		if err != nil {
			return err
		}
		if rows == 0 {
			return fmt.Errorf("application status changed concurrently, expected %s", appDetail.StatusKey)
		}

		// offer_rejected is terminal: close the current application round
		if model.IsTerminalStatusKey(model.StatusKeyOfferRejected) {
			if err := tx.Model(&model.Application{}).Where("id = ?", offer.ApplicationID).Update("is_current", 0).Error; err != nil {
				return err
			}
		}

		// Write application status transition audit record
		if err := s.applications.CreateTransition(ctx, tx, &model.ApplicationStatusTransition{
			ApplicationID:    offer.ApplicationID,
			FromStatus:       appDetail.StatusKey,
			ToStatus:         model.StatusKeyOfferRejected,
			ActorUserID:      req.UserId,
			ActorAccountType: "candidate",
			Reason:           req.Reason,
		}); err != nil {
			return err
		}

		reasonText := req.Reason
		if reasonText == "" {
			reasonText = "候选人未说明具体原因"
		}

		event := &model.OfferEvent{
			OfferID:          offer.ID,
			EventType:        "rejected",
			ActorUserID:      req.UserId,
			ActorAccountType: "candidate",
			Reason:           req.Reason,
			CreatedAt:        now,
		}
		if err := s.offers.CreateEventWithTx(ctx, tx, event); err != nil {
			return err
		}

		// Notify HR who sent the offer
		if offer.SentBy != nil {
			notifyContent := fmt.Sprintf("候选人已拒绝「%s」岗位的 Offer。原因：%s", appDetail.JobTitle, reasonText)
			if err := s.outboxPublisher.WriteEventTx(tx, "notification.create", "offer", uint64(offer.ID), "notification.create", notificationPayload{
				ReceiverID:          *offer.SentBy,
				ReceiverRole:        2,
				ReceiverAccountType: "staff",
				Type:                "offer_rejected",
				Title:               "Offer 已被拒绝",
				Content:             notifyContent,
				Link:                fmt.Sprintf("/hr/jobs/%d/applications", offer.JobID),
				BizType:             "offer",
				BizID:               offer.ID,
			}); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		logger.L().Error("reject offer failed",
			zap.Int64("offer_id", req.OfferId),
			zap.Error(err),
		)
		return nil, err
	}
	s.outboxPublisher.Signal()

	logger.L().Info("offer rejected",
		zap.Int64("offer_id", offer.ID),
		zap.Int64("candidate_user_id", req.UserId),
	)

	return &pb.CommonResponse{Code: errs.OK, Msg: "Offer 已拒绝"}, nil
}

func (s *OfferService) ListMyOffers(ctx context.Context, req *pb.ListMyOffersRequest) (*pb.ListMyOffersResponse, error) {
	if err := s.serviceAuth.VerifyActorMatch(ctx, req.UserId); err != nil {
		return nil, err
	}

	// Candidate can see own offers — no additional permission check needed for self-service.
	// The user_id in the request is verified against the authenticated actor above.

	ps := req.PageSize
	if ps < 1 || ps > 50 {
		ps = 20
	}

	// Count total offers for this candidate (needed for frontend pagination)
	total, err := s.offers.CountByCandidate(ctx, req.UserId)
	if err != nil {
		logger.L().Error("list my offers: count failed",
			zap.Int64("user_id", req.UserId),
			zap.Error(err),
		)
		return nil, err
	}

	rows, cursor, hasMore, err := s.offers.ListByCandidate(ctx, req.UserId, req.Cursor, ps)
	if err != nil {
		logger.L().Error("list my offers failed",
			zap.Int64("user_id", req.UserId),
			zap.Error(err),
		)
		return nil, err
	}

	list := make([]*pb.Offer, 0, len(rows))
	for _, row := range rows {
		list = append(list, toPBOffer(&row))
	}

	return &pb.ListMyOffersResponse{
		Code:       errs.OK,
		Msg:        "success",
		Total:      total,
		List:       list,
		NextCursor: cursor,
		HasMore:    hasMore,
	}, nil
}

func (s *OfferService) ListOfferEvents(ctx context.Context, req *pb.ListOfferEventsRequest) (*pb.ListOfferEventsResponse, error) {
	if err := s.serviceAuth.VerifyActorMatch(ctx, req.HrId); err != nil {
		return nil, err
	}

	// Load offer to verify scope
	offer, err := s.offers.GetByID(ctx, req.OfferId)
	if err != nil {
		return nil, err
	}
	if offer == nil {
		return &pb.ListOfferEventsResponse{Code: errs.ErrBadRequest, Msg: "Offer 不存在"}, nil
	}

	// Check read permission + scope
	if err := s.checkOfferReadScope(ctx, req.HrId, offer.ApplicationID); err != nil {
		return &pb.ListOfferEventsResponse{Code: errs.ErrForbidden, Msg: "无权限查看该 Offer 事件"}, nil
	}

	events, err := s.offers.ListEventsByOfferID(ctx, req.OfferId)
	if err != nil {
		return nil, err
	}

	list := make([]*pb.OfferEventInfo, 0, len(events))
	for _, e := range events {
		list = append(list, &pb.OfferEventInfo{
			Id:               int64(e.ID),
			OfferId:          e.OfferID,
			EventType:        e.EventType,
			ActorUserId:      e.ActorUserID,
			ActorAccountType: e.ActorAccountType,
			Reason:           e.Reason,
			MetadataJson:     e.MetadataJSON,
			CreatedAt:        formatTime(e.CreatedAt),
		})
	}

	return &pb.ListOfferEventsResponse{Code: errs.OK, Msg: "success", List: list}, nil
}

// ── PB Conversion helpers ──────────────────────────────────────────────

func toPBOffer(row *repository.OfferWithDetailsRow) *pb.Offer {
	var expiresAt, decidedAt string
	if row.ExpiresAt != nil {
		expiresAt = row.ExpiresAt.Format(time.RFC3339)
	}
	if row.DecidedAt != nil {
		decidedAt = row.DecidedAt.Format(time.RFC3339)
	}
	var sentBy int64
	if row.SentBy != nil {
		sentBy = *row.SentBy
	}
	return &pb.Offer{
		Id:                  row.ID,
		ApplicationId:       row.ApplicationID,
		CandidateUserId:     row.CandidateUserID,
		JobId:               row.JobID,
		Status:              row.Status,
		Title:               row.Title,
		SalaryRange:         row.SalaryRange,
		Level:               row.Level,
		WorkLocation:        row.WorkLocation,
		StartDate:           row.StartDate,
		ExpiresAt:           expiresAt,
		TermsJson:           row.TermsJSON,
		SentSnapshotJson:    row.SentSnapshotJSON,
		CreatedBy:           row.CreatedBy,
		SentBy:              sentBy,
		DecidedAt:           decidedAt,
		CreatedAt:           formatTime(row.CreatedAt),
		UpdatedAt:           formatTime(row.UpdatedAt),
		JobTitle:            row.JobTitle,
		CandidateName:       row.CandidateName,
		ApplicationStatusKey: row.ApplicationStatusKey,
	}
}
