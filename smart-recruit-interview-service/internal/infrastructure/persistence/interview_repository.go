package persistence

import (
	"context"

	"gorm.io/gorm"

	sharedmodel "smart-recruit-domain-go/model"
	sharedrepo "smart-recruit-domain-go/repository"
	"smart-recruit-interview-service/internal/domain/model"
	"smart-recruit-interview-service/internal/domain/repository"
)

type txContextKey struct{}

func ContextWithTx(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, txContextKey{}, tx)
}

func TxFromContext(ctx context.Context) (*gorm.DB, bool) {
	tx, ok := ctx.Value(txContextKey{}).(*gorm.DB)
	return tx, ok && tx != nil
}

type InterviewRepository struct {
	interviews *sharedrepo.InterviewRepo
}

func NewInterviewRepository(interviews *sharedrepo.InterviewRepo) *InterviewRepository {
	return &InterviewRepository{interviews: interviews}
}

func (r *InterviewRepository) Create(ctx context.Context, interview *model.Interview) error {
	shared := toSharedInterview(interview)
	if err := r.interviews.Create(ctx, shared); err != nil {
		return err
	}
	copyInterviewFields(interview, shared)
	return nil
}

func (r *InterviewRepository) Save(ctx context.Context, interview *model.Interview) error {
	return r.interviews.Update(ctx, toSharedInterview(interview))
}

func (r *InterviewRepository) CreateFeedback(ctx context.Context, feedback *model.Feedback) error {
	shared := toSharedFeedback(feedback)
	if err := r.interviews.CreateFeedback(ctx, shared); err != nil {
		return err
	}
	feedback.ID = shared.ID
	feedback.UpdatedAt = shared.UpdatedAt
	return nil
}

func (r *InterviewRepository) CancelActiveByApplication(ctx context.Context, applicationID int64, reason string) error {
	if tx, ok := TxFromContext(ctx); ok {
		return r.interviews.CancelPendingByApplication(ctx, tx, applicationID, reason)
	}
	return r.interviews.Transaction(ctx, func(tx *gorm.DB) error {
		return r.interviews.CancelPendingByApplication(ctx, tx, applicationID, reason)
	})
}

func (r *InterviewRepository) FindByID(ctx context.Context, interviewID int64) (*model.Interview, error) {
	shared, err := r.interviews.GetModelByID(ctx, interviewID)
	if err != nil || shared == nil {
		return nil, err
	}
	return fromSharedInterview(shared), nil
}

func (r *InterviewRepository) FindDetailsByID(ctx context.Context, interviewID int64) (*repository.InterviewDetails, error) {
	row, err := r.interviews.GetByID(ctx, interviewID)
	if err != nil || row == nil {
		return nil, err
	}
	return fromSharedDetails(row), nil
}

func (r *InterviewRepository) MaxRoundNo(ctx context.Context, applicationID int64) (int32, error) {
	return r.interviews.GetMaxRoundNo(ctx, applicationID)
}

func (r *InterviewRepository) ListByApplication(ctx context.Context, applicationID int64) ([]repository.InterviewDetails, error) {
	rows, err := r.interviews.ListByApplication(ctx, applicationID)
	if err != nil {
		return nil, err
	}
	return fromSharedDetailsRows(rows), nil
}

func (r *InterviewRepository) ListByInterviewer(ctx context.Context, interviewerID int64, status model.InterviewStatus) ([]repository.InterviewDetails, error) {
	rows, err := r.interviews.ListByInterviewer(ctx, interviewerID, string(status))
	if err != nil {
		return nil, err
	}
	return fromSharedDetailsRows(rows), nil
}

func (r *InterviewRepository) ListByCandidate(ctx context.Context, candidateUserID int64) ([]repository.InterviewDetails, error) {
	rows, err := r.interviews.ListByCandidate(ctx, candidateUserID)
	if err != nil {
		return nil, err
	}
	return fromSharedDetailsRows(rows), nil
}

