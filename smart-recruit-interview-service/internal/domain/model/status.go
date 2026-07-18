package model

import "fmt"

type InterviewStatus string

const (
	InterviewStatusPending   InterviewStatus = "pending"
	InterviewStatusScheduled InterviewStatus = "scheduled"
	InterviewStatusCompleted InterviewStatus = "completed"
	InterviewStatusCancelled InterviewStatus = "cancelled"
)

type ApplicationStatus string

const (
	ApplicationStatusApplied            ApplicationStatus = "applied"
	ApplicationStatusViewed             ApplicationStatus = "viewed"
	ApplicationStatusScreening          ApplicationStatus = "screening"
	ApplicationStatusScreenPassed       ApplicationStatus = "screen_passed"
	ApplicationStatusInterviewPending   ApplicationStatus = "interview_pending"
	ApplicationStatusInterviewing       ApplicationStatus = "interviewing"
	ApplicationStatusInterviewPassed    ApplicationStatus = "interview_passed"
	ApplicationStatusInterviewCancelled ApplicationStatus = "interview_cancelled"
	ApplicationStatusOfferPending       ApplicationStatus = "offer_pending"
	ApplicationStatusOfferSent          ApplicationStatus = "offer_sent"
	ApplicationStatusOfferAccepted      ApplicationStatus = "offer_accepted"
	ApplicationStatusOfferRejected      ApplicationStatus = "offer_rejected"
	ApplicationStatusHired              ApplicationStatus = "hired"
	ApplicationStatusRejected           ApplicationStatus = "rejected"
	ApplicationStatusWithdrawn          ApplicationStatus = "withdrawn"
)

var terminalApplicationStatuses = map[ApplicationStatus]bool{
	ApplicationStatusRejected:      true,
	ApplicationStatusWithdrawn:     true,
	ApplicationStatusOfferRejected: true,
	ApplicationStatusHired:         true,
}

var applicationStatusLabels = map[ApplicationStatus]string{
	ApplicationStatusApplied:            "已投递",
	ApplicationStatusViewed:             "已查看",
	ApplicationStatusScreening:          "筛选中",
	ApplicationStatusScreenPassed:       "筛选通过",
	ApplicationStatusInterviewPending:   "待面试",
	ApplicationStatusInterviewing:       "面试中",
	ApplicationStatusInterviewPassed:    "面试通过",
	ApplicationStatusInterviewCancelled: "面试已取消",
	ApplicationStatusOfferPending:       "待发Offer",
	ApplicationStatusOfferSent:          "Offer已发",
	ApplicationStatusOfferAccepted:      "Offer已接受",
	ApplicationStatusOfferRejected:      "Offer被拒",
	ApplicationStatusHired:              "已入职",
	ApplicationStatusRejected:           "淘汰",
	ApplicationStatusWithdrawn:          "候选人撤回",
}

func IsTerminalApplicationStatus(status ApplicationStatus) bool {
	return terminalApplicationStatuses[status]
}

type TransitionError struct {
	From ApplicationStatus
	To   ApplicationStatus
	Msg  string
}

func (e *TransitionError) Error() string {
	return e.Msg
}

func ValidateTransition(from, to ApplicationStatus, allowed map[ApplicationStatus]bool) error {
	if from == to {
		return &TransitionError{From: from, To: to, Msg: fmt.Sprintf("状态未变更：%s", applicationStatusLabel(from))}
	}
	if allowed[from] {
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
