package service

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"logic-grpc-service/model"
	"logic-grpc-service/repository"
)

type RecruitmentLifecycleProcessManager struct {
	applications *repository.ApplicationRepo
}

type RecruitmentLifecycleTransition struct {
	ApplicationID     int64
	FromStatus        string
	ToStatus          string
	LegacyStatus      int32
	ActorUserID       int64
	ActorAccountType  string
	Reason            string
	CloseCurrentRound bool
}

func NewRecruitmentLifecycleProcessManager(applications *repository.ApplicationRepo) *RecruitmentLifecycleProcessManager {
	return &RecruitmentLifecycleProcessManager{applications: applications}
}

func (m *RecruitmentLifecycleProcessManager) ApplyTransitionTx(ctx context.Context, tx *gorm.DB, command RecruitmentLifecycleTransition) (bool, error) {
	if m == nil || m.applications == nil {
		return false, fmt.Errorf("recruitment lifecycle process manager is not configured")
	}
	if command.ApplicationID == 0 {
		return false, fmt.Errorf("application id is required")
	}
	if command.ToStatus == "" {
		return false, fmt.Errorf("target status is required")
	}

	legacyStatus := command.LegacyStatus
	if legacyStatus == 0 {
		legacyStatus = model.StatusKeyToLegacy[command.ToStatus]
	}

	rows, err := m.applications.UpdateStatusAnyWithTx(ctx, tx, command.ApplicationID, command.FromStatus, command.ToStatus, legacyStatus)
	if err != nil {
		return false, err
	}
	if rows == 0 {
		return false, nil
	}

	if command.CloseCurrentRound {
		if err := m.applications.CloseCurrentRoundWithTx(ctx, tx, command.ApplicationID); err != nil {
			return false, err
		}
	}

	return true, m.applications.CreateTransition(ctx, tx, &model.ApplicationStatusTransition{
		ApplicationID:    command.ApplicationID,
		FromStatus:       command.FromStatus,
		ToStatus:         command.ToStatus,
		ActorUserID:      command.ActorUserID,
		ActorAccountType: command.ActorAccountType,
		Reason:           command.Reason,
	})
}