func (r *InterviewRepository) ListFeedbackByInterviews(ctx context.Context, interviewIDs []int64) ([]model.Feedback, error) {
	rows, err := r.interviews.ListFeedbackByInterviews(ctx, interviewIDs)
	if err != nil {
		return nil, err
	}
	feedbacks := make([]model.Feedback, 0, len(rows))
	for _, row := range rows {
		feedbacks = append(feedbacks, *fromSharedFeedback(&row))
	}
	return feedbacks, nil
}

func (r *InterviewRepository) FeedbackExistsByInterviewer(ctx context.Context, interviewID int64, interviewerID int64) (bool, error) {
	return r.interviews.FeedbackExistsByInterviewer(ctx, interviewID, interviewerID)
}

func (r *InterviewRepository) FindFeedbackByInterviewAndInterviewer(ctx context.Context, interviewID int64, interviewerID int64) (*model.Feedback, error) {
	shared, err := r.interviews.GetFeedbackByInterviewAndInterviewer(ctx, interviewID, interviewerID)
	if err != nil || shared == nil {
		return nil, err
	}
	return fromSharedFeedback(shared), nil
}

func (r *InterviewRepository) Transaction(ctx context.Context, fn func(context.Context, repository.InterviewWriter) error) error {
	return r.interviews.Transaction(ctx, func(tx *gorm.DB) error {
		txCtx := ContextWithTx(ctx, tx)
		return fn(txCtx, txWriter{interviews: r.interviews, tx: tx})
	})
}

type txWriter struct {
	interviews *sharedrepo.InterviewRepo
	tx         *gorm.DB
}

func (w txWriter) Create(ctx context.Context, interview *model.Interview) error {
	shared := toSharedInterview(interview)
	if err := w.interviews.CreateWithTx(ctx, w.tx, shared); err != nil {
		return err
	}
	copyInterviewFields(interview, shared)
	return nil
}

func (w txWriter) Save(ctx context.Context, interview *model.Interview) error {
	return w.interviews.UpdateWithTx(ctx, w.tx, toSharedInterview(interview))
}

func (w txWriter) CreateFeedback(ctx context.Context, feedback *model.Feedback) error {
	shared := toSharedFeedback(feedback)
	if err := w.interviews.CreateFeedbackWithTx(ctx, w.tx, shared); err != nil {
		return err
	}
	feedback.ID = shared.ID
	feedback.UpdatedAt = shared.UpdatedAt
	return nil
}

func (w txWriter) CancelActiveByApplication(ctx context.Context, applicationID int64, reason string) error {
	return w.interviews.CancelPendingByApplication(ctx, w.tx, applicationID, reason)
}

func toSharedInterview(interview *model.Interview) *sharedmodel.InterviewSchedule {
	if interview == nil {
		return nil
	}
	return &sharedmodel.InterviewSchedule{
		ID:              interview.ID,
		ApplicationID:   interview.ApplicationID,
		InterviewerID:   interview.InterviewerID,
		RoundNo:         interview.RoundNo,
		ScheduledAt:     interview.ScheduledAt,
		Status:          string(interview.Status),
		CreatedBy:       interview.CreatedBy,
		Title:           interview.Title,
		Mode:            interview.Mode,
		MeetingURL:      interview.MeetingURL,
		Location:        interview.Location,
		DurationMinutes: interview.DurationMinutes,
		CandidateNote:   interview.CandidateNote,
		InternalNote:    interview.InternalNote,
		CancelReason:    interview.CancelReason,
		CreatedAt:       interview.CreatedAt,
		UpdatedAt:       interview.UpdatedAt,
	}
}

