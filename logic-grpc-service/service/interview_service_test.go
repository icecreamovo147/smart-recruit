package service

import (
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"logic-grpc-service/model"
	"logic-grpc-service/pkg/authz"
	"logic-grpc-service/pkg/errs"
	"logic-grpc-service/pkg/metadata"
	"logic-grpc-service/recruitment/pb"
	"logic-grpc-service/repository"
)

// setupInterviewServiceTestDB creates an in-memory SQLite DB with all tables
// needed for interview service tests.
func setupInterviewServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}

	tables := []interface{}{
		&model.User{},
		&model.Role{},
		&model.UserRole{},
		&model.Permission{},
		&model.RolePermission{},
		&model.UserDataScope{},
		&model.Job{},
		&model.Application{},
		&model.CandidateProfile{},
		&model.Resume{},
		&model.InterviewSchedule{},
		&model.InterviewFeedback{},
		&model.AuthorizationAuditLog{},
	}
	for _, table := range tables {
		if err := db.AutoMigrate(table); err != nil {
			t.Fatalf("migrate %T: %v", table, err)
		}
	}
	return db
}

// interviewTestSeed holds references to all entities created during seed setup.
type interviewTestSeed struct {
	InterviewerUser     *model.User
	OtherInterviewer    *model.User
	ScopeUser           *model.User // user with recruiting_all scope
	CandidateUser       *model.User
	UnprivilegedUser    *model.User // user with interview.read but no scope
	Job                 *model.Job
	Application         *model.Application
	AssignedInterview   *model.InterviewSchedule
	UnassignedInterview *model.InterviewSchedule
}

