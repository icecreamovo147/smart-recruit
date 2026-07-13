package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"smart-recruit-recruitment-service/internal/application/command"
	"smart-recruit-recruitment-service/internal/application/port"
	"smart-recruit-recruitment-service/internal/domain/model"
	"smart-recruit-recruitment-service/internal/domain/policy"
	"smart-recruit-recruitment-service/internal/domain/repository"
)

func TestJobServiceCreateAndLifecyclePreservesScopeSemantics(t *testing.T) {
	ctx := context.Background()
	jobs := &fakeJobRepository{
		departments: map[int64]*model.Department{10: {ID: 10, FullName: "Engineering / AI"}},
		locations:   map[int64]*model.JobLocation{20: {ID: 20, Name: "Shanghai"}},
	}
	scopes := &fakeJobScopeChecker{scope: repository.JobScope{Level: repository.JobScopeOwned}}
	validator := &fakeDepartmentLocationValidator{}
	svc, err := NewJobService(jobs, scopes, validator)
	if err != nil {
		t.Fatalf("NewJobService() error = %v", err)
	}

	created, err := svc.CreateJob(ctx, command.CreateJob{
		HRID:         7,
		Title:        " Senior Backend Engineer ",
		DepartmentID: 10,
		LocationID:   20,
		SalaryRange:  "30k-45k",
	})
	if err != nil {
		t.Fatalf("CreateJob() error = %v", err)
	}
	if created.JobID != 1001 {
		t.Fatalf("JobID = %d, want 1001", created.JobID)
	}
	if jobs.created.Title != "Senior Backend Engineer" {
		t.Fatalf("Title = %q, want trimmed title", jobs.created.Title)
	}
	if jobs.created.Department != "Engineering / AI" || jobs.created.Location != "Shanghai" {
		t.Fatalf("job taxonomy snapshots = %q/%q", jobs.created.Department, jobs.created.Location)
	}
	if jobs.created.Status != model.JobStatusOnline {
		t.Fatalf("created status = %d, want online", jobs.created.Status)
	}
	if validator.calledWith != [2]int64{10, 20} {
		t.Fatalf("validator calledWith = %v", validator.calledWith)
	}

	if err := svc.SetJobStatus(ctx, command.SetJobStatus{HRID: 7, JobID: 1001, Online: false}); err != nil {
		t.Fatalf("offline SetJobStatus() error = %v", err)
	}
	if jobs.ownedStatus != model.JobStatusOffline {
		t.Fatalf("offline status = %d, want offline", jobs.ownedStatus)
	}
	scopes.scope = repository.JobScope{Level: repository.JobScopeFull}
	if err := svc.SetJobStatus(ctx, command.SetJobStatus{HRID: 7, JobID: 1001, Online: true}); err != nil {
		t.Fatalf("online SetJobStatus() error = %v", err)
	}
	if jobs.anyStatus != model.JobStatusOnline {
		t.Fatalf("online status = %d, want online", jobs.anyStatus)
	}
}

