package model

import (
	"errors"
	"time"
)

var (
	ErrFeedbackAlreadyExists          = errors.New("您已提交过面试反馈，不可重复提交")
	ErrFeedbackApplicationMismatch    = errors.New("ApplicationId 与面试记录不匹配")
	ErrFeedbackTerminalApplication    = errors.New("该候选人已结束投递流程，无法提交面试反馈")
	ErrFeedbackInvalid                = errors.New("面试反馈不合法")
	ErrFeedbackRecommendationRequired = errors.New("请选择面试推荐结论")
	ErrFeedbackInvalidRecommendation  = errors.New("推荐结论值不合法")
	ErrFeedbackScoreOutOfRange        = errors.New("评分范围为 0-10")
	ErrInterviewerMismatch            = errors.New("您不是该面试的面试官，无法提交反馈")
)

type Feedback struct {
	ID                  int64
	InterviewID         int64
	ApplicationID       int64
	InterviewerID       int64
	Recommendation      string
	Score               int32
	DimensionScoresJSON string
	Comments            string
	SubmittedAt         time.Time
	UpdatedAt           time.Time
}

type FeedbackDetails struct {
	InterviewID         int64
	ApplicationID       int64
	InterviewerID       int64
	Recommendation      string
	Score               int32
	DimensionScoresJSON string
	Comments            string
	SubmittedAt         time.Time
}

var validRecommendations = map[string]bool{
	"positive":             true,
	"negative":             true,
	"pending":              true,
	"strong_recommend":     true,
	"recommend":            true,
	"neutral":              true,
	"not_recommend":        true,
	"strong_not_recommend": true,
}

func NewFeedback(details FeedbackDetails) (*Feedback, error) {
	if details.InterviewID == 0 {
		return nil, errors.New("interview id is required")
	}
	if details.ApplicationID == 0 {
		return nil, errors.New("application id is required")
	}
	if details.InterviewerID == 0 {
		return nil, errors.New("interviewer id is required")
	}
	if details.Recommendation == "" {
		return nil, ErrFeedbackRecommendationRequired
	}
	if details.Score < 0 || details.Score > 10 {
		return nil, ErrFeedbackScoreOutOfRange
	}
	if !validRecommendations[details.Recommendation] {
		return nil, ErrFeedbackInvalidRecommendation
	}
	submittedAt := details.SubmittedAt
	if submittedAt.IsZero() {
		submittedAt = time.Now()
	}
	return &Feedback{
		InterviewID:         details.InterviewID,
		ApplicationID:       details.ApplicationID,
		InterviewerID:       details.InterviewerID,
		Recommendation:      details.Recommendation,
		Score:               details.Score,
		DimensionScoresJSON: details.DimensionScoresJSON,
		Comments:            details.Comments,
		SubmittedAt:         submittedAt,
	}, nil
}