// seedInterviewTestData populates the DB with a complete test scenario:
//   - CandidateUser has an application for Job
//   - InterviewerUser is the assigned interviewer for AssignedInterview
//   - OtherInterviewer is assigned to UnassignedInterview (not InterviewerUser)
//   - ScopeUser has interview.read + recruiting_all scope
//   - UnprivilegedUser has interview.read permission but no data scope
func seedInterviewTestData(t *testing.T, db *gorm.DB) *interviewTestSeed {
	t.Helper()
	now := time.Now()
	adminID := uint64(1)

	// ── Users ──────────────────────────────────────────────────────────

	candidate := &model.User{Username: "candidate", Password: "hash", AccountType: "candidate", Status: "active"}
	interviewer := &model.User{Username: "interviewer", Password: "hash", AccountType: "staff", Status: "active"}
	otherInterviewer := &model.User{Username: "other_interviewer", Password: "hash", AccountType: "staff", Status: "active"}
	scopeUser := &model.User{Username: "scope_user", Password: "hash", AccountType: "staff", Status: "active"}
	unprivileged := &model.User{Username: "unprivileged", Password: "hash", AccountType: "staff", Status: "active"}

	if err := db.Create([]*model.User{candidate, interviewer, otherInterviewer, scopeUser, unprivileged}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}

	// ── Roles ──────────────────────────────────────────────────────────

	interviewerRole := model.Role{RoleKey: "interviewer", Name: "Interviewer", IsSystem: 1}
	hrRole := model.Role{RoleKey: "recruiter", Name: "Recruiter", IsSystem: 1}
	if err := db.Create([]*model.Role{&interviewerRole, &hrRole}).Error; err != nil {
		t.Fatalf("create roles: %v", err)
	}

	// ── Permissions ────────────────────────────────────────────────────

	perms := []model.Permission{
		{PermissionKey: authz.PermInterviewRead, Resource: "interview", Action: "read", Description: "Read interviews"},
		{PermissionKey: authz.PermInterviewFeedback, Resource: "interview", Action: "feedback_submit", Description: "Submit feedback"},
		{PermissionKey: authz.PermInterviewSchedule, Resource: "interview", Action: "schedule", Description: "Schedule interviews"},
	}
	for i := range perms {
		if err := db.Create(&perms[i]).Error; err != nil {
			t.Fatalf("create perm %s: %v", perms[i].PermissionKey, err)
		}
	}

	// Assign all permissions to interviewer role
	for _, p := range perms {
		rp := model.RolePermission{RoleID: interviewerRole.ID, PermissionID: p.ID}
		if err := db.Create(&rp).Error; err != nil {
			t.Fatalf("assign perm %s: %v", p.PermissionKey, err)
		}
	}

	// Assign interview.read and interview.feedback.submit to hr role (no schedule).
	// Reuse the same permission records created above.
	var readPermID uint64
	var fbPermID uint64
	for _, p := range perms {
		if p.PermissionKey == authz.PermInterviewRead {
			readPermID = p.ID
		}
		if p.PermissionKey == authz.PermInterviewFeedback {
			fbPermID = p.ID
		}
	}

	if err := db.Create([]*model.RolePermission{
		{RoleID: hrRole.ID, PermissionID: readPermID},
		{RoleID: hrRole.ID, PermissionID: fbPermID},
	}).Error; err != nil {
		t.Fatalf("assign hr role perms: %v", err)
	}

	// ── Role assignments ───────────────────────────────────────────────

	// Interviewer gets interviewer role
	ur1 := model.UserRole{UserID: uint64(interviewer.ID), RoleID: interviewerRole.ID, AssignedBy: &adminID}
	// OtherInterviewer gets interviewer role
	ur2 := model.UserRole{UserID: uint64(otherInterviewer.ID), RoleID: interviewerRole.ID, AssignedBy: &adminID}
	// ScopeUser gets hr role (which has interview.read)
	ur3 := model.UserRole{UserID: uint64(scopeUser.ID), RoleID: hrRole.ID, AssignedBy: &adminID}
	// UnprivilegedUser gets hr role (which has interview.read)
	ur4 := model.UserRole{UserID: uint64(unprivileged.ID), RoleID: hrRole.ID, AssignedBy: &adminID}

	if err := db.Create([]*model.UserRole{&ur1, &ur2, &ur3, &ur4}).Error; err != nil {
		t.Fatalf("assign roles: %v", err)
	}

	// ── Data scopes ────────────────────────────────────────────────────

	// ScopeUser gets recruiting_all scope
	ds1 := model.UserDataScope{
		UserID:     uint64(scopeUser.ID),
		ScopeKey:   authz.ScopeRecruitingAll,
		AssignedBy: &adminID,
		AssignedAt: now,
	}
	if err := db.Create(&ds1).Error; err != nil {
		t.Fatalf("assign recruiting_all scope: %v", err)
	}

	// ── Job ─────────────────────────────────────────────────────────────

	job := &model.Job{
		HrID:   interviewer.ID,
		Title:  "Software Engineer",
		Status: 1,
	}
	if err := db.Create(job).Error; err != nil {
		t.Fatalf("create job: %v", err)
	}

	// ── Application ─────────────────────────────────────────────────────

	app := &model.Application{
		UserID:    candidate.ID,
		JobID:     job.ID,
		ResumeID:  1,
		Status:    0,
		IsCurrent: 1,
		StatusKey: "new",
	}
	if err := db.Create(app).Error; err != nil {
		t.Fatalf("create application: %v", err)
	}

	// ── Interview schedules ────────────────────────────────────────────

	assignedInterview := &model.InterviewSchedule{
		ApplicationID:   app.ID,
		InterviewerID:   interviewer.ID,
		RoundNo:         1,
		Title:           "Technical Round",
		Mode:            "video",
		DurationMinutes: 60,
		Status:          "scheduled",
		CreatedBy:       &interviewer.ID,
		ScheduledAt:     &now,
	}
	if err := db.Create(assignedInterview).Error; err != nil {
		t.Fatalf("create assigned interview: %v", err)
	}

	unassignedInterview := &model.InterviewSchedule{
		ApplicationID:   app.ID,
		InterviewerID:   otherInterviewer.ID,
		RoundNo:         2,
		Title:           "HR Round",
		Mode:            "video",
		DurationMinutes: 45,
		Status:          "scheduled",
		CreatedBy:       &interviewer.ID,
		ScheduledAt:     &now,
	}
	if err := db.Create(unassignedInterview).Error; err != nil {
		t.Fatalf("create unassigned interview: %v", err)
	}

	return &interviewTestSeed{
		InterviewerUser:     interviewer,
		OtherInterviewer:    otherInterviewer,
		ScopeUser:           scopeUser,
		CandidateUser:       candidate,
		UnprivilegedUser:    unprivileged,
		Job:                 job,
		Application:         app,
		AssignedInterview:   assignedInterview,
		UnassignedInterview: unassignedInterview,
	}
}

