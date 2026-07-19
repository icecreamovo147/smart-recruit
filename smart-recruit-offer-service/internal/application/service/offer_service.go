package service

import (
	"context"
	"errors"
	"fmt"

	"smart-recruit-offer-service/internal/application/command"
	"smart-recruit-offer-service/internal/application/port"
	"smart-recruit-offer-service/internal/application/query"
	domainevent "smart-recruit-offer-service/internal/domain/event"
	"smart-recruit-offer-service/internal/domain/model"
	"smart-recruit-offer-service/internal/domain/repository"
	domainservice "smart-recruit-offer-service/internal/domain/service"
)

var (
	ErrOfferNotFound       = errors.New("Offer 不存在")
	ErrApplicationNotFound = errors.New("投递记录不存在")
	ErrConcurrentStatus    = errors.New("application status changed concurrently")
)

type OfferService struct {
	offers       repository.OfferRepository
	applications port.ApplicationSnapshotReader
	lifecycle    port.ApplicationLifecycle
	outbox       port.OutboxPublisher
	authorizer   port.Authorizer
	clock        port.Clock
}

type Deps struct {
	Offers       repository.OfferRepository
	Applications port.ApplicationSnapshotReader
	Lifecycle    port.ApplicationLifecycle
	Outbox       port.OutboxPublisher
	Authorizer   port.Authorizer
	Clock        port.Clock
}

func NewOfferService(deps Deps) (*OfferService, error) {
	if deps.Offers == nil {
		return nil, errors.New("offer repository is required")
	}
	if deps.Applications == nil {
		return nil, errors.New("application snapshot reader is required")
	}
	if deps.Lifecycle == nil {
		return nil, errors.New("application lifecycle port is required")
	}
	if deps.Outbox == nil {
		return nil, errors.New("outbox publisher is required")
	}
	if deps.Authorizer == nil {
		return nil, errors.New("authorizer is required")
	}
	clock := deps.Clock
	if clock == nil {
		clock = port.SystemClock{}
	}
	return &OfferService{
		offers:       deps.Offers,
		applications: deps.Applications,
		lifecycle:    deps.Lifecycle,
		outbox:       deps.Outbox,
		authorizer:   deps.Authorizer,
		clock:        clock,
	}, nil
}

func (s *OfferService) CreateOffer(ctx context.Context, cmd command.CreateOffer) (int64, error) {
	if err := s.authorizer.VerifyActor(ctx, cmd.HRID); err != nil {
		return 0, err
	}
	if err := s.authorizer.Authorize(ctx, cmd.HRID, port.PermissionOfferManage); err != nil {
		return 0, err
	}
	if err := s.authorizer.CanManageApplication(ctx, cmd.HRID, cmd.ApplicationID); err != nil {
		return 0, err
	}
	snapshot, err := s.applicationSnapshot(ctx, cmd.ApplicationID)
	if err != nil {
		return 0, err
	}
	offer, err := model.NewDraft(model.DraftDetails{
		ApplicationID:   cmd.ApplicationID,
		CandidateUserID: snapshot.CandidateUserID,
		JobID:           snapshot.JobID,
		Title:           cmd.Title,
		SalaryRange:     cmd.SalaryRange,
		Level:           cmd.Level,
		WorkLocation:    cmd.WorkLocation,
		StartDate:       cmd.StartDate,
		ExpiresAt:       cmd.ExpiresAt,
		TermsJSON:       cmd.TermsJSON,
		CreatedBy:       cmd.HRID,
	})
	if err != nil {
		return 0, err
	}
	transition, needsTransition, err := domainservice.CreationTransition(snapshot.StatusKey)
	if err != nil {
		return 0, err
	}

	now := s.clock.Now()
	err = s.offers.Transaction(ctx, func(txCtx context.Context, writer repository.OfferWriter) error {
		if err := writer.Create(txCtx, offer); err != nil {
			return err
		}
		if needsTransition {
			if err := s.applyLifecycle(txCtx, port.LifecycleTransitionCommand{
				ApplicationID:    cmd.ApplicationID,
				FromStatus:       transition.From,
				ToStatus:         transition.To,
				ActorUserID:      cmd.HRID,
				ActorAccountType: "staff",
			}); err != nil {
				return err
			}
		}
		if err := writer.AddEvent(txCtx, domainevent.NewOfferEvent(offer.TenantID, offer.ID, domainevent.TypeCreated, cmd.HRID, "staff", "", now)); err != nil {
			return err
		}
		return s.outbox.Publish(txCtx, candidateNotification(offer, snapshot.JobTitle, "offer_created", "Offer 已生成",
			fmt.Sprintf("您投递的「%s」岗位已生成 Offer，请留意查看。", snapshot.JobTitle)))
	})
	if err != nil {
		return 0, err
	}
	s.outbox.Signal()
	return offer.ID, nil
}

