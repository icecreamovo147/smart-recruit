package persistence

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"smart-recruit-platform-go/errs"
	"smart-recruit-proto/recruitment/pb"
	domainmodel "smart-recruit-recruitment-service/internal/domain/model"
)

func TestNativeUpdateApplicationStatusRejectsInvalidTransitions(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name      string
		current   string
		target    string
		wantCode  int32
		wantNoops bool
	}{
		{name: "invalid status key", current: domainmodel.StatusKeyApplied, target: "bogus", wantCode: errs.ErrBadRequest, wantNoops: true},
		{name: "illegal transition", current: domainmodel.StatusKeyApplied, target: domainmodel.StatusKeyOfferSent, wantCode: errs.ErrBadRequest, wantNoops: true},
		{name: "same status", current: domainmodel.StatusKeyApplied, target: domainmodel.StatusKeyApplied, wantCode: errs.ErrBadRequest, wantNoops: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := newLifecycleFixture(t)
			appID := fixture.seedApplication(t, tt.current, 1, 1)

			resp, err := fixture.application.UpdateApplicationStatus(ctx, &pb.UpdateApplicationStatusRequest{
				HrId:          lifecycleHRID,
				ApplicationId: appID,
				StatusKey:     tt.target,
				Reason:        "state move",
			})
			if err != nil {
				t.Fatalf("UpdateApplicationStatus() error = %v", err)
			}
			if resp.Code != tt.wantCode {
				t.Fatalf("UpdateApplicationStatus() code = %d, want %d; msg=%s", resp.Code, tt.wantCode, resp.Msg)
			}
			if tt.wantNoops {
				fixture.assertApplication(t, appID, tt.current, 1, 1)
				fixture.assertCount(t, &applicationTransitionRecord{}, 0)
				fixture.assertCount(t, &eventOutboxRecord{}, 0)
			}
		})
	}
}

func TestNativeUpdateApplicationStatusRequiresReasons(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		current string
		target  string
	}{
		{name: "rejected", current: domainmodel.StatusKeyApplied, target: domainmodel.StatusKeyRejected},
		{name: "withdrawn", current: domainmodel.StatusKeyApplied, target: domainmodel.StatusKeyWithdrawn},
		{name: "offer rejected", current: domainmodel.StatusKeyOfferSent, target: domainmodel.StatusKeyOfferRejected},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := newLifecycleFixture(t)
			appID := fixture.seedApplication(t, tt.current, 1, 1)

			resp, err := fixture.application.UpdateApplicationStatus(ctx, &pb.UpdateApplicationStatusRequest{
				HrId:          lifecycleHRID,
				ApplicationId: appID,
				StatusKey:     tt.target,
				Reason:        "   ",
			})
			if err != nil {
				t.Fatalf("UpdateApplicationStatus() error = %v", err)
			}
			if resp.Code != errs.ErrBadRequest {
				t.Fatalf("UpdateApplicationStatus() code = %d, want %d; msg=%s", resp.Code, errs.ErrBadRequest, resp.Msg)
			}
			fixture.assertApplication(t, appID, tt.current, 1, 1)
			fixture.assertCount(t, &applicationTransitionRecord{}, 0)
		})
	}
}

