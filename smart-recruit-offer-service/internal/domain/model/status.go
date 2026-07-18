package model

import "fmt"

type OfferStatus string

const (
	OfferStatusDraft     OfferStatus = "draft"
	OfferStatusSent      OfferStatus = "sent"
	OfferStatusWithdrawn OfferStatus = "withdrawn"
	OfferStatusAccepted  OfferStatus = "accepted"
	OfferStatusRejected  OfferStatus = "rejected"
)

type ApplicationStatus string

const (
	ApplicationStatusApplied            ApplicationStatus = "applied"
	ApplicationStatusViewed             ApplicationStatus = "viewed"
	ApplicationStatusScreening          ApplicationStatus = "screening"
	ApplicationStatusScreenPassed       ApplicationStatus = "screen_passed"
	ApplicationStatusInterviewPending   ApplicationStatus = "interview_pending"
	ApplicationStatusInterviewing       ApplicationStatus = "interviewing"
	ApplicationStatusInterviewCancelled ApplicationStatus = "interview_cancelled"
	ApplicationStatusInterviewPassed    ApplicationStatus = "interview_passed"
	ApplicationStatusOfferPending       ApplicationStatus = "offer_pending"
	ApplicationStatusOfferSent          ApplicationStatus = "offer_sent"
	ApplicationStatusOfferAccepted      ApplicationStatus = "offer_accepted"
	ApplicationStatusOfferRejected      ApplicationStatus = "offer_rejected"
	ApplicationStatusHired              ApplicationStatus = "hired"
	ApplicationStatusRejected           ApplicationStatus = "rejected"
	ApplicationStatusWithdrawn          ApplicationStatus = "withdrawn"
)

var applicationStatusLabels = map[ApplicationStatus]string{
	ApplicationStatusApplied:            "已投递",
	ApplicationStatusViewed:             "已查看",
	ApplicationStatusScreening:          "筛选中",
	ApplicationStatusScreenPassed:       "筛选通过",
	ApplicationStatusInterviewPending:   "待面试",
	ApplicationStatusInterviewing:       "面试中",
	ApplicationStatusInterviewCancelled: "面试已取消",
	ApplicationStatusInterviewPassed:    "面试通过",
	ApplicationStatusOfferPending:       "待发Offer",
	ApplicationStatusOfferSent:          "Offer已发",
	ApplicationStatusOfferAccepted:      "Offer已接受",
	ApplicationStatusOfferRejected:      "Offer被拒",
	ApplicationStatusHired:              "已入职",
	ApplicationStatusRejected:           "已拒绝",
	ApplicationStatusWithdrawn:          "已撤回",
}

var allowedApplicationTransitions = map[ApplicationStatus]map[ApplicationStatus]bool{
	ApplicationStatusViewed: {
		ApplicationStatusOfferPending: true,
	},
	ApplicationStatusInterviewPassed: {
		ApplicationStatusOfferPending: true,
	},
	ApplicationStatusOfferPending: {
		ApplicationStatusOfferSent: true,
	},
	ApplicationStatusOfferSent: {
		ApplicationStatusOfferAccepted: true,
		ApplicationStatusOfferRejected: true,
		ApplicationStatusOfferPending:  true,
	},
	ApplicationStatusOfferAccepted: {
		ApplicationStatusHired: true,
	},
	ApplicationStatusOfferRejected: {},
}

type TransitionError struct {
	From ApplicationStatus
	To   ApplicationStatus
	Msg  string
}

func (e *TransitionError) Error() string {
	return e.Msg
}

func ValidateApplicationTransition(from, to ApplicationStatus) error {
	if from == to {
		return &TransitionError{From: from, To: to, Msg: fmt.Sprintf("状态未变更：%s", applicationStatusLabel(from))}
	}
	if targets, ok := allowedApplicationTransitions[from]; ok && targets[to] {
		return nil
	}
	return &TransitionError{
		From: from,
		To:   to,
		Msg:  fmt.Sprintf("不允许从「%s」变更为「%s」", applicationStatusLabel(from), applicationStatusLabel(to)),
	}
}

func applicationStatusLabel(status ApplicationStatus) string {
	if label, ok := applicationStatusLabels[status]; ok {
		return label
	}
	return string(status)
}
