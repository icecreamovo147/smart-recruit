package persistence

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"smart-recruit-proto/recruitment/pb"
)

//go:embed model_catalog_data.json
var bundledModelCatalogJSON []byte

const (
	metadataSourceProviderAPI      = "provider_api"
	metadataSourceProviderDetail   = "provider_detail"
	metadataSourceOfficialDocument = "official_document"
	metadataSourcePlatformPolicy   = "platform_policy"
	metadataSourceUser             = "user"
)

type bundledModelCatalog struct {
	Revision string                    `json:"revision"`
	Entries  []bundledModelCatalogItem `json:"entries"`
}

type bundledModelCatalogItem struct {
	ProviderFamily      string            `json:"provider_family"`
	ModelName           string            `json:"model_name"`
	DisplayName         string            `json:"display_name"`
	ContextWindowTokens *int              `json:"context_window_tokens"`
	MaxInputTokens      *int              `json:"max_input_tokens"`
	MaxOutputTokens     *int              `json:"max_output_tokens"`
	Temperature         *float64          `json:"temperature"`
	TopP                *float64          `json:"top_p"`
	Capabilities        json.RawMessage   `json:"capabilities"`
	FieldSources        map[string]string `json:"field_sources"`
	SourceType          string            `json:"source_type"`
	SourceURL           string            `json:"source_url"`
	VerifiedAt          string            `json:"verified_at"`
	ExpiresAt           string            `json:"expires_at"`
}

type llmModelCatalogRecord struct {
	ID                  int64           `gorm:"primaryKey"`
	ProviderFamily      string          `gorm:"column:provider_family;uniqueIndex:uk_llm_model_catalog_family_model"`
	ModelName           string          `gorm:"column:model_name;uniqueIndex:uk_llm_model_catalog_family_model"`
	DisplayName         sql.NullString  `gorm:"column:display_name"`
	ContextWindowTokens sql.NullInt64   `gorm:"column:context_window_tokens"`
	MaxInputTokens      sql.NullInt64   `gorm:"column:max_input_tokens"`
	MaxOutputTokens     sql.NullInt64   `gorm:"column:max_output_tokens"`
	Temperature         sql.NullFloat64 `gorm:"column:temperature"`
	TopP                sql.NullFloat64 `gorm:"column:top_p"`
	Capabilities        sql.NullString  `gorm:"column:capabilities"`
	FieldSources        sql.NullString  `gorm:"column:field_sources"`
	SourceType          string          `gorm:"column:source_type"`
	SourceURL           sql.NullString  `gorm:"column:source_url"`
	SourceRevision      string          `gorm:"column:source_revision"`
	ContentHash         string          `gorm:"column:content_hash"`
	Status              string          `gorm:"column:status"`
	ManagedBy           string          `gorm:"column:managed_by"`
	VerifiedAt          *time.Time      `gorm:"column:verified_at"`
	ExpiresAt           *time.Time      `gorm:"column:expires_at"`
	CreatedAt           time.Time       `gorm:"column:created_at"`
	UpdatedAt           time.Time       `gorm:"column:updated_at"`
}

func (llmModelCatalogRecord) TableName() string { return "llm_model_catalog" }

type llmModelMetadataObservationRecord struct {
	ID          int64          `gorm:"primaryKey"`
	ProviderID  int64          `gorm:"column:provider_id"`
	ModelName   string         `gorm:"column:model_name"`
	FieldName   string         `gorm:"column:field_name"`
	ValueJSON   string         `gorm:"column:value_json"`
	SourceType  string         `gorm:"column:source_type"`
	SourceRef   sql.NullString `gorm:"column:source_ref"`
	Confidence  float64        `gorm:"column:confidence"`
	ContentHash string         `gorm:"column:content_hash;uniqueIndex:uk_llm_model_observation_hash"`
	ObservedAt  time.Time      `gorm:"column:observed_at"`
	ExpiresAt   *time.Time     `gorm:"column:expires_at"`
	CreatedAt   time.Time      `gorm:"column:created_at"`
}

func (llmModelMetadataObservationRecord) TableName() string {
	return "llm_model_metadata_observations"
}

type metadataFieldSource struct {
	FieldName  string
	SourceType string
	SourceRef  string
	Confidence float64
	ObservedAt time.Time
	Verified   bool
}

