package service

import (
	"context"
	"errors"
	"fmt"

	"smart-recruit-recruitment-service/internal/application/command"
	"smart-recruit-recruitment-service/internal/application/dto"
	"smart-recruit-recruitment-service/internal/application/port"
	"smart-recruit-recruitment-service/internal/domain/model"
	"smart-recruit-recruitment-service/internal/domain/policy"
	"smart-recruit-recruitment-service/internal/domain/repository"
)

var (
	ErrApplicationRepositoryRequired = errors.New("application repository is required")
	ErrProfileRepositoryRequired     = errors.New("candidate profile repository is required")
	ErrOutboxRequired                = errors.New("outbox publisher is required")
	ErrCollaborationRepoRequired     = errors.New("collaboration repository is required")
	ErrCollaborationAuthRequired     = errors.New("collaboration authorizer is required")
	ErrTaxonomyRepositoryRequired    = errors.New("taxonomy repository is required")
	ErrUsageStatsRepositoryRequired  = errors.New("usage stats repository is required")
	ErrUsageStatsAuthRequired        = errors.New("usage stats authorizer is required")
	ErrApplicationConflict           = errors.New("投递状态已变化，请刷新后重试")
)

type ApplicationDeps struct {
	Applications repository.ApplicationRepository
	Profiles     repository.CandidateProfileRepository
	Resumes      repository.ResumeRepository
	Jobs         repository.JobRepository
	Scopes       repository.JobScopeChecker
	Outbox       repository.OutboxPublisher
	Clock        port.Clock
}

type ApplicationLifecycleService struct {
	applications repository.ApplicationRepository
	profiles     repository.CandidateProfileRepository
	resumes      repository.ResumeRepository
	jobs         repository.JobRepository
	scopes       repository.JobScopeChecker
	outbox       repository.OutboxPublisher
	clock        port.Clock
}

func NewApplicationLifecycleService(deps ApplicationDeps) (*ApplicationLifecycleService, error) {
	if deps.Applications == nil {
		return nil, ErrApplicationRepositoryRequired
	}
	if deps.Profiles == nil {
		return nil, ErrProfileRepositoryRequired
	}
	if deps.Resumes == nil {
		return nil, ErrResumeRepositoryRequired
	}
	if deps.Jobs == nil {
		return nil, ErrJobRepositoryRequired
	}
	if deps.Scopes == nil {
		return nil, ErrJobScopeCheckerRequired
	}
	if deps.Outbox == nil {
		return nil, ErrOutboxRequired
	}
	clock := deps.Clock
	if clock == nil {
		clock = port.SystemClock{}
	}
	return &ApplicationLifecycleService{
		applications: deps.Applications,
		profiles:     deps.Profiles,
		resumes:      deps.Resumes,
		jobs:         deps.Jobs,
		scopes:       deps.Scopes,
		outbox:       deps.Outbox,
		clock:        clock,
	}, nil
}

func (s *ApplicationLifecycleService) ApplyJob(ctx context.Context, cmd command.ApplyJob) (dto.ApplyJobResult, error) {
	profile, err := s.profiles.GetByUserID(ctx, cmd.UserID)
	if err != nil {
		return dto.ApplyJobResult{}, err
	}
	resume, err := s.resumes.GetValidByUserID(ctx, cmd.UserID)
	if err != nil {
		return dto.ApplyJobResult{}, err
	}
	job, err := s.jobsForApplication(ctx, cmd.JobID)
	if err != nil {
		return dto.ApplyJobResult{}, err
	}
	if err := policy.ValidateApplyPreconditions(profile, resume, job); err != nil {
		return dto.ApplyJobResult{}, err
	}
	application := &model.Application{
		UserID:    cmd.UserID,
		JobID:     cmd.JobID,
		ResumeID:  resume.ID,
		Status:    model.StatusKeyToLegacy[policy.DefaultApplicationStatusKey()],
		StatusKey: policy.DefaultApplicationStatusKey(),
		AppliedAt: s.clock.Now(),
	}
	if err := s.applications.CreateNewRound(ctx, application, func(applicationID int64) error {
		displayName := policy.CandidateDisplayName(profile.RealName, cmd.UserID)
		content := fmt.Sprintf("%s 投递了「%s」岗位，请及时查看简历。", displayName, job.Title)
		if err := s.outbox.WriteEvent(ctx, model.OutboxEvent{
			Topic:         "application.notification_requested",
			AggregateType: "application",
			AggregateID:   uint64(applicationID),
			EventType:     "notification.create",
			Payload: model.NotificationPayload{
				ReceiverID:          job.HRID,
				ReceiverAccountType: "staff",
				Type:                "new_application",
				Title:               "新的岗位投递",
				Content:             content,
				Link:                fmt.Sprintf("/hr/jobs/%d/applications", job.ID),
				BizType:             "application",
				BizID:               applicationID,
			},
		}); err != nil {
			return err
		}
		return s.outbox.WriteEvent(ctx, model.OutboxEvent{
			Topic:         "application.email_requested",
			AggregateType: "application",
			AggregateID:   uint64(applicationID),
			EventType:     "email.send",
			Payload: model.NotificationPayload{
				ReceiverID:          job.HRID,
				ReceiverAccountType: "staff",
				Type:                "new_application",
				Title:               "新的岗位投递",
				Content:             content,
				Link:                fmt.Sprintf("/hr/jobs/%d/applications", job.ID),
				BizType:             "application",
				BizID:               applicationID,
				JobTitle:            job.Title,
			},
		})
	}); err != nil {
		return dto.ApplyJobResult{}, err
	}
	s.outbox.Signal()
	return dto.ApplyJobResult{ApplicationID: application.ID}, nil
}

