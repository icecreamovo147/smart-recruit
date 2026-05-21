package service

import "testing"

func TestDirectHRAssistantReplyGreeting(t *testing.T) {
	reply, ok := directHRAssistantReply("你好")
	if !ok {
		t.Fatal("expected greeting to be handled without tools")
	}
	if reply == "" {
		t.Fatal("expected non-empty direct reply")
	}
}

func TestDirectHRAssistantReplyHelp(t *testing.T) {
	reply, ok := directHRAssistantReply("你能做什么？")
	if !ok {
		t.Fatal("expected help request to be handled without tools")
	}
	if reply == "" {
		t.Fatal("expected non-empty direct reply")
	}
}

func TestDirectHRAssistantReplyRecruitingQuestion(t *testing.T) {
	if reply, ok := directHRAssistantReply("今天有多少投递？"); ok {
		t.Fatalf("expected recruiting data question to use tools, got direct reply: %q", reply)
	}
}
