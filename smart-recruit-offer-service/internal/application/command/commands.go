package command

import "time"

type CreateOffer struct {
	HRID          int64
	ApplicationID int64
	Title         string
	SalaryRange   string
	Level         string
	WorkLocation  string
	StartDate     string
	ExpiresAt     *time.Time
	TermsJSON     string
}

type UpdateOffer struct {
	HRID         int64
	OfferID      int64
	Title        string
	SalaryRange  string
	Level        string
	WorkLocation string
	StartDate    string
	ExpiresAt    *time.Time
	TermsJSON    string
}

type SendOffer struct {
	HRID    int64
	OfferID int64
}

type WithdrawOffer struct {
	HRID    int64
	OfferID int64
	Reason  string
}

type AcceptOffer struct {
	UserID  int64
	OfferID int64
}

type RejectOffer struct {
	UserID  int64
	OfferID int64
	Reason  string
}