func parseBundledModelCatalog() (bundledModelCatalog, error) {
	var catalog bundledModelCatalog
	if err := json.Unmarshal(bundledModelCatalogJSON, &catalog); err != nil {
		return catalog, fmt.Errorf("decode bundled model catalog: %w", err)
	}
	if strings.TrimSpace(catalog.Revision) == "" {
		return catalog, fmt.Errorf("bundled model catalog revision is required")
	}
	seen := make(map[string]bool, len(catalog.Entries))
	for _, item := range catalog.Entries {
		key := strings.TrimSpace(item.ProviderFamily) + "\x00" + strings.TrimSpace(item.ModelName)
		if strings.TrimSpace(item.ProviderFamily) == "" || strings.TrimSpace(item.ModelName) == "" || seen[key] {
			return catalog, fmt.Errorf("bundled model catalog contains an invalid or duplicate entry")
		}
		seen[key] = true
		if item.SourceURL != "" {
			if !isHTTPSReference(item.SourceURL) {
				return catalog, fmt.Errorf("bundled model catalog source_url must be an HTTPS URL")
			}
		}
		for _, sourceRef := range item.FieldSources {
			if !isHTTPSReference(sourceRef) {
				return catalog, fmt.Errorf("bundled model catalog field source must be an HTTPS URL")
			}
		}
		if !positiveOptionalInt(item.ContextWindowTokens) || !positiveOptionalInt(item.MaxInputTokens) || !positiveOptionalInt(item.MaxOutputTokens) {
			return catalog, fmt.Errorf("bundled model catalog token limits must be positive when provided")
		}
		if item.Temperature != nil && (*item.Temperature < 0 || *item.Temperature > 2) {
			return catalog, fmt.Errorf("bundled model catalog temperature must be between 0 and 2")
		}
		if item.TopP != nil && (*item.TopP < 0 || *item.TopP > 1) {
			return catalog, fmt.Errorf("bundled model catalog top_p must be between 0 and 1")
		}
	}
	return catalog, nil
}

func isHTTPSReference(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	return err == nil && parsed.Scheme == "https" && parsed.Hostname() != ""
}

func positiveOptionalInt(value *int) bool {
	return value == nil || *value > 0
}

// SyncBundledLlmModelCatalog imports the reviewed, versioned JSON catalog with
// an idempotent upsert. Rows managed by administrators are never overwritten.
func (s *NativeStore) SyncBundledLlmModelCatalog(ctx context.Context) error {
	catalog, err := parseBundledModelCatalog()
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		activeKeys := make([]string, 0, len(catalog.Entries))
		for _, item := range catalog.Entries {
			row, err := bundledCatalogRecord(catalog.Revision, item)
			if err != nil {
				return err
			}
			activeKeys = append(activeKeys, row.ProviderFamily+"\x00"+row.ModelName)
			var existing llmModelCatalogRecord
			findErr := tx.Where("provider_family = ? AND model_name = ?", row.ProviderFamily, row.ModelName).First(&existing).Error
			if findErr == nil && existing.ManagedBy != "bundled" {
				continue
			}
			if findErr != nil && findErr != gorm.ErrRecordNotFound {
				return findErr
			}
			result := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "provider_family"}, {Name: "model_name"}},
				DoUpdates: clause.Assignments(map[string]any{
					"display_name": row.DisplayName, "context_window_tokens": row.ContextWindowTokens,
					"max_input_tokens": row.MaxInputTokens, "max_output_tokens": row.MaxOutputTokens,
					"temperature": row.Temperature, "top_p": row.TopP, "capabilities": row.Capabilities,
					"field_sources": row.FieldSources, "source_type": row.SourceType, "source_url": row.SourceURL,
					"source_revision": row.SourceRevision, "content_hash": row.ContentHash, "status": "active",
					"managed_by": "bundled", "verified_at": row.VerifiedAt, "expires_at": row.ExpiresAt,
				}),
			}).Create(&row)
			if result.Error != nil {
				return result.Error
			}
		}
		// Retire only bundled rows removed from this revision. Admin and remote
		// catalog entries remain under their own lifecycle.
		var bundledRows []llmModelCatalogRecord
		if err := tx.Where("managed_by = ?", "bundled").Find(&bundledRows).Error; err != nil {
			return err
		}
		active := make(map[string]bool, len(activeKeys))
		for _, key := range activeKeys {
			active[key] = true
		}
		for _, row := range bundledRows {
			if !active[row.ProviderFamily+"\x00"+row.ModelName] && row.Status != "inactive" {
				if err := tx.Model(&llmModelCatalogRecord{}).Where("id = ?", row.ID).Update("status", "inactive").Error; err != nil {
					return err
				}
			}
		}
		return tx.Where("expires_at IS NOT NULL AND expires_at < ?", time.Now()).Delete(&llmModelMetadataObservationRecord{}).Error
	})
}