func (s *ApplicationLifecycleService) UpdateStatus(ctx context.Context, cmd command.UpdateApplicationStatus) (dto.StatusChangeResult, error) {
	targetKey, err := policy.TargetStatusKey(cmd.StatusKey, cmd.Status)
	if err != nil {
		return dto.StatusChangeResult{}, err
	}
	detail, err := s.applications.GetDetail(ctx, cmd.ApplicationID)
	if err != nil {
		return dto.StatusChangeResult{}, err
	}
	if detail == nil {
		return dto.StatusChangeResult{}, ErrJobForbidden
	}
	currentKey, isRePass, legacyStatus, err := policy.ValidateStatusChange(*detail, targetKey, cmd.Reason)
	if err != nil {
		return dto.StatusChangeResult{}, err
	}
	scope, err := s.scopes.CheckJobScope(ctx, cmd.HRID, detail.JobID)
	if err != nil {
		return dto.StatusChangeResult{}, err
	}
	rows, err := s.applications.UpdateStatus(ctx, cmd.ApplicationID, currentKey, targetKey, legacyStatus, cmd.HRID, scope, isRePass, cmd.Reason, func(rows int64) error {
		if rows == 0 {
			return nil
		}
		notifyType, content := policy.BuildApplicationNotification(targetKey, isRePass, *detail)
		if notifyType == "" {
			return nil
		}
		payload := model.NotificationPayload{
			ReceiverID:          detail.UserID,
			ReceiverAccountType: "candidate",
			Type:                notifyType,
			Title:               "投递进展更新",
			Content:             content,
			Link:                "/applications",
			BizType:             "application",
			BizID:               cmd.ApplicationID,
			JobTitle:            detail.JobTitle,
			RecipientName:       detail.RealName,
		}
		if err := s.outbox.WriteEvent(ctx, model.OutboxEvent{Topic: "application.notification_requested", AggregateType: "application", AggregateID: uint64(cmd.ApplicationID), EventType: "notification.create", Payload: payload}); err != nil {
			return err
		}
		return s.outbox.WriteEvent(ctx, model.OutboxEvent{Topic: "application.email_requested", AggregateType: "application", AggregateID: uint64(cmd.ApplicationID), EventType: "email.send", Payload: payload})
	})
	if err != nil {
		return dto.StatusChangeResult{}, err
	}
	s.outbox.Signal()
	if rows == 0 {
		return dto.StatusChangeResult{}, ErrApplicationConflict
	}
	return dto.StatusChangeResult{FromStatus: currentKey, ToStatus: targetKey, IsRePass: isRePass}, nil
}

func (s *ApplicationLifecycleService) jobsForApplication(ctx context.Context, jobID int64) (*model.Job, error) {
	getter, ok := s.jobs.(interface {
		GetByID(context.Context, int64) (*model.Job, error)
	})
	if !ok {
		return nil, errors.New("job repository GetByID is required for application service")
	}
	return getter.GetByID(ctx, jobID)
}

type CollaborationService struct {
	repo  repository.CollaborationRepository
	authz repository.CollaborationAuthorizer
	clock port.Clock
}

func NewCollaborationService(repo repository.CollaborationRepository, authz repository.CollaborationAuthorizer, clock port.Clock) (*CollaborationService, error) {
	if repo == nil {
		return nil, ErrCollaborationRepoRequired
	}
	if authz == nil {
		return nil, ErrCollaborationAuthRequired
	}
	if clock == nil {
		clock = port.SystemClock{}
	}
	return &CollaborationService{repo: repo, authz: authz, clock: clock}, nil
}