func TestCandidateResumeServiceProfilePresignAndConfirm(t *testing.T) {
	ctx := context.Background()
	storage := newFakeObjectStorage()
	resumes := &fakeResumeRepository{}
	outbox := &fakeOutboxPublisher{}
	usageLogs := &fakeUsageLogRepository{}
	svc, err := NewCandidateResumeService(CandidateResumeDeps{
		Profiles:  &fakeProfileRepository{},
		Resumes:   resumes,
		Storage:   storage,
		Outbox:    outbox,
		UsageLogs: usageLogs,
		UploadIDs: port.UploadIDFunc(func() (string, error) { return "upload-123", nil }),
		Clock:     fixedClock{now: time.Unix(1710000000, 0)},
	})
	if err != nil {
		t.Fatalf("NewCandidateResumeService() error = %v", err)
	}

	profile, err := svc.UpdateProfile(ctx, command.UpdateCandidateProfile{
		UserID:         42,
		RealName:       "Ada",
		Phone:          "13800000000",
		Education:      "Bachelor",
		School:         "THU",
		WorkExperience: "5 years",
		Skills:         "Go, Vue",
	})
	if err != nil {
		t.Fatalf("UpdateProfile() error = %v", err)
	}
	if profile.Profile.IsComplete != 1 {
		t.Fatalf("profile complete = %d, want 1", profile.Profile.IsComplete)
	}

	presign, err := svc.PresignResumeUpload(ctx, command.PresignResumeUpload{
		UserID:   42,
		FileName: "Ada Resume.pdf",
		FileType: "pdf",
	})
	if err != nil {
		t.Fatalf("PresignResumeUpload() error = %v", err)
	}
	wantTmpKey := "resumes/tmp/42/upload-123/Ada_Resume.pdf"
	if presign.UploadID != "upload-123" || presign.OSSKey != wantTmpKey {
		t.Fatalf("presign = %+v, want upload-123/%s", presign, wantTmpKey)
	}
	if storage.savedSessions["upload-123"].ContentType != "application/pdf" {
		t.Fatalf("saved content type = %q", storage.savedSessions["upload-123"].ContentType)
	}
	assertUsageLog(t, usageLogs.entries[0], "oss_presign", "/candidate/resume/presign", wantTmpKey, model.MaxResumeSizeBytes)

	confirmed, err := svc.ConfirmResumeUpload(ctx, command.ConfirmResumeUpload{
		UserID:        42,
		UploadID:      "upload-123",
		OSSKey:        wantTmpKey,
		FileName:      "Ada Resume.pdf",
		FileType:      "pdf",
		FileSize:      1024,
		StrictSession: true,
	})
	if err != nil {
		t.Fatalf("ConfirmResumeUpload() error = %v", err)
	}
	if confirmed.ResumeID != 9001 {
		t.Fatalf("ResumeID = %d, want 9001", confirmed.ResumeID)
	}
	wantPermanentKey := "resumes/42/1710000000_Ada_Resume.pdf"
	if resumes.saved.OSSKey != wantPermanentKey || resumes.saved.FileType != "pdf" || resumes.saved.IsValid != 1 {
		t.Fatalf("saved resume = %+v", resumes.saved)
	}
	if storage.copiedFrom != wantTmpKey || storage.copiedTo != wantPermanentKey || storage.deletedKey != wantTmpKey {
		t.Fatalf("storage copy/delete = %s -> %s deleted %s", storage.copiedFrom, storage.copiedTo, storage.deletedKey)
	}
	if len(outbox.events) != 1 {
		t.Fatalf("outbox events = %d, want 1", len(outbox.events))
	}
	event := outbox.events[0]
	if event.Topic != "resume.parse" || event.AggregateType != "resume" || event.AggregateID != uint64(9001) || event.EventType != "resume.parse" {
		t.Fatalf("outbox event = %+v", event)
	}
	payload, ok := event.Payload.(model.ResumeParsePayload)
	if !ok {
		t.Fatalf("outbox payload type = %T", event.Payload)
	}
	if payload.ResumeID != 9001 || payload.FileType != "pdf" || payload.OSSKey != wantPermanentKey {
		t.Fatalf("outbox payload = %+v", payload)
	}
	if !outbox.signaled {
		t.Fatal("outbox was not signaled")
	}
	assertUsageLog(t, usageLogs.entries[1], "oss_confirm", "/candidate/resume/confirm", wantTmpKey, 1024)
}

func TestCandidateResumeServiceRejectsInvalidStrictSession(t *testing.T) {
	ctx := context.Background()
	storage := newFakeObjectStorage()
	storage.savedSessions["upload-123"] = model.PresignSession{
		UserID:      99,
		OSSKey:      "resumes/tmp/99/upload-123/wrong.pdf",
		FileType:    "pdf",
		ContentType: "application/pdf",
		MaxSize:     model.MaxResumeSizeBytes,
		Status:      "pending",
	}
	svc, err := NewCandidateResumeService(CandidateResumeDeps{
		Resumes:   &fakeResumeRepository{},
		Storage:   storage,
		UploadIDs: port.UploadIDFunc(func() (string, error) { return "upload-123", nil }),
	})
	if err != nil {
		t.Fatalf("NewCandidateResumeService() error = %v", err)
	}

	_, err = svc.ConfirmResumeUpload(ctx, command.ConfirmResumeUpload{
		UserID:        42,
		UploadID:      "upload-123",
		OSSKey:        "resumes/tmp/42/upload-123/resume.pdf",
		FileName:      "resume.pdf",
		FileType:      "pdf",
		FileSize:      1024,
		StrictSession: true,
	})
	if !errors.Is(err, policy.ErrResumeSessionUser) {
		t.Fatalf("ConfirmResumeUpload() error = %v, want ErrResumeSessionUser", err)
	}
}

func assertUsageLog(t *testing.T, got model.UsageLogEntry, serviceType, endpoint, objectKey string, objectSize int64) {
	t.Helper()
	if got.UserID != 42 || got.Role != model.CandidateLegacyRole || got.ServiceType != serviceType || got.Endpoint != endpoint ||
		got.Provider != "fake-oss" || got.ObjectKey != objectKey || got.ObjectSize != objectSize || got.Status != "ok" {
		t.Fatalf("usage log = %+v", got)
	}
}

type fakeJobRepository struct {
	departments map[int64]*model.Department
	locations   map[int64]*model.JobLocation
	created     model.Job
	ownedStatus int32
	anyStatus   int32
	scopeStatus int32
}