func bundledCatalogRecord(revision string, item bundledModelCatalogItem) (llmModelCatalogRecord, error) {
	verifiedAt, err := parseCatalogTime(item.VerifiedAt)
	if err != nil {
		return llmModelCatalogRecord{}, err
	}
	expiresAt, err := parseCatalogTime(item.ExpiresAt)
	if err != nil {
		return llmModelCatalogRecord{}, err
	}
	capabilities := compactJSON(item.Capabilities)
	fieldSources, err := json.Marshal(item.FieldSources)
	if err != nil {
		return llmModelCatalogRecord{}, err
	}
	normalized, err := json.Marshal(item)
	if err != nil {
		return llmModelCatalogRecord{}, err
	}
	hash := sha256.Sum256(append([]byte(revision+"\x00"), normalized...))
	return llmModelCatalogRecord{
		ProviderFamily: strings.TrimSpace(item.ProviderFamily), ModelName: strings.TrimSpace(item.ModelName),
		DisplayName: nullableSQLString(item.DisplayName), ContextWindowTokens: nullableInt(item.ContextWindowTokens),
		MaxInputTokens: nullableInt(item.MaxInputTokens), MaxOutputTokens: nullableInt(item.MaxOutputTokens),
		Temperature: nullableFloat(item.Temperature), TopP: nullableFloat(item.TopP),
		Capabilities: nullableJSONText(capabilities), FieldSources: nullableJSONText(string(fieldSources)),
		SourceType: defaultString(strings.TrimSpace(item.SourceType), metadataSourceOfficialDocument),
		SourceURL:  nullableSQLString(item.SourceURL), SourceRevision: revision, ContentHash: hex.EncodeToString(hash[:]),
		Status: "active", ManagedBy: "bundled", VerifiedAt: verifiedAt, ExpiresAt: expiresAt,
	}, nil
}

func parseCatalogTime(value string) (*time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, fmt.Errorf("invalid catalog time %q: %w", value, err)
	}
	return &parsed, nil
}

func nullableInt(value *int) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*value), Valid: true}
}

func nullableFloat(value *float64) sql.NullFloat64 {
	if value == nil {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: *value, Valid: true}
}

func compactJSON(value json.RawMessage) string {
	if len(value) == 0 || string(value) == "null" {
		return ""
	}
	var compacted bytes.Buffer
	if err := json.Compact(&compacted, value); err != nil {
		return ""
	}
	return compacted.String()
}

func providerCatalogFamilies(provider llmProviderRecord) []string {
	families := make([]string, 0, 4)
	if parsed, err := url.Parse(provider.BaseURL); err == nil {
		host := strings.ToLower(parsed.Hostname())
		switch {
		case host == "api.deepseek.com" || strings.HasSuffix(host, ".deepseek.com"):
			families = append(families, "deepseek")
		case host == "api.openai.com" || strings.HasSuffix(host, ".openai.com"):
			families = append(families, "openai")
		case host == "api.anthropic.com" || strings.HasSuffix(host, ".anthropic.com"):
			families = append(families, "anthropic")
		case host == "generativelanguage.googleapis.com":
			families = append(families, "google_gemini")
		case host == "api.x.ai":
			families = append(families, "xai")
		case host == "api.mistral.ai" || strings.HasSuffix(host, ".mistral.ai"):
			families = append(families, "mistral")
		case host == "dashscope.aliyuncs.com" || host == "dashscope-intl.aliyuncs.com" || strings.HasSuffix(host, ".dashscope.aliyuncs.com"):
			families = append(families, "qwen")
		case host == "open.bigmodel.cn" || strings.HasSuffix(host, ".bigmodel.cn"):
			families = append(families, "zhipu")
		case host == "api.minimax.io" || strings.HasSuffix(host, ".minimax.io") || host == "api.minimaxi.com" || strings.HasSuffix(host, ".minimaxi.com"):
			families = append(families, "minimax")
		case host == "api.xiaomimimo.com" || strings.HasSuffix(host, ".xiaomimimo.com"):
			families = append(families, "xiaomi")
		case host == "api.moonshot.ai" || strings.HasSuffix(host, ".moonshot.ai") || host == "api.moonshot.cn" || strings.HasSuffix(host, ".moonshot.cn"):
			families = append(families, "moonshot")
		}
	}
	families = append(families, provider.ProviderType)
	return uniqueStrings(families)
}

