package service

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"logic-grpc-service/model"
	"logic-grpc-service/pkg/authz"
	"logic-grpc-service/pkg/errs"
	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/recruitment/pb"
	"logic-grpc-service/repository"
)

type CollaborationService struct {
	authzRepo    *repository.AuthzRepo
	collaboration *repository.CollaborationRepo
	applications *repository.ApplicationRepo
	profiles     *repository.ProfileRepo
	jobs         *repository.JobRepo
	users        *repository.UserRepo
	interviews   *repository.InterviewRepo
	offers       *repository.OfferRepo
	resumes      *repository.ResumeRepo
	serviceAuth  *ServiceAuthorizer
	scopeEval    *scopeEvaluator
}

func NewCollaborationService(
	authzRepo *repository.AuthzRepo,
	collaboration *repository.CollaborationRepo,
	applications *repository.ApplicationRepo,
	profiles *repository.ProfileRepo,
	jobs *repository.JobRepo,
	users *repository.UserRepo,
	interviews *repository.InterviewRepo,
	offers *repository.OfferRepo,
	resumes *repository.ResumeRepo,
	serviceAuth *ServiceAuthorizer,
	scopeEval *scopeEvaluator,
) *CollaborationService {
	return &CollaborationService{
		authzRepo:    authzRepo,
		collaboration: collaboration,
		applications: applications,
		profiles:     profiles,
		jobs:         jobs,
		users:        users,
		interviews:   interviews,
		offers:       offers,
		resumes:      resumes,
		serviceAuth:  serviceAuth,
		scopeEval:    scopeEval,
	}
}

// ── Authorization helpers ─────────────────────────────────────────────

func (s *CollaborationService) requireCollabReadScope(ctx context.Context, staffUserID uint64, candidateUserID uint64) error {
	if err := s.serviceAuth.AuthorizePermission(ctx, staffUserID, authz.PermApplicationRead); err != nil {
		return fmt.Errorf("permission denied: application.read required: %w", err)
	}
	return s.requireCandidateAccess(ctx, staffUserID, candidateUserID)
}

func (s *CollaborationService) requireCandidateAccess(ctx context.Context, staffUserID uint64, candidateUserID uint64) error {
	appRows, _, err := s.applications.ListMy(ctx, int64(candidateUserID), 1, 1)
	if err != nil {
		return err
	}
	if len(appRows) == 0 {
		return fmt.Errorf("no applications found for candidate %d", candidateUserID)
	}
	detail, err := s.applications.GetDetail(ctx, appRows[0].ApplicationID)
	if err != nil {
		return err
	}
	if detail == nil {
		return fmt.Errorf("application %d not found", appRows[0].ApplicationID)
	}
	_, err = s.scopeEval.evalScope(ctx, staffUserID, func() (*jobScopeTarget, error) {
		job, err := s.jobs.GetByID(ctx, detail.JobID)
		if err != nil {
			return nil, err
		}
		if job == nil {
			return nil, fmt.Errorf("job %d not found", detail.JobID)
		}
		return &jobScopeTarget{
			ID:           job.ID,
			HrID:         job.HrID,
			DepartmentID: job.DepartmentID,
			LocationID:   job.LocationID,
		}, nil
	})
	if err != nil {
		return fmt.Errorf("scope denied for candidate %d: %w", candidateUserID, err)
	}
	return nil
}

// ── Candidate Workspace ───────────────────────────────────────────────