func fromSharedInterview(interview *sharedmodel.InterviewSchedule) *model.Interview {
	if interview == nil {
		return nil
	}
	return &model.Interview{
		ID:              interview.ID,
		ApplicationID:   interview.ApplicationID,
		InterviewerID:   interview.InterviewerID,
		RoundNo:         interview.RoundNo,
		ScheduledAt:     interview.ScheduledAt,
		Status:          model.InterviewStatus(interview.Status),
		CreatedBy:       interview.CreatedBy,
		Title:           interview.Title,
		Mode:            interview.Mode,
		MeetingURL:      interview.MeetingURL,
		Location:        interview.Location,
		DurationMinutes: interview.DurationMinutes,
		CandidateNote:   interview.CandidateNote,
		InternalNote:    interview.InternalNote,
		CancelReason:    interview.CancelReason,
		CreatedAt:       interview.CreatedAt,
		UpdatedAt:       interview.UpdatedAt,
	}
}

func copyInterviewFields(target *model.Interview, source *sharedmodel.InterviewSchedule) {
	target.ID = source.ID
	target.CreatedAt = source.CreatedAt
	target.UpdatedAt = source.UpdatedAt
}

func fromSharedDetails(row *sharedrepo.InterviewWithDetailsRow) *repository.InterviewDetails {
	if row == nil {
		return nil
	}
	return &repository.InterviewDetails{
		Interview: model.Interview{
			ID:              row.ID,
			ApplicationID:   row.ApplicationID,
			InterviewerID:   row.InterviewerID,
			RoundNo:         row.RoundNo,
			Title:           row.Title,
			Mode:            row.Mode,
			MeetingURL:      row.MeetingURL,
			Location:        row.Location,
			DurationMinutes: row.DurationMinutes,
			CandidateNote:   row.CandidateNote,
			InternalNote:    row.InternalNote,
			CancelReason:    row.CancelReason,
			ScheduledAt:     row.ScheduledAt,
			Status:          model.InterviewStatus(row.Status),
			CreatedBy:       row.CreatedBy,
			CreatedAt:       row.CreatedAt,
			UpdatedAt:       row.UpdatedAt,
		},
		ApplicationStatusKey: model.ApplicationStatus(row.ApplicationStatusKey),
		JobTitle:             row.JobTitle,
		CandidateName:        row.CandidateName,
		CandidatePhone:       row.CandidatePhone,
		InterviewerName:      row.InterviewerName,
		ResumeOssKey:         row.ResumeOssKey,
	}
}

func fromSharedDetailsRows(rows []sharedrepo.InterviewWithDetailsRow) []repository.InterviewDetails {
	details := make([]repository.InterviewDetails, 0, len(rows))
	for i := range rows {
		details = append(details, *fromSharedDetails(&rows[i]))
	}
	return details
}

func toSharedFeedback(feedback *model.Feedback) *sharedmodel.InterviewFeedback {
	if feedback == nil {
		return nil
	}
	return &sharedmodel.InterviewFeedback{
		ID:                  feedback.ID,
		InterviewID:         feedback.InterviewID,
		ApplicationID:       feedback.ApplicationID,
		InterviewerID:       feedback.InterviewerID,
		Recommendation:      feedback.Recommendation,
		Score:               feedback.Score,
		DimensionScoresJSON: feedback.DimensionScoresJSON,
		Comments:            feedback.Comments,
		SubmittedAt:         feedback.SubmittedAt,
		UpdatedAt:           feedback.UpdatedAt,
	}
}

func fromSharedFeedback(feedback *sharedmodel.InterviewFeedback) *model.Feedback {
	if feedback == nil {
		return nil
	}
	return &model.Feedback{
		ID:                  feedback.ID,
		InterviewID:         feedback.InterviewID,
		ApplicationID:       feedback.ApplicationID,
		InterviewerID:       feedback.InterviewerID,
		Recommendation:      feedback.Recommendation,
		Score:               feedback.Score,
		DimensionScoresJSON: feedback.DimensionScoresJSON,
		Comments:            feedback.Comments,
		SubmittedAt:         feedback.SubmittedAt,
		UpdatedAt:           feedback.UpdatedAt,
	}
}
