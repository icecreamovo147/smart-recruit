package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"smart-recruit-commons/oss"
	"smart-recruit-platform-go/errs"
	"smart-recruit-platform-go/metadata"
	"smart-recruit-proto/recruitment/pb"
)

const resumeTestUserID int64 = 42

func TestCandidateResumeGetRequiresCandidateActor(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
	}{
		{name: "missing actor", ctx: context.Background()},
		{name: "mismatched user", ctx: metadata.WithAuthActor(context.Background(), 99, "candidate")},
		{name: "wrong account type", ctx: metadata.WithAuthActor(context.Background(), resumeTestUserID, "staff")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := newResumeFixture(t)
			fixture.seedResume(t, &resumeRecord{
				UserID:     resumeTestUserID,
				OSSKey:     "resumes/42/current.pdf",
				FileName:   "current.pdf",
				FileType:   "pdf",
				FileSize:   1024,
				IsValid:    1,
				UploadedAt: fixture.now,
			})

			resp, err := fixture.candidate.GetResume(tt.ctx, &pb.GetResumeRequest{UserId: resumeTestUserID})
			if err != nil {
				t.Fatalf("GetResume() error = %v", err)
			}
			if resp.Code != errs.ErrForbidden {
				t.Fatalf("GetResume() code = %d, want %d; msg=%s", resp.Code, errs.ErrForbidden, resp.Msg)
			}
			if resp.Resume != nil {
				t.Fatalf("GetResume() returned resume = %+v, want nil", resp.Resume)
			}
			if fixture.storage.getURLCalls != 0 {
				t.Fatalf("GeneratePresignedGetURL calls = %d, want 0", fixture.storage.getURLCalls)
			}
		})
	}
}

func TestCandidateResumePresignAndConfirmRejectActorMismatch(t *testing.T) {
	t.Run("presign", func(t *testing.T) {
		fixture := newResumeFixture(t)
		ctx := metadata.WithAuthActor(context.Background(), 99, "candidate")

		resp, err := fixture.candidate.PresignResumeUpload(ctx, &pb.PresignResumeUploadRequest{
			UserId:   resumeTestUserID,
			FileName: "resume.pdf",
			FileType: "pdf",
		})
		if err != nil {
			t.Fatalf("PresignResumeUpload() error = %v", err)
		}
		if resp.Code != errs.ErrForbidden {
			t.Fatalf("PresignResumeUpload() code = %d, want %d; msg=%s", resp.Code, errs.ErrForbidden, resp.Msg)
		}
		if fixture.storage.putURLCalls != 0 || fixture.storage.saveSessionCalls != 0 {
			t.Fatalf("presign storage calls put=%d save=%d, want 0/0", fixture.storage.putURLCalls, fixture.storage.saveSessionCalls)
		}
	})

	t.Run("confirm", func(t *testing.T) {
		fixture := newResumeFixture(t)
		ctx := metadata.WithAuthActor(context.Background(), 99, "candidate")

		resp, err := fixture.candidate.ConfirmResumeUpload(ctx, validConfirmRequest())
		if err != nil {
			t.Fatalf("ConfirmResumeUpload() error = %v", err)
		}
		if resp.Code != errs.ErrForbidden {
			t.Fatalf("ConfirmResumeUpload() code = %d, want %d; msg=%s", resp.Code, errs.ErrForbidden, resp.Msg)
		}
		if fixture.storage.getSessionCalls != 0 || fixture.storage.copyCalls != 0 {
			t.Fatalf("confirm storage calls session=%d copy=%d, want 0/0", fixture.storage.getSessionCalls, fixture.storage.copyCalls)
		}
	})
}

