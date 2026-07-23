package service

import "smart-recruit-interview-service/internal/domain/model"

type ApplicationTransition struct {
	From model.ApplicationStatus
	To   model.ApplicationStatus
}

func ScheduleTransition(current model.ApplicationStatus) (ApplicationTransition, bool, error) {
	allowed := map[model.ApplicationStatus]bool{
		model.ApplicationStatusViewed:             true,
		model.ApplicationStatusScreenPassed:       true,
		model.ApplicationStatusInterviewCancelled: true,
		model.ApplicationStatusInterviewPassed:    true,
	}
	if current == model.ApplicationStatusInterviewPending {
		return ApplicationTransition{}, false, nil
	}
	if !allowed[current] {
		return ApplicationTransition{}, false, nil
	}
	return ApplicationTransition{From: current, To: model.ApplicationStatusInterviewPending}, true, nil
}

func CancelTransition(current model.ApplicationStatus) (ApplicationTransition, bool) {
	if current != model.ApplicationStatusInterviewPending && current != model.ApplicationStatusInterviewing {
		return ApplicationTransition{}, false
	}
	return ApplicationTransition{From: current, To: model.ApplicationStatusInterviewCancelled}, true
}

func FeedbackTransition(current model.ApplicationStatus) (ApplicationTransition, bool) {
	if current != model.ApplicationStatusInterviewPending {
		return ApplicationTransition{}, false
	}
	return ApplicationTransition{From: current, To: model.ApplicationStatusInterviewing}, true
}
