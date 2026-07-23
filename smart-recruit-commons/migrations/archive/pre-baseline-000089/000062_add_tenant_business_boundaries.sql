-- Establish hard tenant ownership on every enterprise recruitment aggregate.
-- Existing single-enterprise data is assigned to the seeded default tenant.

SET @default_tenant_id := (SELECT id FROM tenants WHERE is_default = 1 LIMIT 1);

ALTER TABLE jobs ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id;
ALTER TABLE applications ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id;
ALTER TABLE application_status_transitions ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id;
ALTER TABLE candidate_match_evaluations ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id;
ALTER TABLE candidate_match_evidence ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id;
ALTER TABLE interview_schedules ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id;
ALTER TABLE interview_feedback ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id;
ALTER TABLE offers ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id;
ALTER TABLE offer_events ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id;
ALTER TABLE departments ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id;
ALTER TABLE job_locations ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id;
ALTER TABLE department_locations ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id;
ALTER TABLE candidate_notes ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id;
ALTER TABLE candidate_tags ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id;
ALTER TABLE candidate_tag_assignments ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id;
ALTER TABLE follow_up_tasks ADD COLUMN tenant_id BIGINT UNSIGNED NULL AFTER id;

UPDATE jobs SET tenant_id = @default_tenant_id WHERE tenant_id IS NULL;
UPDATE applications SET tenant_id = @default_tenant_id WHERE tenant_id IS NULL;
UPDATE application_status_transitions SET tenant_id = @default_tenant_id WHERE tenant_id IS NULL;
UPDATE candidate_match_evaluations SET tenant_id = @default_tenant_id WHERE tenant_id IS NULL;
UPDATE candidate_match_evidence SET tenant_id = @default_tenant_id WHERE tenant_id IS NULL;
UPDATE interview_schedules SET tenant_id = @default_tenant_id WHERE tenant_id IS NULL;
UPDATE interview_feedback SET tenant_id = @default_tenant_id WHERE tenant_id IS NULL;
UPDATE offers SET tenant_id = @default_tenant_id WHERE tenant_id IS NULL;
UPDATE offer_events SET tenant_id = @default_tenant_id WHERE tenant_id IS NULL;
UPDATE departments SET tenant_id = @default_tenant_id WHERE tenant_id IS NULL;
UPDATE job_locations SET tenant_id = @default_tenant_id WHERE tenant_id IS NULL;
UPDATE department_locations SET tenant_id = @default_tenant_id WHERE tenant_id IS NULL;
UPDATE candidate_notes SET tenant_id = @default_tenant_id WHERE tenant_id IS NULL;
UPDATE candidate_tags SET tenant_id = @default_tenant_id WHERE tenant_id IS NULL;
UPDATE candidate_tag_assignments SET tenant_id = @default_tenant_id WHERE tenant_id IS NULL;
UPDATE follow_up_tasks SET tenant_id = @default_tenant_id WHERE tenant_id IS NULL;

ALTER TABLE jobs MODIFY tenant_id BIGINT UNSIGNED NOT NULL, ADD KEY idx_jobs_tenant_status_created (tenant_id, status, created_at, id), ADD CONSTRAINT fk_jobs_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE applications MODIFY tenant_id BIGINT UNSIGNED NOT NULL, ADD KEY idx_applications_tenant_job_status (tenant_id, job_id, status_key, is_current), ADD CONSTRAINT fk_applications_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE application_status_transitions MODIFY tenant_id BIGINT UNSIGNED NOT NULL, ADD KEY idx_app_transitions_tenant_app_created (tenant_id, application_id, created_at), ADD CONSTRAINT fk_app_transitions_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE candidate_match_evaluations MODIFY tenant_id BIGINT UNSIGNED NOT NULL, ADD KEY idx_match_eval_tenant_app (tenant_id, application_id), ADD CONSTRAINT fk_match_eval_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE candidate_match_evidence MODIFY tenant_id BIGINT UNSIGNED NOT NULL, ADD KEY idx_match_evidence_tenant_eval (tenant_id, evaluation_id), ADD CONSTRAINT fk_match_evidence_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE interview_schedules MODIFY tenant_id BIGINT UNSIGNED NOT NULL, ADD KEY idx_interviews_tenant_interviewer (tenant_id, interviewer_id, deleted_at), ADD CONSTRAINT fk_interviews_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE interview_feedback MODIFY tenant_id BIGINT UNSIGNED NOT NULL, ADD KEY idx_feedback_tenant_interview (tenant_id, interview_id), ADD CONSTRAINT fk_feedback_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE offers MODIFY tenant_id BIGINT UNSIGNED NOT NULL, ADD KEY idx_offers_tenant_status_created (tenant_id, status, created_at), ADD CONSTRAINT fk_offers_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE offer_events MODIFY tenant_id BIGINT UNSIGNED NOT NULL, ADD KEY idx_offer_events_tenant_offer (tenant_id, offer_id, created_at), ADD CONSTRAINT fk_offer_events_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE departments DROP INDEX uk_department_parent_name, MODIFY tenant_id BIGINT UNSIGNED NOT NULL, ADD UNIQUE KEY uk_department_tenant_parent_name (tenant_id, parent_id, name), ADD KEY idx_departments_tenant_active (tenant_id, is_active, deleted_at), ADD CONSTRAINT fk_departments_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE job_locations DROP INDEX uk_job_location_name, MODIFY tenant_id BIGINT UNSIGNED NOT NULL, ADD UNIQUE KEY uk_job_location_tenant_name (tenant_id, name), ADD KEY idx_job_locations_tenant_active (tenant_id, is_active, deleted_at), ADD CONSTRAINT fk_job_locations_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE department_locations MODIFY tenant_id BIGINT UNSIGNED NOT NULL, ADD KEY idx_department_locations_tenant (tenant_id, department_id, location_id), ADD CONSTRAINT fk_department_locations_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE candidate_notes MODIFY tenant_id BIGINT UNSIGNED NOT NULL, ADD KEY idx_candidate_notes_tenant_candidate (tenant_id, candidate_user_id, created_at), ADD CONSTRAINT fk_candidate_notes_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE candidate_tags DROP INDEX uk_tag_name, MODIFY tenant_id BIGINT UNSIGNED NOT NULL, ADD UNIQUE KEY uk_candidate_tag_tenant_name (tenant_id, name), ADD CONSTRAINT fk_candidate_tags_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE candidate_tag_assignments MODIFY tenant_id BIGINT UNSIGNED NOT NULL, ADD KEY idx_tag_assignments_tenant_candidate (tenant_id, candidate_user_id), ADD CONSTRAINT fk_tag_assignments_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE follow_up_tasks MODIFY tenant_id BIGINT UNSIGNED NOT NULL, ADD KEY idx_follow_up_tenant_assignee (tenant_id, assignee_user_id, status, due_at), ADD CONSTRAINT fk_follow_up_tasks_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
