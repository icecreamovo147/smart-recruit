package domain

// NotificationID identifies a Notification-owned delivery record.
type NotificationID int64

// ReceiverID identifies the user that receives a notification.
type ReceiverID int64

// AccountType scopes notification visibility, unread counts, and realtime channels.
type AccountType string

const (
	AccountTypeCandidate AccountType = "candidate"
	AccountTypeStaff     AccountType = "staff"
	AccountTypeService   AccountType = "service"
)

// NotificationReadState is the persisted read/unread marker.
type NotificationReadState int32

const (
	NotificationUnread NotificationReadState = 0
	NotificationRead   NotificationReadState = 1
)

// RealtimeEventType is the event envelope type published to notification streams.
type RealtimeEventType string

const (
	RealtimeEventNotificationCreated RealtimeEventType = "notification_created"
)