func (s *NativeStore) enrichModelsFromCatalog(ctx context.Context, provider llmProviderRecord, models []discoveredModel, observedAt time.Time) ([]discoveredModel, error) {
	if len(models) == 0 {
		return models, nil
	}
	names := make([]string, 0, len(models))
	for _, model := range models {
		names = append(names, model.ModelName)
	}
	ownedFamilies := make([]string, 0, len(models))
	for _, model := range models {
		family := strings.ToLower(strings.TrimSpace(model.OwnedBy))
		if family != "" && len(family) <= 64 {
			ownedFamilies = append(ownedFamilies, family)
		}
	}
	families := uniqueStrings(append(ownedFamilies, providerCatalogFamilies(provider)...))
	var rows []llmModelCatalogRecord
	query := s.db.WithContext(ctx).Where("provider_family IN ? AND model_name IN ? AND status = ?", families, names, "active")
	query = query.Where("expires_at IS NULL OR expires_at > ?", observedAt)
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	priority := make(map[string]int)
	for index, family := range families {
		priority[family] = len(families) - index
	}
	byName := make(map[string]llmModelCatalogRecord, len(rows))
	for _, row := range rows {
		current, ok := byName[row.ModelName]
		if !ok || priority[row.ProviderFamily] > priority[current.ProviderFamily] {
			byName[row.ModelName] = row
		}
	}
	for index := range models {
		if row, ok := byName[models[index].ModelName]; ok {
			mergeCatalogModel(&models[index], row, observedAt)
		}
	}
	return models, nil
}

func mergeCatalogModel(model *discoveredModel, row llmModelCatalogRecord, observedAt time.Time) {
	if model.FieldSources == nil {
		model.FieldSources = make(map[string]metadataFieldSource)
	}
	sourceURLs := map[string]string{}
	if row.FieldSources.Valid {
		_ = json.Unmarshal([]byte(row.FieldSources.String), &sourceURLs)
	}
	sourceFor := func(field string) metadataFieldSource {
		ref := nullString(row.SourceURL)
		if sourceURLs[field] != "" {
			ref = sourceURLs[field]
		}
		return metadataFieldSource{FieldName: field, SourceType: row.SourceType, SourceRef: ref, Confidence: 1, ObservedAt: observedAt, Verified: row.VerifiedAt != nil}
	}
	if row.DisplayName.Valid && (model.DisplayName == "" || model.DisplayName == model.ModelName) {
		model.DisplayName = row.DisplayName.String
		model.FieldSources["display_name"] = sourceFor("display_name")
	}
	if !model.ContextWindowTokensKnown && row.ContextWindowTokens.Valid {
		model.ContextWindowTokens, model.ContextWindowTokensKnown = int32(row.ContextWindowTokens.Int64), true
		model.FieldSources["context_window_tokens"] = sourceFor("context_window_tokens")
	}
	if !model.MaxInputTokensKnown && row.MaxInputTokens.Valid {
		model.MaxInputTokens, model.MaxInputTokensKnown = int32(row.MaxInputTokens.Int64), true
		model.FieldSources["max_input_tokens"] = sourceFor("max_input_tokens")
	}
	if !model.MaxOutputTokensKnown && row.MaxOutputTokens.Valid {
		model.MaxOutputTokens, model.MaxOutputTokensKnown = int32(row.MaxOutputTokens.Int64), true
		model.FieldSources["max_output_tokens"] = sourceFor("max_output_tokens")
	}
	if !model.TemperatureKnown && row.Temperature.Valid {
		model.Temperature, model.TemperatureKnown = row.Temperature.Float64, true
		model.FieldSources["temperature"] = sourceFor("temperature")
	}
	if !model.TopPKnown && row.TopP.Valid {
		model.TopP, model.TopPKnown = row.TopP.Float64, true
		model.FieldSources["top_p"] = sourceFor("top_p")
	}
	if model.CapabilitiesJSON == "" && row.Capabilities.Valid {
		model.CapabilitiesJSON = row.Capabilities.String
		model.FieldSources["capabilities"] = sourceFor("capabilities")
	}
}

