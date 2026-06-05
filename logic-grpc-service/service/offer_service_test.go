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

func setupOfferServiceTestDB(t *testing.T) *gorm.DB {
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
		&model.EventOutbox{},
		&model.Offer{},
		&model.OfferEvent{},
	}
	for _, table := range tables {
		if err := db.AutoMigrate(table); err != nil {
			t.Fatalf("migrate %T: %v", table, err)
		}
	}
	return db
}

type offerTestSeed struct {
	HrUser       *model.User
	OtherHrUser  *model.User // unprivileged staff for negative tests
	Candidate    *model.User
	OtherCandidate *model.User // another candidate for ownership tests
	Job          *model.Job
	Application  *model.Application
}

func seedOfferTestData(t *testing.T, db *gorm.DB) *offerTestSeed {
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
		{PermissionKey: authz.PermOfferRead, Resource: "offer", Action: "read", Description: "Read offers"},
		{PermissionKey: authz.PermOfferManage, Resource: "offer", Action: "manage", Description: "Manage offers"},
		{PermissionKey: authz.PermOfferSend, Resource: "offer", Action: "send", Description: "Send offers"},
		{PermissionKey: authz.PermOfferDecisionManage, Resource: "offer", Action: "decision_manage", Description: "Accept/reject offers"},
	}
	for i := range perms {
		if err := db.Create(&perms[i]).Error; err != nil {
			t.Fatalf("create perm %s: %v", perms[i].PermissionKey, err)
		}
	}

	// Recruiter gets all offer permissions
	for _, p := range perms {
		if err := db.Create(&model.RolePermission{RoleID: recruiterRole.ID, PermissionID: p.ID}).Error; err != nil {
			t.Fatalf("assign perm %s to recruiter: %v", p.PermissionKey, err)
		}
	}

	// Candidate only gets offer.decision.manage
	for _, p := range perms {
		if p.PermissionKey == authz.PermOfferDecisionManage {
			if err := db.Create(&model.RolePermission{RoleID: candidateRole.ID, PermissionID: p.ID}).Error; err != nil {
				t.Fatalf("assign perm %s to candidate: %v", p.PermissionKey, err)
			}
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

	// ── Job ─────────────────────────────────────────────────────────────
	job := &model.Job{
		HrID:   hrUser.ID,
		Title:  "Senior Engineer",
		Status: 1,
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
		StatusKey: model.StatusKeyInterviewPassed,
	}
	if err := db.Create(app).Error; err != nil {
		t.Fatalf("create application: %v", err)
	}

	return &offerTestSeed{
		HrUser:        hrUser,
		OtherHrUser:   otherHr,
		Candidate:     candidate,
		OtherCandidate: otherCandidate,
		Job:           job,
		Application:   app,
	}
}

func newOfferServiceForTest(t *testing.T, db *gorm.DB) *OfferService {
	t.Helper()
	authzRepo := repository.NewAuthzRepo(db)
	offerRepo := repository.NewOfferRepo(db)
	appRepo := repository.NewApplicationRepo(db)
	jobRepo := repository.NewJobRepo(db)
	notifRepo := repository.NewNotificationRepo(db)
	outboxRepo := repository.NewOutboxRepo(db)
	scopeEval := &scopeEvaluator{authzRepo: authzRepo}
	serviceAuth := NewServiceAuthorizer(authzRepo, scopeEval)
	outboxPublisher := NewOutboxPublisher(outboxRepo, nil) // nil mqConn — outbox writes work, publishing skipped

	return NewOfferService(authzRepo, offerRepo, appRepo, jobRepo, notifRepo, outboxPublisher, scopeEval, serviceAuth)
}

// helper: createOfferInTest creates a draft offer and returns it for use in subsequent tests.
func createOfferInTest(t *testing.T, svc *OfferService, seed *offerTestSeed, ctx context.Context) int64 {
	t.Helper()
	resp, err := svc.CreateOffer(ctx, &pb.CreateOfferRequest{
		HrId:          seed.HrUser.ID,
		ApplicationId: seed.Application.ID,
		Title:         "Test Offer",
		SalaryRange:   "30k-50k",
		Level:         "P6",
		WorkLocation:  "Beijing",
		StartDate:     "2026-07-01",
		TermsJson:     `{"bonus":"2 months"}`,
	})
	if err != nil {
		t.Fatalf("createOfferInTest: unexpected error: %v", err)
	}
	if resp.Code != errs.OK {
		t.Fatalf("createOfferInTest: expected OK, got code=%d msg=%s", resp.Code, resp.Msg)
	}
	return resp.OfferId
}

// ── CreateOffer Tests ───────────────────────────────────────────────────────

func TestOfferService_CreateOffer_Success(t *testing.T) {
	db := setupOfferServiceTestDB(t)
	seed := seedOfferTestData(t, db)
	svc := newOfferServiceForTest(t, db)

	ctx := metadata.WithAuthActor(context.Background(), seed.HrUser.ID, "staff")
	resp, err := svc.CreateOffer(ctx, &pb.CreateOfferRequest{
		HrId:          seed.HrUser.ID,
		ApplicationId: seed.Application.ID,
		Title:         "Senior Offer",
		SalaryRange:   "40k-60k",
		Level:         "P7",
		WorkLocation:  "Shanghai",
		StartDate:     "2026-08-01",
		TermsJson:     `{"stock":"1000 RSU"}`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Code != errs.OK {
		t.Fatalf("expected OK, got code=%d msg=%s", resp.Code, resp.Msg)
	}
	if resp.OfferId == 0 {
		t.Fatal("expected non-zero offer ID")
	}

	// Verify offer was created in DB
	offer, err := svc.offers.GetModelByID(ctx, resp.OfferId)
	if err != nil {
		t.Fatalf("get offer: %v", err)
	}
	if offer == nil {
		t.Fatal("offer not found in DB")
	}
	if offer.Status != "draft" {
		t.Errorf("expected status=draft, got %s", offer.Status)
	}
	if offer.CandidateUserID != seed.Candidate.ID {
		t.Errorf("CandidateUserID=%d, want %d", offer.CandidateUserID, seed.Candidate.ID)
	}

	// Verify application status transitioned to offer_pending
	appDetail, _ := svc.applications.GetDetail(ctx, seed.Application.ID)
	if appDetail.StatusKey != model.StatusKeyOfferPending {
		t.Errorf("application status_key=%s, want %s", appDetail.StatusKey, model.StatusKeyOfferPending)
	}

	// Verify application status transition record was written
	transitions, _ := svc.applications.ListTransitions(ctx, seed.Application.ID)
	if len(transitions) < 1 {
		t.Fatal("expected at least 1 transition record")
	}
	lastT := transitions[len(transitions)-1]
	if lastT.FromStatus != model.StatusKeyInterviewPassed || lastT.ToStatus != model.StatusKeyOfferPending {
		t.Errorf("transition %s→%s, want %s→%s",
			lastT.FromStatus, lastT.ToStatus,
			model.StatusKeyInterviewPassed, model.StatusKeyOfferPending)
	}
	if lastT.ActorUserID != seed.HrUser.ID {
		t.Errorf("transition actor=%d, want %d", lastT.ActorUserID, seed.HrUser.ID)
	}

	// Verify offer event was created
	events, _ := svc.offers.ListEventsByOfferID(ctx, resp.OfferId)
	if len(events) < 1 {
		t.Fatal("expected at least 1 offer event")
	}
	if events[0].EventType != "created" {
		t.Errorf("event type=%s, want created", events[0].EventType)
	}
}

func TestOfferService_CreateOffer_InvalidTransition(t *testing.T) {
	db := setupOfferServiceTestDB(t)
	seed := seedOfferTestData(t, db)
	svc := newOfferServiceForTest(t, db)

	// Put application in a state where offer_pending is not a valid next state
	seed.Application.StatusKey = model.StatusKeyApplied
	db.Save(seed.Application)

	ctx := metadata.WithAuthActor(context.Background(), seed.HrUser.ID, "staff")
	resp, err := svc.CreateOffer(ctx, &pb.CreateOfferRequest{
		HrId:          seed.HrUser.ID,
		ApplicationId: seed.Application.ID,
		Title:         "Bad Offer",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Code == errs.OK {
		t.Fatal("expected error for invalid transition from applied → offer_pending")
	}
}

func TestOfferService_CreateOffer_NoPermission(t *testing.T) {
	db := setupOfferServiceTestDB(t)
	seed := seedOfferTestData(t, db)
	svc := newOfferServiceForTest(t, db)

	// Candidate tries to create offer (no offer.manage permission)
	ctx := metadata.WithAuthActor(context.Background(), seed.Candidate.ID, "candidate")
	resp, err := svc.CreateOffer(ctx, &pb.CreateOfferRequest{
		HrId:          seed.Candidate.ID,
		ApplicationId: seed.Application.ID,
		Title:         "Candidate Offer",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Code != errs.ErrForbidden {
		t.Fatalf("expected Forbidden, got code=%d msg=%s", resp.Code, resp.Msg)
	}
}

// ── SendOffer Tests ─────────────────────────────────────────────────────────

func TestOfferService_SendOffer_Success(t *testing.T) {
	db := setupOfferServiceTestDB(t)
	seed := seedOfferTestData(t, db)
	svc := newOfferServiceForTest(t, db)

	ctx := metadata.WithAuthActor(context.Background(), seed.HrUser.ID, "staff")
	offerID := createOfferInTest(t, svc, seed, ctx)

	// Send the offer
	resp, err := svc.SendOffer(ctx, &pb.SendOfferRequest{
		HrId:    seed.HrUser.ID,
		OfferId: offerID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Code != errs.OK {
		t.Fatalf("expected OK, got code=%d msg=%s", resp.Code, resp.Msg)
	}

	// Verify offer status
	offer, _ := svc.offers.GetModelByID(ctx, offerID)
	if offer.Status != "sent" {
		t.Errorf("offer status=%s, want sent", offer.Status)
	}
	if offer.SentBy == nil || *offer.SentBy != seed.HrUser.ID {
		t.Errorf("SentBy=%v, want %d", offer.SentBy, seed.HrUser.ID)
	}
	if offer.SentSnapshotJSON == "" {
		t.Error("sent_snapshot_json should not be empty (AC-4)")
	}

	// Verify application status
	appDetail, _ := svc.applications.GetDetail(ctx, seed.Application.ID)
	if appDetail.StatusKey != model.StatusKeyOfferSent {
		t.Errorf("application status=%s, want %s", appDetail.StatusKey, model.StatusKeyOfferSent)
	}

	// Verify transition records
	transitions, _ := svc.applications.ListTransitions(ctx, seed.Application.ID)
	if len(transitions) < 2 {
		t.Fatalf("expected at least 2 transitions, got %d", len(transitions))
	}
	sendT := transitions[len(transitions)-1]
	if sendT.FromStatus != model.StatusKeyOfferPending || sendT.ToStatus != model.StatusKeyOfferSent {
		t.Errorf("send transition %s→%s, want %s→%s",
			sendT.FromStatus, sendT.ToStatus,
			model.StatusKeyOfferPending, model.StatusKeyOfferSent)
	}

	// Verify offer event
	events, _ := svc.offers.ListEventsByOfferID(ctx, offerID)
	foundSent := false
	for _, e := range events {
		if e.EventType == "sent" {
			foundSent = true
			break
		}
	}
	if !foundSent {
		t.Error("expected 'sent' event in offer events")
	}
}

func TestOfferService_SendOffer_CannotSendNonDraft(t *testing.T) {
	db := setupOfferServiceTestDB(t)
	seed := seedOfferTestData(t, db)
	svc := newOfferServiceForTest(t, db)

	ctx := metadata.WithAuthActor(context.Background(), seed.HrUser.ID, "staff")
	offerID := createOfferInTest(t, svc, seed, ctx)

	// Send it once
	svc.SendOffer(ctx, &pb.SendOfferRequest{HrId: seed.HrUser.ID, OfferId: offerID})

	// Try to send again
	resp, err := svc.SendOffer(ctx, &pb.SendOfferRequest{HrId: seed.HrUser.ID, OfferId: offerID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Code != errs.ErrBadRequest {
		t.Fatalf("expected BadRequest for re-sending, got code=%d msg=%s", resp.Code, resp.Msg)
	}
}

// ── AcceptOffer Tests ───────────────────────────────────────────────────────

func TestOfferService_AcceptOffer_Success(t *testing.T) {
	db := setupOfferServiceTestDB(t)
	seed := seedOfferTestData(t, db)
	svc := newOfferServiceForTest(t, db)

	hrCtx := metadata.WithAuthActor(context.Background(), seed.HrUser.ID, "staff")
	offerID := createOfferInTest(t, svc, seed, hrCtx)
	svc.SendOffer(hrCtx, &pb.SendOfferRequest{HrId: seed.HrUser.ID, OfferId: offerID})

	// Candidate accepts
	candidateCtx := metadata.WithAuthActor(context.Background(), seed.Candidate.ID, "candidate")
	resp, err := svc.AcceptOffer(candidateCtx, &pb.AcceptOfferRequest{
		UserId:  seed.Candidate.ID,
		OfferId: offerID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Code != errs.OK {
		t.Fatalf("expected OK, got code=%d msg=%s", resp.Code, resp.Msg)
	}

	// Verify offer status
	offer, _ := svc.offers.GetModelByID(hrCtx, offerID)
	if offer.Status != "accepted" {
		t.Errorf("offer status=%s, want accepted", offer.Status)
	}
	if offer.DecidedAt == nil {
		t.Error("decided_at should be set")
	}

	// Verify application status
	appDetail, _ := svc.applications.GetDetail(hrCtx, seed.Application.ID)
	if appDetail.StatusKey != model.StatusKeyOfferAccepted {
		t.Errorf("application status=%s, want %s", appDetail.StatusKey, model.StatusKeyOfferAccepted)
	}

	// Verify transition with correct actor
	transitions, _ := svc.applications.ListTransitions(hrCtx, seed.Application.ID)
	acceptT := transitions[len(transitions)-1]
	if acceptT.ActorUserID != seed.Candidate.ID {
		t.Errorf("accept transition actor=%d, want %d", acceptT.ActorUserID, seed.Candidate.ID)
	}
	if acceptT.ActorAccountType != "candidate" {
		t.Errorf("accept transition actor type=%s, want candidate", acceptT.ActorAccountType)
	}
}

func TestOfferService_AcceptOffer_WrongCandidate(t *testing.T) {
	db := setupOfferServiceTestDB(t)
	seed := seedOfferTestData(t, db)
	svc := newOfferServiceForTest(t, db)

	hrCtx := metadata.WithAuthActor(context.Background(), seed.HrUser.ID, "staff")
	offerID := createOfferInTest(t, svc, seed, hrCtx)
	svc.SendOffer(hrCtx, &pb.SendOfferRequest{HrId: seed.HrUser.ID, OfferId: offerID})

	// Other candidate tries to accept
	otherCtx := metadata.WithAuthActor(context.Background(), seed.OtherCandidate.ID, "candidate")
	resp, err := svc.AcceptOffer(otherCtx, &pb.AcceptOfferRequest{
		UserId:  seed.OtherCandidate.ID,
		OfferId: offerID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Code != errs.ErrForbidden {
		t.Fatalf("expected Forbidden for wrong candidate, got code=%d msg=%s", resp.Code, resp.Msg)
	}
}

// ── RejectOffer Tests ───────────────────────────────────────────────────────

func TestOfferService_RejectOffer_Success(t *testing.T) {
	db := setupOfferServiceTestDB(t)
	seed := seedOfferTestData(t, db)
	svc := newOfferServiceForTest(t, db)

	hrCtx := metadata.WithAuthActor(context.Background(), seed.HrUser.ID, "staff")
	offerID := createOfferInTest(t, svc, seed, hrCtx)
	svc.SendOffer(hrCtx, &pb.SendOfferRequest{HrId: seed.HrUser.ID, OfferId: offerID})

	// Candidate rejects
	candidateCtx := metadata.WithAuthActor(context.Background(), seed.Candidate.ID, "candidate")
	resp, err := svc.RejectOffer(candidateCtx, &pb.RejectOfferRequest{
		UserId:  seed.Candidate.ID,
		OfferId: offerID,
		Reason:  "Better opportunity elsewhere",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Code != errs.OK {
		t.Fatalf("expected OK, got code=%d msg=%s", resp.Code, resp.Msg)
	}

	// Verify offer status
	offer, _ := svc.offers.GetModelByID(hrCtx, offerID)
	if offer.Status != "rejected" {
		t.Errorf("offer status=%s, want rejected", offer.Status)
	}

	// Verify application status
	appDetail, _ := svc.applications.GetDetail(hrCtx, seed.Application.ID)
	if appDetail.StatusKey != model.StatusKeyOfferRejected {
		t.Errorf("application status=%s, want %s", appDetail.StatusKey, model.StatusKeyOfferRejected)
	}

	// P1 fix: verify is_current=0 for terminal state
	var app model.Application
	db.First(&app, seed.Application.ID)
	if app.IsCurrent != 0 {
		t.Errorf("is_current=%d, want 0 (terminal state must close the current round)", app.IsCurrent)
	}

	// Verify transition record with reason
	transitions, _ := svc.applications.ListTransitions(hrCtx, seed.Application.ID)
	rejectT := transitions[len(transitions)-1]
	if rejectT.ToStatus != model.StatusKeyOfferRejected {
		t.Errorf("reject transition to=%s, want %s", rejectT.ToStatus, model.StatusKeyOfferRejected)
	}
	if rejectT.Reason != "Better opportunity elsewhere" {
		t.Errorf("reject reason=%s, want 'Better opportunity elsewhere'", rejectT.Reason)
	}

	// Verify offer event
	events, _ := svc.offers.ListEventsByOfferID(hrCtx, offerID)
	foundRejected := false
	for _, e := range events {
		if e.EventType == "rejected" {
			foundRejected = true
			break
		}
	}
	if !foundRejected {
		t.Error("expected 'rejected' event")
	}
}

// ── WithdrawOffer Tests ─────────────────────────────────────────────────────

func TestOfferService_WithdrawOffer_SentRevertsApplication(t *testing.T) {
	db := setupOfferServiceTestDB(t)
	seed := seedOfferTestData(t, db)
	svc := newOfferServiceForTest(t, db)

	ctx := metadata.WithAuthActor(context.Background(), seed.HrUser.ID, "staff")
	offerID := createOfferInTest(t, svc, seed, ctx)
	svc.SendOffer(ctx, &pb.SendOfferRequest{HrId: seed.HrUser.ID, OfferId: offerID})

	// Withdraw the sent offer
	resp, err := svc.WithdrawOffer(ctx, &pb.WithdrawOfferRequest{
		HrId:    seed.HrUser.ID,
		OfferId: offerID,
		Reason:  "Budget change",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Code != errs.OK {
		t.Fatalf("expected OK, got code=%d msg=%s", resp.Code, resp.Msg)
	}

	// Verify offer is withdrawn
	offer, _ := svc.offers.GetModelByID(ctx, offerID)
	if offer.Status != "withdrawn" {
		t.Errorf("offer status=%s, want withdrawn", offer.Status)
	}

	// P1 fix: application should revert to offer_pending (not stuck at offer_sent)
	appDetail, _ := svc.applications.GetDetail(ctx, seed.Application.ID)
	if appDetail.StatusKey != model.StatusKeyOfferPending {
		t.Errorf("application status=%s, want %s (should revert to offer_pending after withdraw)",
			appDetail.StatusKey, model.StatusKeyOfferPending)
	}

	// Verify transition record for withdraw
	transitions, _ := svc.applications.ListTransitions(ctx, seed.Application.ID)
	withdrawT := transitions[len(transitions)-1]
	if withdrawT.FromStatus != model.StatusKeyOfferSent || withdrawT.ToStatus != model.StatusKeyOfferPending {
		t.Errorf("withdraw transition %s→%s, want %s→%s",
			withdrawT.FromStatus, withdrawT.ToStatus,
			model.StatusKeyOfferSent, model.StatusKeyOfferPending)
	}

	// Verify offer event
	events, _ := svc.offers.ListEventsByOfferID(ctx, offerID)
	foundWithdrawn := false
	for _, e := range events {
		if e.EventType == "withdrawn" {
			foundWithdrawn = true
			break
		}
	}
	if !foundWithdrawn {
		t.Error("expected 'withdrawn' event")
	}
}

func TestOfferService_WithdrawOffer_DraftNoAppChange(t *testing.T) {
	db := setupOfferServiceTestDB(t)
	seed := seedOfferTestData(t, db)
	svc := newOfferServiceForTest(t, db)

	ctx := metadata.WithAuthActor(context.Background(), seed.HrUser.ID, "staff")
	offerID := createOfferInTest(t, svc, seed, ctx)

	// Application is at offer_pending after create
	appBefore, _ := svc.applications.GetDetail(ctx, seed.Application.ID)

	// Withdraw the draft offer (never sent)
	resp, err := svc.WithdrawOffer(ctx, &pb.WithdrawOfferRequest{
		HrId:    seed.HrUser.ID,
		OfferId: offerID,
		Reason:  "No longer needed",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Code != errs.OK {
		t.Fatalf("expected OK, got code=%d msg=%s", resp.Code, resp.Msg)
	}

	// Offer should be withdrawn
	offer, _ := svc.offers.GetModelByID(ctx, offerID)
	if offer.Status != "withdrawn" {
		t.Errorf("offer status=%s, want withdrawn", offer.Status)
	}

	// Application status should NOT change for draft withdrawal
	appAfter, _ := svc.applications.GetDetail(ctx, seed.Application.ID)
	if appAfter.StatusKey != appBefore.StatusKey {
		t.Errorf("draft withdraw should not change application status: was %s, now %s",
			appBefore.StatusKey, appAfter.StatusKey)
	}
}

// ── Transaction Consistency Tests ───────────────────────────────────────────

func TestOfferService_SendOffer_ConcurrentStatusChangeRollsBack(t *testing.T) {
	db := setupOfferServiceTestDB(t)
	seed := seedOfferTestData(t, db)
	svc := newOfferServiceForTest(t, db)

	ctx := metadata.WithAuthActor(context.Background(), seed.HrUser.ID, "staff")
	offerID := createOfferInTest(t, svc, seed, ctx)

	// Simulate concurrent change: move application to a different status
	// so the UpdateStatusAnyWithTx returns rows=0
	seed.Application.StatusKey = model.StatusKeyRejected
	db.Save(seed.Application)

	resp, err := svc.SendOffer(ctx, &pb.SendOfferRequest{
		HrId:    seed.HrUser.ID,
		OfferId: offerID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should fail because concurrent status change
	if resp.Code == errs.OK {
		t.Fatal("expected failure due to concurrent application status change")
	}

	// Offer should remain in draft (transaction rolled back)
	offer, _ := svc.offers.GetModelByID(ctx, offerID)
	if offer.Status != "draft" {
		t.Errorf("offer status=%s, want draft (transaction should have rolled back)", offer.Status)
	}
}

func TestOfferService_AcceptOffer_InvalidTransitionRollsBack(t *testing.T) {
	db := setupOfferServiceTestDB(t)
	seed := seedOfferTestData(t, db)
	svc := newOfferServiceForTest(t, db)

	hrCtx := metadata.WithAuthActor(context.Background(), seed.HrUser.ID, "staff")
	offerID := createOfferInTest(t, svc, seed, hrCtx)
	// Send offer
	svc.SendOffer(hrCtx, &pb.SendOfferRequest{HrId: seed.HrUser.ID, OfferId: offerID})

	// Simulate concurrent change: application moved away from offer_sent
	seed.Application.StatusKey = model.StatusKeyRejected
	db.Save(seed.Application)

	candidateCtx := metadata.WithAuthActor(context.Background(), seed.Candidate.ID, "candidate")
	resp, err := svc.AcceptOffer(candidateCtx, &pb.AcceptOfferRequest{
		UserId:  seed.Candidate.ID,
		OfferId: offerID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Code == errs.OK {
		t.Fatal("expected failure due to concurrent status change")
	}

	// Offer should remain sent (not accepted) - transaction rolled back
	offer, _ := svc.offers.GetModelByID(hrCtx, offerID)
	if offer.Status != "sent" {
		t.Errorf("offer status=%s, want sent (transaction should have rolled back)", offer.Status)
	}
}
