package persistence

import (
	"context"
	"errors"

	"smart-recruit-platform-go/businessclock"
	"smart-recruit-proto/recruitment/pb"
)

type platformAIControlPlaneServer struct {
	pb.UnimplementedPlatformAIControlPlaneServiceServer
	store *NativeStore
}

func NewPlatformAIControlPlaneServer(store *NativeStore) pb.PlatformAIControlPlaneServiceServer {
	return &platformAIControlPlaneServer{store: store}
}

func (s *platformAIControlPlaneServer) ListPlatformAICapabilities(ctx context.Context, _ *pb.ListPlatformAICapabilitiesRequest) (*pb.ListPlatformAICapabilitiesResponse, error) {
	rows, err := s.store.ListPlatformAICapabilities(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*pb.PlatformAICapabilityInfo, 0, len(rows))
	for _, row := range rows {
		result = append(result, &pb.PlatformAICapabilityInfo{
			Id:                        row.ID,
			CapabilityKey:             row.CapabilityKey,
			Audience:                  row.Audience,
			Name:                      row.Name,
			Description:               row.Description,
			Status:                    row.Status,
			CurrentPublishedVersionId: row.CurrentPublishedVersionID,
		})
	}
	return &pb.ListPlatformAICapabilitiesResponse{Code: 0, Msg: "ok", List: result}, nil
}

func (s *platformAIControlPlaneServer) ListPlatformAICapabilityVersions(ctx context.Context, req *pb.ListPlatformAICapabilityVersionsRequest) (*pb.ListPlatformAICapabilityVersionsResponse, error) {
	rows, err := s.store.ListPlatformAICapabilityVersions(ctx, req.GetCapabilityId())
	if err != nil {
		return nil, err
	}
	result := make([]*pb.PlatformAICapabilityVersionInfo, 0, len(rows))
	for _, row := range rows {
		result = append(result, platformAICapabilityVersionProto(row))
	}
	return &pb.ListPlatformAICapabilityVersionsResponse{Code: 0, Msg: "ok", List: result}, nil
}

func (s *platformAIControlPlaneServer) CreatePlatformAICapabilityDraft(ctx context.Context, req *pb.CreatePlatformAICapabilityDraftRequest) (*pb.PlatformAICapabilityVersionResponse, error) {
	row, err := s.store.CreatePlatformAICapabilityDraft(ctx, req.GetCapabilityId(), req.GetActorUserId(), []byte(req.GetSnapshotJson()), req.GetChangeNote(), req.GetRequestId())
	if err != nil {
		return platformAICapabilityError(err), nil
	}
	return &pb.PlatformAICapabilityVersionResponse{Code: 0, Msg: "ok", Version: platformAICapabilityVersionProto(row)}, nil
}

func (s *platformAIControlPlaneServer) UpdatePlatformAICapabilityDraft(ctx context.Context, req *pb.UpdatePlatformAICapabilityDraftRequest) (*pb.PlatformAICapabilityVersionResponse, error) {
	row, err := s.store.UpdatePlatformAICapabilityDraft(ctx, req.GetVersionId(), req.GetActorUserId(), []byte(req.GetSnapshotJson()), req.GetChangeNote(), req.GetRequestId())
	if err != nil {
		return platformAICapabilityError(err), nil
	}
	return &pb.PlatformAICapabilityVersionResponse{Code: 0, Msg: "ok", Version: platformAICapabilityVersionProto(row)}, nil
}

func (s *platformAIControlPlaneServer) DeletePlatformAICapabilityDraft(ctx context.Context, req *pb.DeletePlatformAICapabilityDraftRequest) (*pb.CommonResponse, error) {
	if err := s.store.DeletePlatformAICapabilityDraft(ctx, req.GetVersionId(), req.GetActorUserId(), req.GetRequestId()); err != nil {
		code, msg := platformAIErrorCode(err)
		return &pb.CommonResponse{Code: code, Msg: msg}, nil
	}
	return &pb.CommonResponse{Code: 0, Msg: "ok"}, nil
}

func (s *platformAIControlPlaneServer) PublishPlatformAICapabilityVersion(ctx context.Context, req *pb.PublishPlatformAICapabilityVersionRequest) (*pb.PlatformAICapabilityVersionResponse, error) {
	row, err := s.store.PublishPlatformAICapabilityVersion(ctx, req.GetVersionId(), req.GetActorUserId(), req.GetRequestId())
	if err != nil {
		return platformAICapabilityError(err), nil
	}
	return &pb.PlatformAICapabilityVersionResponse{Code: 0, Msg: "ok", Version: platformAICapabilityVersionProto(row)}, nil
}

