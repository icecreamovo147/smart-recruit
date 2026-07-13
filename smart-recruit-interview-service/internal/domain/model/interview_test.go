package model

import "testing"

func TestNewScheduledInterviewAppliesDefaults(t *testing.T) {
	interview, err := NewScheduledInterview(ScheduleDetails{
		ApplicationID: 10,
		InterviewerID: 20,
		RoundNo:       2,
		CreatedBy:     30,
	})
	if err != nil {
		t.Fatalf("NewScheduledInterview returned error: %v", err)
	}
	if interview.Status != InterviewStatusScheduled {
		t.Fatalf("status=%s, want scheduled", interview.Status)
	}
	if interview.Title != "第 2 轮面试" {
		t.Fatalf("title=%q, want default round title", interview.Title)
	}
	if interview.Mode != "video" {
		t.Fatalf("mode=%q, want video", interview.Mode)
	}
}

func TestCancelRejectsAlreadyCancelledInterview(t *testing.T) {
	interview := &Interview{Status: InterviewStatusCancelled}
	if err := interview.Cancel("duplicate"); err != ErrInterviewAlreadyCancelled {
		t.Fatalf("Cancel error=%v, want ErrInterviewAlreadyCancelled", err)
	}
}
