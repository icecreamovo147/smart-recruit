package service

import (
	"context"
	"strings"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"logic-grpc-service/model"
	"logic-grpc-service/pkg/crypto"
	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/recruitment/pb"
	"logic-grpc-service/repository"
)

type EmbeddingConfigService struct {
	pb.UnimplementedEmbeddingConfigServiceServer
	providerRepo *repository.EmbeddingProviderRepo
	modelRepo    *repository.EmbeddingModelRepo
	backfillSvc  *EmbeddingBackfillService
	encKey       crypto.EncryptionKey
	embeddingSvc *EmbeddingService
}

func NewEmbeddingConfigService(providerRepo *repository.EmbeddingProviderRepo, modelRepo *repository.EmbeddingModelRepo, encKey crypto.EncryptionKey, backfillSvc *EmbeddingBackfillService, embeddingSvc *EmbeddingService) *EmbeddingConfigService {
	return &EmbeddingConfigService{
		providerRepo: providerRepo,
		modelRepo:    modelRepo,
		backfillSvc:  backfillSvc,
		encKey:       encKey,
		embeddingSvc: embeddingSvc,
	}
}

func (s *EmbeddingConfigService) rebuildProvider(ctx context.Context) {
	if s == nil || s.embeddingSvc == nil {
		return
	}
	s.embeddingSvc.RebuildProvider(ctx)
}

func (s *EmbeddingConfigService) ListEmbeddingProviders(ctx context.Context, req *pb.ListEmbeddingProvidersRequest) (*pb.ListEmbeddingProvidersResponse, error) {
	page := req.GetPage()
	if page <= 0 {
		page = 1
	}
	pageSize := req.GetPageSize()
	if pageSize <= 0 {
		pageSize = 20
	}

	providers, total, err := s.providerRepo.List(ctx, page, pageSize)
	if err != nil {
		logger.L().Error("list embedding providers failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "list embedding providers failed")
	}

	list := make([]*pb.EmbeddingProviderInfo, 0, len(providers))
	for _, p := range providers {
		list = append(list, s.providerToInfo(&p))
	}
	return &pb.ListEmbeddingProvidersResponse{
		Code:  0,
		Msg:   "ok",
		Total: total,
		List:  list,
	}, nil
}

func (s *EmbeddingConfigService) CreateEmbeddingProvider(ctx context.Context, req *pb.CreateEmbeddingProviderRequest) (*pb.EmbeddingProviderResponse, error) {
	if strings.TrimSpace(req.GetName()) == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if strings.TrimSpace(req.GetProviderType()) == "" {
		return nil, status.Error(codes.InvalidArgument, "provider_type is required")
	}
	if strings.TrimSpace(req.GetEndpoint()) == "" {
		return nil, status.Error(codes.InvalidArgument, "endpoint is required")
	}
	if strings.TrimSpace(req.GetApiKey()) == "" {
		return nil, status.Error(codes.InvalidArgument, "api_key is required")
	}

	encrypted, err := crypto.Encrypt(s.encKey, []byte(req.GetApiKey()))
	if err != nil {
		logger.L().Error("encrypt embedding api key failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "encrypt api key failed")
	}

	var extraHeaders *string
	if req.GetExtraHeadersJson() != "" {
		eh := req.GetExtraHeadersJson()
		extraHeaders = &eh
	}

	provider := &model.EmbeddingProviderConfig{
		Name:            strings.TrimSpace(req.GetName()),
		ProviderType:    strings.TrimSpace(req.GetProviderType()),
		Endpoint:        strings.TrimSpace(req.GetEndpoint()),
		APIKeyEncrypted: encrypted,
		ExtraHeaders:    extraHeaders,
		IsEnabled:       1,
	}

	if err := s.providerRepo.Create(ctx, provider); err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") || strings.Contains(err.Error(), "UNIQUE constraint") {
			return nil, status.Error(codes.AlreadyExists, "embedding provider name already exists")
		}
		logger.L().Error("create embedding provider failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "create provider failed")
	}

	s.rebuildProvider(ctx)

	return &pb.EmbeddingProviderResponse{
		Code:     0,
		Msg:      "ok",
		Provider: s.providerToInfo(provider),
	}, nil
}

func (s *EmbeddingConfigService) UpdateEmbeddingProvider(ctx context.Context, req *pb.UpdateEmbeddingProviderRequest) (*pb.EmbeddingProviderResponse, error) {
	provider, err := s.providerRepo.GetByID(ctx, req.GetId())
	if err != nil {
		return nil, status.Error(codes.NotFound, "embedding provider not found")
	}

	if req.GetName() != "" {
		provider.Name = strings.TrimSpace(req.GetName())
	}
	if req.GetProviderType() != "" {
		provider.ProviderType = strings.TrimSpace(req.GetProviderType())
	}
	if req.GetEndpoint() != "" {
		provider.Endpoint = strings.TrimSpace(req.GetEndpoint())
	}
	if req.GetApiKey() != "" {
		encrypted, err := crypto.Encrypt(s.encKey, []byte(req.GetApiKey()))
		if err != nil {
			logger.L().Error("encrypt embedding api key failed", zap.Error(err))
			return nil, status.Error(codes.Internal, "encrypt api key failed")
		}
		provider.APIKeyEncrypted = encrypted
	}
	if req.GetExtraHeadersSet() {
		if req.GetExtraHeadersJson() != "" {
			eh := req.GetExtraHeadersJson()
			provider.ExtraHeaders = &eh
		} else {
			provider.ExtraHeaders = nil
		}
	}
	if req.GetIsEnabledSet() {
		if req.GetIsEnabled() {
			provider.IsEnabled = 1
		} else {
			provider.IsEnabled = 0
		}
	}

	if err := s.providerRepo.Update(ctx, provider); err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") || strings.Contains(err.Error(), "UNIQUE constraint") {
			return nil, status.Error(codes.AlreadyExists, "embedding provider name already exists")
		}
		logger.L().Error("update embedding provider failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "update provider failed")
	}

	s.rebuildProvider(ctx)

	return &pb.EmbeddingProviderResponse{
		Code:     0,
		Msg:      "ok",
		Provider: s.providerToInfo(provider),
	}, nil
}

