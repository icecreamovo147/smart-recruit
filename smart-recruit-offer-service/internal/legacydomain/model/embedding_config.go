package model

import "time"

type EmbeddingProviderConfig struct {
	ID              int64     `gorm:"primaryKey"`
	Name            string    `gorm:"column:name;size:128;not null;uniqueIndex:uk_embedding_providers_name"`
	ProviderType    string    `gorm:"column:provider_type;size:64;not null"`
	Endpoint        string    `gorm:"column:endpoint;size:512;not null"`
	APIKeyEncrypted string    `gorm:"column:api_key_encrypted;type:text;not null"`
	ExtraHeaders    *string   `gorm:"column:extra_headers;type:json"`
	IsEnabled       int32     `gorm:"column:is_enabled;default:1"`
	CreatedAt       time.Time `gorm:"column:created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at"`
}

func (EmbeddingProviderConfig) TableName() string { return "embedding_providers" }

type EmbeddingModelConfig struct {
	ID              int64      `gorm:"primaryKey"`
	ProviderID      int64      `gorm:"column:provider_id;not null;uniqueIndex:uk_embedding_models_provider_model,priority:1"`
	ModelName       string     `gorm:"column:model_name;size:128;not null;uniqueIndex:uk_embedding_models_provider_model,priority:2"`
	DisplayName     string     `gorm:"column:display_name;size:256;not null;default:''"`
	EmbeddingDim    int32      `gorm:"column:embedding_dim;default:0"`
	InputTokenLimit int32      `gorm:"column:input_token_limit;default:0"`
	BatchSize       int32      `gorm:"column:batch_size;default:1"`
	TimeoutSeconds  int32      `gorm:"column:timeout_seconds;default:30"`
	MaxRetries      int32      `gorm:"column:max_retries;default:2"`
	IsEnabled       int32      `gorm:"column:is_enabled;default:1"`
	IsDefault       int32      `gorm:"column:is_default;default:0"`
	LastTestStatus  string     `gorm:"column:last_test_status;size:32;not null;default:'untested'"`
	LastTestError   string     `gorm:"column:last_test_error;type:text"`
	LastTestAt      *time.Time `gorm:"column:last_test_at"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at"`
}

func (EmbeddingModelConfig) TableName() string { return "embedding_models" }
