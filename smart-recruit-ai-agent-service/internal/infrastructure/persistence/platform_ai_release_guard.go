package persistence

import (
	"context"
	"encoding/json"
	"errors"
)

var ErrReleasedConfigurationImmutable = errors.New("configuration is referenced by a published AI capability release; create a new configuration/version and publish a new capability release")

func (s *NativeStore) releasedConfigurationRefs(ctx context.Context) ([]PlatformAIConfigurationRefs, error) {
	if !s.db.Migrator().HasTable("platform_ai_capability_versions") {
		return nil, nil
	}
	var rows []struct {
		SnapshotJSON string `gorm:"column:snapshot_json"`
	}
	if err := s.db.WithContext(ctx).Table("platform_ai_capability_versions").
		Select("snapshot_json").Where("status = ?", PlatformAIReleasePublished).Scan(&rows).Error; err != nil {
		return nil, err
	}
	refs := make([]PlatformAIConfigurationRefs, 0, len(rows))
	for _, row := range rows {
		var snapshot PlatformAICapabilitySnapshot
		if err := json.Unmarshal([]byte(row.SnapshotJSON), &snapshot); err != nil {
			return nil, err
		}
		refs = append(refs, snapshot.ConfigurationRef)
	}
	return refs, nil
}

func (s *NativeStore) assertNotReleasedConfiguration(ctx context.Context, kind string, ids ...int64) error {
	refs, err := s.releasedConfigurationRefs(ctx)
	if err != nil {
		return err
	}
	wanted := int64Set(ids)
	for _, ref := range refs {
		var released []int64
		switch kind {
		case "agent":
			released = ref.AgentIDs
		case "prompt":
			released = ref.PromptTemplateIDs
		case "agent_skill_version":
			released = ref.AgentSkillVersionIDs
		case "mcp_policy":
			released = ref.MCPPolicyIDs
		}
		for _, id := range released {
			if _, ok := wanted[id]; ok {
				return ErrReleasedConfigurationImmutable
			}
		}
	}
	return nil
}

func (s *NativeStore) assertAgentSkillNotReleased(ctx context.Context, skillID int64) error {
	var ids []int64
	if err := s.db.WithContext(ctx).Table("agent_skill_versions").Where("skill_id = ?", skillID).Pluck("id", &ids).Error; err != nil {
		return err
	}
	return s.assertNotReleasedConfiguration(ctx, "agent_skill_version", ids...)
}

func (s *NativeStore) assertMCPServerNotReleased(ctx context.Context, serverID int64) error {
	var ids []int64
	if err := s.db.WithContext(ctx).Table("mcp_tool_policies").Where("server_id = ?", serverID).Pluck("id", &ids).Error; err != nil {
		return err
	}
	return s.assertNotReleasedConfiguration(ctx, "mcp_policy", ids...)
}