func (s *EmbeddingConfigService) DeleteEmbeddingProvider(ctx context.Context, req *pb.DeleteEmbeddingProviderRequest) (*pb.CommonResponse, error) {
	if err := s.providerRepo.Delete(ctx, req.GetId()); err != nil {
		logger.L().Error("delete embedding provider failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "delete provider failed")
	}
	s.rebuildProvider(ctx)
	return &pb.CommonResponse{Code: 0, Msg: "ok"}, nil
}

func (s *EmbeddingConfigService) ListEmbeddingModels(ctx context.Context, req *pb.ListEmbeddingModelsRequest) (*pb.ListEmbeddingModelsResponse, error) {
	page := req.GetPage()
	if page <= 0 {
		page = 1
	}
	pageSize := req.GetPageSize()
	if pageSize <= 0 {
		pageSize = 20
	}

	models, total, err := s.modelRepo.List(ctx, page, pageSize, req.GetProviderId())
	if err != nil {
		logger.L().Error("list embedding models failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "list embedding models failed")
	}

	providerIDs := make([]int64, 0, len(models))
	providerMap := make(map[int64]string)
	for _, m := range models {
		providerIDs = append(providerIDs, m.ProviderID)
	}
	if len(providerIDs) > 0 {
		providers, err := s.providerRepo.FindByIDs(ctx, providerIDs)
		if err == nil {
			for _, p := range providers {
				providerMap[p.ID] = p.Name
			}
		}
	}

	list := make([]*pb.EmbeddingModelInfo, 0, len(models))
	for _, m := range models {
		list = append(list, s.modelToInfo(&m, providerMap[m.ProviderID]))
	}
	return &pb.ListEmbeddingModelsResponse{
		Code:  0,
		Msg:   "ok",
		Total: total,
		List:  list,
	}, nil
}

func (s *EmbeddingConfigService) CreateEmbeddingModel(ctx context.Context, req *pb.CreateEmbeddingModelRequest) (*pb.EmbeddingModelResponse, error) {
	if req.GetProviderId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "provider_id is required")
	}
	if strings.TrimSpace(req.GetModelName()) == "" {
		return nil, status.Error(codes.InvalidArgument, "model_name is required")
	}

	provider, err := s.providerRepo.GetByID(ctx, req.GetProviderId())
	if err != nil {
		return nil, status.Error(codes.NotFound, "embedding provider not found")
	}
	if provider.IsEnabled == 0 {
		return nil, status.Error(codes.FailedPrecondition, "embedding provider is disabled")
	}

	m := &model.EmbeddingModelConfig{
		ProviderID:      req.GetProviderId(),
		ModelName:       strings.TrimSpace(req.GetModelName()),
		DisplayName:     strings.TrimSpace(req.GetDisplayName()),
		EmbeddingDim:    req.GetEmbeddingDim(),
		InputTokenLimit: req.GetInputTokenLimit(),
		BatchSize:       req.GetBatchSize(),
		TimeoutSeconds:  req.GetTimeoutSeconds(),
		MaxRetries:      req.GetMaxRetries(),
		IsEnabled:       1,
	}

	if req.GetIsDefault() {
		if err := s.modelRepo.ClearDefault(ctx); err != nil {
			logger.L().Error("clear default embedding model failed", zap.Error(err))
			return nil, status.Error(codes.Internal, "clear default model failed")
		}
		m.IsDefault = 1
	}

	if err := s.modelRepo.Create(ctx, m); err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") || strings.Contains(err.Error(), "UNIQUE constraint") {
			return nil, status.Error(codes.AlreadyExists, "embedding model already exists for this provider")
		}
		logger.L().Error("create embedding model failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "create model failed")
	}

	s.rebuildProvider(ctx)

	return &pb.EmbeddingModelResponse{
		Code:  0,
		Msg:   "ok",
		Model: s.modelToInfo(m, provider.Name),
	}, nil
}

