package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"smart-recruit-notification-service/internal/application/command"
	"smart-recruit-notification-service/internal/application/port"
	"smart-recruit-notification-service/internal/application/query"
	"smart-recruit-notification-service/internal/domain/model"
	"smart-recruit-notification-service/internal/domain/repository"
)

var testNow = time.Date(2026, 7, 13, 12, 0, 0, 0, time.UTC)

func TestListNotificationsPreservesValidationPaginationAndSoftFailures(t *testing.T) {
	ctx := context.Background()
	repo := &fakeNotificationRepo{
		listRows:       []model.Notification{{ID: 1}},
		listTotal:      1,
		listCursorRows: []model.Notification{{ID: 2}},
		nextCursor:     "next",
		hasMore:        true,
	}
	svc := New(Deps{Notifications: repo, Clock: fakeClock{now: testNow}})

	result, err := svc.ListNotifications(ctx, query.ListNotifications{AccountType: "candidate"})
	if err != nil {
		t.Fatalf("ListNotifications returned error: %v", err)
	}
	if result.Code != query.CodeBadRequest || result.Message != "user_id is required" {
		t.Fatalf("unexpected user validation result: %#v", result)
	}

	result, err = svc.ListNotifications(ctx, query.ListNotifications{UserID: 10})
	if err != nil {
		t.Fatalf("ListNotifications returned error: %v", err)
	}
	if result.Code != query.CodeBadRequest || result.Message != "account_type is required" {
		t.Fatalf("unexpected account validation result: %#v", result)
	}

	result, err = svc.ListNotifications(ctx, query.ListNotifications{UserID: 10, AccountType: "candidate", PageSize: 99, Cursor: "cur"})
	if err != nil {
		t.Fatalf("ListNotifications returned error: %v", err)
	}
	if repo.lastCursorLimit != 20 || result.Code != query.CodeOK || result.NextCursor != "next" || !result.HasMore || result.List[0].ID != 2 {
		t.Fatalf("unexpected cursor result: limit=%d result=%#v", repo.lastCursorLimit, result)
	}

	result, err = svc.ListNotifications(ctx, query.ListNotifications{UserID: 10, AccountType: "candidate", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListNotifications returned error: %v", err)
	}
	if repo.lastPage != 1 || repo.lastPageSize != 10 || result.Total != 1 || result.List[0].ID != 1 {
		t.Fatalf("unexpected page result: repo=%#v result=%#v", repo, result)
	}

	repo.listErr = errors.New("db down")
	result, err = svc.ListNotifications(ctx, query.ListNotifications{UserID: 10, AccountType: "candidate", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListNotifications returned error: %v", err)
	}
	if result.Total != 0 || result.List != nil {
		t.Fatalf("list DB failure should return empty result, got %#v", result)
	}
}

func TestListNotificationsVerifierFailureReturnsForbiddenResult(t *testing.T) {
	svc := New(Deps{
		Notifications: &fakeNotificationRepo{},
		ActorVerifier: fakeActorVerifier{err: errors.New("actor mismatch")},
		Clock:         fakeClock{now: testNow},
	})
	result, err := svc.ListNotifications(context.Background(), query.ListNotifications{UserID: 10, AccountType: "candidate", Cursor: "cur"})
	if err != nil {
		t.Fatalf("ListNotifications returned error: %v", err)
	}
	if result.Code != query.CodeForbidden || result.Message != "actor mismatch" {
		t.Fatalf("unexpected verifier result: %#v", result)
	}
}

func TestUnreadCountAndSummaryKeepLegacyFallbacks(t *testing.T) {
	ctx := context.Background()
	cache := newFakeCache()
	cache.counts["10:candidate"] = 7
	repo := &fakeNotificationRepo{unread: 3}
	svc := New(Deps{Notifications: repo, UnreadCache: cache, Clock: fakeClock{now: testNow}})

	unread, err := svc.UnreadNotificationCount(ctx, query.UnreadCount{UserID: 10, AccountType: "candidate"})
	if err != nil {
		t.Fatalf("UnreadNotificationCount returned error: %v", err)
	}
	if unread.Unread != 7 || repo.unreadCalls != 0 {
		t.Fatalf("cache hit should skip repository: unread=%#v calls=%d", unread, repo.unreadCalls)
	}

	cache.counts = map[string]int64{}
	repo.unreadErr = errors.New("db down")
	unread, err = svc.UnreadNotificationCount(ctx, query.UnreadCount{UserID: 10, AccountType: "candidate"})
	if err != nil {
		t.Fatalf("UnreadNotificationCount returned error: %v", err)
	}
	if unread.Unread != 0 {
		t.Fatalf("unread DB failure should return zero, got %#v", unread)
	}

	repo.unreadErr = nil
	repo.unread = 4
	repo.latestErr = errors.New("latest down")
	summary, err := svc.NotificationSummary(ctx, query.Summary{UserID: 10, AccountType: "candidate"})
	if err != nil {
		t.Fatalf("NotificationSummary returned error: %v", err)
	}
	if summary.Unread != 4 || summary.LatestNotificationID != 0 || summary.LatestCreatedAt != "" {
		t.Fatalf("latest failure should return unread-only summary, got %#v", summary)
	}

	repo.latestErr = nil
	repo.latest = &model.Notification{ID: 99, CreatedAt: testNow}
	summary, err = svc.NotificationSummary(ctx, query.Summary{UserID: 10, AccountType: "candidate"})
	if err != nil {
		t.Fatalf("NotificationSummary returned error: %v", err)
	}
	if summary.LatestNotificationID != 99 || summary.LatestCreatedAt == "" {
		t.Fatalf("summary latest not mapped: %#v", summary)
	}
}

func TestMarkReadAndMarkAllReadKeepLegacySoftSuccess(t *testing.T) {
	ctx := context.Background()
	cache := newFakeCache()
	repo := &fakeNotificationRepo{markRows: 1, markAllRows: []int64{1000, 3}}
	svc := New(Deps{Notifications: repo, UnreadCache: cache, Clock: fakeClock{now: testNow}})

	result, err := svc.MarkNotificationRead(ctx, command.MarkRead{UserID: 10, AccountType: "candidate", NotificationID: 50})
	if err != nil {
		t.Fatalf("MarkNotificationRead returned error: %v", err)
	}
	if result.Code != 0 || result.Message != "success" || !cache.invalidated["10:candidate"] {
		t.Fatalf("unexpected mark success result=%#v invalidated=%v", result, cache.invalidated)
	}

	repo.markRows = 0
	cache.invalidated = map[string]bool{}
	result, err = svc.MarkNotificationRead(ctx, command.MarkRead{UserID: 10, AccountType: "candidate", NotificationID: 51})
	if err != nil {
		t.Fatalf("MarkNotificationRead returned error: %v", err)
	}
	if result.Code != query.CodeForbidden || cache.invalidated["10:candidate"] {
		t.Fatalf("zero affected rows should be forbidden without invalidation: result=%#v invalidated=%v", result, cache.invalidated)
	}

	repo.markErr = errors.New("db down")
	result, err = svc.MarkNotificationRead(ctx, command.MarkRead{UserID: 10, AccountType: "candidate", NotificationID: 52})
	if err != nil {
		t.Fatalf("MarkNotificationRead returned error: %v", err)
	}
	if result.Code != 0 || result.Message != "success" {
		t.Fatalf("DB error should soft-succeed, got %#v", result)
	}

	cache.invalidated = map[string]bool{}
	result, err = svc.MarkAllNotificationsRead(ctx, command.MarkAllRead{UserID: 10, AccountType: "candidate"})
	if err != nil {
		t.Fatalf("MarkAllNotificationsRead returned error: %v", err)
	}
	if result.Message != "success" || repo.markAllCalls != 2 || !cache.invalidated["10:candidate"] {
		t.Fatalf("unexpected mark-all result=%#v calls=%d invalidated=%v", result, repo.markAllCalls, cache.invalidated)
	}
}

func TestHandleNotificationMessageCreatesOnceAndPublishesRealtime(t *testing.T) {
	ctx := context.Background()
	cache := newFakeCache()
	realtime := &fakeRealtime{}
	repo := &fakeNotificationRepo{createOnceCreated: true, unread: 5}
	svc := New(Deps{Notifications: repo, UnreadCache: cache, Realtime: realtime, Clock: fakeClock{now: testNow}})

	err := svc.HandleNotificationMessage(ctx, command.NotificationMessage{
		EventID:    "evt-1",
		ReceiverID: 10,
		Type:       "offer_sent",
		Title:      "Offer",
		Content:    "content",
		Link:       "/offers/1",
	})
	if err != nil {
		t.Fatalf("HandleNotificationMessage returned error: %v", err)
	}
	if repo.createdOnce[0].ReceiverAccountType != model.DefaultAccountType || repo.createdOnce[0].CreatedAt.IsZero() {
		t.Fatalf("notification was not normalized before create: %#v", repo.createdOnce[0])
	}
	if !cache.invalidated["10:candidate"] || cache.counts["10:candidate"] != 5 {
		t.Fatalf("cache side effects missing: %#v", cache)
	}
	if len(realtime.payloads) != 1 || !strings.Contains(realtime.payloads[0], `"notification_created"`) {
		t.Fatalf("missing realtime payload: %#v", realtime.payloads)
	}

	realtime.payloads = nil
	repo.createOnceCreated = false
	if err := svc.HandleNotificationMessage(ctx, command.NotificationMessage{EventID: "evt-1", ReceiverID: 10, Type: "offer_sent"}); err != nil {
		t.Fatalf("duplicate HandleNotificationMessage returned error: %v", err)
	}
	if len(realtime.payloads) != 0 {
		t.Fatalf("duplicate notification should not publish realtime event: %#v", realtime.payloads)
	}
}

func TestHandleEmailMessageCoordinatesTemplateIdempotencyRecipientAndSend(t *testing.T) {
	ctx := context.Background()
	renderer := &fakeRenderer{templates: map[string]bool{"offer_sent": true}}
	sender := &fakeSender{}
	users := &fakeUsers{recipient: &model.EmailRecipient{UserID: 10, Email: "candidate@example.com", Username: "Candidate"}}
	logs := &fakeEmailLogs{}
	svc := New(Deps{EmailLogs: logs, Users: users, Renderer: renderer, Sender: sender, Clock: fakeClock{now: testNow}})

	if err := svc.HandleEmailMessage(ctx, command.EmailMessage{EventID: "no-template", ReceiverID: 10, Type: "unknown"}); err != nil {
		t.Fatalf("missing template should skip without error: %v", err)
	}
	if len(sender.sentTo) != 0 || len(logs.created) != 0 {
		t.Fatalf("missing template should not send/log: sent=%v logs=%v", sender.sentTo, logs.created)
	}

	logs.exists = true
	if err := svc.HandleEmailMessage(ctx, command.EmailMessage{EventID: "evt-1", ReceiverID: 10, Type: "offer_sent", Title: "Offer"}); err != nil {
		t.Fatalf("duplicate email should skip without error: %v", err)
	}
	if len(sender.sentTo) != 0 || logs.created[len(logs.created)-1].Status != model.EmailStatusSkipped {
		t.Fatalf("duplicate should log skipped only: sent=%v logs=%#v", sender.sentTo, logs.created)
	}

	logs.exists = false
	users.recipient = &model.EmailRecipient{UserID: 10, Username: "Candidate"}
	if err := svc.HandleEmailMessage(ctx, command.EmailMessage{EventID: "evt-2", ReceiverID: 10, Type: "offer_sent", Title: "Offer"}); err != nil {
		t.Fatalf("missing recipient email should skip without error: %v", err)
	}
	if logs.created[len(logs.created)-1].Status != model.EmailStatusSkipped {
		t.Fatalf("missing email should log skipped: %#v", logs.created)
	}

	users.recipient = &model.EmailRecipient{UserID: 10, Email: "candidate@example.com", Username: "Candidate"}
	sender.err = errors.New("smtp down")
	err := svc.HandleEmailMessage(ctx, command.EmailMessage{EventID: "evt-3", ReceiverID: 10, Type: "offer_sent", Title: "Offer"})
	if err == nil || err.Error() != "smtp down" {
		t.Fatalf("send failure should be returned, got %v", err)
	}
	if logs.created[len(logs.created)-1].Status != model.EmailStatusFailed || logs.created[len(logs.created)-1].ErrorMsg != "smtp down" {
		t.Fatalf("send failure should log failed: %#v", logs.created[len(logs.created)-1])
	}

	sender.err = nil
	err = svc.HandleEmailMessage(ctx, command.EmailMessage{
		EventID:    "evt-4",
		ReceiverID: 10,
		Type:       "offer_sent",
		Title:      "Offer",
		JobTitle:   "Backend Engineer",
		Link:       "/offers/1",
	})
	if err != nil {
		t.Fatalf("send success returned error: %v", err)
	}
	if sender.sentTo[len(sender.sentTo)-1] != "candidate@example.com" || renderer.lastData.ActionText != "查看并处理 Offer" {
		t.Fatalf("send/template data mismatch: sent=%v data=%#v", sender.sentTo, renderer.lastData)
	}
	if logs.created[len(logs.created)-1].Status != model.EmailStatusSent {
		t.Fatalf("success should log sent: %#v", logs.created[len(logs.created)-1])
	}

	// After SMTP succeeds, a log write failure must not bubble up (would cause MQ redelivery).
	logs.createErr = errors.New("uk_email_event_id duplicate")
	logs.exists = false
	beforeSends := len(sender.sentTo)
	if err := svc.HandleEmailMessage(ctx, command.EmailMessage{
		EventID:    "evt-5",
		ReceiverID: 10,
		Type:       "offer_sent",
		Title:      "Offer",
	}); err != nil {
		t.Fatalf("post-send log failure should not retry: %v", err)
	}
	if len(sender.sentTo) != beforeSends+1 {
		t.Fatalf("expected one additional send after log failure, sent=%d", len(sender.sentTo))
	}
}

func TestInboxIdentityAndRunWithInbox(t *testing.T) {
	body := []byte(`{"event_id":"evt-1","type":"notification.create"}`)
	identity := DeriveInboxIdentity("notification-consumer", body)
	if identity.EventID != "evt-1" || identity.EventType != "notification.create" || identity.IdempotencyKey != "notification-consumer:evt-1" {
		t.Fatalf("unexpected identity: %#v", identity)
	}

	fallback := DeriveInboxIdentity("notification-consumer", []byte(`{"type":"notification.create"}`))
	if !strings.HasPrefix(fallback.EventID, "body_sha256:") || fallback.IdempotencyKey != "notification-consumer:"+fallback.EventID {
		t.Fatalf("unexpected fallback identity: %#v", fallback)
	}

	ctx := context.Background()
	inbox := &fakeInbox{claimed: true, record: &repository.InboxRecord{ID: 9}}
	handled := false
	err := RunWithInbox(ctx, inbox, "notification-consumer", body, func() error {
		handled = true
		return nil
	})
	if err != nil {
		t.Fatalf("RunWithInbox returned error: %v", err)
	}
	if !handled || inbox.processedID != 9 {
		t.Fatalf("expected handler and processed mark: handled=%v inbox=%#v", handled, inbox)
	}

	inbox = &fakeInbox{claimed: false, record: &repository.InboxRecord{ID: 10}}
	handled = false
	if err := RunWithInbox(ctx, inbox, "notification-consumer", body, func() error { handled = true; return nil }); err != nil {
		t.Fatalf("duplicate RunWithInbox returned error: %v", err)
	}
	if handled {
		t.Fatal("duplicate inbox claim should skip handler")
	}

	inbox = &fakeInbox{claimed: true, record: &repository.InboxRecord{ID: 11}}
	handleErr := errors.New("poison")
	err = RunWithInbox(ctx, inbox, "notification-consumer", body, func() error { return handleErr })
	if !errors.Is(err, handleErr) || inbox.failedID != 11 || inbox.failedMsg != "poison" {
		t.Fatalf("expected failure mark and original error: err=%v inbox=%#v", err, inbox)
	}
}

type fakeClock struct{ now time.Time }

func (c fakeClock) Now() time.Time { return c.now }

type fakeNotificationRepo struct {
	created           []model.Notification
	createdOnce       []model.Notification
	createOnceCreated bool
	listRows          []model.Notification
	listTotal         int64
	listErr           error
	listCursorRows    []model.Notification
	nextCursor        string
	hasMore           bool
	lastCursorLimit   int32
	lastPage          int32
	lastPageSize      int32
	unread            int64
	unreadErr         error
	unreadCalls       int
	latest            *model.Notification
	latestErr         error
	markRows          int64
	markErr           error
	markAllRows       []int64
	markAllCalls      int
}

func (r *fakeNotificationRepo) Create(_ context.Context, n *model.Notification) error {
	r.created = append(r.created, *n)
	return nil
}

func (r *fakeNotificationRepo) CreateOnceWithResult(_ context.Context, n *model.Notification) (bool, error) {
	r.createdOnce = append(r.createdOnce, *n)
	return r.createOnceCreated, nil
}

func (r *fakeNotificationRepo) List(_ context.Context, _ int64, _ string, page, pageSize int32) ([]model.Notification, int64, error) {
	r.lastPage = page
	r.lastPageSize = pageSize
	return r.listRows, r.listTotal, r.listErr
}

func (r *fakeNotificationRepo) ListCursor(_ context.Context, _ int64, _ string, _ string, limit int32) ([]model.Notification, string, bool, error) {
	r.lastCursorLimit = limit
	return r.listCursorRows, r.nextCursor, r.hasMore, r.listErr
}

func (r *fakeNotificationRepo) UnreadCount(context.Context, int64, string) (int64, error) {
	r.unreadCalls++
	return r.unread, r.unreadErr
}

func (r *fakeNotificationRepo) Latest(context.Context, int64, string) (*model.Notification, error) {
	return r.latest, r.latestErr
}

func (r *fakeNotificationRepo) MarkRead(context.Context, int64, string, int64) (int64, error) {
	return r.markRows, r.markErr
}

func (r *fakeNotificationRepo) MarkAllReadBatch(context.Context, int64, string, int) (int64, error) {
	r.markAllCalls++
	if len(r.markAllRows) == 0 {
		return 0, nil
	}
	rows := r.markAllRows[0]
	r.markAllRows = r.markAllRows[1:]
	return rows, nil
}

type fakeActorVerifier struct{ err error }

func (v fakeActorVerifier) VerifyActorMatch(context.Context, int64) error { return v.err }

type fakeCache struct {
	counts      map[string]int64
	invalidated map[string]bool
}

func newFakeCache() *fakeCache {
	return &fakeCache{counts: map[string]int64{}, invalidated: map[string]bool{}}
}

func (c *fakeCache) GetUnreadCount(_ context.Context, userID uint64, accountType string) (int64, bool) {
	count, ok := c.counts[cacheKey(userID, accountType)]
	return count, ok
}

func (c *fakeCache) SetUnreadCount(_ context.Context, userID uint64, accountType string, count int64) {
	c.counts[cacheKey(userID, accountType)] = count
}

func (c *fakeCache) Invalidate(_ context.Context, userID uint64, accountType string) {
	c.invalidated[cacheKey(userID, accountType)] = true
	delete(c.counts, cacheKey(userID, accountType))
}

func cacheKey(userID uint64, accountType string) string {
	return fmt.Sprintf("%d:%s", userID, accountType)
}

type fakeRealtime struct{ payloads []string }

func (r *fakeRealtime) PublishNotificationEvent(_ context.Context, _ uint64, _ string, payload string) error {
	r.payloads = append(r.payloads, payload)
	return nil
}

type fakeEmailLogs struct {
	exists    bool
	createErr error
	created   []model.EmailLog
}

func (l *fakeEmailLogs) Create(_ context.Context, log *model.EmailLog) error {
	l.created = append(l.created, *log)
	return l.createErr
}

func (l *fakeEmailLogs) ExistsByEventID(context.Context, string) (bool, error) {
	return l.exists, nil
}

type fakeUsers struct {
	recipient *model.EmailRecipient
	err       error
}

func (u *fakeUsers) GetEmailRecipient(context.Context, int64) (*model.EmailRecipient, error) {
	return u.recipient, u.err
}

type fakeRenderer struct {
	templates map[string]bool
	lastData  port.TemplateData
}

func (r *fakeRenderer) HasTemplate(notificationType string) bool {
	return r.templates[notificationType]
}

func (r *fakeRenderer) Render(_ string, data port.TemplateData) (string, string, error) {
	r.lastData = data
	return "subject", "<p>body</p>", nil
}

type fakeSender struct {
	sentTo []string
	err    error
}

func (s *fakeSender) Send(_ context.Context, to, _, _ string) error {
	if s.err != nil {
		return s.err
	}
	s.sentTo = append(s.sentTo, to)
	return nil
}

type fakeInbox struct {
	claimed     bool
	record      *repository.InboxRecord
	processedID uint64
	failedID    uint64
	failedMsg   string
}

func (i *fakeInbox) Claim(context.Context, repository.InboxClaim) (*repository.InboxRecord, bool, error) {
	return i.record, i.claimed, nil
}

func (i *fakeInbox) MarkProcessed(_ context.Context, id uint64) error {
	i.processedID = id
	return nil
}

func (i *fakeInbox) MarkFailed(_ context.Context, id uint64, msg string) error {
	i.failedID = id
	i.failedMsg = msg
	return nil
}
