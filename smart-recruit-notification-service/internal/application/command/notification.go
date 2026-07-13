package command

type CreateNotification struct {
	EventID             string
	ReceiverID          int64
	ReceiverAccountType string
	ReceiverRole        int32
	Type                string
	Title               string
	Content             string
	Link                string
	BizType             string
	BizID               int64
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
	EventID             string
	ReceiverID          int64
	ReceiverAccountType string
	Type                string
	Title               string
	Content             string
	Link                string
	BizType             string
	BizID               int64
	JobTitle            string
	RecipientName       string
	InterviewDate       string
	InterviewMode       string
	InterviewLink       string
	InterviewLoc        string
	OfferTitle          string
	ExpiryDate          string
}