func (s *CollaborationService) CreateNote(ctx context.Context, cmd command.CreateNote) (dto.CreateNoteResult, error) {
	if err := s.requireCandidatePermission(ctx, cmd.StaffUserID, cmd.CandidateUserID, model.PermissionCollaborationNoteCreate); err != nil {
		return dto.CreateNoteResult{}, err
	}
	var appID *uint64
	if cmd.ApplicationID > 0 {
		appID = &cmd.ApplicationID
	}
	now := s.clock.Now()
	note := &model.CandidateNote{CandidateUserID: cmd.CandidateUserID, ApplicationID: appID, AuthorUserID: cmd.StaffUserID, Content: cmd.Content, Visibility: "internal", CreatedAt: now, UpdatedAt: now}
	if err := s.repo.CreateNote(ctx, note); err != nil {
		return dto.CreateNoteResult{}, err
	}
	return dto.CreateNoteResult{Note: *note}, nil
}

func (s *CollaborationService) CreateTag(ctx context.Context, cmd command.CreateTag) (dto.CreateTagResult, error) {
	if err := s.authz.AuthorizePermission(ctx, cmd.StaffUserID, model.PermissionCollaborationTagManage); err != nil {
		return dto.CreateTagResult{}, err
	}
	tag := &model.CandidateTag{Name: cmd.Name, Color: policy.DefaultTagColor(cmd.Color), CreatedBy: &cmd.StaffUserID}
	if err := s.repo.CreateTag(ctx, tag); err != nil {
		return dto.CreateTagResult{}, err
	}
	return dto.CreateTagResult{Tag: *tag}, nil
}

func (s *CollaborationService) AssignTag(ctx context.Context, cmd command.AssignTag) error {
	if err := s.requireCandidatePermission(ctx, cmd.StaffUserID, cmd.CandidateUserID, model.PermissionCollaborationTagManage); err != nil {
		return err
	}
	return s.repo.AssignTag(ctx, &model.CandidateTagAssignment{TagID: cmd.TagID, CandidateUserID: cmd.CandidateUserID, CreatedBy: &cmd.StaffUserID})
}

func (s *CollaborationService) CreateFollowUpTask(ctx context.Context, cmd command.CreateFollowUpTask) (dto.CreateFollowUpTaskResult, error) {
	if err := s.requireCandidatePermission(ctx, cmd.StaffUserID, cmd.CandidateUserID, model.PermissionCollaborationTaskManage); err != nil {
		return dto.CreateFollowUpTaskResult{}, err
	}
	var appID *uint64
	if cmd.ApplicationID > 0 {
		appID = &cmd.ApplicationID
	}
	now := s.clock.Now()
	task := &model.FollowUpTask{CandidateUserID: cmd.CandidateUserID, ApplicationID: appID, AssigneeUserID: cmd.AssigneeUserID, CreatedBy: cmd.StaffUserID, Title: cmd.Title, Description: cmd.Description, DueAt: cmd.DueAt, Status: "pending", CreatedAt: now, UpdatedAt: now}
	if err := s.repo.CreateTask(ctx, task); err != nil {
		return dto.CreateFollowUpTaskResult{}, err
	}
	return dto.CreateFollowUpTaskResult{Task: *task}, nil
}

func (s *CollaborationService) requireCandidatePermission(ctx context.Context, staffUserID, candidateUserID uint64, permission string) error {
	if err := s.authz.RequireCandidateAccess(ctx, staffUserID, candidateUserID); err != nil {
		return err
	}
	return s.authz.AuthorizePermission(ctx, staffUserID, permission)
}

type TaxonomyService struct {
	repo repository.TaxonomyRepository
}

func NewTaxonomyService(repo repository.TaxonomyRepository) (*TaxonomyService, error) {
	if repo == nil {
		return nil, ErrTaxonomyRepositoryRequired
	}
	return &TaxonomyService{repo: repo}, nil
}

