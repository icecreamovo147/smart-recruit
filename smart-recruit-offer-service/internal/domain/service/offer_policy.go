package service

import "smart-recruit-offer-service/internal/domain/model"

type ApplicationTransition struct {
	From model.ApplicationStatus
	To   model.ApplicationStatus
}

func CreationTransition(current model.ApplicationStatus) (ApplicationTransition, bool, error) {
	if current == model.ApplicationStatusOfferPending {
		return ApplicationTransition{}, false, nil
	}
	if err := model.ValidateApplicationTransition(current, model.ApplicationStatusOfferPending); err != nil {
		return ApplicationTransition{}, false, err
	}
	return ApplicationTransition{From: current, To: model.ApplicationStatusOfferPending}, true, nil
}

func SendTransition(current model.ApplicationStatus) (ApplicationTransition, error) {
	if err := model.ValidateApplicationTransition(current, model.ApplicationStatusOfferSent); err != nil {
		return ApplicationTransition{}, err
	}
	return ApplicationTransition{From: current, To: model.ApplicationStatusOfferSent}, nil
}

func WithdrawTransition(current model.ApplicationStatus) (ApplicationTransition, error) {
	if err := model.ValidateApplicationTransition(current, model.ApplicationStatusOfferPending); err != nil {
		return ApplicationTransition{}, err
	}
	return ApplicationTransition{From: current, To: model.ApplicationStatusOfferPending}, nil
}

func AcceptTransition(current model.ApplicationStatus) (ApplicationTransition, error) {
	if err := model.ValidateApplicationTransition(current, model.ApplicationStatusOfferAccepted); err != nil {
		return ApplicationTransition{}, err
	}
	return ApplicationTransition{From: current, To: model.ApplicationStatusOfferAccepted}, nil
}

func RejectTransition(current model.ApplicationStatus) (ApplicationTransition, error) {
	if err := model.ValidateApplicationTransition(current, model.ApplicationStatusOfferRejected); err != nil {
		return ApplicationTransition{}, err
	}
	return ApplicationTransition{From: current, To: model.ApplicationStatusOfferRejected}, nil
}
