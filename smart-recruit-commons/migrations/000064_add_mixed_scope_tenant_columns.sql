-- Nullable tenant ownership for resources that may be global (candidate or
-- platform managed) or tenant-local. NULL means global; a value means the row
-- belongs exclusively to that tenant.

SET @default_tenant_id := (SELECT id FROM tenants WHERE is_default = 1 LIMIT 1);

ALTER TABLE ai_chat_sessions ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_ai_chat_sessions_tenant_owner (tenant_id, owner_role, owner_id, updated_at), ADD CONSTRAINT fk_ai_chat_sessions_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE ai_chat_history ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_ai_chat_history_tenant_session (tenant_id, session_id, created_at), ADD CONSTRAINT fk_ai_chat_history_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE ai_session_summaries ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_ai_session_summaries_tenant_session (tenant_id, session_id), ADD CONSTRAINT fk_ai_session_summaries_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE ai_tool_traces ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_ai_tool_traces_tenant_session (tenant_id, session_id, created_at), ADD CONSTRAINT fk_ai_tool_traces_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE agent_runs ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_agent_runs_tenant_session (tenant_id, session_id, created_at), ADD CONSTRAINT fk_agent_runs_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE agent_run_events ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_agent_run_events_tenant_run (tenant_id, run_id, seq), ADD CONSTRAINT fk_agent_run_events_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE agent_run_steps ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_agent_run_steps_tenant_run (tenant_id, run_id, step_index), ADD CONSTRAINT fk_agent_run_steps_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE ai_memories ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_ai_memories_tenant_owner (tenant_id, hr_id, scope_type, scope_id), ADD CONSTRAINT fk_ai_memories_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE ai_embeddings ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_ai_embeddings_tenant_object (tenant_id, object_type, object_id), ADD CONSTRAINT fk_ai_embeddings_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);

ALTER TABLE notifications ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_notifications_tenant_receiver (tenant_id, receiver_id, receiver_account_type, created_at), ADD CONSTRAINT fk_notifications_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE event_outbox ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_event_outbox_tenant_status (tenant_id, status, created_at), ADD CONSTRAINT fk_event_outbox_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE email_logs ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_email_logs_tenant_created (tenant_id, created_at), ADD CONSTRAINT fk_email_logs_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE event_inbox ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_event_inbox_tenant_consumer (tenant_id, consumer_name, received_at), ADD CONSTRAINT fk_event_inbox_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE analytics_projection_events ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_projection_events_tenant_projection (tenant_id, projection_name, created_at), ADD CONSTRAINT fk_projection_events_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE third_party_usage_logs ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_usage_logs_tenant_created (tenant_id, created_at), ADD CONSTRAINT fk_usage_logs_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE ai_usage_auth_contexts ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_ai_usage_auth_tenant_actor (tenant_id, actor_user_id, created_at), ADD CONSTRAINT fk_ai_usage_auth_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);

ALTER TABLE llm_providers ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_llm_providers_tenant_enabled (tenant_id, is_enabled), ADD CONSTRAINT fk_llm_providers_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE llm_models ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_llm_models_tenant_provider (tenant_id, provider_id), ADD CONSTRAINT fk_llm_models_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE embedding_providers ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_embedding_providers_tenant_enabled (tenant_id, is_enabled), ADD CONSTRAINT fk_embedding_providers_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE embedding_models ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_embedding_models_tenant_provider (tenant_id, provider_id), ADD CONSTRAINT fk_embedding_models_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE prompt_templates ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_prompt_templates_tenant_agent (tenant_id, agent_type, is_active), ADD CONSTRAINT fk_prompt_templates_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE prompt_versions ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_prompt_versions_tenant_template (tenant_id, template_id, version), ADD CONSTRAINT fk_prompt_versions_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE agent_configs ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_agent_configs_tenant_type (tenant_id, agent_type, is_enabled), ADD CONSTRAINT fk_agent_configs_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE agent_tool_bindings ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_agent_tool_bindings_tenant_agent (tenant_id, agent_id), ADD CONSTRAINT fk_agent_tool_bindings_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE mcp_servers ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_mcp_servers_tenant_enabled (tenant_id, is_enabled), ADD CONSTRAINT fk_mcp_servers_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE mcp_tool_logs ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_mcp_tool_logs_tenant_created (tenant_id, created_at), ADD CONSTRAINT fk_mcp_tool_logs_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE mcp_tool_policies ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_mcp_policies_tenant_server (tenant_id, server_id, tool_name), ADD CONSTRAINT fk_mcp_policies_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE agent_capability_bindings ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_agent_capabilities_tenant_agent (tenant_id, agent_id), ADD CONSTRAINT fk_agent_capabilities_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE ai_skills ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_ai_skills_tenant_enabled (tenant_id, is_enabled), ADD CONSTRAINT fk_ai_skills_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE ai_skill_versions ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_ai_skill_versions_tenant_skill (tenant_id, skill_id), ADD CONSTRAINT fk_ai_skill_versions_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE ai_skill_tools ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_ai_skill_tools_tenant_version (tenant_id, skill_version_id), ADD CONSTRAINT fk_ai_skill_tools_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE agent_skills ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_agent_skills_tenant_enabled (tenant_id, is_enabled), ADD CONSTRAINT fk_agent_skills_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE agent_skill_versions ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id, ADD KEY idx_agent_skill_versions_tenant_skill (tenant_id, skill_id), ADD CONSTRAINT fk_agent_skill_versions_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);

UPDATE ai_chat_sessions SET tenant_id = @default_tenant_id WHERE owner_role = 2;
UPDATE ai_chat_history SET tenant_id = @default_tenant_id WHERE owner_role = 2;
UPDATE ai_session_summaries summary JOIN ai_chat_sessions session ON session.id = summary.session_id SET summary.tenant_id = session.tenant_id;
UPDATE ai_tool_traces trace JOIN ai_chat_sessions session ON session.id = trace.session_id SET trace.tenant_id = session.tenant_id;
UPDATE agent_runs run JOIN ai_chat_sessions session ON session.id = run.session_id SET run.tenant_id = session.tenant_id;
UPDATE agent_run_events event JOIN agent_runs run ON run.id = event.run_id SET event.tenant_id = run.tenant_id;
UPDATE agent_run_steps step JOIN agent_runs run ON run.id = step.run_id SET step.tenant_id = run.tenant_id;
UPDATE ai_memories SET tenant_id = @default_tenant_id;
UPDATE notifications SET tenant_id = @default_tenant_id WHERE receiver_account_type = 'staff';
UPDATE event_outbox SET tenant_id = @default_tenant_id WHERE aggregate_type IN ('job','application','interview','offer','candidate_note','follow_up_task');
UPDATE email_logs mail JOIN event_outbox event ON event.event_id = mail.event_id SET mail.tenant_id = event.tenant_id;
UPDATE third_party_usage_logs SET tenant_id = @default_tenant_id WHERE role >= 2;
UPDATE ai_usage_auth_contexts auth_context JOIN third_party_usage_logs usage_log ON usage_log.id = auth_context.usage_log_id SET auth_context.tenant_id = usage_log.tenant_id;
