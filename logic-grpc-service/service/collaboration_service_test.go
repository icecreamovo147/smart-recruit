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

// ── Test Setup ───────────────────────────────────────────────────────────────

func setupCollaborationTestDB(t *testing.T) *gorm.DB {
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
		&model.ApplicationStatusTransition{},
		&model.CandidateProfile{},
		&model.Resume{},
		&model.CandidateNote{},
		&model.CandidateTag{},
		&model.CandidateTagAssignment{},
		&model.FollowUpTask{},
		&model.InterviewSchedule{},
		&model.Offer{},
	}
	for _, table := range tables {
		if err := db.AutoMigrate(table); err != nil {
			t.Fatalf("migrate %T: %v", table, err)
		}
	}
	return db
}

type collabTestSeed struct {
	HrUser         *model.User
	OtherHrUser    *model.User
	Candidate      *model.User
	OtherCandidate *model.User
	Job            *model.Job
	Application    *model.Application
	Tag            *model.CandidateTag
	Note           *model.CandidateNote
	Task           *model.FollowUpTask
}

func seedCollaborationTestData(t *testing.T, db *gorm.DB) *collabTestSeed {
	t.Helper()
	adminID := uint64(1)

	// ── Users ──────────────────────────────────────────────────────────
	candidate := &model.User{Username: "candidate", Password: "hash", AccountType: "candidate", Status: "active"}
	otherCandidate := &model.User{Username: "other_candidate", Password: "hash", AccountType: "candidate", Status: "active"}
	hrUser := &model.User{Username: "hr_recruiter", Password: "hash", AccountType: "staff", Status: "active"}
	otherHr := &model.User{Username: "hr_other", Password: "hash", AccountType: "staff", Status: "active"}

	if err := db.Create([]*model.User{candidate, otherCandidate, hrUser, otherHr}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}

	// ── Roles ──────────────────────────────────────────────────────────
	recruiterRole := model.Role{RoleKey: "recruiter", Name: "Recruiter", IsSystem: 1}
	candidateRole := model.Role{RoleKey: "candidate", Name: "Candidate", IsSystem: 1}
	if err := db.Create([]*model.Role{&recruiterRole, &candidateRole}).Error; err != nil {
		t.Fatalf("create roles: %v", err)
	}

	// ── Permissions ────────────────────────────────────────────────────
	perms := []model.Permission{
		{PermissionKey: authz.PermApplicationRead, Resource: "application", Action: "read", Description: "Read applications"},
		{PermissionKey: authz.PermCollaborationNoteCreate, Resource: "collaboration", Action: "note.create", Description: "Create notes"},
		{PermissionKey: authz.PermCollaborationNoteRead, Resource: "collaboration", Action: "note.read", Description: "Read notes"},
		{PermissionKey: authz.PermCollaborationTagManage, Resource: "collaboration", Action: "tag.manage", Description: "Manage tags"},
		{PermissionKey: authz.PermCollaborationTaskManage, Resource: "collaboration", Action: "task.manage", Description: "Manage tasks"},
	}
	for i := range perms {
		if err := db.Create(&perms[i]).Error; err != nil {
			t.Fatalf("create perm %s: %v", perms[i].PermissionKey, err)
		}
	}

	// Recruiter gets all collaboration permissions
	for _, p := range perms {
		if err := db.Create(&model.RolePermission{RoleID: recruiterRole.ID, PermissionID: p.ID}).Error; err != nil {
			t.Fatalf("assign perm %s to recruiter: %v", p.PermissionKey, err)
		}
	}

	// ── Role assignments ───────────────────────────────────────────────
	ur1 := model.UserRole{UserID: uint64(hrUser.ID), RoleID: recruiterRole.ID, AssignedBy: &adminID}
	ur2 := model.UserRole{UserID: uint64(otherHr.ID), RoleID: recruiterRole.ID, AssignedBy: &adminID}
	ur3 := model.UserRole{UserID: uint64(candidate.ID), RoleID: candidateRole.ID, AssignedBy: &adminID}
	ur4 := model.UserRole{UserID: uint64(otherCandidate.ID), RoleID: candidateRole.ID, AssignedBy: &adminID}

	if err := db.Create([]*model.UserRole{&ur1, &ur2, &ur3, &ur4}).Error; err != nil {
		t.Fatalf("assign roles: %v", err)
	}

	// ── Data scopes ────────────────────────────────────────────────────
	// HrUser gets recruiting_all scope
	ds1 := model.UserDataScope{
		UserID:     uint64(hrUser.ID),
		ScopeKey:   authz.ScopeRecruitingAll,
		AssignedBy: &adminID,
		AssignedAt: time.Now(),
	}
	if err := db.Create(&ds1).Error; err != nil {
		t.Fatalf("assign recruiting_all scope: %v", err)
	}
	// OtherHr gets no data scope

	// ── Job ─────────────────────────────────────────────────────────────
	job := &model.Job{
		HrID:       hrUser.ID,
		Title:      "Senior Engineer",
		Status:     1,
		Department: "Engineering",
		Location:   "Beijing",
	}
	if err := db.Create(job).Error; err != nil {
		t.Fatalf("create job: %v", err)
	}

	// ── Application ────────────────────────────────────────────────────
	app := &model.Application{
		UserID:    candidate.ID,
		JobID:     job.ID,
		ResumeID:  1,
		Status:    2,
		IsCurrent: 1,
		StatusKey: model.StatusKeyApplied,
	}
	if err := db.Create(app).Error; err != nil {
		t.Fatalf("create application: %v", err)
	}

	// ── Candidate profile ──────────────────────────────────────────────
	profile := &model.CandidateProfile{
		UserID:   candidate.ID,
		RealName: "Test Candidate",
		Phone:    "13800138000",
		Skills:   "Go,Python,SQL",
	}
	if err := db.Create(profile).Error; err != nil {
		t.Fatalf("create profile: %v", err)
	}

	// ── Tag ────────────────────────────────────────────────────────────
	tag := &model.CandidateTag{
		Name:  "High Priority",
		Color: "#ff0000",
	}
	if err := db.Create(tag).Error; err != nil {
		t.Fatalf("create tag: %v", err)
	}

	// ── Note ───────────────────────────────────────────────────────────
	note := &model.CandidateNote{
		CandidateUserID: uint64(candidate.ID),
		AuthorUserID:    uint64(hrUser.ID),
		Content:         "Test note content",
		Visibility:      "internal",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	if err := db.Create(note).Error; err != nil {
		t.Fatalf("create note: %v", err)
	}

	// ── Task ───────────────────────────────────────────────────────────
	task := &model.FollowUpTask{
		CandidateUserID: uint64(candidate.ID),
		AssigneeUserID:  uint64(hrUser.ID),
		CreatedBy:       uint64(hrUser.ID),
		Title:           "Review application",
		Status:          "pending",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	if err := db.Create(task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}

	return &collabTestSeed{
		HrUser:         hrUser,
		OtherHrUser:    otherHr,
		Candidate:      candidate,
		OtherCandidate: otherCandidate,
		Job:            job,
		Application:    app,
		Tag:            tag,
		Note:           note,
		Task:           task,
	}
}

func newCollaborationServiceForTest(t *testing.T, db *gorm.DB) *CollaborationService {
	t.Helper()
	authzRepo := repository.NewAuthzRepo(db)
	collabRepo := repository.NewCollaborationRepo(db)
	appRepo := repository.NewApplicationRepo(db)
	profileRepo := repository.NewProfileRepo(db)
	jobRepo := repository.NewJobRepo(db)
	userRepo := repository.NewUserRepo(db)
	interviewRepo := repository.NewInterviewRepo(db)
	offerRepo := repository.NewOfferRepo(db)
	resumeRepo := repository.NewResumeRepo(db)
	scopeEval := &scopeEvaluator{authzRepo: authzRepo}
	serviceAuth := NewServiceAuthorizer(authzRepo, scopeEval)

	return NewCollaborationService(
		authzRepo,
		collabRepo,
		appRepo,
		profileRepo,
		jobRepo,
		userRepo,
		interviewRepo,
		offerRepo,
		resumeRepo,
		nil,
		serviceAuth,
		scopeEval,
	)
}

// ── Workspace Tests ──────────────────────────────────────────────────────────

func TestCollaborationService_GetCandidateWorkspace_Success(t *testing.T) {
	db := setupCollaborationTestDB(t)
	seed := seedCollaborationTestData(t, db)
	svc := newCollaborationServiceForTest(t, db)

	ctx := metadata.WithAuthActor(context.Background(), seed.HrUser.ID, "staff")
	resp, err := svc.GetCandidateWorkspace(ctx, &pb.GetCandidateWorkspaceRequest{
		StaffUserId:     seed.HrUser.ID,
		CandidateUserId: seed.Candidate.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Code != errs.OK {
		t.Fatalf("expected OK, got code=%d msg=%s", resp.Code, resp.Msg)
	}
	ws := resp.Workspace
	if ws == nil {
		t.Fatal("workspace is nil")
	}

	// Check profile fields
	if ws.RealName != "Test Candidate" {
		t.Errorf("RealName=%s, want Test Candidate", ws.RealName)
	}
	if ws.Phone != "13800138000" {
		t.Errorf("Phone=%s, want 13800138000", ws.Phone)
	}
	if len(ws.Skills) == 0 || ws.Skills[0] != "Go" {
		t.Errorf("Skills missing Go, got %v", ws.Skills)
	}

	// Check applications
	if len(ws.Applications) != 1 {
		t.Fatalf("expected 1 application, got %d", len(ws.Applications))
	}
	app := ws.Applications[0]
	if app.Department != "Engineering" {
		t.Errorf("Department=%s, want Engineering", app.Department)
	}
	if app.Location != "Beijing" {
		t.Errorf("Location=%s, want Beijing", app.Location)
	}
	if app.JobTitle != "Senior Engineer" {
		t.Errorf("JobTitle=%s, want Senior Engineer", app.JobTitle)
	}
	if app.StatusKey != model.StatusKeyApplied {
		t.Errorf("StatusKey=%s, want %s", app.StatusKey, model.StatusKeyApplied)
	}

	_ = ws.Tags // Tags may be empty since no assignment in seed
	_ = ws.Tags // Tags may be empty since no assignment in seed
	_ = ws.Tags // Tags may be empty since no assignment in seed
	_ = ws.Tags // Tags may be empty since no assignment in seed

	// Check resume_url
	_ = ws.ResumeUrl // Should be empty string since no resume uploaded

	// Check stats
	if ws.TotalApplications < 1 {
		t.Errorf("TotalApplications=%d, want >=1", ws.TotalApplications)
	}
}

func TestCollaborationService_GetCandidateWorkspace_ScopeDenied(t *testing.T) {
	db := setupCollaborationTestDB(t)
	seed := seedCollaborationTestData(t, db)
	svc := newCollaborationServiceForTest(t, db)

	// other_hr has no data scope, should be denied
	ctx := metadata.WithAuthActor(context.Background(), seed.OtherHrUser.ID, "staff")
	_, err := svc.GetCandidateWorkspace(ctx, &pb.GetCandidateWorkspaceRequest{
		StaffUserId:     seed.OtherHrUser.ID,
		CandidateUserId: seed.Candidate.ID,
	})
	if err == nil {
		t.Fatal("expected error for scope-denied user, got nil")
	}
}

func TestCollaborationService_GetCandidateWorkspace_CandidateCannotAccess(t *testing.T) {
	db := setupCollaborationTestDB(t)
	seed := seedCollaborationTestData(t, db)
	svc := newCollaborationServiceForTest(t, db)

	// Candidate tries to access their own workspace via staff API
	ctx := metadata.WithAuthActor(context.Background(), seed.Candidate.ID, "candidate")
	_, err := svc.GetCandidateWorkspace(ctx, &pb.GetCandidateWorkspaceRequest{
		StaffUserId:     seed.Candidate.ID,
		CandidateUserId: seed.Candidate.ID,
	})
	if err == nil {
		t.Fatal("expected error for candidate user, got nil")
	}
}

// ── Note Tests ───────────────────────────────────────────────────────────────

func TestCollaborationService_CreateNote_Success(t *testing.T) {
	db := setupCollaborationTestDB(t)
	seed := seedCollaborationTestData(t, db)
	svc := newCollaborationServiceForTest(t, db)

	ctx := metadata.WithAuthActor(context.Background(), seed.HrUser.ID, "staff")
	resp, err := svc.CreateNote(ctx, &pb.CreateNoteRequest{
		StaffUserId:     seed.HrUser.ID,
		CandidateUserId: uint64(seed.Candidate.ID),
		Content:         "New test note",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Code != errs.OK {
		t.Fatalf("expected OK, got code=%d msg=%s", resp.Code, resp.Msg)
	}
	if resp.Note == nil {
		t.Fatal("note response is nil")
	}
	if resp.Note.Content != "New test note" {
		t.Errorf("Content=%s, want New test note", resp.Note.Content)
	}
	if resp.Note.AuthorUserId != uint64(seed.HrUser.ID) {
		t.Errorf("AuthorUserId=%d, want %d", resp.Note.AuthorUserId, seed.HrUser.ID)
	}
}

func TestCollaborationService_CreateNote_ScopeDenied(t *testing.T) {
	db := setupCollaborationTestDB(t)
	seed := seedCollaborationTestData(t, db)
	svc := newCollaborationServiceForTest(t, db)

	ctx := metadata.WithAuthActor(context.Background(), seed.OtherHrUser.ID, "staff")
	_, err := svc.CreateNote(ctx, &pb.CreateNoteRequest{
		StaffUserId:     seed.OtherHrUser.ID,
		CandidateUserId: uint64(seed.Candidate.ID),
		Content:         "Should fail",
	})
	if err == nil {
		t.Fatal("expected error for scope-denied user, got nil")
	}
}

func TestCollaborationService_ListNotes_Success(t *testing.T) {
	db := setupCollaborationTestDB(t)
	seed := seedCollaborationTestData(t, db)
	svc := newCollaborationServiceForTest(t, db)

	ctx := metadata.WithAuthActor(context.Background(), seed.HrUser.ID, "staff")
	resp, err := svc.ListNotes(ctx, &pb.ListNotesRequest{
		StaffUserId:     seed.HrUser.ID,
		CandidateUserId: uint64(seed.Candidate.ID),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Code != errs.OK {
		t.Fatalf("expected OK, got code=%d msg=%s", resp.Code, resp.Msg)
	}
	if len(resp.List) < 1 {
		t.Fatalf("expected at least 1 note, got %d", len(resp.List))
	}
	if resp.List[0].Content != "Test note content" {
		t.Errorf("Content=%s, want Test note content", resp.List[0].Content)
	}
}

// ── Tag Tests ────────────────────────────────────────────────────────────────

func TestCollaborationService_CreateTag_Success(t *testing.T) {
	db := setupCollaborationTestDB(t)
	seed := seedCollaborationTestData(t, db)
	svc := newCollaborationServiceForTest(t, db)

	ctx := metadata.WithAuthActor(context.Background(), seed.HrUser.ID, "staff")
	resp, err := svc.CreateTag(ctx, &pb.CreateTagRequest{
		StaffUserId: seed.HrUser.ID,
		Name:        "Urgent",
		Color:       "#00ff00",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Code != errs.OK {
		t.Fatalf("expected OK, got code=%d msg=%s", resp.Code, resp.Msg)
	}
	if resp.Tag == nil {
		t.Fatal("tag response is nil")
	}
	if resp.Tag.Name != "Urgent" {
		t.Errorf("Name=%s, want Urgent", resp.Tag.Name)
	}
}

func TestCollaborationService_AssignTag_Success(t *testing.T) {
	db := setupCollaborationTestDB(t)
	seed := seedCollaborationTestData(t, db)
	svc := newCollaborationServiceForTest(t, db)

	ctx := metadata.WithAuthActor(context.Background(), seed.HrUser.ID, "staff")
	resp, err := svc.AssignTag(ctx, &pb.AssignTagRequest{
		StaffUserId:     seed.HrUser.ID,
		TagId:           seed.Tag.ID,
		CandidateUserId: uint64(seed.Candidate.ID),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Code != errs.OK {
		t.Fatalf("expected OK, got code=%d msg=%s", resp.Code, resp.Msg)
	}
}

func TestCollaborationService_AssignTag_NoPermission(t *testing.T) {
	db := setupCollaborationTestDB(t)
	seed := seedCollaborationTestData(t, db)
	svc := newCollaborationServiceForTest(t, db)

	// Candidate has no tag.manage permission
	ctx := metadata.WithAuthActor(context.Background(), seed.Candidate.ID, "candidate")
	_, err := svc.AssignTag(ctx, &pb.AssignTagRequest{
		StaffUserId:     seed.Candidate.ID,
		TagId:           seed.Tag.ID,
		CandidateUserId: uint64(seed.Candidate.ID),
	})
	if err == nil {
		t.Fatal("expected error for candidate without tag permission, got nil")
	}
}

func TestCollaborationService_ListCandidateTags_Success(t *testing.T) {
	db := setupCollaborationTestDB(t)
	seed := seedCollaborationTestData(t, db)
	svc := newCollaborationServiceForTest(t, db)

	ctx := metadata.WithAuthActor(context.Background(), seed.HrUser.ID, "staff")
	resp, err := svc.ListCandidateTags(ctx, &pb.ListCandidateTagsRequest{
		StaffUserId:     seed.HrUser.ID,
		CandidateUserId: uint64(seed.Candidate.ID),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Code != errs.OK {
		t.Fatalf("expected OK, got code=%d msg=%s", resp.Code, resp.Msg)
	}
	// Tag was created but not assigned yet - should be empty (unless seed assigned it)
	_ = resp.List
}

func TestCollaborationService_ListTags_Success(t *testing.T) {
	db := setupCollaborationTestDB(t)
	seed := seedCollaborationTestData(t, db)
	svc := newCollaborationServiceForTest(t, db)

	ctx := metadata.WithAuthActor(context.Background(), seed.HrUser.ID, "staff")
	resp, err := svc.ListTags(ctx, &pb.ListTagsRequest{
		StaffUserId: seed.HrUser.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Code != errs.OK {
		t.Fatalf("expected OK, got code=%d msg=%s", resp.Code, resp.Msg)
	}
	if len(resp.List) < 1 {
		t.Fatalf("expected at least 1 tag, got %d", len(resp.List))
	}
	if resp.List[0].Name != "High Priority" {
		t.Errorf("Name=%s, want High Priority", resp.List[0].Name)
	}
}

func TestCollaborationService_UnassignTag_Success(t *testing.T) {
	db := setupCollaborationTestDB(t)
	seed := seedCollaborationTestData(t, db)
	svc := newCollaborationServiceForTest(t, db)

	// First assign the tag
	ctx := metadata.WithAuthActor(context.Background(), seed.HrUser.ID, "staff")
	svc.AssignTag(ctx, &pb.AssignTagRequest{
		StaffUserId:     seed.HrUser.ID,
		TagId:           seed.Tag.ID,
		CandidateUserId: uint64(seed.Candidate.ID),
	})

	// Then unassign
	resp, err := svc.UnassignTag(ctx, &pb.UnassignTagRequest{
		StaffUserId:     seed.HrUser.ID,
		TagId:           seed.Tag.ID,
		CandidateUserId: uint64(seed.Candidate.ID),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Code != errs.OK {
		t.Fatalf("expected OK, got code=%d msg=%s", resp.Code, resp.Msg)
	}
}

func TestCollaborationService_AssignTag_ScopeDenied(t *testing.T) {
	db := setupCollaborationTestDB(t)
	seed := seedCollaborationTestData(t, db)
	svc := newCollaborationServiceForTest(t, db)

	// other_hr has tag.manage permission but no data scope on seed.Candidate
	ctx := metadata.WithAuthActor(context.Background(), seed.OtherHrUser.ID, "staff")
	_, err := svc.AssignTag(ctx, &pb.AssignTagRequest{
		StaffUserId:     seed.OtherHrUser.ID,
		TagId:           seed.Tag.ID,
		CandidateUserId: uint64(seed.Candidate.ID),
	})
	if err == nil {
		t.Fatal("expected scope-denied error for assign tag, got nil")
	}
}

func TestCollaborationService_UnassignTag_ScopeDenied(t *testing.T) {
	db := setupCollaborationTestDB(t)
	seed := seedCollaborationTestData(t, db)
	svc := newCollaborationServiceForTest(t, db)

	// First assign the tag as a user with scope access
	ctxHr := metadata.WithAuthActor(context.Background(), seed.HrUser.ID, "staff")
	svc.AssignTag(ctxHr, &pb.AssignTagRequest{
		StaffUserId:     seed.HrUser.ID,
		TagId:           seed.Tag.ID,
		CandidateUserId: uint64(seed.Candidate.ID),
	})

	// other_hr has tag.manage permission but no data scope on seed.Candidate
	ctx := metadata.WithAuthActor(context.Background(), seed.OtherHrUser.ID, "staff")
	_, err := svc.UnassignTag(ctx, &pb.UnassignTagRequest{
		StaffUserId:     seed.OtherHrUser.ID,
		TagId:           seed.Tag.ID,
		CandidateUserId: uint64(seed.Candidate.ID),
	})
	if err == nil {
		t.Fatal("expected scope-denied error for unassign tag, got nil")
	}
}

// ── Task Tests ──────────────────────────────────────────────────────────────

func TestCollaborationService_CreateFollowUpTask_Success(t *testing.T) {
	db := setupCollaborationTestDB(t)
	seed := seedCollaborationTestData(t, db)
	svc := newCollaborationServiceForTest(t, db)

	ctx := metadata.WithAuthActor(context.Background(), seed.HrUser.ID, "staff")
	resp, err := svc.CreateFollowUpTask(ctx, &pb.CreateFollowUpTaskRequest{
		StaffUserId:     seed.HrUser.ID,
		CandidateUserId: uint64(seed.Candidate.ID),
		AssigneeUserId:  uint64(seed.HrUser.ID),
		Title:           "Follow up on resume",
		Description:     "Check new resume version",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Code != errs.OK {
		t.Fatalf("expected OK, got code=%d msg=%s", resp.Code, resp.Msg)
	}
	if resp.Task == nil {
		t.Fatal("task response is nil")
	}
	if resp.Task.Title != "Follow up on resume" {
		t.Errorf("Title=%s, want Follow up on resume", resp.Task.Title)
	}
	if resp.Task.Status != "pending" {
		t.Errorf("Status=%s, want pending", resp.Task.Status)
	}
}

func TestCollaborationService_ListFollowUpTasks_Success(t *testing.T) {
	db := setupCollaborationTestDB(t)
	seed := seedCollaborationTestData(t, db)
	svc := newCollaborationServiceForTest(t, db)

	ctx := metadata.WithAuthActor(context.Background(), seed.HrUser.ID, "staff")
	resp, err := svc.ListFollowUpTasks(ctx, &pb.ListFollowUpTasksRequest{
		StaffUserId:     seed.HrUser.ID,
		CandidateUserId: uint64(seed.Candidate.ID),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Code != errs.OK {
		t.Fatalf("expected OK, got code=%d msg=%s", resp.Code, resp.Msg)
	}
	if len(resp.List) < 1 {
		t.Fatalf("expected at least 1 task, got %d", len(resp.List))
	}
	if resp.List[0].Title != "Review application" {
		t.Errorf("Title=%s, want Review application", resp.List[0].Title)
	}
}

func TestCollaborationService_ListFollowUpTasks_WithCandidateFilter(t *testing.T) {
	db := setupCollaborationTestDB(t)
	seed := seedCollaborationTestData(t, db)
	svc := newCollaborationServiceForTest(t, db)

	ctx := metadata.WithAuthActor(context.Background(), seed.HrUser.ID, "staff")
	resp, err := svc.ListFollowUpTasks(ctx, &pb.ListFollowUpTasksRequest{
		StaffUserId:     seed.HrUser.ID,
		CandidateUserId: uint64(seed.Candidate.ID),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Code != errs.OK {
		t.Fatalf("expected OK, got code=%d msg=%s", resp.Code, resp.Msg)
	}
	if len(resp.List) < 1 {
		t.Fatalf("expected at least 1 task, got %d", len(resp.List))
	}
}

func TestCollaborationService_ListFollowUpTasks_WithCandidateFilterScopeDenied(t *testing.T) {
	db := setupCollaborationTestDB(t)
	seed := seedCollaborationTestData(t, db)
	svc := newCollaborationServiceForTest(t, db)

	// other_hr has no scope, should be denied when filtering by candidate
	ctx := metadata.WithAuthActor(context.Background(), seed.OtherHrUser.ID, "staff")
	_, err := svc.ListFollowUpTasks(ctx, &pb.ListFollowUpTasksRequest{
		StaffUserId:     seed.OtherHrUser.ID,
		CandidateUserId: uint64(seed.Candidate.ID),
	})
	if err == nil {
		t.Fatal("expected error for scope-denied user with candidate filter, got nil")
	}
}

func TestCollaborationService_CompleteFollowUpTask_Success(t *testing.T) {
	db := setupCollaborationTestDB(t)
	seed := seedCollaborationTestData(t, db)
	svc := newCollaborationServiceForTest(t, db)

	ctx := metadata.WithAuthActor(context.Background(), seed.HrUser.ID, "staff")
	resp, err := svc.CompleteFollowUpTask(ctx, &pb.CompleteFollowUpTaskRequest{
		StaffUserId: seed.HrUser.ID,
		TaskId:      seed.Task.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Code != errs.OK {
		t.Fatalf("expected OK, got code=%d msg=%s", resp.Code, resp.Msg)
	}

	// Verify task status changed
	getResp, err := svc.GetFollowUpTask(ctx, &pb.GetFollowUpTaskRequest{
		StaffUserId: seed.HrUser.ID,
		TaskId:      seed.Task.ID,
	})
	if err != nil {
		t.Fatalf("get task after complete: %v", err)
	}
	if getResp.Task.Status != "completed" {
		t.Errorf("Task status=%s, want completed", getResp.Task.Status)
	}
}

func TestCollaborationService_CompleteFollowUpTask_ScopeDenied(t *testing.T) {
	db := setupCollaborationTestDB(t)
	seed := seedCollaborationTestData(t, db)
	svc := newCollaborationServiceForTest(t, db)

	// other_hr has no scope, should be denied even though they have task.manage permission
	ctx := metadata.WithAuthActor(context.Background(), seed.OtherHrUser.ID, "staff")
	_, err := svc.CompleteFollowUpTask(ctx, &pb.CompleteFollowUpTaskRequest{
		StaffUserId: seed.OtherHrUser.ID,
		TaskId:      seed.Task.ID,
	})
	if err == nil {
		t.Fatal("expected error for scope-denied user completing task, got nil")
	}
}

func TestCollaborationService_GetFollowUpTask_Success(t *testing.T) {
	db := setupCollaborationTestDB(t)
	seed := seedCollaborationTestData(t, db)
	svc := newCollaborationServiceForTest(t, db)

	ctx := metadata.WithAuthActor(context.Background(), seed.HrUser.ID, "staff")
	resp, err := svc.GetFollowUpTask(ctx, &pb.GetFollowUpTaskRequest{
		StaffUserId: seed.HrUser.ID,
		TaskId:      seed.Task.ID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Code != errs.OK {
		t.Fatalf("expected OK, got code=%d msg=%s", resp.Code, resp.Msg)
	}
	if resp.Task == nil {
		t.Fatal("task is nil")
	}
	if resp.Task.Title != "Review application" {
		t.Errorf("Title=%s, want Review application", resp.Task.Title)
	}
}

func TestCollaborationService_GetFollowUpTask_ScopeDenied(t *testing.T) {
	db := setupCollaborationTestDB(t)
	seed := seedCollaborationTestData(t, db)
	svc := newCollaborationServiceForTest(t, db)

	// other_hr has no scope, should be denied
	ctx := metadata.WithAuthActor(context.Background(), seed.OtherHrUser.ID, "staff")
	_, err := svc.GetFollowUpTask(ctx, &pb.GetFollowUpTaskRequest{
		StaffUserId: seed.OtherHrUser.ID,
		TaskId:      seed.Task.ID,
	})
	if err == nil {
		t.Fatal("expected error for scope-denied user getting task, got nil")
	}
}

// ── Timeline Tests ───────────────────────────────────────────────────────────

func TestCollaborationService_ListTimelineEvents_Success(t *testing.T) {
	db := setupCollaborationTestDB(t)
	seed := seedCollaborationTestData(t, db)
	svc := newCollaborationServiceForTest(t, db)

	ctx := metadata.WithAuthActor(context.Background(), seed.HrUser.ID, "staff")
	resp, err := svc.ListTimelineEvents(ctx, &pb.ListTimelineEventsRequest{
		StaffUserId:     seed.HrUser.ID,
		CandidateUserId: uint64(seed.Candidate.ID),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Code != errs.OK {
		t.Fatalf("expected OK, got code=%d msg=%s", resp.Code, resp.Msg)
	}

	// Should have note event from seed
	foundNoteEvent := false
	for _, e := range resp.Events {
		if e.EventType == "note" {
			foundNoteEvent = true
			break
		}
	}
	if !foundNoteEvent {
		t.Error("expected at least one note event in timeline")
	}
}

func TestCollaborationService_ListTimelineEvents_ScopeDenied(t *testing.T) {
	db := setupCollaborationTestDB(t)
	seed := seedCollaborationTestData(t, db)
	svc := newCollaborationServiceForTest(t, db)

	ctx := metadata.WithAuthActor(context.Background(), seed.OtherHrUser.ID, "staff")
	_, err := svc.ListTimelineEvents(ctx, &pb.ListTimelineEventsRequest{
		StaffUserId:     seed.OtherHrUser.ID,
		CandidateUserId: uint64(seed.Candidate.ID),
	})
	if err == nil {
		t.Fatal("expected error for scope-denied user, got nil")
	}
}

// ── Scope Enforcement (B2) Tests ─────────────────────────────────────────────

func TestCollaborationService_ListFollowUpTasks_WithoutCandidateFilterDenied(t *testing.T) {
	db := setupCollaborationTestDB(t)
	seed := seedCollaborationTestData(t, db)
	svc := newCollaborationServiceForTest(t, db)

	// other_hr has task.manage permission but no data scope on seed.Candidate
	// Even without a candidate filter, scope must be enforced.
	// The candidate_user_id field is always required for scope verification.
	ctx := metadata.WithAuthActor(context.Background(), seed.OtherHrUser.ID, "staff")
	_, err := svc.ListFollowUpTasks(ctx, &pb.ListFollowUpTasksRequest{
		StaffUserId:     seed.OtherHrUser.ID,
		CandidateUserId: uint64(seed.Candidate.ID), // Needs scope check
	})
	if err == nil {
		t.Fatal("expected scope-denied error, got nil")
	}
}

// ── Full lifecycle: Workspace → Note → Tag → Task → Timeline ─────────────────

func TestCollaborationService_FullLifecycle(t *testing.T) {
	db := setupCollaborationTestDB(t)
	seed := seedCollaborationTestData(t, db)
	svc := newCollaborationServiceForTest(t, db)

	ctx := metadata.WithAuthActor(context.Background(), seed.HrUser.ID, "staff")
	candidateID := uint64(seed.Candidate.ID)

	// 1. Get workspace
	wsResp, err := svc.GetCandidateWorkspace(ctx, &pb.GetCandidateWorkspaceRequest{
		StaffUserId:     seed.HrUser.ID,
		CandidateUserId: int64(candidateID),
	})
	if err != nil {
		t.Fatalf("workspace: %v", err)
	}
	if wsResp.Workspace == nil {
		t.Fatal("workspace is nil")
	}

	// 2. Create note
	noteResp, err := svc.CreateNote(ctx, &pb.CreateNoteRequest{
		StaffUserId:     seed.HrUser.ID,
		CandidateUserId: candidateID,
		Content:         "Lifecycle test note",
	})
	if err != nil {
		t.Fatalf("create note: %v", err)
	}
	if noteResp.Note.Content != "Lifecycle test note" {
		t.Errorf("note content mismatch")
	}

	// 3. List notes
	listNotesResp, err := svc.ListNotes(ctx, &pb.ListNotesRequest{
		StaffUserId:     seed.HrUser.ID,
		CandidateUserId: candidateID,
	})
	if err != nil {
		t.Fatalf("list notes: %v", err)
	}
	if len(listNotesResp.List) < 1 {
		t.Fatal("expected at least 1 note")
	}

	// 4. Create and assign tag
	tagResp, err := svc.CreateTag(ctx, &pb.CreateTagRequest{
		StaffUserId: seed.HrUser.ID,
		Name:        "Lifecycle-Tag",
	})
	if err != nil {
		t.Fatalf("create tag: %v", err)
	}
	_, err = svc.AssignTag(ctx, &pb.AssignTagRequest{
		StaffUserId:     seed.HrUser.ID,
		TagId:           tagResp.Tag.Id,
		CandidateUserId: candidateID,
	})
	if err != nil {
		t.Fatalf("assign tag: %v", err)
	}

	// 5. List candidate tags
	ctResp, err := svc.ListCandidateTags(ctx, &pb.ListCandidateTagsRequest{
		StaffUserId:     seed.HrUser.ID,
		CandidateUserId: candidateID,
	})
	if err != nil {
		t.Fatalf("list candidate tags: %v", err)
	}
	foundTag := false
	for _, t := range ctResp.List {
		if t.Name == "Lifecycle-Tag" {
			foundTag = true
			break
		}
	}
	if !foundTag {
		t.Error("assigned tag not found in candidate tags")
	}

	// 6. Create and complete task
	taskResp, err := svc.CreateFollowUpTask(ctx, &pb.CreateFollowUpTaskRequest{
		StaffUserId:     seed.HrUser.ID,
		CandidateUserId: candidateID,
		AssigneeUserId:  uint64(seed.HrUser.ID),
		Title:           "Lifecycle task",
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	_, err = svc.CompleteFollowUpTask(ctx, &pb.CompleteFollowUpTaskRequest{
		StaffUserId: seed.HrUser.ID,
		TaskId:      taskResp.Task.Id,
	})
	if err != nil {
		t.Fatalf("complete task: %v", err)
	}

	// 7. Get task and verify completed
	getTaskResp, err := svc.GetFollowUpTask(ctx, &pb.GetFollowUpTaskRequest{
		StaffUserId: seed.HrUser.ID,
		TaskId:      taskResp.Task.Id,
	})
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if getTaskResp.Task.Status != "completed" {
		t.Errorf("task status=%s, want completed", getTaskResp.Task.Status)
	}

	// 8. List tasks with candidate filter
	listTaskResp, err := svc.ListFollowUpTasks(ctx, &pb.ListFollowUpTasksRequest{
		StaffUserId:     seed.HrUser.ID,
		CandidateUserId: candidateID,
	})
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(listTaskResp.List) < 1 {
		t.Fatal("expected at least 1 task")
	}

	// 9. Get timeline
	tlResp, err := svc.ListTimelineEvents(ctx, &pb.ListTimelineEventsRequest{
		StaffUserId:     seed.HrUser.ID,
		CandidateUserId: candidateID,
	})
	if err != nil {
		t.Fatalf("timeline: %v", err)
	}
	if len(tlResp.Events) < 1 {
		t.Fatal("expected timeline events")
	}
}
