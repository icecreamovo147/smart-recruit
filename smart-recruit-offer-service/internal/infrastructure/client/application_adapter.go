package client

import (
	"context"
	"time"

	"gorm.io/gorm"

	"smart-recruit-offer-service/internal/application/port"
	"smart-recruit-offer-service/internal/domain/model"
	"smart-recruit-offer-service/internal/infrastructure/persistence"
	sharedmodel "smart-recruit-offer-service/internal/legacydomain/model"
	sharedrepo "smart-recruit-offer-service/internal/legacydomain/repository"
)

type ApplicationAdapter struct {
	applications *sharedrepo.ApplicationRepo
}

func NewApplicationAdapter(applications *sharedrepo.ApplicationRepo) *ApplicationAdapter {
	return &ApplicationAdapter{applications: applications}
}

func (a *ApplicationAdapter) GetApplicationSnapshot(ctx context.Context, applicationID int64) (*port.ApplicationSnapshot, error) {
	row, err := a.applications.GetDetail(ctx, applicationID)
	if err != nil || row == nil {
		return nil, err
	}
	return &port.ApplicationSnapshot{
		ApplicationID:   row.ApplicationID,
		CandidateUserID: row.UserID,
		JobID:           row.JobID,
		JobTitle:        row.JobTitle,
		StatusKey:       model.ApplicationStatus(row.StatusKey),
	}, nil
}

type ApplicationLifecycleAdapter struct {
	applications *sharedrepo.ApplicationRepo
}

func NewApplicationLifecycleAdapter(applications *sharedrepo.ApplicationRepo) *ApplicationLifecycleAdapter {
	return &ApplicationLifecycleAdapter{applications: applications}
}

func (a *ApplicationLifecycleAdapter) ApplyTransition(ctx context.Context, command port.LifecycleTransitionCommand) (bool, error) {
	if tx, ok := persistence.TxFromContext(ctx); ok {
		return a.applyWithTx(ctx, tx, command)
	}
	var changed bool
	err := a.applications.Transaction(ctx, func(tx *gorm.DB) error {
		var err error
		changed, err = a.applyWithTx(ctx, tx, command)
		return err
	})
	if err != nil {
		return false, err
	}
	return changed, nil
}

func (a *ApplicationLifecycleAdapter) applyWithTx(ctx context.Context, tx *gorm.DB, command port.LifecycleTransitionCommand) (bool, error) {
	rows, err := a.applications.UpdateStatusAnyWithTx(
		ctx,
		tx,
		command.ApplicationID,
		string(command.FromStatus),
		string(command.ToStatus),
		legacyStatus(command.ToStatus),
	)
	if err != nil {
		return false, err
	}
	if rows == 0 {
		return false, nil
	}
	if command.CloseCurrentRound {
		if err := a.applications.CloseCurrentRoundWithTx(ctx, tx, command.ApplicationID); err != nil {
			return false, err
		}
	}
	if err := a.applications.CreateTransition(ctx, tx, transitionRecord(command, time.Now())); err != nil {
		return false, err
	}
	return true, nil
}

func legacyStatus(status model.ApplicationStatus) int32 {
	if value, ok := sharedmodel.StatusKeyToLegacy[string(status)]; ok {
		return value
	}
	return 0
}

func transitionRecord(command port.LifecycleTransitionCommand, now time.Time) *sharedmodel.ApplicationStatusTransition {
	return &sharedmodel.ApplicationStatusTransition{
		ApplicationID:    command.ApplicationID,
		FromStatus:       string(command.FromStatus),
		ToStatus:         string(command.ToStatus),
		ActorUserID:      command.ActorUserID,
		ActorAccountType: command.ActorAccountType,
		Reason:           command.Reason,
		CreatedAt:        now,
	}
}