func (s *platformAIControlPlaneServer) ListPlatformAIRuntimeModels(ctx context.Context, req *pb.ListPlatformAIRuntimeModelsRequest) (*pb.ListPlatformAIRuntimeModelsResponse, error) {
	models, version, err := s.store.ListAllowedRuntimeModels(ctx, req.GetCapabilityKey(), req.GetAudience(), req.GetCapabilityVersionId())
	if err != nil {
		code, msg := platformAIErrorCode(err)
		return &pb.ListPlatformAIRuntimeModelsResponse{Code: code, Msg: msg}, nil
	}
	result := make([]*pb.PlatformAIRuntimeModelInfo, 0, len(models))
	for _, model := range models {
		result = append(result, &pb.PlatformAIRuntimeModelInfo{
			Id:                  model.ID,
			ModelName:           model.ModelName,
			DisplayName:         model.DisplayName,
			ProviderId:          model.ProviderID,
			ProviderName:        model.ProviderName,
			IsDefault:           model.IsDefault,
			MaxTokens:           model.MaxTokens,
			ContextWindowTokens: model.ContextWindowTokens,
		})
	}
	return &pb.ListPlatformAIRuntimeModelsResponse{Code: 0, Msg: "ok", CapabilityVersionId: version.ID, SnapshotHash: version.SnapshotHash, List: result}, nil
}

func (s *platformAIControlPlaneServer) ResolvePlatformAIRuntimeModel(ctx context.Context, req *pb.ResolvePlatformAIRuntimeModelRequest) (*pb.ResolvePlatformAIRuntimeModelResponse, error) {
	resolution, err := s.store.ResolveRuntimeModel(ctx, req.GetCapabilityKey(), req.GetAudience(), req.GetCapabilityVersionId(), req.GetRequestedModelId())
	if err != nil {
		code, msg := platformAIErrorCode(err)
		return &pb.ResolvePlatformAIRuntimeModelResponse{Code: code, Msg: msg, RequestedModelId: req.GetRequestedModelId()}, nil
	}
	return &pb.ResolvePlatformAIRuntimeModelResponse{
		Code:                0,
		Msg:                 "ok",
		CapabilityKey:       resolution.CapabilityKey,
		Audience:            resolution.Audience,
		CapabilityVersionId: resolution.CapabilityVersionID,
		SnapshotHash:        resolution.SnapshotHash,
		RequestedModelId:    resolution.RequestedModelID,
		EffectiveModelId:    resolution.EffectiveModelID,
		ModelFallbackReason: resolution.FallbackReason,
		ModelName:           resolution.ModelName,
		DisplayName:         resolution.DisplayName,
		ProviderId:          resolution.ProviderID,
		ProviderName:        resolution.ProviderName,
	}, nil
}

func (s *platformAIControlPlaneServer) QueryPlatformAIConfigAuditLogs(ctx context.Context, req *pb.QueryPlatformAIConfigAuditLogsRequest) (*pb.QueryPlatformAIConfigAuditLogsResponse, error) {
	rows, total, err := s.store.QueryPlatformAIConfigAuditLogs(ctx, int(req.GetPage()), int(req.GetPageSize()), req.GetResourceType(), req.GetCapabilityId())
	if err != nil {
		return nil, err
	}
	result := make([]*pb.PlatformAIConfigAuditLogInfo, 0, len(rows))
	for _, row := range rows {
		result = append(result, &pb.PlatformAIConfigAuditLogInfo{
			Id:                  row.ID,
			ActorUserId:         row.ActorUserID,
			Action:              row.Action,
			ResourceType:        row.ResourceType,
			ResourceId:          row.ResourceID,
			CapabilityId:        row.CapabilityID,
			CapabilityVersionId: row.CapabilityVersionID,
			BeforeSnapshot:      row.BeforeSnapshot,
			AfterSnapshot:       row.AfterSnapshot,
			RequestId:           row.RequestID,
			CreatedAt:           businessclock.FormatRFC3339(row.CreatedAt),
		})
	}
	return &pb.QueryPlatformAIConfigAuditLogsResponse{Code: 0, Msg: "ok", Total: total, List: result}, nil
}

func platformAICapabilityVersionProto(row PlatformAICapabilityVersion) *pb.PlatformAICapabilityVersionInfo {
	publishedAt := ""
	if row.PublishedAt != nil {
		publishedAt = businessclock.FormatRFC3339(*row.PublishedAt)
	}
	return &pb.PlatformAICapabilityVersionInfo{
		Id:           row.ID,
		CapabilityId: row.CapabilityID,
		Version:      int32(row.Version),
		Status:       row.Status,
		SnapshotJson: row.SnapshotJSON,
		SnapshotHash: row.SnapshotHash,
		ChangeNote:   row.ChangeNote,
		PublishedAt:  publishedAt,
	}
}

func platformAICapabilityError(err error) *pb.PlatformAICapabilityVersionResponse {
	code, msg := platformAIErrorCode(err)
	return &pb.PlatformAICapabilityVersionResponse{Code: code, Msg: msg}
}

func platformAIErrorCode(err error) (int32, string) {
	switch {
	case errors.Is(err, ErrCapabilityNotFound):
		return 404, err.Error()
	case errors.Is(err, ErrCapabilityVersionChanged):
		return 409, err.Error()
	case errors.Is(err, ErrCapabilityUnavailable):
		return 503, err.Error()
	case errors.Is(err, ErrModelNotAllowed):
		return 400, err.Error()
	default:
		return 400, err.Error()
	}
}
