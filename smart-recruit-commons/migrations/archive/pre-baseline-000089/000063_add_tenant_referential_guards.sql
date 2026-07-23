-- Composite keys make tenant consistency a database invariant across
-- recruitment aggregates, not merely an application-layer convention.

ALTER TABLE jobs ADD UNIQUE KEY uk_jobs_tenant_id (tenant_id, id);
ALTER TABLE applications ADD UNIQUE KEY uk_applications_tenant_id (tenant_id, id), ADD CONSTRAINT fk_applications_tenant_job FOREIGN KEY (tenant_id, job_id) REFERENCES jobs (tenant_id, id);
ALTER TABLE application_status_transitions ADD CONSTRAINT fk_app_transitions_tenant_application FOREIGN KEY (tenant_id, application_id) REFERENCES applications (tenant_id, id);

ALTER TABLE candidate_match_evaluations ADD UNIQUE KEY uk_match_eval_tenant_id (tenant_id, id), ADD CONSTRAINT fk_match_eval_tenant_application FOREIGN KEY (tenant_id, application_id) REFERENCES applications (tenant_id, id), ADD CONSTRAINT fk_match_eval_tenant_job FOREIGN KEY (tenant_id, job_id) REFERENCES jobs (tenant_id, id);
ALTER TABLE candidate_match_evidence ADD CONSTRAINT fk_match_evidence_tenant_evaluation FOREIGN KEY (tenant_id, evaluation_id) REFERENCES candidate_match_evaluations (tenant_id, id) ON DELETE CASCADE;

ALTER TABLE interview_schedules ADD UNIQUE KEY uk_interviews_tenant_id (tenant_id, id), ADD CONSTRAINT fk_interviews_tenant_application FOREIGN KEY (tenant_id, application_id) REFERENCES applications (tenant_id, id);
ALTER TABLE interview_feedback ADD CONSTRAINT fk_feedback_tenant_interview FOREIGN KEY (tenant_id, interview_id) REFERENCES interview_schedules (tenant_id, id), ADD CONSTRAINT fk_feedback_tenant_application FOREIGN KEY (tenant_id, application_id) REFERENCES applications (tenant_id, id);

ALTER TABLE offers ADD UNIQUE KEY uk_offers_tenant_id (tenant_id, id), ADD CONSTRAINT fk_offers_tenant_application FOREIGN KEY (tenant_id, application_id) REFERENCES applications (tenant_id, id), ADD CONSTRAINT fk_offers_tenant_job FOREIGN KEY (tenant_id, job_id) REFERENCES jobs (tenant_id, id);
ALTER TABLE offer_events ADD CONSTRAINT fk_offer_events_tenant_offer FOREIGN KEY (tenant_id, offer_id) REFERENCES offers (tenant_id, id) ON DELETE CASCADE;

ALTER TABLE departments ADD UNIQUE KEY uk_departments_tenant_id (tenant_id, id);
ALTER TABLE job_locations ADD UNIQUE KEY uk_job_locations_tenant_id (tenant_id, id);
ALTER TABLE department_locations ADD CONSTRAINT fk_department_locations_tenant_department FOREIGN KEY (tenant_id, department_id) REFERENCES departments (tenant_id, id), ADD CONSTRAINT fk_department_locations_tenant_location FOREIGN KEY (tenant_id, location_id) REFERENCES job_locations (tenant_id, id);

ALTER TABLE candidate_tags ADD UNIQUE KEY uk_candidate_tags_tenant_id (tenant_id, id);
ALTER TABLE candidate_notes ADD CONSTRAINT fk_candidate_notes_tenant_application FOREIGN KEY (tenant_id, application_id) REFERENCES applications (tenant_id, id);
ALTER TABLE candidate_tag_assignments ADD CONSTRAINT fk_tag_assignments_tenant_tag FOREIGN KEY (tenant_id, tag_id) REFERENCES candidate_tags (tenant_id, id);
ALTER TABLE follow_up_tasks ADD CONSTRAINT fk_follow_up_tenant_application FOREIGN KEY (tenant_id, application_id) REFERENCES applications (tenant_id, id);
