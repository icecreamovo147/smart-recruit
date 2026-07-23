package service

import (
	"context"
	"testing"
	"time"

	"smart-recruit-recruitment-service/internal/application/command"
	"smart-recruit-recruitment-service/internal/domain/model"
	"smart-recruit-recruitment-service/internal/domain/repository"
)

func TestApplicationLifecycleApplyAndUpdateStatus(t *testing.T) {
	ctx := context.Background()
	outbox := &fakeOutboxPublisher{}
	apps := &fakeApplicationRepository{
		detail: &model.ApplicationDetail{
			ApplicationID: 7001,
			UserID:        42,
			JobID:         501,
			JobTitle:      "Backend Engineer",
			RealName:      "Ada",
			StatusKey:     model.StatusKeyApplied,
			RoundNo:       1,
			IsCurrent:     1,
		},
	}
	jobs := &fakeJobRepository{jobs: map[int64]*model.Job{501: {ID: 501, HRID: 7, Title: "Backend Engineer", Status: model.JobStatusOnline}}}
	svc, err := NewApplicationLifecycleService(ApplicationDeps{
		Applications: apps,
		Profiles:     &fakeProfileRepository{profile: &model.CandidateProfile{UserID: 42, RealName: "Ada", IsComplete: 1}},
		Resumes:      &fakeResumeRepository{valid: &model.Resume{ID: 801, UserID: 42, IsValid: 1}},
		Jobs:         jobs,
		Scopes:       &fakeJobScopeChecker{scope: repository.JobScope{Level: repository.JobScopeOwned}},
		Outbox:       outbox,
		Clock:        fixedClock{now: time.Unix(1710000000, 0)},
	})
	if err != nil {
		t.Fatalf("NewApplicationLifecycleService() error = %v", err)
	}

	applied, err := svc.ApplyJob(ctx, command.ApplyJob{UserID: 42, JobID: 501})
	if err != nil {
		t.Fatalf("ApplyJob() error = %v", err)
	}
	if applied.ApplicationID != 7001 {
		t.Fatalf("ApplicationID = %d, want 7001", applied.ApplicationID)
	}
	if apps.created.StatusKey != model.StatusKeyApplied || apps.created.ResumeID != 801 || apps.created.RoundNo != 1 {
		t.Fatalf("created application = %+v", apps.created)
	}
	if len(outbox.events) != 2 || !outbox.signaled {
		t.Fatalf("apply outbox events/signaled = %d/%v, want 2/true", len(outbox.events), outbox.signaled)
	}
	payload, ok := outbox.events[0].Payload.(model.NotificationPayload)
	if !ok || payload.Type != "new_application" || payload.ReceiverID != 7 || payload.BizID != 7001 {
		t.Fatalf("apply notification payload = %#v", outbox.events[0].Payload)
	}

	snapshot, err := svc.GetSnapshot(ctx, 7001)
	if err != nil {
		t.Fatalf("GetSnapshot() error = %v", err)
	}
	if snapshot.ApplicationID != 7001 || snapshot.CandidateUserID != 42 || snapshot.StatusKey != model.StatusKeyApplied || !snapshot.IsCurrent || snapshot.JobHRID != 7 {
		t.Fatalf("snapshot = %+v", snapshot)
	}

	outbox.events = nil
	outbox.signaled = false
	changed, err := svc.UpdateStatus(ctx, command.UpdateApplicationStatus{
		HRID:          7,
		ApplicationID: 7001,
		StatusKey:     model.StatusKeyScreenPassed,
	})
	if err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}
	if changed.FromStatus != model.StatusKeyApplied || changed.ToStatus != model.StatusKeyScreenPassed || changed.IsRePass {
		t.Fatalf("status change = %+v", changed)
	}
	if apps.updatedTarget != model.StatusKeyScreenPassed || apps.updatedLegacy != model.StatusKeyToLegacy[model.StatusKeyScreenPassed] {
		t.Fatalf("updated target/legacy = %s/%d", apps.updatedTarget, apps.updatedLegacy)
	}
	if len(outbox.events) != 2 || !outbox.signaled {
		t.Fatalf("status outbox events/signaled = %d/%v, want 2/true", len(outbox.events), outbox.signaled)
	}
	payload, ok = outbox.events[0].Payload.(model.NotificationPayload)
	if !ok || payload.Type != "application_approved" || payload.ReceiverID != 42 || payload.BizID != 7001 {
		t.Fatalf("status notification payload = %#v", outbox.events[0].Payload)
	}

	if _, err := svc.ApplyLifecycleTransition(ctx, command.ApplyApplicationLifecycleTransition{
		ActorUserID:       7,
		ApplicationID:     7001,
		ExpectedStatusKey: model.StatusKeyRejected,
		TargetStatusKey:   model.StatusKeyOfferPending,
	}); err == nil {
		t.Fatal("ApplyLifecycleTransition() expected status conflict")
	}
	result, err := svc.ApplyLifecycleTransition(ctx, command.ApplyApplicationLifecycleTransition{
		ActorUserID:       42,
		ActorAccountType:  "candidate",
		ApplicationID:     7001,
		ExpectedStatusKey: model.StatusKeyApplied,
		TargetStatusKey:   model.StatusKeyScreenPassed,
		CloseCurrentRound: true,
	})
	if err != nil {
		t.Fatalf("ApplyLifecycleTransition() error = %v", err)
	}
	if !result.Changed || result.CurrentStatusKey != model.StatusKeyScreenPassed || apps.closedRound != 7001 {
		t.Fatalf("lifecycle transition result=%+v closedRound=%d", result, apps.closedRound)
	}
}