func TestCandidateResumeConfirmRequiresUploadID(t *testing.T) {
	fixture := newResumeFixture(t)
	req := validConfirmRequest()
	req.UploadId = ""

	resp, err := fixture.candidate.ConfirmResumeUpload(resumeActorCtx(), req)
	if err != nil {
		t.Fatalf("ConfirmResumeUpload() error = %v", err)
	}
	if resp.Code != errs.ErrBadRequest {
		t.Fatalf("ConfirmResumeUpload() code = %d, want %d; msg=%s", resp.Code, errs.ErrBadRequest, resp.Msg)
	}
	if fixture.storage.getSessionCalls != 0 || fixture.storage.verifyCalls != 0 || fixture.storage.copyCalls != 0 {
		t.Fatalf("storage calls session=%d verify=%d copy=%d, want 0/0/0", fixture.storage.getSessionCalls, fixture.storage.verifyCalls, fixture.storage.copyCalls)
	}
	fixture.assertResumeCount(t, 0)
	fixture.assertOutboxCount(t, 0)
}

func TestCandidateResumeConfirmRejectsStrictSessionMismatches(t *testing.T) {
	tests := []struct {
		name    string
		session oss.PresignSession
		request *pb.ConfirmResumeUploadRequest
	}{
		{
			name:    "user mismatch",
			session: validOSSSessionWith(func(session *oss.PresignSession) { session.UserID = 99 }),
			request: validConfirmRequest(),
		},
		{
			name:    "key mismatch",
			session: validOSSSessionWith(func(session *oss.PresignSession) { session.OssKey = "resumes/tmp/42/upload-123/other.pdf" }),
			request: validConfirmRequest(),
		},
		{
			name:    "type mismatch",
			session: validOSSSessionWith(func(session *oss.PresignSession) { session.FileType = "docx" }),
			request: validConfirmRequest(),
		},
		{
			name:    "status mismatch",
			session: validOSSSessionWith(func(session *oss.PresignSession) { session.Status = "confirmed" }),
			request: validConfirmRequest(),
		},
		{
			name:    "size mismatch",
			session: validOSSSessionWith(func(session *oss.PresignSession) { session.MaxSize = 1024 }),
			request: validConfirmRequestWith(func(req *pb.ConfirmResumeUploadRequest) { req.FileSize = 1025 }),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := newResumeFixture(t)
			fixture.storage.sessions[tt.request.UploadId] = tt.session

			resp, err := fixture.candidate.ConfirmResumeUpload(resumeActorCtx(), tt.request)
			if err != nil {
				t.Fatalf("ConfirmResumeUpload() error = %v", err)
			}
			if resp.Code != errs.ErrBadRequest {
				t.Fatalf("ConfirmResumeUpload() code = %d, want %d; msg=%s", resp.Code, errs.ErrBadRequest, resp.Msg)
			}
			if fixture.storage.verifyCalls != 0 || fixture.storage.copyCalls != 0 {
				t.Fatalf("storage calls verify=%d copy=%d, want 0/0", fixture.storage.verifyCalls, fixture.storage.copyCalls)
			}
			fixture.assertResumeCount(t, 0)
			fixture.assertOutboxCount(t, 0)
		})
	}
}

