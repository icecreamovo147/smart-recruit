CREATE TABLE IF NOT EXISTS embedding_providers (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name VARCHAR(128) NOT NULL,
  provider_type VARCHAR(64) NOT NULL,
  endpoint VARCHAR(512) NOT NULL,
  api_key_encrypted TEXT NOT NULL,
  extra_headers JSON NULL,
  is_enabled TINYINT(1) NOT NULL DEFAULT 1,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_embedding_providers_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Embedding provider configuration';

CREATE TABLE IF NOT EXISTS embedding_models (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  provider_id BIGINT UNSIGNED NOT NULL,
  model_name VARCHAR(128) NOT NULL,
  display_name VARCHAR(256) NOT NULL DEFAULT '',
  embedding_dim INT NOT NULL DEFAULT 0,
  input_token_limit INT NOT NULL DEFAULT 0,
  batch_size INT NOT NULL DEFAULT 1,
  timeout_seconds INT NOT NULL DEFAULT 30,
  max_retries INT NOT NULL DEFAULT 2,
  is_enabled TINYINT(1) NOT NULL DEFAULT 1,
  is_default TINYINT(1) NOT NULL DEFAULT 0,
  last_test_status VARCHAR(32) NOT NULL DEFAULT 'untested',
  last_test_error TEXT NULL,
  last_test_at DATETIME NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_embedding_models_provider_model (provider_id, model_name),
  KEY idx_embedding_models_default (is_default),
  KEY idx_embedding_models_enabled_default (is_enabled, is_default),
  CONSTRAINT fk_embedding_models_provider FOREIGN KEY (provider_id) REFERENCES embedding_providers (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Embedding model configuration';