func TestNativeUpdateApplicationStatusWritesTransitionAndApplicationOutbox(t *testing.T) {
	ctx := context.Background()
	fixture := newLifecycleFixture(t)
	appID := fixture.seedApplication(t, domainmodel.StatusKeyApplied, 1, 1)

	resp, err := fixture.application.UpdateApplicationStatus(ctx, &pb.UpdateApplicationStatusRequest{
		HrId:          lifecycleHRID,
		ApplicationId: appID,
		StatusKey:     domainmodel.StatusKeyScreenPassed,
		Reason:        "qualified",
	})
	if err != nil {
		t.Fatalf("UpdateApplicationStatus() error = %v", err)
	}
	if resp.Code != errs.OK {
		t.Fatalf("UpdateApplicationStatus() code = %d, want %d; msg=%s", resp.Code, errs.OK, resp.Msg)
	}
	fixture.assertApplication(t, appID, domainmodel.StatusKeyScreenPassed, 1, 1)
	transition := fixture.onlyTransition(t)
	if transition.FromStatus != domainmodel.StatusKeyApplied || transition.ToStatus != domainmodel.StatusKeyScreenPassed {
		t.Fatalf("transition = %s -> %s", transition.FromStatus, transition.ToStatus)
	}
	if transition.ActorUserID != lifecycleHRID || transition.ActorAccountType != "staff" {
		t.Fatalf("transition actor = %d/%s", transition.ActorUserID, transition.ActorAccountType)
	}
	outbox := fixture.outboxRows(t)
	if len(outbox) != 2 {
		t.Fatalf("outbox rows = %d, want 2", len(outbox))
	}
	fixture.assertOutboxPayload(t, outbox[0], "application_approved", "candidate")
	fixture.assertOutboxPayload(t, outbox[1], "application_approved", "candidate")
}

func TestNativeUpdateApplicationStatusRePassReopensCurrentRound(t *testing.T) {
	ctx := context.Background()
	fixture := newLifecycleFixture(t)
	appID := fixture.seedApplication(t, domainmodel.StatusKeyRejected, 1, 0)

	resp, err := fixture.application.UpdateApplicationStatus(ctx, &pb.UpdateApplicationStatusRequest{
		HrId:          lifecycleHRID,
		ApplicationId: appID,
		StatusKey:     domainmodel.StatusKeyScreenPassed,
		Reason:        "manual review passed",
	})
	if err != nil {
		t.Fatalf("UpdateApplicationStatus() error = %v", err)
	}
	if resp.Code != errs.OK {
		t.Fatalf("UpdateApplicationStatus() code = %d, want %d; msg=%s", resp.Code, errs.OK, resp.Msg)
	}
	fixture.assertApplication(t, appID, domainmodel.StatusKeyScreenPassed, 2, 1)
	transition := fixture.onlyTransition(t)
	if transition.FromStatus != domainmodel.StatusKeyRejected || transition.ToStatus != domainmodel.StatusKeyScreenPassed {
		t.Fatalf("transition = %s -> %s", transition.FromStatus, transition.ToStatus)
	}
}

func TestNativeUpdateApplicationStatusCancelsActiveInterviewsOnRejectedAndWithdrawn(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name   string
		target string
	}{
		{name: "rejected", target: domainmodel.StatusKeyRejected},
		{name: "withdrawn", target: domainmodel.StatusKeyWithdrawn},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := newLifecycleFixture(t)
			appID := fixture.seedApplication(t, domainmodel.StatusKeyScreenPassed, 1, 1)
			fixture.seedInterview(t, appID, "pending")
			fixture.seedInterview(t, appID, "scheduled")
			completedID := fixture.seedInterview(t, appID, "completed")

			resp, err := fixture.application.UpdateApplicationStatus(ctx, &pb.UpdateApplicationStatusRequest{
				HrId:          lifecycleHRID,
				ApplicationId: appID,
				StatusKey:     tt.target,
				Reason:        "role closed",
			})
			if err != nil {
				t.Fatalf("UpdateApplicationStatus() error = %v", err)
			}
			if resp.Code != errs.OK {
				t.Fatalf("UpdateApplicationStatus() code = %d, want %d; msg=%s", resp.Code, errs.OK, resp.Msg)
			}
			var rows []lifecycleInterviewScheduleRecord
			if err := fixture.db.Order("id ASC").Find(&rows).Error; err != nil {
				t.Fatalf("load interviews: %v", err)
			}
			for _, row := range rows {
				if row.ID == completedID {
					if row.Status != "completed" {
						t.Fatalf("completed interview status = %s, want completed", row.Status)
					}
					continue
				}
				if row.Status != "cancelled" || row.CancelReason != "role closed" {
					t.Fatalf("active interview %d = %s/%q, want cancelled/reason", row.ID, row.Status, row.CancelReason)
				}
			}
		})
	}
}