func (s *CollaborationService) GetCandidateWorkspace(ctx context.Context, req *pb.GetCandidateWorkspaceRequest) (*pb.GetCandidateWorkspaceResponse, error) {
	candidateUID := uint64(req.CandidateUserId)
	staffUID := uint64(req.StaffUserId)

	if err := s.requireCollabReadScope(ctx, staffUID, candidateUID); err != nil {
		return nil, err
	}

	// Get profile
	profile, err := s.profiles.GetByUserID(ctx, int64(candidateUID))
	if err != nil {
		logger.L().Error("get candidate profile failed", zap.Uint64("candidate_user_id", candidateUID), zap.Error(err))
		return nil, err
	}

	var profileRealName, profilePhone, profileEducation, profileSchool, profileWorkExperience string
	var profileSkills []string
	if profile != nil {
		profileRealName = profile.RealName
		profilePhone = profile.Phone
		profileEducation = profile.Education
		profileSchool = profile.School
		profileWorkExperience = profile.WorkExperience
		if profile.Skills != "" {
			profileSkills = splitSkills(profile.Skills)
		}
	}

	// Get resume URL
	resumeURL := ""
	resume, err := s.resumes.GetValidByUserID(ctx, int64(candidateUID))
	if err != nil {
		logger.L().Error("get resume failed", zap.Uint64("candidate_user_id", candidateUID), zap.Error(err))
	} else if resume != nil {
		resumeURL = resume.OSSKey
	}

	// Get applications with department/location
	appRows, _, err := s.applications.ListMy(ctx, int64(candidateUID), 1, 100)
	if err != nil {
		return nil, err
	}

	// Build job_id -> job map for department/location lookup
	jobIDs := make([]int64, 0, len(appRows))
	jobIDSet := make(map[int64]struct{})
	for _, a := range appRows {
		if _, ok := jobIDSet[a.JobID]; !ok {
			jobIDs = append(jobIDs, a.JobID)
			jobIDSet[a.JobID] = struct{}{}
		}
	}
	jobMap := make(map[int64]*model.Job)
	for _, jid := range jobIDs {
		job, err := s.jobs.GetByID(ctx, jid)
		if err != nil {
			logger.L().Error("get job failed", zap.Int64("job_id", jid), zap.Error(err))
			continue
		}
		if job != nil {
			jobMap[jid] = job
		}
	}

	workspaceApps := make([]*pb.CandidateWorkspaceApplication, 0, len(appRows))
	for _, a := range appRows {
		appDept := ""
		appLoc := ""
		if job, ok := jobMap[a.JobID]; ok {
			appDept = job.Department
			appLoc = job.Location
		}
		workspaceApps = append(workspaceApps, &pb.CandidateWorkspaceApplication{
			ApplicationId: a.ApplicationID,
			JobId:         a.JobID,
			JobTitle:      a.JobTitle,
			Department:    appDept,
			Location:      appLoc,
			StatusKey:     a.StatusKey,
			RoundNo:       a.RoundNo,
			IsCurrent:     a.IsCurrent,
			AppliedAt:     a.AppliedAt.Format(time.RFC3339),
		})
	}

	// Get tags
	pbTagsResp, err := s.ListCandidateTags(ctx, &pb.ListCandidateTagsRequest{
		StaffUserId:     req.StaffUserId,
		CandidateUserId: candidateUID,
	})
	if err != nil {
		pbTagsResp = nil
	}
	var pbTags []*pb.CandidateTagInfo
	if pbTagsResp != nil {
		pbTags = pbTagsResp.List
	}

	// Get interview details
	interviewRows, err := s.interviews.ListByCandidate(ctx, int64(candidateUID))
	if err != nil {
		logger.L().Error("list candidate interviews failed", zap.Uint64("candidate_user_id", candidateUID), zap.Error(err))
		interviewRows = nil
	}
	pbInterviews := make([]*pb.CandidateWorkspaceInterview, 0, len(interviewRows))
	for _, iv := range interviewRows {
		var scheduledAt string
		if iv.ScheduledAt != nil {
			scheduledAt = iv.ScheduledAt.Format(time.RFC3339)
		}
		pbInterviews = append(pbInterviews, &pb.CandidateWorkspaceInterview{
			InterviewId:     iv.ID,
			ApplicationId:   iv.ApplicationID,
			Title:           iv.Title,
			Mode:            iv.Mode,
			Status:          iv.Status,
			ScheduledAt:     scheduledAt,
			InterviewerName: iv.InterviewerName,
			JobTitle:        iv.JobTitle,
			RoundNo:         iv.RoundNo,
		})
	}

	// Get offer details
	offerRows, _, _, err := s.offers.ListByCandidate(ctx, int64(candidateUID), "", 100)
	if err != nil {
		logger.L().Error("list candidate offers failed", zap.Uint64("candidate_user_id", candidateUID), zap.Error(err))
		offerRows = nil
	}
	pbOffers := make([]*pb.CandidateWorkspaceOffer, 0, len(offerRows))
	for _, of := range offerRows {
		pbOffers = append(pbOffers, &pb.CandidateWorkspaceOffer{
			OfferId:       of.ID,
			ApplicationId: of.ApplicationID,
			Title:         of.Title,
			Status:        of.Status,
			SalaryRange:   of.SalaryRange,
			Level:         of.Level,
			WorkLocation:  of.WorkLocation,
			StartDate:     of.StartDate,
			JobTitle:      of.JobTitle,
		})
	}

	// Get counts
	totalApps, _ := s.collaboration.CountApplicationsByUser(ctx, candidateUID)
	totalInterviews, _ := s.collaboration.CountInterviewsByCandidate(ctx, candidateUID)
	totalOffers, _ := s.collaboration.CountOffersByCandidate(ctx, candidateUID)
	latestActivity, _ := s.collaboration.GetLatestActivity(ctx, candidateUID)

	var latestActivityStr string
	if latestActivity != nil {
		latestActivityStr = latestActivity.Format(time.RFC3339)
	}

	workspace := &pb.CandidateWorkspace{
		RealName:          profileRealName,
		Phone:             profilePhone,
		Education:         profileEducation,
		School:            profileSchool,
		WorkExperience:    profileWorkExperience,
		Skills:            profileSkills,
		Applications:      workspaceApps,
		Tags:              pbTags,
		TotalApplications: totalApps,
		TotalInterviews:   totalInterviews,
		TotalOffers:       totalOffers,
		LatestActivityAt:  latestActivityStr,
		ResumeUrl:         resumeURL,
		Interviews:        pbInterviews,
		Offers:            pbOffers,
	}

	return &pb.GetCandidateWorkspaceResponse{
		Code:      errs.OK,
		Msg:       "success",
		Workspace: workspace,
	}, nil
}