func (s *EmbeddingConfigService) UpdateEmbeddingModel(ctx context.Context, req *pb.UpdateEmbeddingModelRequest) (*pb.EmbeddingModelResponse, error) {
	m, err := s.modelRepo.GetByID(ctx, req.GetId())
	if err != nil {
		return nil, status.Error(codes.NotFound, "embedding model not found")
	}

	if req.GetModelName() != "" {
		m.ModelName = strings.TrimSpace(req.GetModelName())
	}
	if req.GetDisplayName() != "" {
		m.DisplayName = strings.TrimSpace(req.GetDisplayName())
	}
	if req.GetEmbeddingDimSet() {
		m.EmbeddingDim = req.GetEmbeddingDim()
	}
	if req.GetInputTokenLimitSet() {
		m.InputTokenLimit = req.GetInputTokenLimit()
	}
	if req.GetBatchSizeSet() {
		m.BatchSize = req.GetBatchSize()
	}
	if req.GetTimeoutSecondsSet() {
		m.TimeoutSeconds = req.GetTimeoutSeconds()
	}
	if req.GetMaxRetriesSet() {
		m.MaxRetries = req.GetMaxRetries()
	}
	if req.GetIsEnabledSet() {
		if req.GetIsEnabled() {
			m.IsEnabled = 1
		} else {
			m.IsEnabled = 0
		}
	}
	if req.GetIsDefaultSet() && req.GetIsDefault() {
		if err := s.modelRepo.ClearDefault(ctx); err != nil {
			logger.L().Error("clear default embedding model failed", zap.Error(err))
			return nil, status.Error(codes.Internal, "clear default model failed")
		}
		m.IsDefault = 1
	} else if req.GetIsDefaultSet() {
		m.IsDefault = 0
	}

	if err := s.modelRepo.Update(ctx, m); err != nil {
		logger.L().Error("update embedding model failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "update model failed")
	}

	s.rebuildProvider(ctx)

	provider, _ := s.providerRepo.GetByID(ctx, m.ProviderID)
	providerName := ""
	if provider != nil {
		providerName = provider.Name
	}

	return &pb.EmbeddingModelResponse{
		Code:  0,
		Msg:   "ok",
		Model: s.modelToInfo(m, providerName),
	}, nil
}

func (s *EmbeddingConfigService) SetDefaultEmbeddingModel(ctx context.Context, req *pb.SetDefaultEmbeddingModelRequest) (*pb.CommonResponse, error) {
	m, err := s.modelRepo.GetByID(ctx, req.GetId())
	if err != nil {
		return nil, status.Error(codes.NotFound, "embedding model not found")
	}

	if err := s.modelRepo.ClearDefault(ctx); err != nil {
		logger.L().Error("clear default embedding model failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "clear default model failed")
	}

	if err := s.modelRepo.UpdatePartial(ctx, m.ID, map[string]any{"is_default": 1}); err != nil {
		logger.L().Error("set default embedding model failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "set default model failed")
	}

	s.rebuildProvider(ctx)

	return &pb.CommonResponse{Code: 0, Msg: "ok"}, nil
}

func (s *EmbeddingConfigService) TestEmbeddingModel(ctx context.Context, req *pb.TestEmbeddingModelRequest) (*pb.TestEmbeddingModelResponse, error) {
	provider, err := s.providerRepo.GetByID(ctx, req.GetProviderId())
	if err != nil {
		return nil, status.Error(codes.NotFound, "embedding provider not found")
	}

	modelConfig, err := s.modelRepo.GetByID(ctx, req.GetModelId())
	if err != nil {
		return nil, status.Error(codes.NotFound, "embedding model not found")
	}

	apiKeyBytes, err := crypto.Decrypt(s.encKey, provider.APIKeyEncrypted)
	if err != nil {
		logger.L().Error("decrypt embedding api key failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "decrypt api key failed")
	}
	apiKey := string(apiKeyBytes)

	timeout := time.Duration(modelConfig.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = DefaultBailianTimeout
	}
	maxRetries := int(modelConfig.MaxRetries)
	if maxRetries < 0 {
		maxRetries = 0
	}

	providerImpl := NewBailianTextEmbeddingProvider(
		provider.Endpoint,
		apiKey,
		WithBailianModel(modelConfig.ModelName),
		WithBailianTimeout(timeout),
		WithBailianMaxRetries(maxRetries),
	)

	testText := req.GetTestText()
	if testText == "" {
		testText = "test"
	}

	start := time.Now()
	vector, err := providerImpl.EmbedText(ctx, testText)
	latency := time.Since(start).Milliseconds()

	lastTestAt := time.Now()
	lastTestStatus := "failed"
	var lastTestError string

	resp := &pb.TestEmbeddingModelResponse{
		Success:   false,
		Dimension: 0,
		LatencyMs: latency,
		Detail:    "ok",
	}

	if err != nil {
		errMsg := err.Error()
		if len(errMsg) > 200 {
			errMsg = errMsg[:200]
		}
		lastTestError = errMsg
		resp.Msg = "test failed"
		resp.Detail = errMsg
	} else {
		lastTestStatus = "success"
		resp.Success = true
		resp.Dimension = int32(len(vector.Vector))
		resp.RequestId = ""
	}

	updateMap := map[string]any{
		"last_test_status": lastTestStatus,
		"last_test_at":     lastTestAt,
	}
	if lastTestError != "" {
		updateMap["last_test_error"] = lastTestError
	} else {
		updateMap["last_test_error"] = ""
	}
	if updErr := s.modelRepo.UpdatePartial(ctx, modelConfig.ID, updateMap); updErr != nil {
		logger.L().Warn("update embedding model test status failed", zap.Error(updErr))
	}

	return resp, nil
}

func (s *EmbeddingConfigService) BackfillEmbeddings(ctx context.Context, req *pb.BackfillEmbeddingsRequest) (*pb.BackfillEmbeddingsResponse, error) {
	if s.backfillSvc == nil {
		return nil, status.Error(codes.Unavailable, "backfill service not available")
	}

	result, err := s.backfillSvc.Run(ctx, BackfillInput{
		ObjectType: req.GetObjectType(),
		Limit:      int(req.GetLimit()),
		BatchSize:  int(req.GetBatchSize()),
		Force:      req.GetForce(),
		DryRun:     req.GetDryRun(),
		ModelID:    req.GetModelId(),
	})
	if err != nil {
		logger.L().Error("backfill failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "backfill failed")
	}

	return &pb.BackfillEmbeddingsResponse{
		Code:         0,
		Msg:          "ok",
		SuccessCount: int32(result.SuccessCount),
		FailedCount:  int32(result.FailedCount),
		SkippedCount: int32(result.SkippedCount),
	}, nil
}

func (s *EmbeddingConfigService) providerToInfo(p *model.EmbeddingProviderConfig) *pb.EmbeddingProviderInfo {
	apiKeyMasked := ""
	if keyBytes, err := crypto.Decrypt(s.encKey, p.APIKeyEncrypted); err == nil {
		apiKeyMasked = crypto.MaskAPIKey(string(keyBytes))
	}

	return &pb.EmbeddingProviderInfo{
		Id:               p.ID,
		Name:             p.Name,
		ProviderType:     p.ProviderType,
		Endpoint:         p.Endpoint,
		ApiKeyMasked:     apiKeyMasked,
		ExtraHeadersJson: "",
		IsEnabled:        p.IsEnabled == 1,
		CreatedAt:        p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        p.UpdatedAt.Format(time.RFC3339),
	}
}

func (s *EmbeddingConfigService) modelToInfo(m *model.EmbeddingModelConfig, providerName string) *pb.EmbeddingModelInfo {
	var lastTestAt string
	if m.LastTestAt != nil {
		lastTestAt = m.LastTestAt.Format(time.RFC3339)
	}

	return &pb.EmbeddingModelInfo{
		Id:              m.ID,
		ProviderId:      m.ProviderID,
		ModelName:       m.ModelName,
		DisplayName:     m.DisplayName,
		EmbeddingDim:    m.EmbeddingDim,
		InputTokenLimit: m.InputTokenLimit,
		BatchSize:       m.BatchSize,
		TimeoutSeconds:  m.TimeoutSeconds,
		MaxRetries:      m.MaxRetries,
		IsEnabled:       m.IsEnabled == 1,
		IsDefault:       m.IsDefault == 1,
		LastTestStatus:  m.LastTestStatus,
		LastTestError:   m.LastTestError,
		LastTestAt:      lastTestAt,
		CreatedAt:       m.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       m.UpdatedAt.Format(time.RFC3339),
		ProviderName:    providerName,
	}
}
