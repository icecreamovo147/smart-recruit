package service

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"logic-grpc-service/pkg/logger"
)

type fallbackRequirementExtractor struct {
	primary  RequirementExtractorV2
	fallback RequirementExtractorV2
	enabled  bool
}

func NewFallbackRequirementExtractor(primary, fallback RequirementExtractorV2, opts ...ReqFallbackOption) RequirementExtractorV2 {
	e := &fallbackRequirementExtractor{
		primary:  primary,
		fallback: fallback,
		enabled:  true,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

type ReqFallbackOption func(*fallbackRequirementExtractor)

func WithReqFallbackEnabled(enabled bool) ReqFallbackOption {
	return func(e *fallbackRequirementExtractor) {
		e.enabled = enabled
	}
}

func (e *fallbackRequirementExtractor) Extract(ctx context.Context, jobTitle, department, description, requirements string) (*JobRequirementProfile, string, error) {
	result, err := e.ExtractWithMetadata(ctx, jobTitle, department, description, requirements)
	if err != nil {
		return nil, "", err
	}
	return result.Profile, result.InputHash, nil
}

func (e *fallbackRequirementExtractor) ExtractWithMetadata(ctx context.Context, jobTitle, department, description, requirements string) (*JobRequirementExtractResult, error) {
	if e.primary == nil {
		return e.fallback.ExtractWithMetadata(ctx, jobTitle, department, description, requirements)
	}

	start := time.Now()
	result, err := e.primary.ExtractWithMetadata(ctx, jobTitle, department, description, requirements)

	if err == nil && result != nil && result.Profile != nil {
		if err := result.Profile.Validate(); err == nil {
			result.Metadata.FallbackUsed = false
			result.Metadata.Duration = time.Since(start)
			logger.L().Info("requirement extractor: primary succeeded",
				zap.String("extractor_type", result.Metadata.ExtractorType),
				zap.Int("requirements", len(result.Profile.Requirements)),
				zap.Duration("duration", result.Metadata.Duration),
			)
			return result, nil
		}
		logger.L().Warn("requirement extractor: primary output failed validation",
			zap.Error(err),
		)
	} else if err != nil {
		logger.L().Warn("requirement extractor: primary failed",
			zap.Error(err),
		)
	}

	if !e.enabled {
		logger.L().Info("requirement extractor: fallback disabled, propagating primary error")
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("requirement extractor: primary failed and fallback disabled")
	}

	fallbackResult, fallbackErr := e.fallback.ExtractWithMetadata(ctx, jobTitle, department, description, requirements)
	if fallbackErr != nil {
		return nil, fallbackErr
	}

	fallbackResult.Metadata.FallbackUsed = true
	fallbackResult.Metadata.Duration = time.Since(start)
	logger.L().Info("requirement extractor: fallback used",
		zap.String("extractor_type", fallbackResult.Metadata.ExtractorType),
		zap.Int("requirements", len(fallbackResult.Profile.Requirements)),
		zap.Duration("duration", fallbackResult.Metadata.Duration),
	)
	return fallbackResult, nil
}
