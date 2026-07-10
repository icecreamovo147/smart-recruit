package service

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"

	"logic-grpc-service/pkg/logger"
)

type FallbackResumeProfileExtractor struct {
	primary  ResumeProfileExtractor
	fallback ResumeProfileExtractor
	enabled  bool
}

type FallbackExtractorOption func(*FallbackResumeProfileExtractor)

func WithFallbackEnabled(enabled bool) FallbackExtractorOption {
	return func(f *FallbackResumeProfileExtractor) {
		f.enabled = enabled
	}
}

func NewFallbackResumeProfileExtractor(primary, fallback ResumeProfileExtractor, opts ...FallbackExtractorOption) *FallbackResumeProfileExtractor {
	f := &FallbackResumeProfileExtractor{
		primary:  primary,
		fallback: fallback,
		enabled:  true,
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

func (f *FallbackResumeProfileExtractor) Extract(ctx context.Context, text string) (string, error) {
	result, err := f.ExtractWithMetadata(ctx, text)
	if err != nil {
		return "", err
	}
	return result.RawJSON, nil
}

func (f *FallbackResumeProfileExtractor) ExtractWithMetadata(ctx context.Context, text string) (ResumeProfileExtractResult, error) {
	start := time.Now()

	var primaryResult ResumeProfileExtractResult
	var primaryErr error

	if v2, ok := f.primary.(ResumeProfileExtractorV2); ok {
		primaryResult, primaryErr = v2.ExtractWithMetadata(ctx, text)
	} else {
		var raw string
		raw, primaryErr = f.primary.Extract(ctx, text)
		if primaryErr == nil {
			primaryResult = ResumeProfileExtractResult{RawJSON: raw}
		}
	}

	if primaryErr == nil {
		primaryResult.Metadata.Duration = time.Since(start)
		logger.L().Info("fallback extractor: primary succeeded",
			zap.String("extractor_type", "llm"),
			zap.String("parser_version", primaryResult.Metadata.ParserVersion),
			zap.Duration("duration", time.Since(start)),
		)
		return primaryResult, nil
	}

	if !f.enabled || f.fallback == nil {
		logger.L().Warn("fallback extractor: primary failed and fallback disabled",
			zap.String("parser_version", primaryResult.Metadata.ParserVersion),
			zap.Error(primaryErr),
		)
		return ResumeProfileExtractResult{}, primaryErr
	}

	logger.L().Warn("fallback extractor: primary failed, falling back to heuristic",
		zap.Error(primaryErr),
		zap.Duration("primary_duration", time.Since(start)),
	)

	var fallbackResult ResumeProfileExtractResult
	var fallbackErr error

	if v2, ok := f.fallback.(ResumeProfileExtractorV2); ok {
		fallbackResult, fallbackErr = v2.ExtractWithMetadata(ctx, text)
	} else {
		var raw string
		raw, fallbackErr = f.fallback.Extract(ctx, text)
		if fallbackErr == nil {
			fallbackResult = ResumeProfileExtractResult{RawJSON: raw}
		}
	}

	if fallbackErr != nil {
		logger.L().Error("fallback extractor: both primary and fallback failed",
			zap.String("parser_version", fallbackResult.Metadata.ParserVersion),
			zap.Error(fallbackErr),
		)
		return ResumeProfileExtractResult{}, errors.Join(errors.New("primary failed"), fallbackErr)
	}

	fallbackResult.Metadata.FallbackUsed = true
	fallbackResult.Metadata.ExtractorType = "heuristic_fallback"
	fallbackResult.Metadata.Duration = time.Since(start)
	if fallbackResult.Metadata.ParserVersion == "" {
		fallbackResult.Metadata.ParserVersion = heuristicParserVersion
	}

	logger.L().Warn("fallback extractor: used heuristic fallback",
		zap.Duration("total_duration", time.Since(start)),
		zap.String("extractor_type", "heuristic_fallback"),
		zap.String("parser_version", fallbackResult.Metadata.ParserVersion),
	)
	return fallbackResult, nil
}
