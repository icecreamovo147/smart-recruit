package command

type CreateNotification struct {
	EventID             string `json:"event_id,omitempty"`
	IdempotencyKey      string `json:"idempotency_key,omitempty"`
	ReceiverID          int64  `json:"receiver_id,omitempty"`
	ReceiverAccountType string `json:"receiver_account_type,omitempty"`
	ReceiverRole        int32  `json:"receiver_role,omitempty"`
	Type                string `json:"type,omitempty"`
	Title               string `json:"title,omitempty"`
	Content             string `json:"content,omitempty"`
	Link                string `json:"link,omitempty"`
	BizType             string `json:"biz_type,omitempty"`
	BizID               int64  `json:"biz_id,omitempty"`
}

type NotificationMessage = CreateNotification

type MarkRead struct {
	UserID         int64
	AccountType    string
	NotificationID int64
}

type MarkAllRead struct {
	UserID      int64
	AccountType string
}

type EmailMessage struct {
	EventID             string `json:"event_id,omitempty"`
	IdempotencyKey      string `json:"idempotency_key,omitempty"`
	ReceiverID          int64  `json:"receiver_id,omitempty"`
	ReceiverAccountType string `json:"receiver_account_type,omitempty"`
	Type                string `json:"type,omitempty"`
	Title               string `json:"title,omitempty"`
	Content             string `json:"content,omitempty"`
	Link                string `json:"link,omitempty"`
	BizType             string `json:"biz_type,omitempty"`
	BizID               int64  `json:"biz_id,omitempty"`
	JobTitle            string `json:"job_title,omitempty"`
	RecipientName       string `json:"recipient_name,omitempty"`
	InterviewDate       string `json:"interview_date,omitempty"`
	InterviewMode       string `json:"interview_mode,omitempty"`
	InterviewLink       string `json:"interview_link,omitempty"`
	InterviewLoc        string `json:"interview_loc,omitempty"`
	OfferTitle          string `json:"offer_title,omitempty"`
	ExpiryDate          string `json:"expiry_date,omitempty"`
}