func TestCandidateResumeConfirmHappyPathSavesResumeAndOutbox(t *testing.T) {
	fixture := newResumeFixture(t)
	fixture.seedResume(t, &resumeRecord{
		UserID:     resumeTestUserID,
		OSSKey:     "resumes/42/old.pdf",
		FileName:   "old.pdf",
		FileType:   "pdf",
		FileSize:   512,
		IsValid:    1,
		UploadedAt: fixture.now.Add(-time.Hour),
	})
	req := validConfirmRequest()
	fixture.storage.sessions[req.UploadId] = validOSSSession()

	resp, err := fixture.candidate.ConfirmResumeUpload(resumeActorCtx(), req)
	if err != nil {
		t.Fatalf("ConfirmResumeUpload() error = %v", err)
	}
	if resp.Code != errs.OK {
		t.Fatalf("ConfirmResumeUpload() code = %d, want %d; msg=%s", resp.Code, errs.OK, resp.Msg)
	}
	if resp.ResumeId == 0 {
		t.Fatalf("ConfirmResumeUpload() resume id = 0, want saved id")
	}
	if fixture.storage.copiedFrom != req.OssKey {
		t.Fatalf("CopyObject source = %q, want %q", fixture.storage.copiedFrom, req.OssKey)
	}
	if !strings.HasPrefix(fixture.storage.copiedTo, "resumes/42/") || !strings.HasSuffix(fixture.storage.copiedTo, "_resume.pdf") {
		t.Fatalf("CopyObject destination = %q, want permanent resume key", fixture.storage.copiedTo)
	}
	if fixture.storage.deletedKey != req.OssKey {
		t.Fatalf("DeleteObject key = %q, want %q", fixture.storage.deletedKey, req.OssKey)
	}

	var resumes []resumeRecord
	if err := fixture.db.Order("uploaded_at ASC, id ASC").Find(&resumes).Error; err != nil {
		t.Fatalf("load resumes: %v", err)
	}
	if len(resumes) != 2 {
		t.Fatalf("resume count = %d, want 2", len(resumes))
	}
	if resumes[0].IsValid != 0 {
		t.Fatalf("old resume is_valid = %d, want 0", resumes[0].IsValid)
	}
	newResume := resumes[1]
	if newResume.UserID != resumeTestUserID || newResume.OSSKey != fixture.storage.copiedTo || newResume.FileName != req.FileName || newResume.FileType != "pdf" || newResume.FileSize != req.FileSize || newResume.IsValid != 1 {
		t.Fatalf("new resume = %+v, want saved permanent resume", newResume)
	}

	outbox := fixture.onlyOutbox(t)
	if outbox.EventType != "resume.parse" || outbox.RoutingKey != "resume.parse" || outbox.AggregateType != "resume" || outbox.AggregateID != uint64(newResume.ID) {
		t.Fatalf("outbox = %+v, want resume.parse for saved resume", outbox)
	}
	var payload struct {
		ResumeID int64  `json:"resume_id"`
		FileType string `json:"file_type"`
		OSSKey   string `json:"oss_key"`
	}
	if err := json.Unmarshal([]byte(outbox.Payload), &payload); err != nil {
		t.Fatalf("unmarshal outbox payload: %v", err)
	}
	if payload.ResumeID != newResume.ID || payload.FileType != "pdf" || payload.OSSKey != newResume.OSSKey {
		t.Fatalf("outbox payload = %+v, want saved resume details", payload)
	}
}

type resumeFixture struct {
	db        *gorm.DB
	candidate *candidateAdapter
	storage   *resumeStorage
	now       time.Time
}

