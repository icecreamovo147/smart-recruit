package mapper

import (
	"strings"
	"time"

	"smart-recruit-interview-service/internal/application/port"
	"smart-recruit-interview-service/internal/domain/model"
	"smart-recruit-interview-service/internal/domain/repository"
	"smart-recruit-platform-go/businessclock"
	"smart-recruit-proto/recruitment/pb"
)

func ParseOptionalRFC3339(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := businessclock.Parse(value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func ToPBInterview(details repository.InterviewDetails) *pb.InterviewSchedule {
	interview := details.Interview
	var scheduledAt string
	if interview.ScheduledAt != nil {
		scheduledAt = businessclock.FormatRFC3339(*interview.ScheduledAt)
	}
	return &pb.InterviewSchedule{
		InterviewId:          interview.ID,
		ApplicationId:        interview.ApplicationID,
		InterviewerId:        interview.InterviewerID,
		RoundNo:              interview.RoundNo,
		Title:                interview.Title,
		Mode:                 interview.Mode,
		MeetingUrl:           interview.MeetingURL,
		Location:             interview.Location,
		DurationMinutes:      interview.DurationMinutes,
		CandidateNote:        interview.CandidateNote,
		InternalNote:         interview.InternalNote,
		CancelReason:         interview.CancelReason,
		ScheduledAt:          scheduledAt,
		Status:               string(interview.Status),
		CreatedBy:            int64PtrValue(interview.CreatedBy),
		CreatedAt:            formatTime(interview.CreatedAt),
		UpdatedAt:            formatTime(interview.UpdatedAt),
		InterviewerName:      details.InterviewerName,
		ApplicationStatusKey: string(details.ApplicationStatusKey),
		JobTitle:             details.JobTitle,
		CandidateName:        details.CandidateName,
		CandidatePhone:       details.CandidatePhone,
		ResumeUrl:            details.ResumeURL,
		HasFeedback:          details.HasFeedbackForRequest,
	}
}

func ToPBInterviews(rows []repository.InterviewDetails) []*pb.InterviewSchedule {
	list := make([]*pb.InterviewSchedule, 0, len(rows))
	for _, row := range rows {
		list = append(list, ToPBInterview(row))
	}
	return list
}

func ToPBFeedback(feedback *model.Feedback) *pb.InterviewFeedback {
	if feedback == nil {
		return nil
	}
	return &pb.InterviewFeedback{
		FeedbackId:          feedback.ID,
		InterviewId:         feedback.InterviewID,
		ApplicationId:       feedback.ApplicationID,
		InterviewerId:       feedback.InterviewerID,
		Recommendation:      feedback.Recommendation,
		Score:               feedback.Score,
		DimensionScoresJson: feedback.DimensionScoresJSON,
		Comments:            feedback.Comments,
		SubmittedAt:         formatTime(feedback.SubmittedAt),
		UpdatedAt:           formatTime(feedback.UpdatedAt),
	}
}

func ToPBStaffUsers(page port.StaffPage) []*pb.StaffUserInfo {
	list := make([]*pb.StaffUserInfo, 0, len(page.List))
	for _, user := range page.List {
		list = append(list, &pb.StaffUserInfo{
			UserId:       user.UserID,
			Username:     user.Username,
			Email:        user.Email,
			Status:       user.Status,
			AccountType:  user.AccountType,
			Roles:        user.Roles,
			TokenVersion: user.TokenVersion,
			CreatedAt:    formatTime(user.CreatedAt),
		})
	}
	return list
}

func int64PtrValue(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return businessclock.FormatRFC3339(value)
}
