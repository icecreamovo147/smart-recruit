package persistence

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	appmemory "smart-recruit-ai-agent-service/internal/application/memory"
	domainmemory "smart-recruit-ai-agent-service/internal/domain/memory"
)

type MemoryListFilter struct {
	TenantID   *uint64
	OwnerRole  domainmemory.OwnerRole
	OwnerID    uint64
	ScopeType  string
	ScopeID    uint64
	MemoryType string
	Status     domainmemory.Status
	PIILevel   domainmemory.PIILevel
	Page       int
	PageSize   int
}

type MemoryRecallFilter struct {
	TenantID    *uint64
	OwnerRole   domainmemory.OwnerRole
	OwnerID     uint64
	Scopes      []domainmemory.Scope
	ScopeType   string
	ScopeID     uint64
	Query       string
	Limit       int
	MemoryTypes []string
}

type MemoryRecallItem struct {
	Memory domainmemory.Memory
	Score  float64
	Reason string
}

type aiMemoryRecord struct {
	ID              uint64     `gorm:"column:id;primaryKey"`
	TenantID        *uint64    `gorm:"column:tenant_id"`
	OwnerRole       int32      `gorm:"column:owner_role"`
	OwnerID         uint64     `gorm:"column:owner_id"`
	HRID            uint64     `gorm:"column:hr_id"`
	ScopeType       string     `gorm:"column:scope_type"`
	ScopeID         uint64     `gorm:"column:scope_id"`
	MemoryType      string     `gorm:"column:memory_type"`
	Content         string     `gorm:"column:content"`
	Source          string     `gorm:"column:source"`
	Confidence      float64    `gorm:"column:confidence"`
	Importance      float64    `gorm:"column:importance"`
	Status          string     `gorm:"column:status"`
	DeletedAt       *time.Time `gorm:"column:deleted_at"`
	ContentHash     *string    `gorm:"column:content_hash"`
	PIILevel        string     `gorm:"column:pii_level"`
	SourceSessionID *uint64    `gorm:"column:source_session_id"`
	SourceMessageID *uint64    `gorm:"column:source_message_id"`
	SourceRunID     *uint64    `gorm:"column:source_run_id"`
	CreatedBy       *uint64    `gorm:"column:created_by"`
	RevokedBy       *uint64    `gorm:"column:revoked_by"`
	RevokeReason    *string    `gorm:"column:revoke_reason"`
	ExpiresAt       *time.Time `gorm:"column:expires_at"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at"`
}

func (aiMemoryRecord) TableName() string { return "ai_memories" }

func (s *NativeStore) CreateMemory(ctx context.Context, memory domainmemory.Memory) (domainmemory.Memory, error) {
	if s == nil || s.db == nil {
		return domainmemory.Memory{}, gorm.ErrInvalidDB
	}
	if err := memory.OwnerKey().Validate(); err != nil {
		return domainmemory.Memory{}, err
	}
	if err := domainmemory.ValidateScope(memory.OwnerRole, memory.Scope); err != nil {
		return domainmemory.Memory{}, err
	}
	if memory.ContentHash == "" {
		memory.ContentHash = domainmemory.ContentHash(memory.Content)
	}
	if memory.Status == "" {
		memory.Status = domainmemory.StatusActive
	}
	if memory.PIILevel == "" {
		memory.PIILevel = domainmemory.ClassifyPIILevel(memory.Content)
	}
	if !domainmemory.WriteAllowedPIILevel(memory.PIILevel, memory.AllowHighPII) {
		return domainmemory.Memory{}, fmt.Errorf("memory content contains high PII and was rejected")
	}
	now := time.Now()
	row := memoryToRecord(memory, now)
	if existing, found, err := s.findMemoryByDedupKey(ctx, memory.OwnerKey(), memory.Scope, memory.ContentHash); err != nil {
		return domainmemory.Memory{}, err
	} else if found {
		existing.Content = memory.Content
		existing.Confidence = memory.Confidence
		existing.Importance = memory.Importance
		existing.Source = memory.Source
		existing.MemoryType = memory.MemoryType
		existing.PIILevel = memory.PIILevel
		existing.SourceSessionID = memory.SourceSessionID
		existing.SourceMessageID = memory.SourceMessageID
		existing.SourceRunID = memory.SourceRunID
		existing.UpdatedAt = now
		return s.UpdateMemory(ctx, existing)
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return domainmemory.Memory{}, err
	}
	return recordToMemory(row), nil
}