func (r *fakeJobRepository) Create(_ context.Context, job *model.Job) error {
	job.ID = 1001
	r.created = *job
	return nil
}

func (r *fakeJobRepository) SetStatusOwned(_ context.Context, _, _ int64, status int32) (int64, error) {
	r.ownedStatus = status
	return 1, nil
}

func (r *fakeJobRepository) SetStatusAny(_ context.Context, _ int64, status int32) (int64, error) {
	r.anyStatus = status
	return 1, nil
}

func (r *fakeJobRepository) SetStatusInScope(_ context.Context, _ int64, _, _ []uint64, status int32) (int64, error) {
	r.scopeStatus = status
	return 1, nil
}

func (r *fakeJobRepository) LookupDepartment(_ context.Context, departmentID int64) (*model.Department, error) {
	return r.departments[departmentID], nil
}

func (r *fakeJobRepository) LookupLocation(_ context.Context, locationID int64) (*model.JobLocation, error) {
	return r.locations[locationID], nil
}

type fakeJobScopeChecker struct {
	scope repository.JobScope
}

func (c *fakeJobScopeChecker) CheckJobScope(context.Context, int64, int64) (repository.JobScope, error) {
	return c.scope, nil
}

type fakeDepartmentLocationValidator struct {
	calledWith [2]int64
}

func (v *fakeDepartmentLocationValidator) ValidateDepartmentLocation(_ context.Context, departmentID, locationID int64) error {
	v.calledWith = [2]int64{departmentID, locationID}
	return nil
}

type fakeProfileRepository struct {
	saved model.CandidateProfile
}

func (r *fakeProfileRepository) GetByUserID(context.Context, int64) (*model.CandidateProfile, error) {
	return nil, nil
}

func (r *fakeProfileRepository) Upsert(_ context.Context, profile *model.CandidateProfile) error {
	r.saved = *profile
	return nil
}

type fakeResumeRepository struct {
	saved model.Resume
}

func (r *fakeResumeRepository) GetValidByUserID(context.Context, int64) (*model.Resume, error) {
	return nil, nil
}

func (r *fakeResumeRepository) ConfirmUpload(_ context.Context, resume *model.Resume, afterCreate func(resumeID int64) error) error {
	resume.ID = 9001
	r.saved = *resume
	if afterCreate != nil {
		return afterCreate(resume.ID)
	}
	return nil
}

type fakeObjectStorage struct {
	savedSessions map[string]model.PresignSession
	copiedFrom    string
	copiedTo      string
	deletedKey    string
}

func newFakeObjectStorage() *fakeObjectStorage {
	return &fakeObjectStorage{savedSessions: map[string]model.PresignSession{}}
}

func (s *fakeObjectStorage) GeneratePresignedPutURL(ossKey, contentType string) (string, int64, error) {
	return "https://upload.example/" + ossKey + "?ct=" + contentType, 1710000900, nil
}

func (s *fakeObjectStorage) SavePresignSession(_ context.Context, uploadID string, session model.PresignSession) error {
	s.savedSessions[uploadID] = session
	return nil
}

func (s *fakeObjectStorage) GetAndDeletePresignSession(_ context.Context, uploadID string) (*model.PresignSession, error) {
	session, ok := s.savedSessions[uploadID]
	if !ok {
		return nil, errors.New("missing session")
	}
	delete(s.savedSessions, uploadID)
	return &session, nil
}

func (s *fakeObjectStorage) VerifyObject(context.Context, string) error { return nil }

func (s *fakeObjectStorage) VerifyObjectSize(context.Context, string, int64) error { return nil }

func (s *fakeObjectStorage) CopyObject(_ context.Context, srcKey, dstKey string) error {
	s.copiedFrom = srcKey
	s.copiedTo = dstKey
	return nil
}

func (s *fakeObjectStorage) DeleteObject(_ context.Context, ossKey string) error {
	s.deletedKey = ossKey
	return nil
}

func (s *fakeObjectStorage) ProviderName() string { return "fake-oss" }

type fakeOutboxPublisher struct {
	events   []model.OutboxEvent
	signaled bool
}

func (p *fakeOutboxPublisher) WriteEvent(_ context.Context, event model.OutboxEvent) error {
	p.events = append(p.events, event)
	return nil
}

func (p *fakeOutboxPublisher) Signal() { p.signaled = true }

type fakeUsageLogRepository struct {
	entries []model.UsageLogEntry
}

func (r *fakeUsageLogRepository) CreateUsageLog(_ context.Context, entry model.UsageLogEntry) error {
	r.entries = append(r.entries, entry)
	return nil
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time { return c.now }
