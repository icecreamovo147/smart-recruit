package grpc

import (
	"context"
	"errors"
	"fmt"

	"smart-recruit-platform-go/errs"
	"smart-recruit-proto/recruitment/pb"
	"smart-recruit-recruitment-service/internal/application/command"
	"smart-recruit-recruitment-service/internal/application/dto"
	appservice "smart-recruit-recruitment-service/internal/application/service"
	"smart-recruit-recruitment-service/internal/domain/policy"
)

type ApplicationOwnerContractUsecase interface {
	GetSnapshot(context.Context, int64) (dto.ApplicationSnapshot, error)
	ApplyLifecycleTransition(context.Context, command.ApplyApplicationLifecycleTransition) (dto.ApplicationLifecycleTransitionResult, error)
}

type ApplicationOwnerContractAdapter struct {
	usecase ApplicationOwnerContractUsecase
}

func NewApplicationOwnerContractAdapter(usecase ApplicationOwnerContractUsecase) (*ApplicationOwnerContractAdapter, error) {
	if usecase == nil {
		return nil, fmt.Errorf("application owner contract usecase is required")
	}
	return &ApplicationOwnerContractAdapter{usecase: usecase}, nil
}

func (a *ApplicationOwnerContractAdapter) GetApplicationSnapshot(ctx context.Context, req *pb.GetApplicationSnapshotRequest) (*pb.GetApplicationSnapshotResponse, error) {
	snapshot, err := a.usecase.GetSnapshot(ctx, req.ApplicationId)
	if err != nil {
		return applicationSnapshotError(err), nil
	}
	return &pb.GetApplicationSnapshotResponse{
		Code:            errs.OK,
		Msg:             "success",
		ApplicationId:   snapshot.ApplicationID,
		CandidateUserId: snapshot.CandidateUserID,
		JobId:           snapshot.JobID,
		JobTitle:        snapshot.JobTitle,
		CandidateName:   snapshot.CandidateName,
		ResumeId:        snapshot.ResumeID,
		LegacyStatus:    snapshot.LegacyStatus,
		StatusKey:       snapshot.StatusKey,
		RoundNo:         snapshot.RoundNo,
		IsCurrent:       snapshot.IsCurrent,
		JobHrId:         snapshot.JobHRID,
		DepartmentId:    int64Value(snapshot.DepartmentID),
		LocationId:      int64Value(snapshot.LocationID),
	}, nil
}

func (a *ApplicationOwnerContractAdapter) ApplyApplicationLifecycleTransition(ctx context.Context, req *pb.ApplyApplicationLifecycleTransitionRequest) (*pb.ApplyApplicationLifecycleTransitionResponse, error) {
	result, err := a.usecase.ApplyLifecycleTransition(ctx, command.ApplyApplicationLifecycleTransition{
		ActorUserID:        req.ActorUserId,
		ActorAccountType:   req.ActorAccountType,
		ApplicationID:      req.ApplicationId,
		ExpectedStatusKey:  req.ExpectedStatusKey,
		TargetStatusKey:    req.TargetStatusKey,
		LegacyTargetStatus: req.LegacyTargetStatus,
		Reason:             req.Reason,
		CloseCurrentRound:  req.CloseCurrentRound,
	})
	if err != nil {
		return applicationTransitionError(err), nil
	}
	return &pb.ApplyApplicationLifecycleTransitionResponse{
		Code:             errs.OK,
		Msg:              "success",
		Changed:          result.Changed,
		FromStatusKey:    result.FromStatusKey,
		CurrentStatusKey: result.CurrentStatusKey,
	}, nil
}

func applicationSnapshotError(err error) *pb.GetApplicationSnapshotResponse {
	switch {
	case errors.Is(err, appservice.ErrJobForbidden):
		return &pb.GetApplicationSnapshotResponse{Code: errs.ErrForbidden, Msg: err.Error()}
	default:
		return &pb.GetApplicationSnapshotResponse{Code: errs.ErrInternal, Msg: err.Error()}
	}
}

func applicationTransitionError(err error) *pb.ApplyApplicationLifecycleTransitionResponse {
	switch {
	case errors.Is(err, appservice.ErrApplicationConflict), errors.As(err, new(*policy.TransitionError)):
		return &pb.ApplyApplicationLifecycleTransitionResponse{Code: errs.ErrConflict, Msg: err.Error()}
	case errors.Is(err, appservice.ErrJobForbidden):
		return &pb.ApplyApplicationLifecycleTransitionResponse{Code: errs.ErrForbidden, Msg: err.Error()}
	default:
		return &pb.ApplyApplicationLifecycleTransitionResponse{Code: errs.ErrInternal, Msg: err.Error()}
	}
}

func int64Value(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}
