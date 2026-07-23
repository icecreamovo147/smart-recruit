package mq

import (
	"context"
	"testing"

	"smart-recruit-notification-service/internal/application/command"
)

func TestNotificationConsumerHandlesCompatiblePayloads(t *testing.T) {
	tests := []struct {
		name string
		body string
		want command.NotificationMessage
	}{
		{
			name: "canonical top-level payload",
			body: `{
				"event_id":"evt-canonical",
				"idempotency_key":"idem-canonical",
				"receiver_id":10,
				"receiver_account_type":"staff",
				"receiver_role":2,
				"type":"new_application",
				"title":"新的岗位投递",
				"content":"候选人投递了岗位",
				"link":"/hr/jobs/1/applications",
				"biz_type":"application",
				"biz_id":100
			}`,
			want: command.NotificationMessage{
				EventID:             "evt-canonical",
				IdempotencyKey:      "idem-canonical",
				ReceiverID:          10,
				ReceiverAccountType: "staff",
				ReceiverRole:        2,
				Type:                "new_application",
				Title:               "新的岗位投递",
				Content:             "候选人投递了岗位",
				Link:                "/hr/jobs/1/applications",
				BizType:             "application",
				BizID:               100,
			},
		},
		{
			name: "nested payload envelope",
			body: `{
				"event_id":"evt-nested",
				"idempotency_key":"idem-nested",
				"type":"notification.create",
				"payload":{
					"receiver_id":11,
					"receiver_account_type":"candidate",
					"type":"application_approved",
					"title":"投递进展更新",
					"content":"筛选已通过",
					"link":"/applications",
					"biz_type":"application",
					"biz_id":101
				}
			}`,
			want: command.NotificationMessage{
				EventID:             "evt-nested",
				IdempotencyKey:      "idem-nested",
				ReceiverID:          11,
				ReceiverAccountType: "candidate",
				Type:                "application_approved",
				Title:               "投递进展更新",
				Content:             "筛选已通过",
				Link:                "/applications",
				BizType:             "application",
				BizID:               101,
			},
		},
		{
			name: "legacy user id category payload",
			body: `{
				"event_id":"evt-legacy",
				"user_id":12,
				"account_type":"staff",
				"category":"new_application",
				"title":"旧格式通知",
				"content":"旧格式消息体",
				"link":"/legacy",
				"related_type":"application",
				"related_id":102,
				"related_title":"后端工程师",
				"extra":"{}"
			}`,
			want: command.NotificationMessage{
				EventID:             "evt-legacy",
				ReceiverID:          12,
				ReceiverAccountType: "staff",
				Type:                "new_application",
				Title:               "旧格式通知",
				Content:             "旧格式消息体",
				Link:                "/legacy",
				BizType:             "application",
				BizID:               102,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &captureNotificationHandler{}
			consumer := NewNotificationConsumer(handler, nil)

			if err := consumer.handle(context.Background(), []byte(tt.body)); err != nil {
				t.Fatalf("handle returned error: %v", err)
			}
			if len(handler.messages) != 1 {
				t.Fatalf("handled messages = %d, want 1", len(handler.messages))
			}
			assertNotificationMessage(t, handler.messages[0], tt.want)
		})
	}
}

func TestEmailConsumerHandlesNestedPayload(t *testing.T) {
	handler := &captureEmailHandler{}
	consumer := NewEmailConsumer(handler, nil)
	body := `{
		"event_id":"evt-email",
		"idempotency_key":"idem-email",
		"type":"email.send",
		"payload":{
			"receiver_id":20,
			"receiver_account_type":"staff",
			"type":"new_application",
			"title":"新的岗位投递",
			"content":"候选人投递了岗位",
			"link":"/hr/jobs/1/applications",
			"biz_type":"application",
			"biz_id":200,
			"job_title":"后端工程师",
			"recipient_name":"招聘负责人"
		}
	}`

	if err := consumer.handle(context.Background(), []byte(body)); err != nil {
		t.Fatalf("handle returned error: %v", err)
	}
	if len(handler.messages) != 1 {
		t.Fatalf("handled messages = %d, want 1", len(handler.messages))
	}
	got := handler.messages[0]
	if got.EventID != "evt-email" ||
		got.IdempotencyKey != "idem-email" ||
		got.ReceiverID != 20 ||
		got.ReceiverAccountType != "staff" ||
		got.Type != "new_application" ||
		got.Title != "新的岗位投递" ||
		got.Content != "候选人投递了岗位" ||
		got.Link != "/hr/jobs/1/applications" ||
		got.BizType != "application" ||
		got.BizID != 200 ||
		got.JobTitle != "后端工程师" ||
		got.RecipientName != "招聘负责人" {
		t.Fatalf("email message mismatch: %#v", got)
	}
}

func assertNotificationMessage(t *testing.T, got, want command.NotificationMessage) {
	t.Helper()
	if got.EventID != want.EventID ||
		got.IdempotencyKey != want.IdempotencyKey ||
		got.ReceiverID != want.ReceiverID ||
		got.ReceiverAccountType != want.ReceiverAccountType ||
		got.ReceiverRole != want.ReceiverRole ||
		got.Type != want.Type ||
		got.Title != want.Title ||
		got.Content != want.Content ||
		got.Link != want.Link ||
		got.BizType != want.BizType ||
		got.BizID != want.BizID {
		t.Fatalf("notification message mismatch:\n got: %#v\nwant: %#v", got, want)
	}
}

type captureNotificationHandler struct {
	messages []command.NotificationMessage
}

func (h *captureNotificationHandler) HandleNotificationMessage(_ context.Context, msg command.NotificationMessage) error {
	h.messages = append(h.messages, msg)
	return nil
}

type captureEmailHandler struct {
	messages []command.EmailMessage
}

func (h *captureEmailHandler) HandleEmailMessage(_ context.Context, msg command.EmailMessage) error {
	h.messages = append(h.messages, msg)
	return nil
}