func TestNativeOwnerLifecycleExpectedStatusConflictDoesNotWrite(t *testing.T) {
	ctx := context.Background()
	fixture := newLifecycleFixture(t)
	appID := fixture.seedApplication(t, domainmodel.StatusKeyApplied, 1, 1)

	resp, err := fixture.owner.ApplyApplicationLifecycleTransition(ctx, &pb.ApplyApplicationLifecycleTransitionRequest{
		ActorUserId:       lifecycleHRID,
		ActorAccountType:  "staff",
		ApplicationId:     appID,
		ExpectedStatusKey: domainmodel.StatusKeyViewed,
		TargetStatusKey:   domainmodel.StatusKeyScreenPassed,
		Reason:            "stale write",
	})
	if err != nil {
		t.Fatalf("ApplyApplicationLifecycleTransition() error = %v", err)
	}
	if resp.Code != errs.ErrConflict {
		t.Fatalf("ApplyApplicationLifecycleTransition() code = %d, want %d; msg=%s", resp.Code, errs.ErrConflict, resp.Msg)
	}
	fixture.assertApplication(t, appID, domainmodel.StatusKeyApplied, 1, 1)
	fixture.assertCount(t, &applicationTransitionRecord{}, 0)
}

func TestNativeOwnerLifecycleRejectsBlankAndInvalidActorWithoutWrites(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name             string
		actorAccountType string
	}{
		{name: "blank actor", actorAccountType: ""},
		{name: "invalid actor", actorAccountType: "robot"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := newLifecycleFixture(t)
			appID := fixture.seedApplication(t, domainmodel.StatusKeyApplied, 1, 1)

			resp, err := fixture.owner.ApplyApplicationLifecycleTransition(ctx, &pb.ApplyApplicationLifecycleTransitionRequest{
				ActorUserId:       lifecycleHRID,
				ActorAccountType:  tt.actorAccountType,
				ApplicationId:     appID,
				ExpectedStatusKey: domainmodel.StatusKeyApplied,
				TargetStatusKey:   domainmodel.StatusKeyScreenPassed,
				Reason:            "qualified",
			})
			if err != nil {
				t.Fatalf("ApplyApplicationLifecycleTransition() error = %v", err)
			}
			if resp.Code != errs.ErrBadRequest {
				t.Fatalf("ApplyApplicationLifecycleTransition() code = %d, want %d; msg=%s", resp.Code, errs.ErrBadRequest, resp.Msg)
			}
			if resp.Changed {
				t.Fatalf("ApplyApplicationLifecycleTransition() changed = true, want false")
			}
			fixture.assertApplication(t, appID, domainmodel.StatusKeyApplied, 1, 1)
			fixture.assertCount(t, &applicationTransitionRecord{}, 0)
			fixture.assertCount(t, &eventOutboxRecord{}, 0)
		})
	}
}

