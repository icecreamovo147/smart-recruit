package persistence

import (
	"context"
	"strconv"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"smart-recruit-commons/oss"
	sharedauthz "smart-recruit-commons/pkg/authz"
	"smart-recruit-platform-go/errs"
	"smart-recruit-platform-go/metadata"
	"smart-recruit-proto/recruitment/pb"
	domainmodel "smart-recruit-recruitment-service/internal/domain/model"
)

func TestNativeRecruitmentScopeOwnJobs(t *testing.T) {
	ctx := context.Background()
	fixture := newScopeFixture(t)
	fixture.seedScope(t, 101, sharedauthz.ScopeOwnJobs, "", 0)
	fixture.seedJob(t, 1001, 101, 10, 20)
	fixture.seedJob(t, 1002, 202, 10, 20)

	list, err := fixture.job.ListHRJobs(ctx, &pb.ListHRJobsRequest{HrId: 101, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListHRJobs() error = %v", err)
	}
	if list.Code != errs.OK || list.Total != 1 || len(list.List) != 1 || list.List[0].JobId != 1001 {
		t.Fatalf("ListHRJobs() = code %d total %d ids %v, want only 1001", list.Code, list.Total, jobIDs(list.List))
	}

	assertCommonOK(t, func() (*pb.CommonResponse, error) {
		return fixture.job.UpdateJob(ctx, &pb.UpdateJobRequest{HrId: 101, JobId: 1001, Title: "Owned Updated"})
	})
	assertCommonCode(t, errs.ErrForbidden, func() (*pb.CommonResponse, error) {
		return fixture.job.UpdateJob(ctx, &pb.UpdateJobRequest{HrId: 101, JobId: 1002, Title: "Denied"})
	})
	assertCommonOK(t, func() (*pb.CommonResponse, error) {
		return fixture.job.OfflineJob(ctx, &pb.OfflineJobRequest{HrId: 101, JobId: 1001})
	})
	fixture.assertJobStatus(t, 1001, 0)
	assertCommonOK(t, func() (*pb.CommonResponse, error) {
		return fixture.job.OnlineJob(ctx, &pb.OfflineJobRequest{HrId: 101, JobId: 1001})
	})
	fixture.assertJobStatus(t, 1001, 1)
	assertCommonCode(t, errs.ErrForbidden, func() (*pb.CommonResponse, error) {
		return fixture.job.OfflineJob(ctx, &pb.OfflineJobRequest{HrId: 101, JobId: 1002})
	})
	fixture.assertJobStatus(t, 1002, 1)
}

func TestNativeRecruitmentScopeFullAccess(t *testing.T) {
	for _, scopeKey := range []string{sharedauthz.ScopeRecruitingAll, sharedauthz.ScopeSystemAll} {
		t.Run(scopeKey, func(t *testing.T) {
			ctx := context.Background()
			fixture := newScopeFixture(t)
			fixture.seedScope(t, 101, scopeKey, "", 0)
			fixture.seedJob(t, 1001, 101, 10, 20)
			fixture.seedJob(t, 1002, 202, 11, 21)

			list, err := fixture.job.ListHRJobs(ctx, &pb.ListHRJobsRequest{HrId: 101, Page: 1, PageSize: 10})
			if err != nil {
				t.Fatalf("ListHRJobs() error = %v", err)
			}
			if list.Code != errs.OK || list.Total != 2 {
				t.Fatalf("ListHRJobs() = code %d total %d, want OK total 2", list.Code, list.Total)
			}
			assertCommonOK(t, func() (*pb.CommonResponse, error) {
				return fixture.job.UpdateJob(ctx, &pb.UpdateJobRequest{HrId: 101, JobId: 1002, Title: "Full Updated"})
			})
			assertCommonOK(t, func() (*pb.CommonResponse, error) {
				return fixture.job.OfflineJob(ctx, &pb.OfflineJobRequest{HrId: 101, JobId: 1002})
			})
			fixture.assertJobStatus(t, 1002, 0)
		})
	}
}

func TestNativeRecruitmentScopeDepartmentLocationOR(t *testing.T) {
	ctx := context.Background()
	fixture := newScopeFixture(t)
	fixture.seedScope(t, 303, sharedauthz.ScopeDepartment, "department", 10)
	fixture.seedScope(t, 303, sharedauthz.ScopeLocation, "location", 20)
	fixture.seedJob(t, 1001, 201, 10, 99)
	fixture.seedJob(t, 1002, 202, 99, 20)
	fixture.seedJob(t, 1003, 203, 11, 21)

	list, err := fixture.job.ListHRJobs(ctx, &pb.ListHRJobsRequest{HrId: 303, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListHRJobs() error = %v", err)
	}
	if list.Code != errs.OK || list.Total != 2 {
		t.Fatalf("ListHRJobs() = code %d total %d ids %v, want jobs 1001 and 1002", list.Code, list.Total, jobIDs(list.List))
	}
	assertCommonOK(t, func() (*pb.CommonResponse, error) {
		return fixture.job.UpdateJob(ctx, &pb.UpdateJobRequest{HrId: 303, JobId: 1001, Title: "Dept Updated"})
	})
	assertCommonOK(t, func() (*pb.CommonResponse, error) {
		return fixture.job.OfflineJob(ctx, &pb.OfflineJobRequest{HrId: 303, JobId: 1002})
	})
	assertCommonCode(t, errs.ErrForbidden, func() (*pb.CommonResponse, error) {
		return fixture.job.UpdateJob(ctx, &pb.UpdateJobRequest{HrId: 303, JobId: 1003, Title: "Denied"})
	})
	fixture.assertJobStatus(t, 1002, 0)
	fixture.assertJobTitle(t, 1003, "Job 1003")
}

func TestNativeRecruitmentScopeNoValidScopeFailsClosed(t *testing.T) {
	ctx := context.Background()
	fixture := newScopeFixture(t)
	fixture.seedScope(t, 404, sharedauthz.ScopeSelf, "", 0)
	fixture.seedJob(t, 1001, 404, 10, 20)

	list, err := fixture.job.ListHRJobs(ctx, &pb.ListHRJobsRequest{HrId: 404, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListHRJobs() error = %v", err)
	}
	if list.Code != errs.ErrForbidden || len(list.List) != 0 {
		t.Fatalf("ListHRJobs() = code %d len %d, want forbidden empty", list.Code, len(list.List))
	}
	assertCommonCode(t, errs.ErrForbidden, func() (*pb.CommonResponse, error) {
		return fixture.job.UpdateJob(ctx, &pb.UpdateJobRequest{HrId: 404, JobId: 1001, Title: "Denied"})
	})
	assertCommonCode(t, errs.ErrForbidden, func() (*pb.CommonResponse, error) {
		return fixture.job.OfflineJob(ctx, &pb.OfflineJobRequest{HrId: 404, JobId: 1001})
	})
	fixture.assertJobStatus(t, 1001, 1)
}

func TestNativeRecruitmentScopeApplicationsDeniedAcrossScope(t *testing.T) {
	ctx := context.Background()
	fixture := newScopeFixture(t)
	fixture.seedScope(t, 101, sharedauthz.ScopeOwnJobs, "", 0)
	fixture.seedJob(t, 1001, 101, 10, 20)
	fixture.seedJob(t, 1002, 202, 10, 20)
	appID := fixture.seedApplication(t, 1002, 3001, domainmodel.StatusKeyApplied)
	fixture.seedTransition(t, appID, domainmodel.StatusKeyApplied, domainmodel.StatusKeyViewed, 202)

	list, err := fixture.application.ListJobApplications(ctx, &pb.ListJobApplicationsRequest{HrId: 101, JobId: 1002, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListJobApplications() error = %v", err)
	}
	if list.Code != errs.ErrForbidden {
		t.Fatalf("ListJobApplications() code = %d, want forbidden", list.Code)
	}
	update, err := fixture.application.UpdateApplicationStatus(ctx, &pb.UpdateApplicationStatusRequest{
		HrId:          101,
		ApplicationId: appID,
		StatusKey:     domainmodel.StatusKeyScreenPassed,
		Reason:        "qualified",
	})
	if err != nil {
		t.Fatalf("UpdateApplicationStatus() error = %v", err)
	}
	if update.Code != errs.ErrForbidden {
		t.Fatalf("UpdateApplicationStatus() code = %d, want forbidden", update.Code)
	}
	fixture.assertApplicationStatus(t, appID, domainmodel.StatusKeyApplied)

	transitions, err := fixture.application.ListApplicationStatusTransitions(ctx, &pb.ListApplicationStatusTransitionsRequest{HrId: 101, ApplicationId: appID})
	if err != nil {
		t.Fatalf("ListApplicationStatusTransitions() error = %v", err)
	}
	if transitions.Code != errs.ErrForbidden {
		t.Fatalf("ListApplicationStatusTransitions() code = %d, want forbidden", transitions.Code)
	}
}

func TestNativeRecruitmentScopeApplicationLifecycleActors(t *testing.T) {
	ctx := context.Background()
	fixture := newScopeFixture(t)
	fixture.seedScope(t, 101, sharedauthz.ScopeOwnJobs, "", 0)
	fixture.seedJob(t, 1001, 101, 10, 20)
	fixture.seedJob(t, 1002, 202, 10, 20)
	staffDeniedAppID := fixture.seedApplication(t, 1002, 3001, domainmodel.StatusKeyApplied)

	staffSnapshotCtx := metadata.WithAuthActor(ctx, 101, "staff")
	snapshot, err := fixture.owner.GetApplicationSnapshot(staffSnapshotCtx, &pb.GetApplicationSnapshotRequest{ApplicationId: staffDeniedAppID})
	if err != nil {
		t.Fatalf("GetApplicationSnapshot() error = %v", err)
	}
	if snapshot.Code != errs.ErrForbidden {
		t.Fatalf("GetApplicationSnapshot() code = %d, want forbidden", snapshot.Code)
	}
	staffTransition, err := fixture.owner.ApplyApplicationLifecycleTransition(ctx, &pb.ApplyApplicationLifecycleTransitionRequest{
		ActorUserId:       101,
		ActorAccountType:  "staff",
		ApplicationId:     staffDeniedAppID,
		ExpectedStatusKey: domainmodel.StatusKeyApplied,
		TargetStatusKey:   domainmodel.StatusKeyScreenPassed,
		Reason:            "qualified",
	})
	if err != nil {
		t.Fatalf("ApplyApplicationLifecycleTransition(staff) error = %v", err)
	}
	if staffTransition.Code != errs.ErrForbidden || staffTransition.Changed {
		t.Fatalf("ApplyApplicationLifecycleTransition(staff) = %+v, want forbidden unchanged", staffTransition)
	}
	fixture.assertApplicationStatus(t, staffDeniedAppID, domainmodel.StatusKeyApplied)

	serviceAppID := fixture.seedApplication(t, 1002, 3002, domainmodel.StatusKeyApplied)
	serviceTransition, err := fixture.owner.ApplyApplicationLifecycleTransition(ctx, &pb.ApplyApplicationLifecycleTransitionRequest{
		ActorUserId:       9001,
		ActorAccountType:  "service",
		ApplicationId:     serviceAppID,
		ExpectedStatusKey: domainmodel.StatusKeyApplied,
		TargetStatusKey:   domainmodel.StatusKeyScreenPassed,
		Reason:            "interview scheduled",
	})
	if err != nil {
		t.Fatalf("ApplyApplicationLifecycleTransition(service) error = %v", err)
	}
	if serviceTransition.Code != errs.OK || !serviceTransition.Changed {
		t.Fatalf("ApplyApplicationLifecycleTransition(service) = %+v, want OK changed", serviceTransition)
	}

	candidateAppID := fixture.seedApplication(t, 1002, 3003, domainmodel.StatusKeyOfferSent)
	candidateTransition, err := fixture.owner.ApplyApplicationLifecycleTransition(ctx, &pb.ApplyApplicationLifecycleTransitionRequest{
		ActorUserId:       3003,
		ActorAccountType:  "candidate",
		ApplicationId:     candidateAppID,
		ExpectedStatusKey: domainmodel.StatusKeyOfferSent,
		TargetStatusKey:   domainmodel.StatusKeyOfferRejected,
		Reason:            "accepted another offer",
		CloseCurrentRound: true,
	})
	if err != nil {
		t.Fatalf("ApplyApplicationLifecycleTransition(candidate) error = %v", err)
	}
	if candidateTransition.Code != errs.OK || !candidateTransition.Changed {
		t.Fatalf("ApplyApplicationLifecycleTransition(candidate) = %+v, want OK changed", candidateTransition)
	}
	fixture.assertApplicationStatus(t, candidateAppID, domainmodel.StatusKeyOfferRejected)
}

type scopeFixture struct {
	db          *gorm.DB
	job         *jobAdapter
	application *applicationAdapter
	owner       *applicationOwnerAdapter
	now         time.Time
	nextAppID   int64
}

func newScopeFixture(t *testing.T) *scopeFixture {
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
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	now := time.Date(2026, 7, 14, 10, 0, 0, 0, time.UTC)
	store := &nativeStore{db: db, storage: scopeStorage{}, now: func() time.Time { return now }}
	application := &applicationAdapter{nativeStore: store}
	return &scopeFixture{
		db:          db,
		job:         &jobAdapter{nativeStore: store},
		application: application,
		owner:       &applicationOwnerAdapter{applicationAdapter: application},
		now:         now,
		nextAppID:   1,
	}
}

func (f *scopeFixture) seedScope(t *testing.T, userID int64, scopeKey, resourceType string, resourceID uint64) {
	t.Helper()
	row := &userDataScopeRecord{UserID: uint64(userID), ScopeKey: scopeKey, ResourceType: resourceType, ResourceID: resourceID, AssignedAt: f.now}
	if err := f.db.Create(row).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
}

func (f *scopeFixture) seedJob(t *testing.T, jobID, hrID, departmentID, locationID int64) {
	t.Helper()
	row := &jobRecord{
		ID:           jobID,
		HrID:         hrID,
		Title:        "Job " + strconv.FormatInt(jobID, 10),
		Department:   "Department",
		DepartmentID: positivePtr(departmentID),
		Location:     "Location",
		LocationID:   positivePtr(locationID),
		Status:       1,
		CreatedAt:    f.now.Add(time.Duration(jobID) * time.Second),
		UpdatedAt:    f.now,
	}
	if err := f.db.Create(row).Error; err != nil {
		t.Fatalf("seed job: %v", err)
	}
}

func (f *scopeFixture) seedApplication(t *testing.T, jobID, userID int64, statusKey string) int64 {
	t.Helper()
	if err := f.db.FirstOrCreate(&candidateProfileRecord{UserID: userID}, candidateProfileRecord{UserID: userID, RealName: "Candidate", IsComplete: 1}).Error; err != nil {
		t.Fatalf("seed profile: %v", err)
	}
	resumeID := userID + 10000
	if err := f.db.FirstOrCreate(&resumeRecord{ID: resumeID}, resumeRecord{ID: resumeID, UserID: userID, FileName: "cv.pdf", FileType: "pdf", IsValid: 1, UploadedAt: f.now}).Error; err != nil {
		t.Fatalf("seed resume: %v", err)
	}
	appID := f.nextAppID
	f.nextAppID++
	row := &applicationRecord{
		ID:        appID,
		UserID:    userID,
		JobID:     jobID,
		ResumeID:  resumeID,
		Status:    domainmodel.StatusKeyToLegacy[statusKey],
		StatusKey: statusKey,
		RoundNo:   1,
		IsCurrent: 1,
		AppliedAt: f.now,
	}
	if err := f.db.Create(row).Error; err != nil {
		t.Fatalf("seed application: %v", err)
	}
	return appID
}

func (f *scopeFixture) seedTransition(t *testing.T, applicationID int64, from, to string, actorID int64) {
	t.Helper()
	row := &applicationTransitionRecord{ApplicationID: applicationID, FromStatus: from, ToStatus: to, ActorUserID: actorID, ActorAccountType: "staff", CreatedAt: f.now}
	if err := f.db.Create(row).Error; err != nil {
		t.Fatalf("seed transition: %v", err)
	}
}

func (f *scopeFixture) assertJobStatus(t *testing.T, jobID int64, want int32) {
	t.Helper()
	var job jobRecord
	if err := f.db.First(&job, jobID).Error; err != nil {
		t.Fatalf("load job: %v", err)
	}
	if job.Status != want {
		t.Fatalf("job %d status = %d, want %d", jobID, job.Status, want)
	}
}

func (f *scopeFixture) assertJobTitle(t *testing.T, jobID int64, want string) {
	t.Helper()
	var job jobRecord
	if err := f.db.First(&job, jobID).Error; err != nil {
		t.Fatalf("load job: %v", err)
	}
	if job.Title != want {
		t.Fatalf("job %d title = %q, want %q", jobID, job.Title, want)
	}
}

func (f *scopeFixture) assertApplicationStatus(t *testing.T, appID int64, want string) {
	t.Helper()
	var app applicationRecord
	if err := f.db.First(&app, appID).Error; err != nil {
		t.Fatalf("load application: %v", err)
	}
	if app.StatusKey != want || app.Status != domainmodel.StatusKeyToLegacy[want] {
		t.Fatalf("application %d status = %d/%s, want %d/%s", appID, app.Status, app.StatusKey, domainmodel.StatusKeyToLegacy[want], want)
	}
}

func assertCommonOK(t *testing.T, call func() (*pb.CommonResponse, error)) {
	t.Helper()
	resp, err := call()
	if err != nil {
		t.Fatalf("response error = %v", err)
	}
	if resp.Code != errs.OK {
		t.Fatalf("response code = %d, want OK; msg=%s", resp.Code, resp.Msg)
	}
}

func assertCommonCode(t *testing.T, want int32, call func() (*pb.CommonResponse, error)) {
	t.Helper()
	resp, err := call()
	if err != nil {
		t.Fatalf("response error = %v", err)
	}
	if resp.Code != want {
		t.Fatalf("response code = %d, want %d; msg=%s", resp.Code, want, resp.Msg)
	}
}

func jobIDs(jobs []*pb.Job) []int64 {
	ids := make([]int64, 0, len(jobs))
	for _, job := range jobs {
		ids = append(ids, job.JobId)
	}
	return ids
}

type scopeStorage struct{}

func (scopeStorage) SetPresignCache(*oss.PresignCache) {}

func (scopeStorage) ProviderName() string { return "test" }

func (scopeStorage) GeneratePresignedPutURL(string, string) (string, time.Time, error) {
	return "put-url", time.Now(), nil
}

func (scopeStorage) GeneratePresignedGetURL(ossKey string) (string, error) {
	return "get://" + ossKey, nil
}

func (scopeStorage) VerifyObject(context.Context, string) error { return nil }

func (scopeStorage) VerifyObjectSize(context.Context, string, int64) error { return nil }

func (scopeStorage) DownloadObject(context.Context, string) ([]byte, error) { return nil, nil }

func (scopeStorage) CopyObject(context.Context, string, string) error { return nil }

func (scopeStorage) DeleteObject(context.Context, string) error { return nil }

func (scopeStorage) SavePresignSession(context.Context, oss.PresignSession) (string, error) {
	return "upload-id", nil
}

func (scopeStorage) SavePresignSessionWithID(context.Context, string, oss.PresignSession) error {
	return nil
}

func (scopeStorage) GetAndDeletePresignSession(context.Context, string) (*oss.PresignSession, error) {
	return nil, nil
}