func (s *OfferService) UpdateOffer(ctx context.Context, cmd command.UpdateOffer) error {
	if err := s.authorizer.VerifyActor(ctx, cmd.HRID); err != nil {
		return err
	}
	offer, err := s.offerModel(ctx, cmd.OfferID)
	if err != nil {
		return err
	}
	if err := s.authorizer.Authorize(ctx, cmd.HRID, port.PermissionOfferManage); err != nil {
		return err
	}
	if err := s.authorizer.CanManageApplication(ctx, cmd.HRID, offer.ApplicationID); err != nil {
		return err
	}
	if err := offer.ApplyDraftPatch(model.DraftPatch{
		Title:        cmd.Title,
		SalaryRange:  cmd.SalaryRange,
		Level:        cmd.Level,
		WorkLocation: cmd.WorkLocation,
		StartDate:    cmd.StartDate,
		ExpiresAt:    cmd.ExpiresAt,
		TermsJSON:    cmd.TermsJSON,
	}); err != nil {
		return err
	}
	now := s.clock.Now()
	return s.offers.Transaction(ctx, func(txCtx context.Context, writer repository.OfferWriter) error {
		if err := writer.Save(txCtx, offer); err != nil {
			return err
		}
		return writer.AddEvent(txCtx, domainevent.NewOfferEvent(offer.TenantID, offer.ID, domainevent.TypeUpdated, cmd.HRID, "staff", "", now))
	})
}

func (s *OfferService) SendOffer(ctx context.Context, cmd command.SendOffer) error {
	if err := s.authorizer.VerifyActor(ctx, cmd.HRID); err != nil {
		return err
	}
	offer, err := s.offerModel(ctx, cmd.OfferID)
	if err != nil {
		return err
	}
	if err := s.authorizer.Authorize(ctx, cmd.HRID, port.PermissionOfferSend); err != nil {
		return err
	}
	if err := s.authorizer.CanManageApplication(ctx, cmd.HRID, offer.ApplicationID); err != nil {
		return err
	}
	snapshot, err := s.applicationSnapshot(ctx, offer.ApplicationID)
	if err != nil {
		return err
	}
	transition, err := domainservice.SendTransition(snapshot.StatusKey)
	if err != nil {
		return err
	}
	if err := offer.MarkSent(cmd.HRID); err != nil {
		return err
	}
	now := s.clock.Now()
	err = s.offers.Transaction(ctx, func(txCtx context.Context, writer repository.OfferWriter) error {
		if err := writer.Save(txCtx, offer); err != nil {
			return err
		}
		if err := s.applyLifecycle(txCtx, port.LifecycleTransitionCommand{
			ApplicationID:    offer.ApplicationID,
			FromStatus:       transition.From,
			ToStatus:         transition.To,
			ActorUserID:      cmd.HRID,
			ActorAccountType: "staff",
		}); err != nil {
			return err
		}
		if err := writer.AddEvent(txCtx, domainevent.NewOfferEvent(offer.TenantID, offer.ID, domainevent.TypeSent, cmd.HRID, "staff", "", now)); err != nil {
			return err
		}
		content := fmt.Sprintf("您投递的「%s」岗位的 Offer 已发送，请及时查看并做出决定。", snapshot.JobTitle)
		if err := s.outbox.Publish(txCtx, candidateNotification(offer, snapshot.JobTitle, "offer_sent", "Offer 已发送", content)); err != nil {
			return err
		}
		return s.outbox.Publish(txCtx, candidateEmail(offer, snapshot.JobTitle, "offer_sent", "Offer 已发送", content))
	})
	if err != nil {
		return err
	}
	s.outbox.Signal()
	return nil
}