func TestCollaborationTaxonomyAndUsageStatsServices(t *testing.T) {
	ctx := context.Background()
	clock := fixedClock{now: time.Date(2026, 7, 13, 9, 0, 0, 0, time.UTC)}
	authz := &fakeCollaborationAuthorizer{}
	collabRepo := &fakeCollaborationRepository{}
	collab, err := NewCollaborationService(collabRepo, authz, clock)
	if err != nil {
		t.Fatalf("NewCollaborationService() error = %v", err)
	}
	note, err := collab.CreateNote(ctx, command.CreateNote{StaffUserID: 7, CandidateUserID: 42, ApplicationID: 7001, Content: "Strong Go background"})
	if err != nil {
		t.Fatalf("CreateNote() error = %v", err)
	}
	if note.Note.Visibility != "internal" || note.Note.AuthorUserID != 7 || collabRepo.note.Content != "Strong Go background" {
		t.Fatalf("note = %+v", note.Note)
	}
	tag, err := collab.CreateTag(ctx, command.CreateTag{StaffUserID: 7, Name: "High Potential"})
	if err != nil {
		t.Fatalf("CreateTag() error = %v", err)
	}
	if tag.Tag.Color != "#409eff" {
		t.Fatalf("tag color = %q, want default", tag.Tag.Color)
	}
	if err := collab.AssignTag(ctx, command.AssignTag{StaffUserID: 7, CandidateUserID: 42, TagID: 11}); err != nil {
		t.Fatalf("AssignTag() error = %v", err)
	}
	if collabRepo.assignment.TagID != 11 || collabRepo.assignment.CandidateUserID != 42 {
		t.Fatalf("assignment = %+v", collabRepo.assignment)
	}
	if !authz.requiredAccess || len(authz.permissions) != 3 {
		t.Fatalf("authz access/permissions = %v/%v", authz.requiredAccess, authz.permissions)
	}

	taxRepo := &fakeTaxonomyRepository{activeLocations: []model.JobLocation{{ID: 20}, {ID: 21}}, fullName: "Engineering"}
	taxonomy, err := NewTaxonomyService(taxRepo)
	if err != nil {
		t.Fatalf("NewTaxonomyService() error = %v", err)
	}
	dep, err := taxonomy.CreateDepartment(ctx, command.CreateDepartment{AdminID: 7, Name: " Engineering ", SortOrder: 3})
	if err != nil {
		t.Fatalf("CreateDepartment() error = %v", err)
	}
	if dep.Department.ID != 301 || dep.Department.InheritLocations != 0 || dep.Department.Path != "/301/" || dep.Department.FullName != "Engineering" {
		t.Fatalf("department = %+v", dep.Department)
	}
	if len(taxRepo.replacedLocations) != 2 {
		t.Fatalf("replaced locations = %v, want default active locations", taxRepo.replacedLocations)
	}

	usageRepo := &fakeUsageStatsRepository{}
	usageAuth := &fakeUsageStatsAuthorizer{}
	usage, err := NewUsageStatsService(usageRepo, usageAuth, clock)
	if err != nil {
		t.Fatalf("NewUsageStatsService() error = %v", err)
	}
	if _, err := usage.GetStats(ctx, command.GetUsageStats{ActorUserID: 7}); err != nil {
		t.Fatalf("GetStats() error = %v", err)
	}
	if usageRepo.dimension != "model" || usageAuth.permission != model.PermissionAuditUsageRead {
		t.Fatalf("usage dimension/permission = %s/%s", usageRepo.dimension, usageAuth.permission)
	}
	if got, want := usageRepo.start, clock.now.AddDate(0, 0, -30); !got.Equal(want) {
		t.Fatalf("usage start = %s, want %s", got, want)
	}
	if _, err := usage.GetTrend(ctx, command.GetUsageTrend{ActorUserID: 7}); err != nil {
		t.Fatalf("GetTrend() error = %v", err)
	}
	if usageRepo.granularity != "day" {
		t.Fatalf("granularity = %q, want day", usageRepo.granularity)
	}
}

type fakeApplicationRepository struct {
	created       model.Application
	detail        *model.ApplicationDetail
	updatedTarget string
	updatedLegacy int32
	closedRound   int64
}

func (r *fakeApplicationRepository) CreateNewRound(_ context.Context, application *model.Application, afterCreate func(applicationID int64) error) error {
	application.ID = 7001
	application.RoundNo = 1
	application.IsCurrent = 1
	r.created = *application
	if afterCreate != nil {
		return afterCreate(application.ID)
	}
	return nil
}

func (r *fakeApplicationRepository) GetDetail(context.Context, int64) (*model.ApplicationDetail, error) {
	return r.detail, nil
}