func (s *TaxonomyService) CreateDepartment(ctx context.Context, cmd command.CreateDepartment) (dto.CreateDepartmentResult, error) {
	var parent *model.DepartmentNode
	var err error
	if cmd.ParentID > 0 {
		parent, err = s.repo.GetDepartment(ctx, cmd.ParentID)
		if err != nil {
			return dto.CreateDepartmentResult{}, err
		}
	}
	activeLocations, err := s.repo.ListActiveLocations(ctx)
	if err != nil {
		return dto.CreateDepartmentResult{}, err
	}
	locationIDs := make([]int64, 0, len(activeLocations))
	for _, location := range activeLocations {
		locationIDs = append(locationIDs, location.ID)
	}
	dep, defaultLocIDs, err := policy.BuildDepartmentForCreate(parent, cmd.ParentID, cmd.Name, cmd.SortOrder, cmd.AdminID, locationIDs)
	if err != nil {
		return dto.CreateDepartmentResult{}, err
	}
	if deleted, _ := s.repo.FindDeletedDepartment(ctx, cmd.ParentID, dep.Name); deleted != nil {
		if err := s.repo.ReactivateDepartment(ctx, deleted.ID, cmd.AdminID, dep.Name, cmd.SortOrder); err != nil {
			return dto.CreateDepartmentResult{}, err
		}
		deleted.Name = dep.Name
		deleted.SortOrder = cmd.SortOrder
		deleted.IsActive = 1
		return dto.CreateDepartmentResult{Department: *deleted}, nil
	}
	if err := s.repo.CreateDepartment(ctx, &dep); err != nil {
		return dto.CreateDepartmentResult{}, err
	}
	if dep.ParentID == 0 && len(defaultLocIDs) > 0 {
		_ = s.repo.ReplaceDepartmentLocations(ctx, cmd.AdminID, dep.ID, defaultLocIDs)
	}
	parentPath := "/"
	parentDepth := 0
	if parent != nil {
		parentPath = parent.Path
		parentDepth = parent.Depth
	}
	dep.Path = fmt.Sprintf("%s%d/", parentPath, dep.ID)
	dep.Depth = parentDepth + 1
	_ = s.repo.UpdateDepartmentFields(ctx, dep.ID, map[string]any{"path": dep.Path, "depth": dep.Depth})
	fullName, _ := s.repo.BuildDepartmentFullName(ctx, dep.ID)
	dep.FullName = fullName
	_ = s.repo.UpdateDepartmentFields(ctx, dep.ID, map[string]any{"full_name": fullName})
	return dto.CreateDepartmentResult{Department: dep}, nil
}

type UsageStatsAuthorizer interface {
	AuthorizePermission(ctx context.Context, actorUserID uint64, permission string) error
}

type UsageStatsService struct {
	repo  repository.UsageStatsRepository
	authz UsageStatsAuthorizer
	clock port.Clock
}

func NewUsageStatsService(repo repository.UsageStatsRepository, authz UsageStatsAuthorizer, clock port.Clock) (*UsageStatsService, error) {
	if repo == nil {
		return nil, ErrUsageStatsRepositoryRequired
	}
	if authz == nil {
		return nil, ErrUsageStatsAuthRequired
	}
	if clock == nil {
		clock = port.SystemClock{}
	}
	return &UsageStatsService{repo: repo, authz: authz, clock: clock}, nil
}

func (s *UsageStatsService) GetStats(ctx context.Context, cmd command.GetUsageStats) (dto.UsageStatsResult, error) {
	if err := s.authz.AuthorizePermission(ctx, cmd.ActorUserID, model.PermissionAuditUsageRead); err != nil {
		return dto.UsageStatsResult{}, err
	}
	start, end, err := policy.ParseUsageTimeRange(s.clock.Now(), cmd.StartTime, cmd.EndTime, 30)
	if err != nil {
		return dto.UsageStatsResult{}, err
	}
	switch cmd.Dimension {
	case "user":
		rows, err := s.repo.GetStatsByUser(ctx, start, end)
		return dto.UsageStatsResult{List: rows}, err
	case "session":
		rows, err := s.repo.GetStatsBySession(ctx, start, end)
		return dto.UsageStatsResult{List: rows}, err
	default:
		rows, err := s.repo.GetStatsByModel(ctx, start, end)
		return dto.UsageStatsResult{List: rows}, err
	}
}

func (s *UsageStatsService) GetTrend(ctx context.Context, cmd command.GetUsageTrend) (dto.UsageTrendResult, error) {
	if err := s.authz.AuthorizePermission(ctx, cmd.ActorUserID, model.PermissionAuditUsageRead); err != nil {
		return dto.UsageTrendResult{}, err
	}
	start, end, err := policy.ParseUsageTimeRange(s.clock.Now(), cmd.StartTime, cmd.EndTime, 30)
	if err != nil {
		return dto.UsageTrendResult{}, err
	}
	granularity := cmd.Granularity
	if granularity == "" {
		granularity = "day"
	}
	rows, err := s.repo.GetTrend(ctx, start, end, granularity)
	return dto.UsageTrendResult{List: rows}, err
}