func (s *OfferService) WithdrawOffer(ctx context.Context, cmd command.WithdrawOffer) error {
	if err := s.authorizer.VerifyActor(ctx, cmd.HRID); err != nil {
		return err
	}
	offer, err := s.offerModel(ctx, cmd.OfferID)
	if err != nil {
		return err
	}
	if err := s.authorizer.Authorize(ctx, cmd.HRID, port.PermissionOfferManage); err != nil {
		return err
	}
	if err := s.authorizer.CanManageApplication(ctx, cmd.HRID, offer.ApplicationID); err != nil {
		return err
	}
	snapshot, err := s.applicationSnapshot(ctx, offer.ApplicationID)
	if err != nil {
		return err
	}
	needsAppRevert, err := offer.MarkWithdrawn()
	if err != nil {
		return err
	}
	var transition domainservice.ApplicationTransition
	if needsAppRevert {
		transition, err = domainservice.WithdrawTransition(snapshot.StatusKey)
		if err != nil {
			return err
		}
	}
	now := s.clock.Now()
	err = s.offers.Transaction(ctx, func(txCtx context.Context, writer repository.OfferWriter) error {
		if err := writer.Save(txCtx, offer); err != nil {
			return err
		}
		if needsAppRevert {
			if err := s.applyLifecycle(txCtx, port.LifecycleTransitionCommand{
				ApplicationID:    offer.ApplicationID,
				FromStatus:       transition.From,
				ToStatus:         transition.To,
				ActorUserID:      cmd.HRID,
				ActorAccountType: "staff",
				Reason:           cmd.Reason,
			}); err != nil {
				return err
			}
		}
		if err := writer.AddEvent(txCtx, domainevent.NewOfferEvent(offer.TenantID, offer.ID, domainevent.TypeWithdrawn, cmd.HRID, "staff", cmd.Reason, now)); err != nil {
			return err
		}
		reason := cmd.Reason
		if reason == "" {
			reason = "暂无说明"
		}
		content := fmt.Sprintf("您投递的「%s」岗位的 Offer 已被撤回。原因：%s", snapshot.JobTitle, reason)
		if err := s.outbox.Publish(txCtx, candidateNotification(offer, snapshot.JobTitle, "offer_withdrawn", "Offer 已撤回", content)); err != nil {
			return err
		}
		return s.outbox.Publish(txCtx, candidateEmail(offer, snapshot.JobTitle, "offer_withdrawn", "Offer 已撤回", content))
	})
	if err != nil {
		return err
	}
	s.outbox.Signal()
	return nil
}

func (s *OfferService) AcceptOffer(ctx context.Context, cmd command.AcceptOffer) error {
	if err := s.authorizer.VerifyActor(ctx, cmd.UserID); err != nil {
		return err
	}
	if err := s.authorizer.Authorize(ctx, cmd.UserID, port.PermissionOfferDecisionManage); err != nil {
		return err
	}
	offer, err := s.offerModel(ctx, cmd.OfferID)
	if err != nil {
		return err
	}
	if err := offer.MarkAccepted(cmd.UserID, s.clock.Now()); err != nil {
		return err
	}
	snapshot, err := s.applicationSnapshot(ctx, offer.ApplicationID)
	if err != nil {
		return err
	}
	transition, err := domainservice.AcceptTransition(snapshot.StatusKey)
	if err != nil {
		return err
	}
	now := s.clock.Now()
	err = s.offers.Transaction(ctx, func(txCtx context.Context, writer repository.OfferWriter) error {
		if err := writer.Save(txCtx, offer); err != nil {
			return err
		}
		if err := s.applyLifecycle(txCtx, port.LifecycleTransitionCommand{
			ApplicationID:    offer.ApplicationID,
			FromStatus:       transition.From,
			ToStatus:         transition.To,
			ActorUserID:      cmd.UserID,
			ActorAccountType: "candidate",
		}); err != nil {
			return err
		}
		if err := writer.AddEvent(txCtx, domainevent.NewOfferEvent(offer.TenantID, offer.ID, domainevent.TypeAccepted, cmd.UserID, "candidate", "", now)); err != nil {
			return err
		}
		if offer.SentBy == nil {
			return nil
		}
		return s.outbox.Publish(txCtx, staffNotification(offer, *offer.SentBy, snapshot.JobTitle, "offer_accepted", "Offer 已被接受",
			fmt.Sprintf("候选人已接受「%s」岗位的 Offer。", snapshot.JobTitle)))
	})
	if err != nil {
		return err
	}
	s.outbox.Signal()
	return nil
}

