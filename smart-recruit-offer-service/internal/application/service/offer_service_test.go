package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"smart-recruit-offer-service/internal/application/command"
	"smart-recruit-offer-service/internal/application/port"
	"smart-recruit-offer-service/internal/application/query"
	domainevent "smart-recruit-offer-service/internal/domain/event"
	"smart-recruit-offer-service/internal/domain/model"
	"smart-recruit-offer-service/internal/domain/repository"
)

func TestCreateOfferCreatesDraftAndPendingTransition(t *testing.T) {
	fixture := newOfferFixture(t)
	offerID, err := fixture.service.CreateOffer(context.Background(), command.CreateOffer{
		HRID:          100,
		ApplicationID: 10,
		Title:         "Offer Title",
		SalaryRange:   "30k-40k",
	})
	if err != nil {
		t.Fatalf("CreateOffer returned error: %v", err)
	}
	offer := fixture.repo.offers[offerID]
	if offer == nil {
		t.Fatalf("created offer %d not persisted", offerID)
	}
	if offer.Status != model.OfferStatusDraft || offer.CandidateUserID != 20 || offer.JobID != 30 {
		t.Fatalf("unexpected created offer: %+v", offer)
	}
	if got := fixture.lifecycle.last.ToStatus; got != model.ApplicationStatusOfferPending {
		t.Fatalf("application transition to=%s, want offer_pending", got)
	}
	if got := fixture.repo.events[0].EventType; got != domainevent.TypeCreated {
		t.Fatalf("event type=%s, want created", got)
	}
	if got := fixture.outbox.messages[0].Type; got != "offer_created" {
		t.Fatalf("outbox type=%s, want offer_created", got)
	}
	if !fixture.outbox.signaled {
		t.Fatal("expected outbox signal")
	}
}

func TestSendOfferUpdatesSnapshotAndPublishesNotificationAndEmail(t *testing.T) {
	fixture := newOfferFixture(t)
	offer := fixture.seedOffer(model.OfferStatusDraft)
	fixture.applications.snapshots[offer.ApplicationID].StatusKey = model.ApplicationStatusOfferPending

	if err := fixture.service.SendOffer(context.Background(), command.SendOffer{HRID: 100, OfferID: offer.ID}); err != nil {
		t.Fatalf("SendOffer returned error: %v", err)
	}
	if offer.Status != model.OfferStatusSent {
		t.Fatalf("status=%s, want sent", offer.Status)
	}
	if offer.SentBy == nil || *offer.SentBy != 100 {
		t.Fatalf("sent_by=%v, want 100", offer.SentBy)
	}
	if offer.SentSnapshotJSON == "" {
		t.Fatal("expected sent snapshot")
	}
	if got := fixture.lifecycle.last.ToStatus; got != model.ApplicationStatusOfferSent {
		t.Fatalf("application transition to=%s, want offer_sent", got)
	}
	if len(fixture.outbox.messages) != 2 {
		t.Fatalf("outbox messages=%d, want 2", len(fixture.outbox.messages))
	}
	if fixture.outbox.messages[1].RoutingKey != "email.send" {
		t.Fatalf("second routing key=%s, want email.send", fixture.outbox.messages[1].RoutingKey)
	}
}

func TestUpdateOfferAppliesDraftPatchAndWritesEvent(t *testing.T) {
	fixture := newOfferFixture(t)
	offer := fixture.seedOffer(model.OfferStatusDraft)

	if err := fixture.service.UpdateOffer(context.Background(), command.UpdateOffer{
		HRID:        100,
		OfferID:     offer.ID,
		Title:       "Updated Offer",
		SalaryRange: "35k-45k",
	}); err != nil {
		t.Fatalf("UpdateOffer returned error: %v", err)
	}
	if offer.Title != "Updated Offer" || offer.SalaryRange != "35k-45k" {
		t.Fatalf("draft patch not applied: title=%q salary=%q", offer.Title, offer.SalaryRange)
	}
	if got := fixture.repo.events[0].EventType; got != domainevent.TypeUpdated {
		t.Fatalf("event type=%s, want updated", got)
	}
}

func TestWithdrawSentRevertsApplicationToPending(t *testing.T) {
	fixture := newOfferFixture(t)
	offer := fixture.seedOffer(model.OfferStatusSent)
	fixture.applications.snapshots[offer.ApplicationID].StatusKey = model.ApplicationStatusOfferSent

	if err := fixture.service.WithdrawOffer(context.Background(), command.WithdrawOffer{HRID: 100, OfferID: offer.ID, Reason: "comp changed"}); err != nil {
		t.Fatalf("WithdrawOffer returned error: %v", err)
	}
	if offer.Status != model.OfferStatusWithdrawn {
		t.Fatalf("status=%s, want withdrawn", offer.Status)
	}
	if got := fixture.lifecycle.last.ToStatus; got != model.ApplicationStatusOfferPending {
		t.Fatalf("application transition to=%s, want offer_pending", got)
	}
	if got := fixture.repo.events[0].Reason; got != "comp changed" {
		t.Fatalf("event reason=%q, want comp changed", got)
	}
}