func (s *NativeStore) GetMemory(ctx context.Context, owner domainmemory.OwnerKey, id uint64) (domainmemory.Memory, bool, error) {
	if s == nil || s.db == nil {
		return domainmemory.Memory{}, false, gorm.ErrInvalidDB
	}
	if err := owner.Validate(); err != nil {
		return domainmemory.Memory{}, false, err
	}
	var row aiMemoryRecord
	query := s.db.WithContext(ctx).Where("id = ?", id)
	query = applyMemoryOwnerFilter(query, owner)
	err := query.First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domainmemory.Memory{}, false, nil
	}
	if err != nil {
		return domainmemory.Memory{}, false, err
	}
	return recordToMemory(row), true, nil
}

func (s *NativeStore) ListMemories(ctx context.Context, filter MemoryListFilter) ([]domainmemory.Memory, int64, error) {
	if s == nil || s.db == nil {
		return nil, 0, gorm.ErrInvalidDB
	}
	query := s.db.WithContext(ctx).Model(&aiMemoryRecord{})
	query = applyMemoryOwnerFilter(query, domainmemory.OwnerKey{
		TenantID: filter.TenantID,
		Role:     filter.OwnerRole,
		ID:       filter.OwnerID,
	})
	if filter.ScopeType != "" {
		query = query.Where("scope_type = ?", filter.ScopeType)
		if filter.ScopeID > 0 || filter.ScopeType == domainmemory.ScopeHR {
			query = query.Where("scope_id = ?", filter.ScopeID)
		}
	}
	if filter.MemoryType != "" {
		query = query.Where("memory_type = ?", filter.MemoryType)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.PIILevel != "" {
		query = query.Where("pii_level = ?", filter.PIILevel)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	var rows []aiMemoryRecord
	if err := query.Order("updated_at DESC, id DESC").Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domainmemory.Memory, 0, len(rows))
	for _, row := range rows {
		out = append(out, recordToMemory(row))
	}
	return out, total, nil
}

func (s *NativeStore) UpdateMemory(ctx context.Context, memory domainmemory.Memory) (domainmemory.Memory, error) {
	if s == nil || s.db == nil {
		return domainmemory.Memory{}, gorm.ErrInvalidDB
	}
	if memory.ID == 0 {
		return domainmemory.Memory{}, fmt.Errorf("memory id is required")
	}
	if err := memory.OwnerKey().Validate(); err != nil {
		return domainmemory.Memory{}, err
	}
	existing, found, err := s.GetMemory(ctx, memory.OwnerKey(), memory.ID)
	if err != nil {
		return domainmemory.Memory{}, err
	}
	if !found {
		return domainmemory.Memory{}, gorm.ErrRecordNotFound
	}
	if memory.Status != "" && memory.Status != existing.Status {
		if err := domainmemory.ValidateStatusTransition(existing.Status, memory.Status); err != nil {
			return domainmemory.Memory{}, err
		}
	}
	if strings.TrimSpace(memory.Content) != "" {
		memory.ContentHash = domainmemory.ContentHash(memory.Content)
		memory.PIILevel = domainmemory.ClassifyPIILevel(memory.Content)
		if !domainmemory.WriteAllowedPIILevel(memory.PIILevel, memory.AllowHighPII) {
			return domainmemory.Memory{}, fmt.Errorf("memory content contains high PII and was rejected")
		}
	}
	now := time.Now()
	updates := map[string]any{"updated_at": now}
	if strings.TrimSpace(memory.Content) != "" {
		updates["content"] = memory.Content
		updates["content_hash"] = memory.ContentHash
		updates["pii_level"] = string(memory.PIILevel)
	}
	if memory.Confidence > 0 {
		updates["confidence"] = memory.Confidence
	}
	if memory.Importance > 0 {
		updates["importance"] = memory.Importance
	}
	if memory.Status != "" {
		updates["status"] = string(memory.Status)
	}
	if memory.MemoryType != "" {
		updates["memory_type"] = memory.MemoryType
	}
	if memory.ExpiresAt != nil {
		updates["expires_at"] = memory.ExpiresAt
	}
	query := s.db.WithContext(ctx).Model(&aiMemoryRecord{}).Where("id = ?", memory.ID)
	query = applyMemoryOwnerFilter(query, memory.OwnerKey())
	result := query.Updates(updates)
	if result.Error != nil {
		return domainmemory.Memory{}, result.Error
	}
	return s.mustGetMemory(ctx, memory.OwnerKey(), memory.ID)
}

func (s *NativeStore) RevokeMemory(ctx context.Context, owner domainmemory.OwnerKey, id, revokedBy uint64, reason string) error {
	if s == nil || s.db == nil {
		return gorm.ErrInvalidDB
	}
	if err := owner.Validate(); err != nil {
		return err
	}
	existing, found, err := s.GetMemory(ctx, owner, id)
	if err != nil {
		return err
	}
	if !found {
		return gorm.ErrRecordNotFound
	}
	if err := domainmemory.ValidateStatusTransition(existing.Status, domainmemory.StatusRevoked); err != nil {
		return err
	}
	now := time.Now()
	updates := map[string]any{
		"status":        string(domainmemory.StatusRevoked),
		"deleted_at":    now,
		"revoked_by":    revokedBy,
		"revoke_reason": strings.TrimSpace(reason),
		"updated_at":    now,
	}
	query := s.db.WithContext(ctx).Model(&aiMemoryRecord{}).Where("id = ?", id)
	query = applyMemoryOwnerFilter(query, owner)
	return query.Updates(updates).Error
}

func (s *NativeStore) ListActiveForRecall(ctx context.Context, filter MemoryRecallFilter) ([]MemoryRecallItem, error) {
	if s == nil || s.db == nil {
		return nil, gorm.ErrInvalidDB
	}
	query := s.db.WithContext(ctx).Model(&aiMemoryRecord{})
	query = applyMemoryOwnerFilter(query, domainmemory.OwnerKey{
		TenantID: filter.TenantID,
		Role:     filter.OwnerRole,
		ID:       filter.OwnerID,
	})
	query = query.Where("status = ?", domainmemory.StatusActive)
	query = query.Where("(expires_at IS NULL OR expires_at > ?)", time.Now())
	if len(filter.Scopes) > 0 {
		parts := make([]string, 0, len(filter.Scopes))
		args := make([]any, 0, len(filter.Scopes)*2)
		for _, scope := range filter.Scopes {
			parts = append(parts, "(scope_type = ? AND scope_id = ?)")
			args = append(args, scope.Type, scope.ID)
		}
		query = query.Where(strings.Join(parts, " OR "), args...)
	} else if filter.ScopeType != "" {
		query = query.Where("scope_type = ?", filter.ScopeType)
		if filter.ScopeID > 0 || filter.ScopeType == domainmemory.ScopeHR {
			query = query.Where("scope_id = ?", filter.ScopeID)
		}
	}
	if len(filter.MemoryTypes) > 0 {
		query = query.Where("memory_type IN ?", filter.MemoryTypes)
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	var rows []aiMemoryRecord
	if err := query.Order("importance DESC, updated_at DESC, id DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]MemoryRecallItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, MemoryRecallItem{Memory: recordToMemory(row)})
	}
	return out, nil
}

func (s *NativeStore) ExpireAndCleanup(ctx context.Context, revokedRetention time.Duration) (appmemory.CleanupResult, error) {
	if s == nil || s.db == nil {
		return appmemory.CleanupResult{}, gorm.ErrInvalidDB
	}
	if revokedRetention <= 0 {
		revokedRetention = 30 * 24 * time.Hour
	}
	now := time.Now()
	cutoff := now.Add(-revokedRetention)
	result := appmemory.CleanupResult{}

	var expiredRows []aiMemoryRecord
	if err := s.db.WithContext(ctx).
		Where("status = ? AND expires_at IS NOT NULL AND expires_at <= ?", domainmemory.StatusActive, now).
		Find(&expiredRows).Error; err != nil {
		return appmemory.CleanupResult{}, err
	}
	for _, row := range expiredRows {
		if err := domainmemory.ValidateStatusTransition(domainmemory.Status(row.Status), domainmemory.StatusArchived); err != nil {
			continue
		}
		if err := s.db.WithContext(ctx).Model(&aiMemoryRecord{}).Where("id = ?", row.ID).
			Updates(map[string]any{"status": string(domainmemory.StatusArchived), "updated_at": now}).Error; err != nil {
			return appmemory.CleanupResult{}, err
		}
		result.ExpiredArchived++
		result.InvalidatedMemoryIDs = append(result.InvalidatedMemoryIDs, row.ID)
	}

	var purgeRows []aiMemoryRecord
	if err := s.db.WithContext(ctx).
		Where("status = ? AND deleted_at IS NOT NULL AND deleted_at <= ?", domainmemory.StatusRevoked, cutoff).
		Find(&purgeRows).Error; err != nil {
		return appmemory.CleanupResult{}, err
	}
	for _, row := range purgeRows {
		if err := s.db.WithContext(ctx).Delete(&aiMemoryRecord{}, row.ID).Error; err != nil {
			return appmemory.CleanupResult{}, err
		}
		result.RevokedPurged++
		result.InvalidatedMemoryIDs = append(result.InvalidatedMemoryIDs, row.ID)
	}
	return result, nil
}

func applyMemoryOwnerFilter(query *gorm.DB, owner domainmemory.OwnerKey) *gorm.DB {
	if err := owner.Validate(); err != nil {
		return query.Where("1 = 0")
	}
	query = query.Where("owner_role = ? AND owner_id = ?", int32(owner.Role), owner.ID)
	if owner.TenantID == nil {
		return query.Where("tenant_id IS NULL")
	}
	return query.Where("tenant_id = ?", *owner.TenantID)
}

func (s *NativeStore) findMemoryByDedupKey(ctx context.Context, owner domainmemory.OwnerKey, scope domainmemory.Scope, hash string) (domainmemory.Memory, bool, error) {
	var row aiMemoryRecord
	query := s.db.WithContext(ctx).
		Where("scope_type = ? AND scope_id = ? AND content_hash = ? AND status = ?",
			scope.Type, scope.ID, hash, domainmemory.StatusActive)
	query = applyMemoryOwnerFilter(query, owner)
	err := query.First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domainmemory.Memory{}, false, nil
	}
	if err != nil {
		return domainmemory.Memory{}, false, err
	}
	return recordToMemory(row), true, nil
}

func (s *NativeStore) mustGetMemory(ctx context.Context, owner domainmemory.OwnerKey, id uint64) (domainmemory.Memory, error) {
	memory, found, err := s.GetMemory(ctx, owner, id)
	if err != nil {
		return domainmemory.Memory{}, err
	}
	if !found {
		return domainmemory.Memory{}, gorm.ErrRecordNotFound
	}
	return memory, nil
}

func memoryToRecord(memory domainmemory.Memory, now time.Time) aiMemoryRecord {
	hash := memory.ContentHash
	row := aiMemoryRecord{
		TenantID:        memory.TenantID,
		OwnerRole:       int32(memory.OwnerRole),
		OwnerID:         memory.OwnerID,
		HRID:            domainmemory.CompatibilityHRID(memory.OwnerRole, memory.OwnerID),
		ScopeType:       memory.Scope.Type,
		ScopeID:         memory.Scope.ID,
		MemoryType:      memory.MemoryType,
		Content:         memory.Content,
		Source:          memory.Source,
		Confidence:      memory.Confidence,
		Importance:      memory.Importance,
		Status:          string(memory.Status),
		ContentHash:     &hash,
		PIILevel:        string(memory.PIILevel),
		SourceSessionID: memory.SourceSessionID,
		SourceMessageID: memory.SourceMessageID,
		SourceRunID:     memory.SourceRunID,
		CreatedBy:       memory.CreatedBy,
		ExpiresAt:       memory.ExpiresAt,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if memory.Source == "" {
		row.Source = "agent"
	}
	if row.Confidence <= 0 {
		row.Confidence = 1
	}
	if row.Importance <= 0 {
		row.Importance = 1
	}
	if memory.CreatedAt.IsZero() {
		row.CreatedAt = now
	} else {
		row.CreatedAt = memory.CreatedAt
	}
	if memory.UpdatedAt.IsZero() {
		row.UpdatedAt = now
	} else {
		row.UpdatedAt = memory.UpdatedAt
	}
	return row
}

func recordToMemory(row aiMemoryRecord) domainmemory.Memory {
	hash := ""
	if row.ContentHash != nil {
		hash = *row.ContentHash
	}
	revokeReason := ""
	if row.RevokeReason != nil {
		revokeReason = *row.RevokeReason
	}
	return domainmemory.Memory{
		ID:              row.ID,
		TenantID:        row.TenantID,
		OwnerRole:       domainmemory.OwnerRole(row.OwnerRole),
		OwnerID:         row.OwnerID,
		HRID:            row.HRID,
		Scope:           domainmemory.Scope{Type: row.ScopeType, ID: row.ScopeID},
		MemoryType:      row.MemoryType,
		Content:         row.Content,
		Source:          row.Source,
		Confidence:      row.Confidence,
		Importance:      row.Importance,
		Status:          domainmemory.Status(row.Status),
		PIILevel:        domainmemory.PIILevel(row.PIILevel),
		ContentHash:     hash,
		SourceSessionID: row.SourceSessionID,
		SourceMessageID: row.SourceMessageID,
		SourceRunID:     row.SourceRunID,
		CreatedBy:       row.CreatedBy,
		RevokedBy:       row.RevokedBy,
		RevokeReason:    revokeReason,
		ExpiresAt:       row.ExpiresAt,
		DeletedAt:       row.DeletedAt,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}