func (s *OfferService) RejectOffer(ctx context.Context, cmd command.RejectOffer) error {
	if err := s.authorizer.VerifyActor(ctx, cmd.UserID); err != nil {
		return err
	}
	if err := s.authorizer.Authorize(ctx, cmd.UserID, port.PermissionOfferDecisionManage); err != nil {
		return err
	}
	offer, err := s.offerModel(ctx, cmd.OfferID)
	if err != nil {
		return err
	}
	if err := offer.MarkRejected(cmd.UserID, s.clock.Now()); err != nil {
		return err
	}
	snapshot, err := s.applicationSnapshot(ctx, offer.ApplicationID)
	if err != nil {
		return err
	}
	transition, err := domainservice.RejectTransition(snapshot.StatusKey)
	if err != nil {
		return err
	}
	now := s.clock.Now()
	err = s.offers.Transaction(ctx, func(txCtx context.Context, writer repository.OfferWriter) error {
		if err := writer.Save(txCtx, offer); err != nil {
			return err
		}
		if err := s.applyLifecycle(txCtx, port.LifecycleTransitionCommand{
			ApplicationID:     offer.ApplicationID,
			FromStatus:        transition.From,
			ToStatus:          transition.To,
			ActorUserID:       cmd.UserID,
			ActorAccountType:  "candidate",
			Reason:            cmd.Reason,
			CloseCurrentRound: true,
		}); err != nil {
			return err
		}
		if err := writer.AddEvent(txCtx, domainevent.NewOfferEvent(offer.TenantID, offer.ID, domainevent.TypeRejected, cmd.UserID, "candidate", cmd.Reason, now)); err != nil {
			return err
		}
		if offer.SentBy == nil {
			return nil
		}
		reason := cmd.Reason
		if reason == "" {
			reason = "候选人未说明具体原因"
		}
		return s.outbox.Publish(txCtx, staffNotification(offer, *offer.SentBy, snapshot.JobTitle, "offer_rejected", "Offer 已被拒绝",
			fmt.Sprintf("候选人已拒绝「%s」岗位的 Offer。原因：%s", snapshot.JobTitle, reason)))
	})
	if err != nil {
		return err
	}
	s.outbox.Signal()
	return nil
}

func (s *OfferService) GetOffer(ctx context.Context, q query.GetOffer) (*repository.OfferDetails, error) {
	if err := s.authorizer.VerifyActor(ctx, q.UserID); err != nil {
		return nil, err
	}
	offer, err := s.offers.FindDetailsByID(ctx, q.OfferID)
	if err != nil {
		return nil, err
	}
	if offer == nil {
		return nil, ErrOfferNotFound
	}
	if offer.CandidateUserID != q.UserID {
		if err := s.authorizer.Authorize(ctx, q.UserID, port.PermissionOfferRead); err != nil {
			return nil, err
		}
		if err := s.authorizer.CanReadApplication(ctx, q.UserID, offer.ApplicationID); err != nil {
			return nil, err
		}
	}
	return offer, nil
}

func (s *OfferService) ListOffersByApplication(ctx context.Context, q query.ListOffersByApplication) ([]repository.OfferDetails, error) {
	if err := s.authorizer.VerifyActor(ctx, q.HRID); err != nil {
		return nil, err
	}
	if err := s.authorizer.Authorize(ctx, q.HRID, port.PermissionOfferRead); err != nil {
		return nil, err
	}
	if err := s.authorizer.CanReadApplication(ctx, q.HRID, q.ApplicationID); err != nil {
		return nil, err
	}
	return s.offers.ListByApplication(ctx, q.ApplicationID)
}