func TestAcceptOfferUpdatesLifecycleAndNotifiesStaff(t *testing.T) {
	fixture := newOfferFixture(t)
	offer := fixture.seedOffer(model.OfferStatusSent)
	sentBy := int64(100)
	offer.SentBy = &sentBy
	fixture.applications.snapshots[offer.ApplicationID].StatusKey = model.ApplicationStatusOfferSent

	if err := fixture.service.AcceptOffer(context.Background(), command.AcceptOffer{UserID: offer.CandidateUserID, OfferID: offer.ID}); err != nil {
		t.Fatalf("AcceptOffer returned error: %v", err)
	}
	if offer.Status != model.OfferStatusAccepted || offer.DecidedAt == nil {
		t.Fatalf("accept state not applied: %+v", offer)
	}
	if got := fixture.lifecycle.last.ToStatus; got != model.ApplicationStatusOfferAccepted {
		t.Fatalf("application transition to=%s, want offer_accepted", got)
	}
	if got := fixture.outbox.messages[0].ReceiverAccountType; got != "staff" {
		t.Fatalf("receiver account type=%s, want staff", got)
	}
}

func TestRejectOfferClosesCurrentRound(t *testing.T) {
	fixture := newOfferFixture(t)
	offer := fixture.seedOffer(model.OfferStatusSent)
	sentBy := int64(100)
	offer.SentBy = &sentBy
	fixture.applications.snapshots[offer.ApplicationID].StatusKey = model.ApplicationStatusOfferSent

	if err := fixture.service.RejectOffer(context.Background(), command.RejectOffer{UserID: offer.CandidateUserID, OfferID: offer.ID, Reason: "accepted other offer"}); err != nil {
		t.Fatalf("RejectOffer returned error: %v", err)
	}
	if offer.Status != model.OfferStatusRejected {
		t.Fatalf("status=%s, want rejected", offer.Status)
	}
	if !fixture.lifecycle.last.CloseCurrentRound {
		t.Fatal("expected reject transition to close current round")
	}
	if got := fixture.repo.events[0].Reason; got != "accepted other offer" {
		t.Fatalf("event reason=%q, want accepted other offer", got)
	}
}

func TestListOfferEventsChecksReadScope(t *testing.T) {
	fixture := newOfferFixture(t)
	offer := fixture.seedOffer(model.OfferStatusSent)
	fixture.repo.events = append(fixture.repo.events, domainevent.NewOfferEvent(offer.TenantID, offer.ID, domainevent.TypeSent, 100, "staff", "", fixture.service.clock.Now()))

	events, err := fixture.service.ListOfferEvents(context.Background(), query.ListOfferEvents{HRID: 100, OfferID: offer.ID})
	if err != nil {
		t.Fatalf("ListOfferEvents returned error: %v", err)
	}
	if len(events) != 1 || events[0].EventType != domainevent.TypeSent {
		t.Fatalf("events=%+v, want one sent event", events)
	}
	if fixture.authorizer.readChecks != 2 {
		t.Fatalf("read checks=%d, want permission and scope checks", fixture.authorizer.readChecks)
	}
}

func TestGetOfferAllowsCandidateSelfService(t *testing.T) {
	fixture := newOfferFixture(t)
	offer := fixture.seedOffer(model.OfferStatusSent)
	details, err := fixture.service.GetOffer(context.Background(), query.GetOffer{UserID: offer.CandidateUserID, OfferID: offer.ID})
	if err != nil {
		t.Fatalf("GetOffer returned error: %v", err)
	}
	if details.ID != offer.ID {
		t.Fatalf("offer id=%d, want %d", details.ID, offer.ID)
	}
	if fixture.authorizer.readChecks != 0 {
		t.Fatalf("candidate self-service should not require read scope, got %d checks", fixture.authorizer.readChecks)
	}
}