func TestNativeOwnerLifecyclePreservesCandidateActorAndClosesRound(t *testing.T) {
	ctx := context.Background()
	fixture := newLifecycleFixture(t)
	appID := fixture.seedApplication(t, domainmodel.StatusKeyOfferSent, 1, 1)

	resp, err := fixture.owner.ApplyApplicationLifecycleTransition(ctx, &pb.ApplyApplicationLifecycleTransitionRequest{
		ActorUserId:       lifecycleCandidateID,
		ActorAccountType:  "candidate",
		ApplicationId:     appID,
		ExpectedStatusKey: domainmodel.StatusKeyOfferSent,
		TargetStatusKey:   domainmodel.StatusKeyOfferRejected,
		Reason:            "accepted another offer",
		CloseCurrentRound: true,
	})
	if err != nil {
		t.Fatalf("ApplyApplicationLifecycleTransition() error = %v", err)
	}
	if resp.Code != errs.OK || !resp.Changed {
		t.Fatalf("ApplyApplicationLifecycleTransition() = %+v, want OK changed", resp)
	}
	if resp.FromStatusKey != domainmodel.StatusKeyOfferSent || resp.CurrentStatusKey != domainmodel.StatusKeyOfferRejected {
		t.Fatalf("transition response = %s -> %s", resp.FromStatusKey, resp.CurrentStatusKey)
	}
	fixture.assertApplication(t, appID, domainmodel.StatusKeyOfferRejected, 1, 0)
	transition := fixture.onlyTransition(t)
	if transition.ActorUserID != lifecycleCandidateID || transition.ActorAccountType != "candidate" {
		t.Fatalf("transition actor = %d/%s, want candidate", transition.ActorUserID, transition.ActorAccountType)
	}
	if transition.ToStatus != domainmodel.StatusKeyOfferRejected {
		t.Fatalf("transition to = %s, want offer_rejected", transition.ToStatus)
	}
}

const (
	lifecycleHRID        int64 = 2001
	lifecycleCandidateID int64 = 3001
	lifecycleJobID       int64 = 4001
	lifecycleResumeID    int64 = 5001
)

type lifecycleFixture struct {
	db          *gorm.DB
	application *applicationAdapter
	owner       *applicationOwnerAdapter
	now         time.Time
	nextAppID   int64
}