// newInterviewServiceForTest creates an InterviewService wired with real repos
// that enforce permissions (authzRepo + scopeEval are non-nil).
func newInterviewServiceForTest(t *testing.T, db *gorm.DB) *InterviewService {
	t.Helper()
	authzRepo := repository.NewAuthzRepo(db)
	interviewRepo := repository.NewInterviewRepo(db)
	userRepo := repository.NewUserRepo(db)
	appRepo := repository.NewApplicationRepo(db)
	jobRepo := repository.NewJobRepo(db)
	scopeEval := &scopeEvaluator{authzRepo: authzRepo}
	serviceAuth := NewServiceAuthorizer(authzRepo, scopeEval)

	// OutboxPublisher with nil repo — methods that don't call outbox work fine.
	outboxPublisher := &OutboxPublisher{}

	return NewInterviewService(authzRepo, interviewRepo, userRepo, appRepo, jobRepo, nil, outboxPublisher, scopeEval, serviceAuth)
}

// ── GetInterview ───────────────────────────────────────────────────────────

func TestInterviewService_GetInterview_InterviewerCanReadAssigned(t *testing.T) {
	db := setupInterviewServiceTestDB(t)
	seed := seedInterviewTestData(t, db)
	svc := newInterviewServiceForTest(t, db)

	ctx := metadata.WithAuthActor(context.Background(), seed.InterviewerUser.ID, "staff")
	resp, err := svc.GetInterview(ctx, &pb.GetInterviewRequest{
		UserId:      seed.InterviewerUser.ID,
		InterviewId: seed.AssignedInterview.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Code != errs.OK {
		t.Fatalf("expected OK, got code=%d msg=%s", resp.Code, resp.Msg)
	}
	if resp.Interview == nil {
		t.Fatal("expected interview in response")
	}
	if resp.Interview.InterviewId != seed.AssignedInterview.ID {
		t.Errorf("InterviewId = %d, want %d", resp.Interview.InterviewId, seed.AssignedInterview.ID)
	}
}

func TestInterviewService_GetInterview_InterviewerDeniedForUnassigned(t *testing.T) {
	db := setupInterviewServiceTestDB(t)
	seed := seedInterviewTestData(t, db)
	svc := newInterviewServiceForTest(t, db)

	// InterviewerUser tries to read an interview assigned to OtherInterviewer
	ctx := metadata.WithAuthActor(context.Background(), seed.InterviewerUser.ID, "staff")
	resp, err := svc.GetInterview(ctx, &pb.GetInterviewRequest{
		UserId:      seed.InterviewerUser.ID,
		InterviewId: seed.UnassignedInterview.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Code != errs.ErrForbidden {
		t.Fatalf("expected Forbidden for unassigned interview, got code=%d msg=%s", resp.Code, resp.Msg)
	}
}

func TestInterviewService_GetInterview_CandidateCanReadOwnInterview(t *testing.T) {
	db := setupInterviewServiceTestDB(t)
	seed := seedInterviewTestData(t, db)
	svc := newInterviewServiceForTest(t, db)

	// Candidate reads their own interview
	ctx := metadata.WithAuthActor(context.Background(), seed.CandidateUser.ID, "candidate")
	resp, err := svc.GetInterview(ctx, &pb.GetInterviewRequest{
		UserId:      seed.CandidateUser.ID,
		InterviewId: seed.AssignedInterview.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Code != errs.OK {
		t.Fatalf("expected OK, got code=%d msg=%s", resp.Code, resp.Msg)
	}
	if resp.Interview == nil {
		t.Fatal("expected interview in response")
	}
	if resp.Interview.InterviewId != seed.AssignedInterview.ID {
		t.Errorf("InterviewId = %d, want %d", resp.Interview.InterviewId, seed.AssignedInterview.ID)
	}
}

func TestInterviewService_GetInterview_ScopeUserCanReadAnyInterview(t *testing.T) {
	db := setupInterviewServiceTestDB(t)
	seed := seedInterviewTestData(t, db)
	svc := newInterviewServiceForTest(t, db)

	// User with recruiting_all scope can read any interview
	ctx := metadata.WithAuthActor(context.Background(), seed.ScopeUser.ID, "staff")
	resp, err := svc.GetInterview(ctx, &pb.GetInterviewRequest{
		UserId:      seed.ScopeUser.ID,
		InterviewId: seed.UnassignedInterview.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Code != errs.OK {
		t.Fatalf("expected OK for scope user, got code=%d msg=%s", resp.Code, resp.Msg)
	}
	if resp.Interview == nil {
		t.Fatal("expected interview in response")
	}
}

func TestInterviewService_GetInterview_UnprivilegedUserDenied(t *testing.T) {
	db := setupInterviewServiceTestDB(t)
	seed := seedInterviewTestData(t, db)
	svc := newInterviewServiceForTest(t, db)

	// User with interview.read but no scope keys should be denied for unassigned interview
	ctx := metadata.WithAuthActor(context.Background(), seed.UnprivilegedUser.ID, "staff")
	resp, err := svc.GetInterview(ctx, &pb.GetInterviewRequest{
		UserId:      seed.UnprivilegedUser.ID,
		InterviewId: seed.UnassignedInterview.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Code != errs.ErrForbidden {
		t.Fatalf("expected Forbidden for unprivileged user, got code=%d msg=%s", resp.Code, resp.Msg)
	}
}

// ── ListMyInterviews ───────────────────────────────────────────────────────

func TestInterviewService_ListMyInterviews_ReturnsAssignedOnly(t *testing.T) {
	db := setupInterviewServiceTestDB(t)
	seed := seedInterviewTestData(t, db)
	svc := newInterviewServiceForTest(t, db)

	ctx := metadata.WithAuthActor(context.Background(), seed.InterviewerUser.ID, "staff")
	resp, err := svc.ListMyInterviews(ctx, &pb.ListMyInterviewsRequest{
		InterviewerId: seed.InterviewerUser.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Code != errs.OK {
		t.Fatalf("expected OK, got code=%d msg=%s", resp.Code, resp.Msg)
	}
	if len(resp.List) != 1 {
		t.Fatalf("expected 1 interview, got %d", len(resp.List))
	}
	if resp.List[0].InterviewId != seed.AssignedInterview.ID {
		t.Errorf("InterviewId = %d, want %d", resp.List[0].InterviewId, seed.AssignedInterview.ID)
	}
}

// ── SubmitFeedback ─────────────────────────────────────────────────────────

func TestInterviewService_SubmitFeedback_RequiresAssignment(t *testing.T) {
	db := setupInterviewServiceTestDB(t)
	seed := seedInterviewTestData(t, db)
	svc := newInterviewServiceForTest(t, db)

	// InterviewerUser tries to submit feedback for an interview they are NOT assigned to
	ctx := metadata.WithAuthActor(context.Background(), seed.InterviewerUser.ID, "staff")
	resp, err := svc.SubmitFeedback(ctx, &pb.SubmitFeedbackRequest{
		InterviewerId:  seed.InterviewerUser.ID,
		InterviewId:    seed.UnassignedInterview.ID,
		ApplicationId:  seed.Application.ID,
		Recommendation: "positive",
		Score:          8,
		Comments:       "Good candidate",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Code != errs.ErrForbidden {
		t.Fatalf("expected Forbidden for unassigned feedback, got code=%d msg=%s", resp.Code, resp.Msg)
	}
}

func TestInterviewService_SubmitFeedback_Success(t *testing.T) {
	db := setupInterviewServiceTestDB(t)
	seed := seedInterviewTestData(t, db)
	svc := newInterviewServiceForTest(t, db)

	ctx := metadata.WithAuthActor(context.Background(), seed.InterviewerUser.ID, "staff")
	resp, err := svc.SubmitFeedback(ctx, &pb.SubmitFeedbackRequest{
		InterviewerId:  seed.InterviewerUser.ID,
		InterviewId:    seed.AssignedInterview.ID,
		ApplicationId:  seed.Application.ID,
		Recommendation: "positive",
		Score:          8,
		Comments:       "Strong technical skills",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Code != errs.OK {
		t.Fatalf("expected OK, got code=%d msg=%s", resp.Code, resp.Msg)
	}

	// Verify the interview status changed to completed
	var interview model.InterviewSchedule
	if err := db.Where("id = ?", seed.AssignedInterview.ID).First(&interview).Error; err != nil {
		t.Fatalf("fetch interview: %v", err)
	}
	if interview.Status != "completed" {
		t.Errorf("interview status = %q, want 'completed'", interview.Status)
	}
}

func TestInterviewService_SubmitFeedback_Immutability(t *testing.T) {
	db := setupInterviewServiceTestDB(t)
	seed := seedInterviewTestData(t, db)
	svc := newInterviewServiceForTest(t, db)

	ctx := metadata.WithAuthActor(context.Background(), seed.InterviewerUser.ID, "staff")

	// First submission succeeds
	resp, err := svc.SubmitFeedback(ctx, &pb.SubmitFeedbackRequest{
		InterviewerId:  seed.InterviewerUser.ID,
		InterviewId:    seed.AssignedInterview.ID,
		ApplicationId:  seed.Application.ID,
		Recommendation: "positive",
		Score:          8,
	})
	if err != nil {
		t.Fatalf("first submit: %v", err)
	}
	if resp.Code != errs.OK {
		t.Fatalf("expected OK, got code=%d msg=%s", resp.Code, resp.Msg)
	}

	// Second submission should be rejected as duplicate
	resp, err = svc.SubmitFeedback(ctx, &pb.SubmitFeedbackRequest{
		InterviewerId:  seed.InterviewerUser.ID,
		InterviewId:    seed.AssignedInterview.ID,
		ApplicationId:  seed.Application.ID,
		Recommendation: "negative",
		Score:          3,
	})
	if err != nil {
		t.Fatalf("second submit: %v", err)
	}
	if resp.Code != errs.ErrConflict {
		t.Fatalf("expected Conflict for duplicate feedback, got code=%d msg=%s", resp.Code, resp.Msg)
	}
}

func TestInterviewService_SubmitFeedback_DoesNotAlterApplicationStatus(t *testing.T) {
	db := setupInterviewServiceTestDB(t)
	seed := seedInterviewTestData(t, db)
	svc := newInterviewServiceForTest(t, db)

	// Capture original app status
	originalStatusKey := seed.Application.StatusKey

	ctx := metadata.WithAuthActor(context.Background(), seed.InterviewerUser.ID, "staff")
	resp, err := svc.SubmitFeedback(ctx, &pb.SubmitFeedbackRequest{
		InterviewerId:  seed.InterviewerUser.ID,
		InterviewId:    seed.AssignedInterview.ID,
		ApplicationId:  seed.Application.ID,
		Recommendation: "positive",
		Score:          8,
		Comments:       "Good",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Code != errs.OK {
		t.Fatalf("expected OK, got code=%d msg=%s", resp.Code, resp.Msg)
	}

	// Application status should remain unchanged
	var app model.Application
	if err := db.Where("id = ?", seed.Application.ID).First(&app).Error; err != nil {
		t.Fatalf("fetch application: %v", err)
	}
	if app.StatusKey != originalStatusKey {
		t.Errorf("application status_key changed from %q to %q", originalStatusKey, app.StatusKey)
	}
}

// ── GetInterview: own_jobs scope enforcement ─────────────────────────────────

func TestInterviewService_GetInterview_OwnJobsScopeMustOwnJob(t *testing.T) {
	db := setupInterviewServiceTestDB(t)
	seed := seedInterviewTestData(t, db)
	svc := newInterviewServiceForTest(t, db)

	// Create a user with interview.read + own_jobs scope, but assign them
	// a DIFFERENT job (not the one the application belongs to).
	ownJobsUser := &model.User{Username: "own_jobs_other", Password: "hash", AccountType: "staff", Status: "active"}
	if err := db.Create(ownJobsUser).Error; err != nil {
		t.Fatalf("create ownJobsUser: %v", err)
	}

	// Give them the interviewer role (has interview.read)
	adminID := uint64(1)
	ur := model.UserRole{UserID: uint64(ownJobsUser.ID), RoleID: 2, AssignedBy: &adminID}
	if err := db.Create(&ur).Error; err != nil {
		t.Fatalf("assign role: %v", err)
	}

	// Give them own_jobs scope (they have the scope key but don't own THIS job)
	ds := model.UserDataScope{
		UserID:     uint64(ownJobsUser.ID),
		ScopeKey:   authz.ScopeOwnJobs,
		AssignedBy: &adminID,
		AssignedAt: time.Now(),
	}
	if err := db.Create(&ds).Error; err != nil {
		t.Fatalf("assign own_jobs scope: %v", err)
	}

	// Create a DIFFERENT job owned by this user (not the interview's job)
	otherJob := &model.Job{HrID: ownJobsUser.ID, Title: "Other Job", Status: 1}
	if err := db.Create(otherJob).Error; err != nil {
		t.Fatalf("create other job: %v", err)
	}

	// Application's job (seed.Job) has HrID = seed.InterviewerUser.ID, NOT ownJobsUser.ID.
	// So BelongsToHR should return false, and access should be denied.
	ctx := metadata.WithAuthActor(context.Background(), ownJobsUser.ID, "staff")
	resp, err := svc.GetInterview(ctx, &pb.GetInterviewRequest{
		UserId:      ownJobsUser.ID,
		InterviewId: seed.AssignedInterview.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Code != errs.ErrForbidden {
		t.Fatalf("expected Forbidden for own_jobs user who does NOT own this job, got code=%d msg=%s", resp.Code, resp.Msg)
	}

	_ = otherJob // silence unused warning
}

// ── GetInterview: candidate InternalNote filtering ────────────────────────────

func TestInterviewService_GetInterview_CandidateInternalNoteFiltered(t *testing.T) {
	db := setupInterviewServiceTestDB(t)
	seed := seedInterviewTestData(t, db)

	// Set an internal note on the interview
	db.Model(&model.InterviewSchedule{}).
		Where("id = ?", seed.AssignedInterview.ID).
		Update("internal_note", "sensitive internal info")

	svc := newInterviewServiceForTest(t, db)

	ctx := metadata.WithAuthActor(context.Background(), seed.CandidateUser.ID, "candidate")
	resp, err := svc.GetInterview(ctx, &pb.GetInterviewRequest{
		UserId:      seed.CandidateUser.ID,
		InterviewId: seed.AssignedInterview.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Code != errs.OK {
		t.Fatalf("expected OK, got code=%d msg=%s", resp.Code, resp.Msg)
	}
	if resp.Interview == nil {
		t.Fatal("expected interview in response")
	}
	if resp.Interview.InternalNote != "" {
		t.Errorf("candidate GetInterview has internal_note=%q, expected empty (defense-in-depth)", resp.Interview.InternalNote)
	}
}

// ── SubmitFeedback: ApplicationId validation ──────────────────────────────────

func TestInterviewService_SubmitFeedback_ApplicationIdMismatch(t *testing.T) {
	db := setupInterviewServiceTestDB(t)
	seed := seedInterviewTestData(t, db)
	svc := newInterviewServiceForTest(t, db)

	// InterviewerUser submits feedback with wrong ApplicationId
	ctx := metadata.WithAuthActor(context.Background(), seed.InterviewerUser.ID, "staff")
	resp, err := svc.SubmitFeedback(ctx, &pb.SubmitFeedbackRequest{
		InterviewerId:  seed.InterviewerUser.ID,
		InterviewId:    seed.AssignedInterview.ID,
		ApplicationId:  9999, // wrong — does not match interview's ApplicationID
		Recommendation: "positive",
		Score:          8,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Code != errs.ErrBadRequest {
		t.Fatalf("expected BadRequest for ApplicationId mismatch, got code=%d msg=%s", resp.Code, resp.Msg)
	}
}

// ── ListCandidateInterviews ────────────────────────────────────────────────

func TestInterviewService_ListCandidateInterviews_FiltersInternalNote(t *testing.T) {
	db := setupInterviewServiceTestDB(t)
	seed := seedInterviewTestData(t, db)

	// Set an internal note on the interview
	db.Model(&model.InterviewSchedule{}).
		Where("id = ?", seed.AssignedInterview.ID).
		Update("internal_note", "sensitive internal info")

	svc := newInterviewServiceForTest(t, db)

	ctx := metadata.WithAuthActor(context.Background(), seed.CandidateUser.ID, "candidate")
	resp, err := svc.ListCandidateInterviews(ctx, &pb.ListCandidateInterviewsRequest{
		UserId: seed.CandidateUser.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Code != errs.OK {
		t.Fatalf("expected OK, got code=%d msg=%s", resp.Code, resp.Msg)
	}
	if len(resp.List) == 0 {
		t.Fatal("expected at least one interview")
	}

	for _, iv := range resp.List {
		if iv.InternalNote != "" {
			t.Errorf("candidate-facing interview %d has internal_note=%q, expected empty", iv.InterviewId, iv.InternalNote)
		}
	}
}