// ── Notes ──────────────────────────────────────────────────────────────

func (s *CollaborationService) CreateNote(ctx context.Context, req *pb.CreateNoteRequest) (*pb.CreateNoteResponse, error) {
	staffUID := uint64(req.StaffUserId)
	candidateUID := req.CandidateUserId

	if err := s.requireCollabReadScope(ctx, staffUID, candidateUID); err != nil {
		return nil, err
	}
	if err := s.serviceAuth.AuthorizePermission(ctx, staffUID, authz.PermCollaborationNoteCreate); err != nil {
		return nil, fmt.Errorf("permission denied: collaboration.note.create required: %w", err)
	}

	var appID *uint64
	if req.ApplicationId > 0 {
		appID = &req.ApplicationId
	}

	now := time.Now()
	note := &model.CandidateNote{
		CandidateUserID: candidateUID,
		ApplicationID:   appID,
		AuthorUserID:    staffUID,
		Content:         req.Content,
		Visibility:      "internal",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.collaboration.CreateNote(ctx, note); err != nil {
		logger.L().Error("create note failed", zap.Error(err))
		return nil, err
	}

	authorName := s.getUsername(ctx, staffUID)

	return &pb.CreateNoteResponse{
		Code: errs.OK,
		Msg:  "success",
		Note: toPBNote(note, authorName),
	}, nil
}

func (s *CollaborationService) ListNotes(ctx context.Context, req *pb.ListNotesRequest) (*pb.ListNotesResponse, error) {
	staffUID := uint64(req.StaffUserId)
	candidateUID := req.CandidateUserId

	if err := s.requireCollabReadScope(ctx, staffUID, candidateUID); err != nil {
		return nil, err
	}
	if err := s.serviceAuth.AuthorizePermission(ctx, staffUID, authz.PermCollaborationNoteRead); err != nil {
		return nil, fmt.Errorf("permission denied: collaboration.note.read required: %w", err)
	}

	var appID *uint64
	if req.ApplicationId > 0 {
		appID = &req.ApplicationId
	}

	notes, err := s.collaboration.ListNotes(ctx, candidateUID, appID)
	if err != nil {
		logger.L().Error("list notes failed", zap.Error(err))
		return nil, err
	}

	pbNotes := make([]*pb.CandidateNoteInfo, 0, len(notes))
	for _, n := range notes {
		authorName := s.getUsername(ctx, n.AuthorUserID)
		pbNotes = append(pbNotes, toPBNote(&n, authorName))
	}

	return &pb.ListNotesResponse{
		Code: errs.OK,
		Msg:  "success",
		List: pbNotes,
	}, nil
}

// ── Tags ───────────────────────────────────────────────────────────────

func (s *CollaborationService) CreateTag(ctx context.Context, req *pb.CreateTagRequest) (*pb.CreateTagResponse, error) {
	staffUID := uint64(req.StaffUserId)

	if err := s.serviceAuth.AuthorizePermission(ctx, staffUID, authz.PermCollaborationTagManage); err != nil {
		return nil, fmt.Errorf("permission denied: collaboration.tag.manage required: %w", err)
	}

	color := req.Color
	if color == "" {
		color = "#409eff"
	}

	tag := &model.CandidateTag{
		Name:      req.Name,
		Color:     color,
		CreatedBy: &staffUID,
	}

	if err := s.collaboration.CreateTag(ctx, tag); err != nil {
		logger.L().Error("create tag failed", zap.Error(err))
		return nil, err
	}

	return &pb.CreateTagResponse{
		Code: errs.OK,
		Msg:  "success",
		Tag:  toPBTag(tag),
	}, nil
}

func (s *CollaborationService) ListTags(ctx context.Context, req *pb.ListTagsRequest) (*pb.ListTagsResponse, error) {
	staffUID := uint64(req.StaffUserId)

	if err := s.serviceAuth.AuthorizePermission(ctx, staffUID, authz.PermCollaborationTagManage); err != nil {
		return nil, fmt.Errorf("permission denied: collaboration.tag.manage required: %w", err)
	}

	tags, err := s.collaboration.ListTags(ctx)
	if err != nil {
		logger.L().Error("list tags failed", zap.Error(err))
		return nil, err
	}

	pbTags := make([]*pb.CandidateTagInfo, 0, len(tags))
	for _, t := range tags {
		pbTags = append(pbTags, toPBTag(&t))
	}

	return &pb.ListTagsResponse{
		Code: errs.OK,
		Msg:  "success",
		List: pbTags,
	}, nil
}

func (s *CollaborationService) AssignTag(ctx context.Context, req *pb.AssignTagRequest) (*pb.CommonResponse, error) {
	staffUID := uint64(req.StaffUserId)

	if err := s.serviceAuth.AuthorizePermission(ctx, staffUID, authz.PermCollaborationTagManage); err != nil {
		return nil, fmt.Errorf("permission denied: collaboration.tag.manage required: %w", err)
	}

	assignment := &model.CandidateTagAssignment{
		TagID:           req.TagId,
		CandidateUserID: req.CandidateUserId,
		CreatedBy:       &staffUID,
	}

	if err := s.collaboration.AssignTag(ctx, assignment); err != nil {
		logger.L().Error("assign tag failed", zap.Error(err))
		return nil, err
	}

	return &pb.CommonResponse{Code: errs.OK, Msg: "success"}, nil
}

func (s *CollaborationService) ListCandidateTags(ctx context.Context, req *pb.ListCandidateTagsRequest) (*pb.ListCandidateTagsResponse, error) {
	staffUID := uint64(req.StaffUserId)
	candidateUID := req.CandidateUserId

	if err := s.requireCollabReadScope(ctx, staffUID, candidateUID); err != nil {
		return nil, err
	}

	tags, err := s.collaboration.ListCandidateTags(ctx, candidateUID)
	if err != nil {
		logger.L().Error("list candidate tags failed", zap.Error(err))
		return nil, err
	}

	pbTags := make([]*pb.CandidateTagInfo, 0, len(tags))
	for _, t := range tags {
		pbTags = append(pbTags, toPBTag(&t))
	}

	return &pb.ListCandidateTagsResponse{
		Code: errs.OK,
		Msg:  "success",
		List: pbTags,
	}, nil
}

func (s *CollaborationService) UnassignTag(ctx context.Context, req *pb.UnassignTagRequest) (*pb.CommonResponse, error) {
	staffUID := uint64(req.StaffUserId)

	if err := s.serviceAuth.AuthorizePermission(ctx, staffUID, authz.PermCollaborationTagManage); err != nil {
		return nil, fmt.Errorf("permission denied: collaboration.tag.manage required: %w", err)
	}

	if err := s.collaboration.UnassignTag(ctx, req.TagId, req.CandidateUserId); err != nil {
		logger.L().Error("unassign tag failed", zap.Error(err))
		return nil, err
	}

	return &pb.CommonResponse{Code: errs.OK, Msg: "success"}, nil
}

// ── Follow-up Tasks ────────────────────────────────────────────────────

func (s *CollaborationService) CreateFollowUpTask(ctx context.Context, req *pb.CreateFollowUpTaskRequest) (*pb.CreateFollowUpTaskResponse, error) {
	staffUID := uint64(req.StaffUserId)
	candidateUID := req.CandidateUserId

	if err := s.requireCollabReadScope(ctx, staffUID, candidateUID); err != nil {
		return nil, err
	}
	if err := s.serviceAuth.AuthorizePermission(ctx, staffUID, authz.PermCollaborationTaskManage); err != nil {
		return nil, fmt.Errorf("permission denied: collaboration.task.manage required: %w", err)
	}

	var dueAt *time.Time
	if req.DueAt != "" {
		t, err := time.Parse(time.RFC3339, req.DueAt)
		if err != nil {
			return nil, fmt.Errorf("invalid due_at format: %w", err)
		}
		dueAt = &t
	}

	var appID *uint64
	if req.ApplicationId > 0 {
		appID = &req.ApplicationId
	}

	now := time.Now()
	task := &model.FollowUpTask{
		CandidateUserID: candidateUID,
		ApplicationID:   appID,
		AssigneeUserID:  req.AssigneeUserId,
		CreatedBy:       staffUID,
		Title:           req.Title,
		Description:     req.Description,
		DueAt:           dueAt,
		Status:          "pending",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.collaboration.CreateTask(ctx, task); err != nil {
		logger.L().Error("create follow-up task failed", zap.Error(err))
		return nil, err
	}

	assigneeName := s.getUsername(ctx, task.AssigneeUserID)
	candidateName := s.getUsername(ctx, task.CandidateUserID)

	return &pb.CreateFollowUpTaskResponse{
		Code: errs.OK,
		Msg:  "success",
		Task: toPBTask(task, assigneeName, candidateName),
	}, nil
}

// B2 fix: add scope check for ListFollowUpTasks when candidate_user_id filter is provided.
func (s *CollaborationService) ListFollowUpTasks(ctx context.Context, req *pb.ListFollowUpTasksRequest) (*pb.ListFollowUpTasksResponse, error) {
	staffUID := uint64(req.StaffUserId)

	if err := s.serviceAuth.AuthorizePermission(ctx, staffUID, authz.PermCollaborationTaskManage); err != nil {
		return nil, fmt.Errorf("permission denied: collaboration.task.manage required: %w", err)
	}

	// B2: Verify candidate scope if candidate filter is provided
	if req.CandidateUserId > 0 {
		if err := s.requireCollabReadScope(ctx, staffUID, req.CandidateUserId); err != nil {
			return nil, err
		}
	}

	filter := repository.TaskFilter{Status: req.Status}
	if req.CandidateUserId > 0 {
		filter.CandidateUserID = &req.CandidateUserId
	}
	if req.AssigneeUserId > 0 {
		filter.AssigneeUserID = &req.AssigneeUserId
	}

	tasks, err := s.collaboration.ListTasks(ctx, filter)
	if err != nil {
		logger.L().Error("list follow-up tasks failed", zap.Error(err))
		return nil, err
	}

	pbTasks := make([]*pb.FollowUpTaskInfo, 0, len(tasks))
	for _, t := range tasks {
		assigneeName := s.getUsername(ctx, t.AssigneeUserID)
		candidateName := s.getUsername(ctx, t.CandidateUserID)
		pbTasks = append(pbTasks, toPBTask(&t, assigneeName, candidateName))
	}

	return &pb.ListFollowUpTasksResponse{
		Code: errs.OK,
		Msg:  "success",
		List: pbTasks,
	}, nil
}

// B2 fix: look up task first to get candidate_user_id, then verify scope.
func (s *CollaborationService) CompleteFollowUpTask(ctx context.Context, req *pb.CompleteFollowUpTaskRequest) (*pb.CommonResponse, error) {
	staffUID := uint64(req.StaffUserId)

	if err := s.serviceAuth.AuthorizePermission(ctx, staffUID, authz.PermCollaborationTaskManage); err != nil {
		return nil, fmt.Errorf("permission denied: collaboration.task.manage required: %w", err)
	}

	// B2: Look up task first to get candidate_user_id
	task, err := s.collaboration.GetTask(ctx, req.TaskId)
	if err != nil {
		logger.L().Error("get follow-up task failed", zap.Error(err))
		return nil, err
	}
	if task == nil {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "task not found"}, nil
	}
	if err := s.requireCollabReadScope(ctx, staffUID, task.CandidateUserID); err != nil {
		return nil, err
	}

	now := time.Now()
	if err := s.collaboration.UpdateTaskStatus(ctx, req.TaskId, "completed", &now); err != nil {
		logger.L().Error("complete follow-up task failed", zap.Error(err))
		return nil, err
	}

	return &pb.CommonResponse{Code: errs.OK, Msg: "success"}, nil
}

// B2 fix: look up task first to get candidate_user_id, then verify scope.
func (s *CollaborationService) GetFollowUpTask(ctx context.Context, req *pb.GetFollowUpTaskRequest) (*pb.GetFollowUpTaskResponse, error) {
	staffUID := uint64(req.StaffUserId)

	if err := s.serviceAuth.AuthorizePermission(ctx, staffUID, authz.PermCollaborationTaskManage); err != nil {
		return nil, fmt.Errorf("permission denied: collaboration.task.manage required: %w", err)
	}

	task, err := s.collaboration.GetTask(ctx, req.TaskId)
	if err != nil {
		logger.L().Error("get follow-up task failed", zap.Error(err))
		return nil, err
	}
	if task == nil {
		return &pb.GetFollowUpTaskResponse{Code: errs.ErrBadRequest, Msg: "task not found"}, nil
	}

	// B2: Verify candidate scope after getting the task
	if err := s.requireCollabReadScope(ctx, staffUID, task.CandidateUserID); err != nil {
		return nil, err
	}

	assigneeName := s.getUsername(ctx, task.AssigneeUserID)
	candidateName := s.getUsername(ctx, task.CandidateUserID)

	return &pb.GetFollowUpTaskResponse{
		Code: errs.OK,
		Msg:  "success",
		Task: toPBTask(task, assigneeName, candidateName),
	}, nil
}

// ── Timeline Events (B3: FR-2) ────────────────────────────────────────

func (s *CollaborationService) ListTimelineEvents(ctx context.Context, req *pb.ListTimelineEventsRequest) (*pb.ListTimelineEventsResponse, error) {
	staffUID := uint64(req.StaffUserId)
	candidateUID := req.CandidateUserId

	if err := s.requireCollabReadScope(ctx, staffUID, candidateUID); err != nil {
		return nil, err
	}

	var events []*pb.TimelineEventInfo

	// 1. Notes
	notes, err := s.collaboration.ListNotes(ctx, candidateUID, nil)
	if err != nil {
		logger.L().Error("list notes for timeline failed", zap.Error(err))
	} else {
		for _, n := range notes {
			events = append(events, &pb.TimelineEventInfo{
				Id:            fmt.Sprintf("note-%d", n.ID),
				EventType:     "note",
				Title:         s.getUsername(ctx, n.AuthorUserID) + " 添加了备注",
				Description:   n.Content,
				Timestamp:     n.CreatedAt.Format(time.RFC3339),
				ActorName:     s.getUsername(ctx, n.AuthorUserID),
				ApplicationId: uint64PtrToInt64(n.ApplicationID),
			})
		}
	}

	// 2. Application status transitions
	appRows, _, err := s.applications.ListMy(ctx, int64(candidateUID), 1, 100)
	if err == nil {
		for _, a := range appRows {
			transitions, err := s.applications.ListTransitions(ctx, a.ApplicationID)
			if err != nil {
				continue
			}
			for _, t := range transitions {
				actorName := s.getUsername(ctx, uint64(t.ActorUserID))
				desc := t.Reason
				events = append(events, &pb.TimelineEventInfo{
					Id:            fmt.Sprintf("transition-%d", t.ID),
					EventType:     "status_transition",
					Title:         actorName + " 变更状态: " + t.FromStatus + " → " + t.ToStatus,
					Description:   desc,
					Timestamp:     t.CreatedAt.Format(time.RFC3339),
					ActorName:     actorName,
					ApplicationId: t.ApplicationID,
				})
			}
		}
	}

	// 3. Interviews
	interviewRows, err := s.interviews.ListByCandidate(ctx, int64(candidateUID))
	if err != nil {
		logger.L().Error("list interviews for timeline failed", zap.Error(err))
	} else {
		for _, iv := range interviewRows {
			var scheduledAt string
			if iv.ScheduledAt != nil {
				scheduledAt = iv.ScheduledAt.Format(time.RFC3339)
			}
			title := "面试: " + iv.Title
			if iv.Status == "cancelled" {
				title = "面试已取消: " + iv.Title
			}
			desc := "面试官: " + iv.InterviewerName
			if scheduledAt != "" {
				desc += " | 时间: " + scheduledAt
			}
			events = append(events, &pb.TimelineEventInfo{
				Id:            fmt.Sprintf("interview-%d", iv.ID),
				EventType:     "interview",
				Title:         title,
				Description:   desc,
				Timestamp:     iv.CreatedAt.Format(time.RFC3339),
				ActorName:     iv.InterviewerName,
				ApplicationId: iv.ApplicationID,
			})
		}
	}

	// 4. Offers
	offerRows, _, _, err := s.offers.ListByCandidate(ctx, int64(candidateUID), "", 100)
	if err != nil {
		logger.L().Error("list offers for timeline failed", zap.Error(err))
	} else {
		for _, of := range offerRows {
			title := "Offer: " + of.Title
			switch of.Status {
			case "draft":
				title = "Offer已创建: " + of.Title
			case "sent":
				title = "Offer已发送: " + of.Title
			case "accepted":
				title = "Offer已接受: " + of.Title
			case "rejected":
				title = "Offer已拒绝: " + of.Title
			case "withdrawn":
				title = "Offer已撤回: " + of.Title
			}
			desc := "薪酬: " + of.SalaryRange + " | 职级: " + of.Level
			events = append(events, &pb.TimelineEventInfo{
				Id:            fmt.Sprintf("offer-%d", of.ID),
				EventType:     "offer",
				Title:         title,
				Description:   desc,
				Timestamp:     of.CreatedAt.Format(time.RFC3339),
				ActorName:     s.getUsername(ctx, uint64(of.CreatedBy)),
				ApplicationId: of.ApplicationID,
			})
		}
	}

	// Sort by timestamp descending (most recent first)
	sortTimelineEvents(events)

	return &pb.ListTimelineEventsResponse{
		Code:   errs.OK,
		Msg:    "success",
		Events: events,
	}, nil
}

// ── Helpers ───────────────────────────────────────────────────────────

func (s *CollaborationService) getUsername(ctx context.Context, userID uint64) string {
	user, err := s.users.GetByID(ctx, int64(userID))
	if err != nil || user == nil {
		return fmt.Sprintf("用户%d", userID)
	}
	return user.Username
}

func sortTimelineEvents(events []*pb.TimelineEventInfo) {
	for i := 0; i < len(events); i++ {
		for j := i + 1; j < len(events); j++ {
			t1, _ := time.Parse(time.RFC3339, events[i].Timestamp)
			t2, _ := time.Parse(time.RFC3339, events[j].Timestamp)
			if t2.After(t1) {
				events[i], events[j] = events[j], events[i]
			}
		}
	}
}

func uint64PtrToInt64(p *uint64) int64 {
	if p == nil {
		return 0
	}
	return int64(*p)
}

// ── Converters ─────────────────────────────────────────────────────────

func toPBNote(note *model.CandidateNote, authorName string) *pb.CandidateNoteInfo {
	pbNote := &pb.CandidateNoteInfo{
		Id:              note.ID,
		CandidateUserId: note.CandidateUserID,
		AuthorUserId:    note.AuthorUserID,
		Content:         note.Content,
		Visibility:      note.Visibility,
		CreatedAt:       note.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       note.UpdatedAt.Format(time.RFC3339),
		AuthorName:      authorName,
	}
	if note.ApplicationID != nil {
		pbNote.ApplicationId = *note.ApplicationID
	}
	return pbNote
}

func toPBTag(tag *model.CandidateTag) *pb.CandidateTagInfo {
	pbTag := &pb.CandidateTagInfo{
		Id:    tag.ID,
		Name:  tag.Name,
		Color: tag.Color,
	}
	if tag.CreatedBy != nil {
		pbTag.CreatedBy = *tag.CreatedBy
	}
	return pbTag
}

func toPBTask(task *model.FollowUpTask, assigneeName, candidateName string) *pb.FollowUpTaskInfo {
	pbTask := &pb.FollowUpTaskInfo{
		Id:              task.ID,
		CandidateUserId: task.CandidateUserID,
		AssigneeUserId:  task.AssigneeUserID,
		CreatedBy:       task.CreatedBy,
		Title:           task.Title,
		Description:     task.Description,
		Status:          task.Status,
		AssigneeName:    assigneeName,
		CandidateName:   candidateName,
		CreatedAt:       task.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       task.UpdatedAt.Format(time.RFC3339),
	}
	if task.ApplicationID != nil {
		pbTask.ApplicationId = *task.ApplicationID
	}
	if task.DueAt != nil {
		pbTask.DueAt = task.DueAt.Format(time.RFC3339)
	}
	if task.CompletedAt != nil {
		pbTask.CompletedAt = task.CompletedAt.Format(time.RFC3339)
	}
	return pbTask
}