func (s *OfferService) ListMyOffers(ctx context.Context, q query.ListMyOffers) (repository.CandidateOfferPage, error) {
	if err := s.authorizer.VerifyActor(ctx, q.UserID); err != nil {
		return repository.CandidateOfferPage{}, err
	}
	pageSize := q.PageSize
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}
	return s.offers.ListByCandidate(ctx, q.UserID, q.Cursor, pageSize)
}

func (s *OfferService) ListOfferEvents(ctx context.Context, q query.ListOfferEvents) ([]domainevent.OfferEvent, error) {
	if err := s.authorizer.VerifyActor(ctx, q.HRID); err != nil {
		return nil, err
	}
	offer, err := s.offers.FindDetailsByID(ctx, q.OfferID)
	if err != nil {
		return nil, err
	}
	if offer == nil {
		return nil, ErrOfferNotFound
	}
	if err := s.authorizer.Authorize(ctx, q.HRID, port.PermissionOfferRead); err != nil {
		return nil, err
	}
	if err := s.authorizer.CanReadApplication(ctx, q.HRID, offer.ApplicationID); err != nil {
		return nil, err
	}
	return s.offers.ListEvents(ctx, q.OfferID)
}

func (s *OfferService) offerModel(ctx context.Context, offerID int64) (*model.Offer, error) {
	offer, err := s.offers.FindByID(ctx, offerID)
	if err != nil {
		return nil, err
	}
	if offer == nil {
		return nil, ErrOfferNotFound
	}
	return offer, nil
}

func (s *OfferService) applicationSnapshot(ctx context.Context, applicationID int64) (*port.ApplicationSnapshot, error) {
	snapshot, err := s.applications.GetApplicationSnapshot(ctx, applicationID)
	if err != nil {
		return nil, err
	}
	if snapshot == nil {
		return nil, ErrApplicationNotFound
	}
	return snapshot, nil
}

func (s *OfferService) applyLifecycle(ctx context.Context, command port.LifecycleTransitionCommand) error {
	changed, err := s.lifecycle.ApplyTransition(ctx, command)
	if err != nil {
		return err
	}
	if !changed {
		return fmt.Errorf("%w, expected %s", ErrConcurrentStatus, command.FromStatus)
	}
	return nil
}

func candidateNotification(offer *model.Offer, jobTitle string, typ string, title string, content string) port.OutboxMessage {
	return port.OutboxMessage{
		TenantID:            offer.TenantID,
		EventType:           "offer.notification_requested",
		AggregateType:       "offer",
		AggregateID:         offer.ID,
		RoutingKey:          "notification.create",
		ReceiverID:          offer.CandidateUserID,
		ReceiverRole:        1,
		ReceiverAccountType: "candidate",
		Type:                typ,
		Title:               title,
		Content:             content,
		Link:                "/applications",
		BizType:             "offer",
		BizID:               offer.ID,
		JobTitle:            jobTitle,
	}
}

func candidateEmail(offer *model.Offer, jobTitle string, typ string, title string, content string) port.OutboxMessage {
	message := candidateNotification(offer, jobTitle, typ, title, content)
	message.EventType = "offer.email_requested"
	message.RoutingKey = "email.send"
	return message
}

func staffNotification(offer *model.Offer, receiverID int64, jobTitle string, typ string, title string, content string) port.OutboxMessage {
	return port.OutboxMessage{
		TenantID:            offer.TenantID,
		EventType:           "offer.notification_requested",
		AggregateType:       "offer",
		AggregateID:         offer.ID,
		RoutingKey:          "notification.create",
		ReceiverID:          receiverID,
		ReceiverRole:        2,
		ReceiverAccountType: "staff",
		Type:                typ,
		Title:               title,
		Content:             content,
		Link:                fmt.Sprintf("/hr/jobs/%d/applications", offer.JobID),
		BizType:             "offer",
		BizID:               offer.ID,
		JobTitle:            jobTitle,
	}
}