func annotateProviderModels(models []discoveredModel, sourceType, sourceRef string, observedAt time.Time) []discoveredModel {
	for index := range models {
		model := &models[index]
		if model.FieldSources == nil {
			model.FieldSources = make(map[string]metadataFieldSource)
		}
		fields := []string{"model_name"}
		if model.DisplayName != "" && model.DisplayName != model.ModelName {
			fields = append(fields, "display_name")
		}
		if model.OwnedBy != "" {
			fields = append(fields, "owned_by")
		}
		if model.ContextWindowTokensKnown {
			fields = append(fields, "context_window_tokens")
		}
		if model.MaxInputTokensKnown {
			fields = append(fields, "max_input_tokens")
		}
		if model.MaxOutputTokensKnown {
			fields = append(fields, "max_output_tokens")
		}
		if model.TemperatureKnown {
			fields = append(fields, "temperature")
		}
		if model.TopPKnown {
			fields = append(fields, "top_p")
		}
		if model.CapabilitiesJSON != "" {
			fields = append(fields, "capabilities")
		}
		for _, field := range fields {
			model.FieldSources[field] = metadataFieldSource{FieldName: field, SourceType: sourceType, SourceRef: sourceRef, Confidence: 1, ObservedAt: observedAt, Verified: sourceType == metadataSourceProviderDetail}
		}
	}
	return models
}

func (s *NativeStore) storeModelMetadataObservations(ctx context.Context, providerID int64, models []discoveredModel, expiresAt time.Time) error {
	rows := make([]llmModelMetadataObservationRecord, 0, len(models)*3)
	for _, model := range models {
		fields := discoveredModelValues(model)
		keys := make([]string, 0, len(fields))
		for field := range fields {
			keys = append(keys, field)
		}
		sort.Strings(keys)
		for _, field := range keys {
			source, ok := model.FieldSources[field]
			if !ok {
				continue
			}
			valueJSON, err := json.Marshal(fields[field])
			if err != nil {
				return err
			}
			hashInput := fmt.Sprintf("%d\x00%s\x00%s\x00%s\x00%s\x00%s", providerID, model.ModelName, field, source.SourceType, source.SourceRef, valueJSON)
			hash := sha256.Sum256([]byte(hashInput))
			observedAt := source.ObservedAt
			if observedAt.IsZero() {
				observedAt = time.Now()
			}
			rows = append(rows, llmModelMetadataObservationRecord{
				ProviderID: providerID, ModelName: model.ModelName, FieldName: field, ValueJSON: string(valueJSON),
				SourceType: source.SourceType, SourceRef: nullableSQLString(source.SourceRef), Confidence: source.Confidence,
				ContentHash: hex.EncodeToString(hash[:]), ObservedAt: observedAt, ExpiresAt: &expiresAt,
			})
		}
	}
	if len(rows) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "content_hash"}},
		DoUpdates: clause.AssignmentColumns([]string{"observed_at", "expires_at"}),
	}).CreateInBatches(rows, 200).Error
}

func discoveredModelValues(model discoveredModel) map[string]any {
	values := map[string]any{"model_name": model.ModelName}
	if model.DisplayName != "" && model.DisplayName != model.ModelName {
		values["display_name"] = model.DisplayName
	}
	if model.OwnedBy != "" {
		values["owned_by"] = model.OwnedBy
	}
	if model.ContextWindowTokensKnown {
		values["context_window_tokens"] = model.ContextWindowTokens
	}
	if model.MaxInputTokensKnown {
		values["max_input_tokens"] = model.MaxInputTokens
	}
	if model.MaxOutputTokensKnown {
		values["max_output_tokens"] = model.MaxOutputTokens
	}
	if model.TemperatureKnown {
		values["temperature"] = model.Temperature
	}
	if model.TopPKnown {
		values["top_p"] = model.TopP
	}
	if model.CapabilitiesJSON != "" {
		var capabilities any
		if json.Unmarshal([]byte(model.CapabilitiesJSON), &capabilities) == nil {
			values["capabilities"] = capabilities
		}
	}
	return values
}

func metadataFieldSourcesToPB(sources map[string]metadataFieldSource) []*pb.ModelMetadataFieldSource {
	keys := make([]string, 0, len(sources))
	for key := range sources {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]*pb.ModelMetadataFieldSource, 0, len(keys))
	for _, key := range keys {
		source := sources[key]
		result = append(result, &pb.ModelMetadataFieldSource{FieldName: key, SourceType: source.SourceType, SourceRef: source.SourceRef, Confidence: source.Confidence, ObservedAt: formatTime(source.ObservedAt), Verified: source.Verified})
	}
	return result
}