func newLifecycleFixture(t *testing.T) *lifecycleFixture {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&jobRecord{},
		&candidateProfileRecord{},
		&resumeRecord{},
		&applicationRecord{},
		&applicationTransitionRecord{},
		&eventOutboxRecord{},
		&userDataScopeRecord{},
		&lifecycleInterviewScheduleRecord{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	now := time.Date(2026, 7, 14, 10, 0, 0, 0, time.UTC)
	store := &nativeStore{db: db, now: func() time.Time { return now }}
	application := &applicationAdapter{nativeStore: store}
	fixture := &lifecycleFixture{db: db, application: application, owner: &applicationOwnerAdapter{applicationAdapter: application}, now: now, nextAppID: 1}
	fixture.seedBaseRecords(t)
	return fixture
}

func (f *lifecycleFixture) seedBaseRecords(t *testing.T) {
	t.Helper()
	job := &jobRecord{ID: lifecycleJobID, HrID: lifecycleHRID, Title: "Backend Engineer", Department: "Engineering", Location: "Shanghai", Status: 1}
	profile := &candidateProfileRecord{UserID: lifecycleCandidateID, RealName: "Ada Candidate", IsComplete: 1}
	resume := &resumeRecord{ID: lifecycleResumeID, UserID: lifecycleCandidateID, OSSKey: "resumes/3001/cv.pdf", FileName: "cv.pdf", FileType: "pdf", IsValid: 1, UploadedAt: f.now}
	if err := f.db.Create(job).Error; err != nil {
		t.Fatalf("seed job: %v", err)
	}
	if err := f.db.Create(profile).Error; err != nil {
		t.Fatalf("seed profile: %v", err)
	}
	if err := f.db.Create(resume).Error; err != nil {
		t.Fatalf("seed resume: %v", err)
	}
	f.seedScope(t, lifecycleHRID, "own_jobs", "", 0)
}

func (f *lifecycleFixture) seedScope(t *testing.T, userID int64, scopeKey, resourceType string, resourceID uint64) {
	t.Helper()
	scope := &userDataScopeRecord{UserID: uint64(userID), ScopeKey: scopeKey, ResourceType: resourceType, ResourceID: resourceID, AssignedAt: f.now}
	if err := f.db.Create(scope).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
}

func (f *lifecycleFixture) seedApplication(t *testing.T, statusKey string, roundNo int32, isCurrent int32) int64 {
	t.Helper()
	appID := f.nextAppID
	f.nextAppID++
	app := &applicationRecord{
		ID:        appID,
		UserID:    lifecycleCandidateID,
		JobID:     lifecycleJobID,
		ResumeID:  lifecycleResumeID,
		Status:    domainmodel.StatusKeyToLegacy[statusKey],
		StatusKey: statusKey,
		RoundNo:   roundNo,
		IsCurrent: isCurrent,
		AppliedAt: f.now,
		UpdatedAt: f.now,
	}
	if err := f.db.Create(app).Error; err != nil {
		t.Fatalf("seed application: %v", err)
	}
	return appID
}

func (f *lifecycleFixture) seedInterview(t *testing.T, applicationID int64, status string) int64 {
	t.Helper()
	row := &lifecycleInterviewScheduleRecord{
		ApplicationID: applicationID,
		InterviewerID: 9001,
		RoundNo:       1,
		Status:        status,
		CreatedAt:     f.now,
		UpdatedAt:     f.now,
	}
	if err := f.db.Create(row).Error; err != nil {
		t.Fatalf("seed interview: %v", err)
	}
	return row.ID
}

func (f *lifecycleFixture) assertApplication(t *testing.T, appID int64, statusKey string, roundNo int32, isCurrent int32) {
	t.Helper()
	var app applicationRecord
	if err := f.db.First(&app, appID).Error; err != nil {
		t.Fatalf("load app: %v", err)
	}
	if app.StatusKey != statusKey || app.Status != domainmodel.StatusKeyToLegacy[statusKey] || app.RoundNo != roundNo || app.IsCurrent != isCurrent {
		t.Fatalf("app = status %d/%s round %d current %d, want %d/%s round %d current %d",
			app.Status, app.StatusKey, app.RoundNo, app.IsCurrent,
			domainmodel.StatusKeyToLegacy[statusKey], statusKey, roundNo, isCurrent)
	}
}

func (f *lifecycleFixture) onlyTransition(t *testing.T) applicationTransitionRecord {
	t.Helper()
	var rows []applicationTransitionRecord
	if err := f.db.Find(&rows).Error; err != nil {
		t.Fatalf("load transitions: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("transition rows = %d, want 1", len(rows))
	}
	return rows[0]
}

func (f *lifecycleFixture) outboxRows(t *testing.T) []eventOutboxRecord {
	t.Helper()
	var rows []eventOutboxRecord
	if err := f.db.Order("id ASC").Find(&rows).Error; err != nil {
		t.Fatalf("load outbox: %v", err)
	}
	return rows
}

func (f *lifecycleFixture) assertOutboxPayload(t *testing.T, row eventOutboxRecord, typ string, accountType string) {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal([]byte(row.Payload), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload["receiver_account_type"] != accountType || payload["account_type"] != accountType {
		t.Fatalf("payload account type = %v/%v, want %s", payload["receiver_account_type"], payload["account_type"], accountType)
	}
	if payload["type"] != typ || payload["category"] != typ {
		t.Fatalf("payload type = %v/%v, want %s", payload["type"], payload["category"], typ)
	}
	if payload["biz_type"] != "application" || payload["related_type"] != "application" {
		t.Fatalf("payload biz type = %v/%v, want application", payload["biz_type"], payload["related_type"])
	}
}

func (f *lifecycleFixture) assertCount(t *testing.T, model any, want int64) {
	t.Helper()
	var count int64
	if err := f.db.Model(model).Count(&count).Error; err != nil {
		t.Fatalf("count %T: %v", model, err)
	}
	if count != want {
		t.Fatalf("count %T = %d, want %d", model, count, want)
	}
}

type lifecycleInterviewScheduleRecord struct {
	ID            int64 `gorm:"primaryKey"`
	ApplicationID int64
	InterviewerID int64
	RoundNo       int32
	Status        string
	CancelReason  string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     *time.Time
}

func (lifecycleInterviewScheduleRecord) TableName() string { return "interview_schedules" }
