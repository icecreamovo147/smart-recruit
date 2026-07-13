package service

import (
	"context"
	"time"

	"go.uber.org/zap"

	"smart-recruit-domain-go/pkg/crypto"
	"smart-recruit-platform-go/logger"
	"smart-recruit-platform-go/serviceconfig"
	"smart-recruit-recruitment-service/internal/legacydomain/model"
	"smart-recruit-recruitment-service/internal/legacydomain/repository"
)

type EmbeddingProviderFactory struct {
	cfg          config.Config
	modelRepo    *repository.EmbeddingModelRepo
	providerRepo *repository.EmbeddingProviderRepo
}

func NewEmbeddingProviderFactory(cfg config.Config, modelRepo *repository.EmbeddingModelRepo, providerRepo *repository.EmbeddingProviderRepo) *EmbeddingProviderFactory {
	return &EmbeddingProviderFactory{
		cfg:          cfg,
		modelRepo:    modelRepo,
		providerRepo: providerRepo,
	}
}

func (f *EmbeddingProviderFactory) Build(ctx context.Context, encKey crypto.EncryptionKey) EmbeddingProvider {
	if f.cfg.Embedding.Enabled != nil && !*f.cfg.Embedding.Enabled {
		logger.L().Warn("[embedding-factory] embedding disabled by config, using unavailable provider")
		return UnavailableEmbeddingProvider{}
	}

	defaultModel, err := f.modelRepo.GetDefaultModel(ctx)
	if err != nil {
		logger.L().Warn("[embedding-factory] no default embedding model found, using unavailable provider", zap.Error(err))
		return UnavailableEmbeddingProvider{}
	}

	if defaultModel.IsEnabled == 0 {
		logger.L().Warn("[embedding-factory] default embedding model is disabled, using unavailable provider",
			zap.Int64("model_id", defaultModel.ID),
			zap.String("model_name", defaultModel.ModelName))
		return UnavailableEmbeddingProvider{}
	}

	providerConfig, err := f.providerRepo.GetByID(ctx, defaultModel.ProviderID)
	if err != nil {
		logger.L().Warn("[embedding-factory] embedding provider not found for default model, using unavailable provider",
			zap.Int64("provider_id", defaultModel.ProviderID),
			zap.Error(err))
		return UnavailableEmbeddingProvider{}
	}

	if providerConfig.IsEnabled == 0 {
		logger.L().Warn("[embedding-factory] embedding provider is disabled, using unavailable provider",
			zap.Int64("provider_id", providerConfig.ID),
			zap.String("provider_name", providerConfig.Name))
		return UnavailableEmbeddingProvider{}
	}

	switch providerConfig.ProviderType {
	case "bailian":
		return f.buildBailian(providerConfig, defaultModel, encKey)
	default:
		logger.L().Warn("[embedding-factory] unsupported provider type, using unavailable provider",
			zap.String("provider_type", providerConfig.ProviderType))
		return UnavailableEmbeddingProvider{}
	}
}

func (f *EmbeddingProviderFactory) buildBailian(providerConfig *model.EmbeddingProviderConfig, modelConfig *model.EmbeddingModelConfig, encKey crypto.EncryptionKey) EmbeddingProvider {
	apiKeyBytes, err := crypto.Decrypt(encKey, providerConfig.APIKeyEncrypted)
	if err != nil {
		logger.L().Warn("[embedding-factory] failed to decrypt provider API key, using unavailable provider",
			zap.Int64("provider_id", providerConfig.ID),
			zap.Error(err))
		return UnavailableEmbeddingProvider{}
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
		providerConfig.Endpoint,
		apiKey,
		WithBailianModel(modelConfig.ModelName),
		WithBailianTimeout(timeout),
		WithBailianMaxRetries(maxRetries),
		WithBailianDimensions(int(modelConfig.EmbeddingDim)),
	)

	logger.L().Info("[embedding-factory] successfully built BailianTextEmbeddingProvider",
		zap.Int64("model_id", modelConfig.ID),
		zap.String("model_name", modelConfig.ModelName),
		zap.Int("embedding_dim", int(modelConfig.EmbeddingDim)),
		zap.Int64("provider_id", providerConfig.ID),
		zap.String("endpoint", providerConfig.Endpoint))

	return providerImpl
}