func newResumeFixture(t *testing.T) *resumeFixture {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&resumeRecord{}, &eventOutboxRecord{}, &usageLogRecord{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	now := time.Date(2026, 7, 14, 10, 0, 0, 0, time.UTC)
	storage := newResumeStorage()
	store := &nativeStore{db: db, storage: storage, now: func() time.Time { return now }}
	return &resumeFixture{
		db:        db,
		candidate: &candidateAdapter{nativeStore: store},
		storage:   storage,
		now:       now,
	}
}

func (f *resumeFixture) seedResume(t *testing.T, row *resumeRecord) {
	t.Helper()
	if err := f.db.Create(row).Error; err != nil {
		t.Fatalf("seed resume: %v", err)
	}
}

func (f *resumeFixture) assertResumeCount(t *testing.T, want int64) {
	t.Helper()
	var count int64
	if err := f.db.Model(&resumeRecord{}).Count(&count).Error; err != nil {
		t.Fatalf("count resumes: %v", err)
	}
	if count != want {
		t.Fatalf("resume count = %d, want %d", count, want)
	}
}

func (f *resumeFixture) assertOutboxCount(t *testing.T, want int64) {
	t.Helper()
	var count int64
	if err := f.db.Model(&eventOutboxRecord{}).Count(&count).Error; err != nil {
		t.Fatalf("count outbox: %v", err)
	}
	if count != want {
		t.Fatalf("outbox count = %d, want %d", count, want)
	}
}

func (f *resumeFixture) onlyOutbox(t *testing.T) eventOutboxRecord {
	t.Helper()
	var rows []eventOutboxRecord
	if err := f.db.Find(&rows).Error; err != nil {
		t.Fatalf("load outbox: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("outbox rows = %d, want 1", len(rows))
	}
	return rows[0]
}

func resumeActorCtx() context.Context {
	return metadata.WithAuthActor(context.Background(), resumeTestUserID, "candidate")
}

func validConfirmRequest() *pb.ConfirmResumeUploadRequest {
	return validConfirmRequestWith(nil)
}

func validConfirmRequestWith(mutator func(*pb.ConfirmResumeUploadRequest)) *pb.ConfirmResumeUploadRequest {
	req := &pb.ConfirmResumeUploadRequest{
		UserId:   resumeTestUserID,
		UploadId: "upload-123",
		OssKey:   "resumes/tmp/42/upload-123/resume.pdf",
		FileName: "resume.pdf",
		FileType: "pdf",
		FileSize: 2048,
	}
	if mutator != nil {
		mutator(req)
	}
	return req
}

func validOSSSession() oss.PresignSession {
	return validOSSSessionWith(nil)
}

func validOSSSessionWith(mutator func(*oss.PresignSession)) oss.PresignSession {
	session := oss.PresignSession{
		UserID:      resumeTestUserID,
		OssKey:      "resumes/tmp/42/upload-123/resume.pdf",
		FileName:    "resume.pdf",
		FileType:    "pdf",
		ContentType: "application/pdf",
		MaxSize:     oss.MaxResumeSizeBytes,
		Status:      "pending",
	}
	if mutator != nil {
		mutator(&session)
	}
	return session
}

type resumeStorage struct {
	sessions         map[string]oss.PresignSession
	getURLCalls      int
	putURLCalls      int
	saveSessionCalls int
	getSessionCalls  int
	verifyCalls      int
	verifySizeCalls  int
	copyCalls        int
	copiedFrom       string
	copiedTo         string
	deletedKey       string
}

func newResumeStorage() *resumeStorage {
	return &resumeStorage{sessions: map[string]oss.PresignSession{}}
}

func (s *resumeStorage) SetPresignCache(*oss.PresignCache) {}

func (s *resumeStorage) ProviderName() string { return "test-oss" }

func (s *resumeStorage) GeneratePresignedPutURL(ossKey, contentType string) (string, time.Time, error) {
	s.putURLCalls++
	return "put://" + ossKey + "?content_type=" + contentType, time.Date(2026, 7, 14, 10, 15, 0, 0, time.UTC), nil
}

func (s *resumeStorage) GeneratePresignedGetURL(ossKey string) (string, error) {
	s.getURLCalls++
	return "get://" + ossKey, nil
}

func (s *resumeStorage) VerifyObject(context.Context, string) error {
	s.verifyCalls++
	return nil
}

func (s *resumeStorage) VerifyObjectSize(context.Context, string, int64) error {
	s.verifySizeCalls++
	return nil
}

func (s *resumeStorage) DownloadObject(context.Context, string) ([]byte, error) {
	return nil, nil
}

func (s *resumeStorage) CopyObject(_ context.Context, srcKey, dstKey string) error {
	s.copyCalls++
	s.copiedFrom = srcKey
	s.copiedTo = dstKey
	return nil
}

func (s *resumeStorage) DeleteObject(_ context.Context, ossKey string) error {
	s.deletedKey = ossKey
	return nil
}

func (s *resumeStorage) SavePresignSession(_ context.Context, session oss.PresignSession) (string, error) {
	s.saveSessionCalls++
	s.sessions["generated-upload-id"] = session
	return "generated-upload-id", nil
}

func (s *resumeStorage) SavePresignSessionWithID(_ context.Context, uploadID string, session oss.PresignSession) error {
	s.saveSessionCalls++
	s.sessions[uploadID] = session
	return nil
}

func (s *resumeStorage) GetAndDeletePresignSession(_ context.Context, uploadID string) (*oss.PresignSession, error) {
	s.getSessionCalls++
	session, ok := s.sessions[uploadID]
	if !ok {
		return nil, errors.New("missing session")
	}
	delete(s.sessions, uploadID)
	return &session, nil
}
