package client

import (
	"context"
	"fmt"

	"smart-recruit-interview-service/internal/application/port"
	"smart-recruit-interview-service/internal/domain/model"
	"smart-recruit-platform-go/errs"
	"smart-recruit-proto/recruitment/pb"
)

type ApplicationAdapter struct {
	applications pb.ApplicationOwnerServiceClient
}

func NewApplicationAdapter(applications pb.ApplicationOwnerServiceClient) *ApplicationAdapter {
	return &ApplicationAdapter{applications: applications}
}

func (a *ApplicationAdapter) GetApplicationSnapshot(ctx context.Context, applicationID int64) (*port.ApplicationSnapshot, error) {
	resp, err := a.applications.GetApplicationSnapshot(ctx, &pb.GetApplicationSnapshotRequest{ApplicationId: applicationID})
	if err != nil {
		return nil, err
	}
	if resp.Code != errs.OK {
		return nil, fmt.Errorf("get application snapshot: %s", resp.Msg)
	}
	return &port.ApplicationSnapshot{
		ApplicationID:   resp.ApplicationId,
		CandidateUserID: resp.CandidateUserId,
		JobID:           resp.JobId,
		JobTitle:        resp.JobTitle,
		CandidateName:   resp.CandidateName,
		StatusKey:       model.ApplicationStatus(resp.StatusKey),
		CurrentRoundNo:  resp.RoundNo,
		JobHRID:         resp.JobHrId,
		DepartmentID:    zeroAsNil(resp.DepartmentId),
		LocationID:      zeroAsNil(resp.LocationId),
	}, nil
}

type ApplicationLifecycleAdapter struct {
	applications pb.ApplicationOwnerServiceClient
}

func NewApplicationLifecycleAdapter(applications pb.ApplicationOwnerServiceClient) *ApplicationLifecycleAdapter {
	return &ApplicationLifecycleAdapter{applications: applications}
}

func (a *ApplicationLifecycleAdapter) ApplyTransition(ctx context.Context, command port.LifecycleTransitionCommand) (bool, error) {
	resp, err := a.applications.ApplyApplicationLifecycleTransition(ctx, &pb.ApplyApplicationLifecycleTransitionRequest{
		ActorUserId:       command.ActorUserID,
		ActorAccountType:  command.ActorAccountType,
		ApplicationId:     command.ApplicationID,
		ExpectedStatusKey: string(command.FromStatus),
		TargetStatusKey:   string(command.ToStatus),
		Reason:            command.Reason,
	})
	if err != nil {
		return false, err
	}
	if resp.Code != errs.OK {
		return false, fmt.Errorf("apply application lifecycle transition: %s", resp.Msg)
	}
	return resp.Changed, nil
}

func zeroAsNil(value int64) *int64 {
	if value == 0 {
		return nil
	}
	return &value
}
