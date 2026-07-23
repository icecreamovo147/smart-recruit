package policy

import (
	"errors"
	"testing"

	"smart-recruit-recruitment-service/internal/domain/model"
)

func TestRecruitmentPolicyValidatesResumeFile(t *testing.T) {
	if err := ValidateResumeFile("resume.pdf", "pdf"); err != nil {
		t.Fatalf("ValidateResumeFile(pdf) error = %v", err)
	}
	if err := ValidateResumeFile("resume.doc", "doc"); !errors.Is(err, ErrResumeFileTypeInvalid) {
		t.Fatalf("ValidateResumeFile(doc) error = %v, want ErrResumeFileTypeInvalid", err)
	}
}

func TestRecruitmentPolicyStrictResumeSession(t *testing.T) {
	session := &model.PresignSession{
		UserID:      42,
		OSSKey:      "resumes/tmp/42/upload/resume.pdf",
		FileType:    "pdf",
		ContentType: "application/pdf",
		MaxSize:     model.MaxResumeSizeBytes,
		Status:      "pending",
	}
	if err := ValidateStrictResumeSession(42, "upload", session.OSSKey, "pdf", 1024, session); err != nil {
		t.Fatalf("ValidateStrictResumeSession() error = %v", err)
	}
	if err := ValidateStrictResumeSession(42, "upload", session.OSSKey, "pdf", model.MaxResumeSizeBytes+1, session); !errors.Is(err, ErrResumeFileTooLarge) {
		t.Fatalf("ValidateStrictResumeSession() error = %v, want ErrResumeFileTooLarge", err)
	}
	if err := ValidateLegacyResumeKey(42, "resumes/99/resume.pdf"); !errors.Is(err, ErrResumeLegacyKeyMismatch) {
		t.Fatalf("ValidateLegacyResumeKey() error = %v, want ErrResumeLegacyKeyMismatch", err)
	}
}