func newOfferFixture(t *testing.T) *offerFixture {
	t.Helper()
	repo := &fakeOfferRepository{offers: map[int64]*model.Offer{}, nextID: 1}
	apps := &fakeApplications{snapshots: map[int64]*port.ApplicationSnapshot{
		10: {
			ApplicationID:   10,
			CandidateUserID: 20,
			JobID:           30,
			JobTitle:        "Backend Engineer",
			StatusKey:       model.ApplicationStatusInterviewPassed,
		},
	}}
	lifecycle := &fakeLifecycle{}
	outbox := &fakeOutbox{}
	authorizer := &fakeAuthorizer{}
	service, err := NewOfferService(Deps{
		Offers:       repo,
		Applications: apps,
		Lifecycle:    lifecycle,
		Outbox:       outbox,
		Authorizer:   authorizer,
		Clock:        fixedClock{now: time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatalf("NewOfferService returned error: %v", err)
	}
	return &offerFixture{
		service:      service,
		repo:         repo,
		applications: apps,
		lifecycle:    lifecycle,
		outbox:       outbox,
		authorizer:   authorizer,
	}
}

type offerFixture struct {
	service      *OfferService
	repo         *fakeOfferRepository
	applications *fakeApplications
	lifecycle    *fakeLifecycle
	outbox       *fakeOutbox
	authorizer   *fakeAuthorizer
}

func (f *offerFixture) seedOffer(status model.OfferStatus) *model.Offer {
	offer := &model.Offer{
		ID:              f.repo.nextID,
		ApplicationID:   10,
		CandidateUserID: 20,
		JobID:           30,
		Status:          status,
		Title:           "Offer Title",
		SalaryRange:     "30k-40k",
		Level:           "P5",
		WorkLocation:    "Shanghai",
		StartDate:       "2026-08-01",
		TermsJSON:       `{"probation":"3 months"}`,
		CreatedBy:       100,
	}
	f.repo.nextID++
	f.repo.offers[offer.ID] = offer
	return offer
}

type fakeOfferRepository struct {
	offers map[int64]*model.Offer
	events []domainevent.OfferEvent
	nextID int64
}

func (r *fakeOfferRepository) Create(_ context.Context, offer *model.Offer) error {
	if offer.ID == 0 {
		offer.ID = r.nextID
		r.nextID++
	}
	r.offers[offer.ID] = offer
	return nil
}

func (r *fakeOfferRepository) Save(_ context.Context, offer *model.Offer) error {
	if offer.ID == 0 {
		return errors.New("offer id is required")
	}
	r.offers[offer.ID] = offer
	return nil
}

func (r *fakeOfferRepository) AddEvent(_ context.Context, event domainevent.OfferEvent) error {
	r.events = append(r.events, event)
	return nil
}

func (r *fakeOfferRepository) FindByID(_ context.Context, offerID int64) (*model.Offer, error) {
	return r.offers[offerID], nil
}

func (r *fakeOfferRepository) FindDetailsByID(_ context.Context, offerID int64) (*repository.OfferDetails, error) {
	offer := r.offers[offerID]
	if offer == nil {
		return nil, nil
	}
	return &repository.OfferDetails{Offer: *offer, JobTitle: "Backend Engineer", ApplicationStatusKey: model.ApplicationStatusOfferSent}, nil
}

func (r *fakeOfferRepository) ListByApplication(_ context.Context, applicationID int64) ([]repository.OfferDetails, error) {
	var rows []repository.OfferDetails
	for _, offer := range r.offers {
		if offer.ApplicationID == applicationID {
			rows = append(rows, repository.OfferDetails{Offer: *offer})
		}
	}
	return rows, nil
}

func (r *fakeOfferRepository) ListByCandidate(_ context.Context, candidateUserID int64, cursor string, limit int32) (repository.CandidateOfferPage, error) {
	var rows []repository.OfferDetails
	for _, offer := range r.offers {
		if offer.CandidateUserID == candidateUserID {
			rows = append(rows, repository.OfferDetails{Offer: *offer})
		}
	}
	return repository.CandidateOfferPage{Offers: rows, Total: int64(len(rows))}, nil
}

func (r *fakeOfferRepository) ListEvents(_ context.Context, offerID int64) ([]domainevent.OfferEvent, error) {
	var rows []domainevent.OfferEvent
	for _, event := range r.events {
		if event.OfferID == offerID {
			rows = append(rows, event)
		}
	}
	return rows, nil
}

func (r *fakeOfferRepository) Transaction(ctx context.Context, fn func(context.Context, repository.OfferWriter) error) error {
	return fn(ctx, r)
}

type fakeApplications struct {
	snapshots map[int64]*port.ApplicationSnapshot
}

func (a *fakeApplications) GetApplicationSnapshot(_ context.Context, applicationID int64) (*port.ApplicationSnapshot, error) {
	return a.snapshots[applicationID], nil
}

type fakeLifecycle struct {
	last port.LifecycleTransitionCommand
}

func (l *fakeLifecycle) ApplyTransition(_ context.Context, command port.LifecycleTransitionCommand) (bool, error) {
	l.last = command
	return true, nil
}

type fakeOutbox struct {
	messages []port.OutboxMessage
	signaled bool
}

func (o *fakeOutbox) Publish(_ context.Context, message port.OutboxMessage) error {
	o.messages = append(o.messages, message)
	return nil
}

func (o *fakeOutbox) Signal() {
	o.signaled = true
}

type fakeAuthorizer struct {
	readChecks int
}

func (a *fakeAuthorizer) VerifyActor(_ context.Context, actorID int64) error {
	if actorID == 0 {
		return errors.New("actor id is required")
	}
	return nil
}

func (a *fakeAuthorizer) Authorize(_ context.Context, _ int64, permission port.Permission) error {
	if permission == port.PermissionOfferRead {
		a.readChecks++
	}
	return nil
}

func (a *fakeAuthorizer) CanManageApplication(context.Context, int64, int64) error {
	return nil
}

func (a *fakeAuthorizer) CanReadApplication(context.Context, int64, int64) error {
	a.readChecks++
	return nil
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}