func (r *fakeApplicationRepository) UpdateStatus(_ context.Context, applicationID int64, currentKey, targetKey string, legacyStatus int32, actorUserID int64, scope repository.JobScope, isRePass bool, reason string, afterUpdate func(rows int64) error) (int64, error) {
	r.updatedTarget = targetKey
	r.updatedLegacy = legacyStatus
	if afterUpdate != nil {
		return 1, afterUpdate(1)
	}
	return 1, nil
}

func (r *fakeApplicationRepository) CloseCurrentRound(_ context.Context, applicationID int64) error {
	r.closedRound = applicationID
	return nil
}

func (r *fakeApplicationRepository) ListTransitions(context.Context, int64) ([]model.ApplicationStatusTransition, error) {
	return nil, nil
}

type fakeCollaborationRepository struct {
	note       model.CandidateNote
	tag        model.CandidateTag
	assignment model.CandidateTagAssignment
	task       model.FollowUpTask
}

func (r *fakeCollaborationRepository) CreateNote(_ context.Context, note *model.CandidateNote) error {
	note.ID = 1
	r.note = *note
	return nil
}

func (r *fakeCollaborationRepository) ListNotes(context.Context, uint64, *uint64) ([]model.CandidateNote, error) {
	return nil, nil
}

func (r *fakeCollaborationRepository) CreateTag(_ context.Context, tag *model.CandidateTag) error {
	tag.ID = 11
	r.tag = *tag
	return nil
}

func (r *fakeCollaborationRepository) AssignTag(_ context.Context, assignment *model.CandidateTagAssignment) error {
	r.assignment = *assignment
	return nil
}

func (r *fakeCollaborationRepository) CreateTask(_ context.Context, task *model.FollowUpTask) error {
	task.ID = 21
	r.task = *task
	return nil
}

type fakeCollaborationAuthorizer struct {
	requiredAccess bool
	permissions    []string
}

func (a *fakeCollaborationAuthorizer) RequireCandidateAccess(context.Context, uint64, uint64) error {
	a.requiredAccess = true
	return nil
}

func (a *fakeCollaborationAuthorizer) AuthorizePermission(_ context.Context, _ uint64, permission string) error {
	a.permissions = append(a.permissions, permission)
	return nil
}

type fakeTaxonomyRepository struct {
	activeLocations   []model.JobLocation
	created           model.DepartmentNode
	replacedLocations []int64
	fullName          string
}

func (r *fakeTaxonomyRepository) GetDepartment(context.Context, int64) (*model.DepartmentNode, error) {
	return nil, nil
}

func (r *fakeTaxonomyRepository) FindDeletedDepartment(context.Context, int64, string) (*model.DepartmentNode, error) {
	return nil, nil
}

func (r *fakeTaxonomyRepository) ReactivateDepartment(context.Context, int64, int64, string, int) error {
	return nil
}

func (r *fakeTaxonomyRepository) CreateDepartment(_ context.Context, department *model.DepartmentNode) error {
	department.ID = 301
	r.created = *department
	return nil
}

func (r *fakeTaxonomyRepository) ListActiveLocations(context.Context) ([]model.JobLocation, error) {
	return r.activeLocations, nil
}

func (r *fakeTaxonomyRepository) ReplaceDepartmentLocations(_ context.Context, _ int64, _ int64, locationIDs []int64) error {
	r.replacedLocations = append([]int64(nil), locationIDs...)
	return nil
}

func (r *fakeTaxonomyRepository) UpdateDepartmentFields(context.Context, int64, map[string]any) error {
	return nil
}

func (r *fakeTaxonomyRepository) BuildDepartmentFullName(context.Context, int64) (string, error) {
	return r.fullName, nil
}

type fakeUsageStatsRepository struct {
	dimension   string
	granularity string
	start       time.Time
}

func (r *fakeUsageStatsRepository) GetStatsByModel(_ context.Context, startTime, _ time.Time) ([]model.UsageStatsRow, error) {
	r.dimension = "model"
	r.start = startTime
	return []model.UsageStatsRow{{Name: "qwen", TotalTokens: 10}}, nil
}

func (r *fakeUsageStatsRepository) GetStatsByUser(_ context.Context, startTime, _ time.Time) ([]model.UsageStatsRow, error) {
	r.dimension = "user"
	r.start = startTime
	return nil, nil
}

func (r *fakeUsageStatsRepository) GetStatsBySession(_ context.Context, startTime, _ time.Time) ([]model.UsageStatsRow, error) {
	r.dimension = "session"
	r.start = startTime
	return nil, nil
}

func (r *fakeUsageStatsRepository) GetTrend(_ context.Context, startTime, _ time.Time, granularity string) ([]model.UsageTrendRow, error) {
	r.start = startTime
	r.granularity = granularity
	return []model.UsageTrendRow{{Date: "2026-07-13", TotalTokens: 10}}, nil
}

type fakeUsageStatsAuthorizer struct {
	permission string
}

func (a *fakeUsageStatsAuthorizer) AuthorizePermission(_ context.Context, _ uint64, permission string) error {
	a.permission = permission
	return nil
}
