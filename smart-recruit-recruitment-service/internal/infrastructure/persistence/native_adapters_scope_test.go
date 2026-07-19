package persistence

import (
	"context"
	"strconv"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"smart-recruit-commons/oss"
	sharedauthz "smart-recruit-commons/pkg/authz"
	commonsquota "smart-recruit-commons/quota"
	"smart-recruit-platform-go/errs"
	"smart-recruit-platform-go/metadata"
	"smart-recruit-proto/recruitment/pb"
	domainmodel "smart-recruit-recruitment-service/internal/domain/model"
)

func TestNativeRecruitmentScopeOwnJobs(t *testing.T) {
	ctx := context.Background()
	fixture := newScopeFixture(t)
	fixture.seedScope(t, 101, sharedauthz.ScopeOwnJobs, "", 0)
	fixture.seedJob(t, 1001, 101, 10, 20)
	fixture.seedJob(t, 1002, 202, 10, 20)

	list, err := fixture.job.ListHRJobs(ctx, &pb.ListHRJobsRequest{HrId: 101, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListHRJobs() error = %v", err)
	}
	if list.Code != errs.OK || list.Total != 1 || len(list.List) != 1 || list.List[0].JobId != 1001 {
		t.Fatalf("ListHRJobs() = code %d total %d ids %v, want only 1001", list.Code, list.Total, jobIDs(list.List))
	}

	assertCommonOK(t, func() (*pb.CommonResponse, error) {
		return fixture.job.UpdateJob(ctx, &pb.UpdateJobRequest{HrId: 101, JobId: 1001, Title: "Owned Updated"})
	})
	assertCommonCode(t, errs.ErrForbidden, func() (*pb.CommonResponse, error) {
		return fixture.job.UpdateJob(ctx, &pb.UpdateJobRequest{HrId: 101, JobId: 1002, Title: "Denied"})
	})
	assertCommonOK(t, func() (*pb.CommonResponse, error) {
		return fixture.job.OfflineJob(ctx, &pb.OfflineJobRequest{HrId: 101, JobId: 1001})
	})
	fixture.assertJobStatus(t, 1001, 0)
	assertCommonOK(t, func() (*pb.CommonResponse, error) {
		return fixture.job.OnlineJob(ctx, &pb.OfflineJobRequest{HrId: 101, JobId: 1001})
	})
	fixture.assertJobStatus(t, 1001, 1)
	assertCommonCode(t, errs.ErrForbidden, func() (*pb.CommonResponse, error) {
		return fixture.job.OfflineJob(ctx, &pb.OfflineJobRequest{HrId: 101, JobId: 1002})
	})
	fixture.assertJobStatus(t, 1002, 1)
}

func TestNativeRecruitmentScopeFullAccess(t *testing.T) {
	for _, scopeKey := range []string{sharedauthz.ScopeRecruitingAll, sharedauthz.ScopeSystemAll} {
		t.Run(scopeKey, func(t *testing.T) {
			ctx := context.Background()
			fixture := newScopeFixture(t)
			fixture.seedScope(t, 101, scopeKey, "", 0)
			fixture.seedJob(t, 1001, 101, 10, 20)
			fixture.seedJob(t, 1002, 202, 11, 21)

			list, err := fixture.job.ListHRJobs(ctx, &pb.ListHRJobsRequest{HrId: 101, Page: 1, PageSize: 10})
			if err != nil {
				t.Fatalf("ListHRJobs() error = %v", err)
			}
			if list.Code != errs.OK || list.Total != 2 {
				t.Fatalf("ListHRJobs() = code %d total %d, want OK total 2", list.Code, list.Total)
			}
			assertCommonOK(t, func() (*pb.CommonResponse, error) {
				return fixture.job.UpdateJob(ctx, &pb.UpdateJobRequest{HrId: 101, JobId: 1002, Title: "Full Updated"})
			})
			assertCommonOK(t, func() (*pb.CommonResponse, error) {
				return fixture.job.OfflineJob(ctx, &pb.OfflineJobRequest{HrId: 101, JobId: 1002})
			})
			fixture.assertJobStatus(t, 1002, 0)
		})
	}
}

func TestNativeOnlineJobEnforcesPublishedJobQuota(t *testing.T) {
	ctx := context.Background()
	fixture := newScopeFixture(t)
	fixture.seedScope(t, 101, sharedauthz.ScopeOwnJobs, "", 0)
	fixture.seedJob(t, 1001, 101, 10, 20)
	fixture.seedJob(t, 1002, 101, 10, 20)
	assertCommonOK(t, func() (*pb.CommonResponse, error) {
		return fixture.job.OfflineJob(ctx, &pb.OfflineJobRequest{HrId: 101, JobId: 1002})
	})

	ddl := []string{
		"CREATE TABLE platform_plan_versions (id INTEGER PRIMARY KEY)",
		"CREATE TABLE tenant_subscriptions (id INTEGER PRIMARY KEY, tenant_id INTEGER, plan_version_id INTEGER, status TEXT, starts_at DATETIME, ends_at DATETIME)",
		"CREATE TABLE platform_plan_entitlements (plan_version_id INTEGER, entitlement_key TEXT, value_json TEXT, enforcement_mode TEXT)",
		"CREATE TABLE tenant_entitlement_overrides (tenant_id INTEGER, entitlement_key TEXT, value_json TEXT, expires_at DATETIME)",
		"INSERT INTO platform_plan_versions (id) VALUES (3)",
		"INSERT INTO tenant_subscriptions (id, tenant_id, plan_version_id, status, starts_at) VALUES (1, 1, 3, 'active', '2026-07-01 00:00:00')",
		"INSERT INTO platform_plan_entitlements (plan_version_id, entitlement_key, value_json, enforcement_mode) VALUES (3, 'jobs.published.max', '1', 'hard')",
	}
	for _, statement := range ddl {
		if err := fixture.db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	fixture.job.quota = commonsquota.NewChecker(fixture.db)
	assertCommonCode(t, errs.ErrForbidden, func() (*pb.CommonResponse, error) {
		return fixture.job.OnlineJob(ctx, &pb.OfflineJobRequest{HrId: 101, JobId: 1002})
	})
	fixture.assertJobStatus(t, 1002, 0)
}

func TestNativeRecruitmentScopeDepartmentLocationOR(t *testing.T) {
	ctx := context.Background()
	fixture := newScopeFixture(t)
	fixture.seedScope(t, 303, sharedauthz.ScopeDepartment, "department", 10)
	fixture.seedScope(t, 303, sharedauthz.ScopeLocation, "location", 20)
	fixture.seedJob(t, 1001, 201, 10, 99)
	fixture.seedJob(t, 1002, 202, 99, 20)
	fixture.seedJob(t, 1003, 203, 11, 21)

	list, err := fixture.job.ListHRJobs(ctx, &pb.ListHRJobsRequest{HrId: 303, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListHRJobs() error = %v", err)
	}
	if list.Code != errs.OK || list.Total != 2 {
		t.Fatalf("ListHRJobs() = code %d total %d ids %v, want jobs 1001 and 1002", list.Code, list.Total, jobIDs(list.List))
	}
	assertCommonOK(t, func() (*pb.CommonResponse, error) {
		return fixture.job.UpdateJob(ctx, &pb.UpdateJobRequest{HrId: 303, JobId: 1001, Title: "Dept Updated"})
	})
	assertCommonOK(t, func() (*pb.CommonResponse, error) {
		return fixture.job.OfflineJob(ctx, &pb.OfflineJobRequest{HrId: 303, JobId: 1002})
	})
	assertCommonCode(t, errs.ErrForbidden, func() (*pb.CommonResponse, error) {
		return fixture.job.UpdateJob(ctx, &pb.UpdateJobRequest{HrId: 303, JobId: 1003, Title: "Denied"})
	})
	fixture.assertJobStatus(t, 1002, 0)
	fixture.assertJobTitle(t, 1003, "Job 1003")
}

func TestNativeRecruitmentScopeNoValidScopeFailsClosed(t *testing.T) {
	ctx := context.Background()
	fixture := newScopeFixture(t)
	fixture.seedScope(t, 404, sharedauthz.ScopeSelf, "", 0)
	fixture.seedJob(t, 1001, 404, 10, 20)

	list, err := fixture.job.ListHRJobs(ctx, &pb.ListHRJobsRequest{HrId: 404, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListHRJobs() error = %v", err)
	}
	if list.Code != errs.ErrForbidden || len(list.List) != 0 {
		t.Fatalf("ListHRJobs() = code %d len %d, want forbidden empty", list.Code, len(list.List))
	}
	assertCommonCode(t, errs.ErrForbidden, func() (*pb.CommonResponse, error) {
		return fixture.job.UpdateJob(ctx, &pb.UpdateJobRequest{HrId: 404, JobId: 1001, Title: "Denied"})
	})
	assertCommonCode(t, errs.ErrForbidden, func() (*pb.CommonResponse, error) {
		return fixture.job.OfflineJob(ctx, &pb.OfflineJobRequest{HrId: 404, JobId: 1001})
	})
	fixture.assertJobStatus(t, 1001, 1)
}

func TestNativeRecruitmentScopeApplicationsDeniedAcrossScope(t *testing.T) {
	ctx := context.Background()
	fixture := newScopeFixture(t)
	fixture.seedScope(t, 101, sharedauthz.ScopeOwnJobs, "", 0)
	fixture.seedJob(t, 1001, 101, 10, 20)
	fixture.seedJob(t, 1002, 202, 10, 20)
	appID := fixture.seedApplication(t, 1002, 3001, domainmodel.StatusKeyApplied)
	fixture.seedTransition(t, appID, domainmodel.StatusKeyApplied, domainmodel.StatusKeyViewed, 202)

	list, err := fixture.application.ListJobApplications(ctx, &pb.ListJobApplicationsRequest{HrId: 101, JobId: 1002, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListJobApplications() error = %v", err)
	}
	if list.Code != errs.ErrForbidden {
		t.Fatalf("ListJobApplications() code = %d, want forbidden", list.Code)
	}
	update, err := fixture.application.UpdateApplicationStatus(ctx, &pb.UpdateApplicationStatusRequest{
		HrId:          101,
		ApplicationId: appID,
		StatusKey:     domainmodel.StatusKeyScreenPassed,
		Reason:        "qualified",
	})
	if err != nil {
		t.Fatalf("UpdateApplicationStatus() error = %v", err)
	}
	if update.Code != errs.ErrForbidden {
		t.Fatalf("UpdateApplicationStatus() code = %d, want forbidden", update.Code)
	}
	fixture.assertApplicationStatus(t, appID, domainmodel.StatusKeyApplied)

	transitions, err := fixture.application.ListApplicationStatusTransitions(ctx, &pb.ListApplicationStatusTransitionsRequest{HrId: 101, ApplicationId: appID})
	if err != nil {
		t.Fatalf("ListApplicationStatusTransitions() error = %v", err)
	}
	if transitions.Code != errs.ErrForbidden {
		t.Fatalf("ListApplicationStatusTransitions() code = %d, want forbidden", transitions.Code)
	}
}

func TestNativeRecruitmentScopeApplicationLifecycleActors(t *testing.T) {
	ctx := context.Background()
	fixture := newScopeFixture(t)
	fixture.seedScope(t, 101, sharedauthz.ScopeOwnJobs, "", 0)
	fixture.seedJob(t, 1001, 101, 10, 20)
	fixture.seedJob(t, 1002, 202, 10, 20)
	staffDeniedAppID := fixture.seedApplication(t, 1002, 3001, domainmodel.StatusKeyApplied)

	staffSnapshotCtx := metadata.WithAuthActor(ctx, 101, "staff")
	snapshot, err := fixture.owner.GetApplicationSnapshot(staffSnapshotCtx, &pb.GetApplicationSnapshotRequest{ApplicationId: staffDeniedAppID})
	if err != nil {
		t.Fatalf("GetApplicationSnapshot() error = %v", err)
	}
	if snapshot.Code != errs.ErrForbidden {
		t.Fatalf("GetApplicationSnapshot() code = %d, want forbidden", snapshot.Code)
	}
	staffTransition, err := fixture.owner.ApplyApplicationLifecycleTransition(ctx, &pb.ApplyApplicationLifecycleTransitionRequest{
		ActorUserId:       101,
		ActorAccountType:  "staff",
		ApplicationId:     staffDeniedAppID,
		ExpectedStatusKey: domainmodel.StatusKeyApplied,
		TargetStatusKey:   domainmodel.StatusKeyScreenPassed,
		Reason:            "qualified",
	})
	if err != nil {
		t.Fatalf("ApplyApplicationLifecycleTransition(staff) error = %v", err)
	}
	if staffTransition.Code != errs.ErrForbidden || staffTransition.Changed {
		t.Fatalf("ApplyApplicationLifecycleTransition(staff) = %+v, want forbidden unchanged", staffTransition)
	}
	fixture.assertApplicationStatus(t, staffDeniedAppID, domainmodel.StatusKeyApplied)

	serviceAuthorizedStaffAppID := fixture.seedApplication(t, 1002, 3004, domainmodel.StatusKeyApplied)
	serviceCtx := metadata.WithTenantActor(ctx, metadata.TenantContext{TenantID: 1, AccountType: "service", ClientApp: "interview-service"})
	serviceAuthorizedStaffTransition, err := fixture.owner.ApplyApplicationLifecycleTransition(serviceCtx, &pb.ApplyApplicationLifecycleTransitionRequest{
		ActorUserId:       101,
		ActorAccountType:  "staff",
		ApplicationId:     serviceAuthorizedStaffAppID,
		ExpectedStatusKey: domainmodel.StatusKeyApplied,
		TargetStatusKey:   domainmodel.StatusKeyScreenPassed,
		Reason:            "interviewer submitted feedback",
	})
	if err != nil {
		t.Fatalf("ApplyApplicationLifecycleTransition(service authorized staff actor) error = %v", err)
	}
	if serviceAuthorizedStaffTransition.Code != errs.OK || !serviceAuthorizedStaffTransition.Changed {
		t.Fatalf("ApplyApplicationLifecycleTransition(service authorized staff actor) = %+v, want OK changed", serviceAuthorizedStaffTransition)
	}
	fixture.assertApplicationStatus(t, serviceAuthorizedStaffAppID, domainmodel.StatusKeyScreenPassed)
	var serviceAuthorizedTransition applicationTransitionRecord
	if err := fixture.db.Where("application_id = ?", serviceAuthorizedStaffAppID).First(&serviceAuthorizedTransition).Error; err != nil {
		t.Fatalf("load service-authorized transition: %v", err)
	}
	if serviceAuthorizedTransition.ActorUserID != 101 || serviceAuthorizedTransition.ActorAccountType != "staff" {
		t.Fatalf("service-authorized audit actor = %d/%s, want 101/staff", serviceAuthorizedTransition.ActorUserID, serviceAuthorizedTransition.ActorAccountType)
	}

	untrustedServiceAppID := fixture.seedApplication(t, 1002, 3005, domainmodel.StatusKeyApplied)
	untrustedServiceCtx := metadata.WithTenantActor(ctx, metadata.TenantContext{TenantID: 1, AccountType: "service", ClientApp: "unknown-service"})
	untrustedServiceTransition, err := fixture.owner.ApplyApplicationLifecycleTransition(untrustedServiceCtx, &pb.ApplyApplicationLifecycleTransitionRequest{
		ActorUserId:       101,
		ActorAccountType:  "staff",
		ApplicationId:     untrustedServiceAppID,
		ExpectedStatusKey: domainmodel.StatusKeyApplied,
		TargetStatusKey:   domainmodel.StatusKeyScreenPassed,
		Reason:            "untrusted service attempt",
	})
	if err != nil {
		t.Fatalf("ApplyApplicationLifecycleTransition(untrusted service) error = %v", err)
	}
	if untrustedServiceTransition.Code != errs.ErrForbidden || untrustedServiceTransition.Changed {
		t.Fatalf("ApplyApplicationLifecycleTransition(untrusted service) = %+v, want forbidden unchanged", untrustedServiceTransition)
	}
	fixture.assertApplicationStatus(t, untrustedServiceAppID, domainmodel.StatusKeyApplied)

	serviceAppID := fixture.seedApplication(t, 1002, 3002, domainmodel.StatusKeyApplied)
	serviceTransition, err := fixture.owner.ApplyApplicationLifecycleTransition(ctx, &pb.ApplyApplicationLifecycleTransitionRequest{
		ActorUserId:       9001,
		ActorAccountType:  "service",
		ApplicationId:     serviceAppID,
		ExpectedStatusKey: domainmodel.StatusKeyApplied,
		TargetStatusKey:   domainmodel.StatusKeyScreenPassed,
		Reason:            "interview scheduled",
	})
	if err != nil {
		t.Fatalf("ApplyApplicationLifecycleTransition(service) error = %v", err)
	}
	if serviceTransition.Code != errs.OK || !serviceTransition.Changed {
		t.Fatalf("ApplyApplicationLifecycleTransition(service) = %+v, want OK changed", serviceTransition)
	}

	candidateAppID := fixture.seedApplication(t, 1002, 3003, domainmodel.StatusKeyOfferSent)
	candidateTransition, err := fixture.owner.ApplyApplicationLifecycleTransition(ctx, &pb.ApplyApplicationLifecycleTransitionRequest{
		ActorUserId:       3003,
		ActorAccountType:  "candidate",
		ApplicationId:     candidateAppID,
		ExpectedStatusKey: domainmodel.StatusKeyOfferSent,
		TargetStatusKey:   domainmodel.StatusKeyOfferRejected,
		Reason:            "accepted another offer",
		CloseCurrentRound: true,
	})
	if err != nil {
		t.Fatalf("ApplyApplicationLifecycleTransition(candidate) error = %v", err)
	}
	if candidateTransition.Code != errs.OK || !candidateTransition.Changed {
		t.Fatalf("ApplyApplicationLifecycleTransition(candidate) = %+v, want OK changed", candidateTransition)
	}
	fixture.assertApplicationStatus(t, candidateAppID, domainmodel.StatusKeyOfferRejected)
}

func TestNativeCollaborationPermissionMissingFailsClosed(t *testing.T) {
	ctx := context.Background()
	fixture := newScopeFixture(t)
	fixture.seedScope(t, 101, sharedauthz.ScopeOwnJobs, "", 0)
	fixture.seedJob(t, 1001, 101, 10, 20)
	fixture.seedApplication(t, 1001, 3001, domainmodel.StatusKeyApplied)

	workspace, err := fixture.collaboration.GetCandidateWorkspace(ctx, &pb.GetCandidateWorkspaceRequest{StaffUserId: 101, CandidateUserId: 3001})
	if err != nil {
		t.Fatalf("GetCandidateWorkspace() error = %v", err)
	}
	if workspace.Code != errs.ErrForbidden {
		t.Fatalf("GetCandidateWorkspace() code = %d, want forbidden", workspace.Code)
	}
	tag, err := fixture.collaboration.CreateTag(ctx, &pb.CreateTagRequest{StaffUserId: 101, Name: "Hot"})
	if err != nil {
		t.Fatalf("CreateTag() error = %v", err)
	}
	if tag.Code != errs.ErrForbidden {
		t.Fatalf("CreateTag() code = %d, want forbidden", tag.Code)
	}
}

func TestNativeCollaborationScopeDeniedAcrossCandidate(t *testing.T) {
	ctx := context.Background()
	fixture := newScopeFixture(t)
	fixture.seedScope(t, 101, sharedauthz.ScopeOwnJobs, "", 0)
	fixture.seedPermissions(t, 101, collaborationPermissionSet()...)
	fixture.seedJob(t, 1001, 101, 10, 20)
	fixture.seedJob(t, 1002, 202, 10, 20)
	fixture.seedApplication(t, 1002, 3001, domainmodel.StatusKeyApplied)
	fixture.seedTag(t, 11, "VIP")
	fixture.seedTagAssignment(t, 11, 3001, 202)
	fixture.seedFollowUpTask(t, 91, 3001, 202, "pending")
	fixture.assertTagAssignmentCount(t, 1)
	fixture.assertFollowUpTaskCount(t, 1)

	workspace, err := fixture.collaboration.GetCandidateWorkspace(ctx, &pb.GetCandidateWorkspaceRequest{StaffUserId: 101, CandidateUserId: 3001})
	if err != nil {
		t.Fatalf("GetCandidateWorkspace() error = %v", err)
	}
	if workspace.Code != errs.ErrForbidden {
		t.Fatalf("GetCandidateWorkspace() code = %d, want forbidden", workspace.Code)
	}
	note, err := fixture.collaboration.CreateNote(ctx, &pb.CreateNoteRequest{StaffUserId: 101, CandidateUserId: 3001, Content: "denied"})
	if err != nil {
		t.Fatalf("CreateNote() error = %v", err)
	}
	if note.Code != errs.ErrForbidden {
		t.Fatalf("CreateNote() code = %d, want forbidden", note.Code)
	}
	fixture.assertNoteCount(t, 0)
	notes, err := fixture.collaboration.ListNotes(ctx, &pb.ListNotesRequest{StaffUserId: 101, CandidateUserId: 3001})
	if err != nil {
		t.Fatalf("ListNotes() error = %v", err)
	}
	if notes.Code != errs.ErrForbidden {
		t.Fatalf("ListNotes() code = %d, want forbidden", notes.Code)
	}
	assertCommonCode(t, errs.ErrForbidden, func() (*pb.CommonResponse, error) {
		return fixture.collaboration.AssignTag(ctx, &pb.AssignTagRequest{StaffUserId: 101, CandidateUserId: 3001, TagId: 11})
	})
	fixture.assertTagAssignmentCount(t, 1)
	assertCommonCode(t, errs.ErrForbidden, func() (*pb.CommonResponse, error) {
		return fixture.collaboration.UnassignTag(ctx, &pb.UnassignTagRequest{StaffUserId: 101, CandidateUserId: 3001, TagId: 11})
	})
	fixture.assertTagAssignmentCount(t, 1)
	candidateTags, err := fixture.collaboration.ListCandidateTags(ctx, &pb.ListCandidateTagsRequest{StaffUserId: 101, CandidateUserId: 3001})
	if err != nil {
		t.Fatalf("ListCandidateTags() error = %v", err)
	}
	if candidateTags.Code != errs.ErrForbidden {
		t.Fatalf("ListCandidateTags() code = %d, want forbidden", candidateTags.Code)
	}
	createdTask, err := fixture.collaboration.CreateFollowUpTask(ctx, &pb.CreateFollowUpTaskRequest{StaffUserId: 101, CandidateUserId: 3001, AssigneeUserId: 101, Title: "denied"})
	if err != nil {
		t.Fatalf("CreateFollowUpTask() error = %v", err)
	}
	if createdTask.Code != errs.ErrForbidden {
		t.Fatalf("CreateFollowUpTask() code = %d, want forbidden", createdTask.Code)
	}
	fixture.assertFollowUpTaskCount(t, 1)
	tasks, err := fixture.collaboration.ListFollowUpTasks(ctx, &pb.ListFollowUpTasksRequest{StaffUserId: 101, CandidateUserId: 3001})
	if err != nil {
		t.Fatalf("ListFollowUpTasks() error = %v", err)
	}
	if tasks.Code != errs.ErrForbidden {
		t.Fatalf("ListFollowUpTasks() code = %d, want forbidden", tasks.Code)
	}
	task, err := fixture.collaboration.GetFollowUpTask(ctx, &pb.GetFollowUpTaskRequest{StaffUserId: 101, TaskId: 91})
	if err != nil {
		t.Fatalf("GetFollowUpTask() error = %v", err)
	}
	if task.Code != errs.ErrForbidden {
		t.Fatalf("GetFollowUpTask() code = %d, want forbidden", task.Code)
	}
	assertCommonCode(t, errs.ErrForbidden, func() (*pb.CommonResponse, error) {
		return fixture.collaboration.CompleteFollowUpTask(ctx, &pb.CompleteFollowUpTaskRequest{StaffUserId: 101, TaskId: 91})
	})
	fixture.assertFollowUpStatus(t, 91, "pending")
	timeline, err := fixture.collaboration.ListTimelineEvents(ctx, &pb.ListTimelineEventsRequest{StaffUserId: 101, CandidateUserId: 3001})
	if err != nil {
		t.Fatalf("ListTimelineEvents() error = %v", err)
	}
	if timeline.Code != errs.ErrForbidden {
		t.Fatalf("ListTimelineEvents() code = %d, want forbidden", timeline.Code)
	}
}

func TestNativeCollaborationScopeAllowsOwnedCandidateWithPermissions(t *testing.T) {
	ctx := metadata.WithTenantActor(context.Background(), metadata.TenantContext{TenantID: 1, MembershipID: 1, UserID: 101, AccountType: "staff", ClientApp: "staff"})
	fixture := newScopeFixture(t)
	fixture.seedScope(t, 101, sharedauthz.ScopeOwnJobs, "", 0)
	fixture.seedPermissions(t, 101, collaborationPermissionSet()...)
	fixture.seedJob(t, 1001, 101, 10, 20)
	appID := fixture.seedApplication(t, 1001, 3001, domainmodel.StatusKeyApplied)
	fixture.seedTransition(t, appID, domainmodel.StatusKeyApplied, domainmodel.StatusKeyInterviewing, 101)
	interviewID := fixture.seedInterview(t, appID, 202, fixture.now.Add(time.Hour))
	fixture.seedInterviewFeedback(t, interviewID, appID, 202)
	fixture.seedOffer(t, appID, 3001, 1001, "sent", fixture.now.Add(4*time.Hour))

	workspace, err := fixture.collaboration.GetCandidateWorkspace(ctx, &pb.GetCandidateWorkspaceRequest{StaffUserId: 101, CandidateUserId: 3001})
	if err != nil {
		t.Fatalf("GetCandidateWorkspace() error = %v", err)
	}
	if workspace.Code != errs.OK || workspace.Workspace == nil || workspace.Workspace.TotalApplications != 1 {
		t.Fatalf("GetCandidateWorkspace() = code %d workspace %+v, want OK", workspace.Code, workspace.Workspace)
	}
	if len(workspace.Workspace.Applications) != 1 || workspace.Workspace.Applications[0].Department != "Department" || workspace.Workspace.Applications[0].Location != "Location" {
		t.Fatalf("workspace applications = %+v, want department/location details", workspace.Workspace.Applications)
	}
	if workspace.Workspace.TotalInterviews != 1 || len(workspace.Workspace.Interviews) != 1 {
		t.Fatalf("workspace interviews = total %d len %d, want one", workspace.Workspace.TotalInterviews, len(workspace.Workspace.Interviews))
	}
	gotInterview := workspace.Workspace.Interviews[0]
	if gotInterview.InterviewId != interviewID || gotInterview.ApplicationId != appID || gotInterview.Title != "Technical Round" || gotInterview.Mode != "video" || !gotInterview.HasFeedback {
		t.Fatalf("workspace interview = %+v, want seeded interview with feedback", gotInterview)
	}
	if gotInterview.FeedbackRecommendation != "positive" || gotInterview.FeedbackScore != 9 || gotInterview.FeedbackDimensionScoresJson != `{"technical":9}` || gotInterview.FeedbackComments != "strong signal" {
		t.Fatalf("workspace interview feedback = %+v, want positive feedback", gotInterview)
	}
	if workspace.Workspace.TotalOffers != 1 || len(workspace.Workspace.Offers) != 1 {
		t.Fatalf("workspace offers = total %d len %d, want one", workspace.Workspace.TotalOffers, len(workspace.Workspace.Offers))
	}
	gotOffer := workspace.Workspace.Offers[0]
	if gotOffer.ApplicationId != appID || gotOffer.Title != "Senior Engineer Offer" || gotOffer.Status != "sent" || gotOffer.SalaryRange != "30k-40k" || gotOffer.Level != "P6" || gotOffer.WorkLocation != "Shanghai" || gotOffer.StartDate != "2026-08-01" || gotOffer.JobTitle != "Job 1001" {
		t.Fatalf("workspace offer = %+v, want seeded offer", gotOffer)
	}
	if workspace.Workspace.LatestActivityAt != fixture.now.Add(5*time.Hour).Format(time.RFC3339) {
		t.Fatalf("LatestActivityAt = %q, want feedback updated_at", workspace.Workspace.LatestActivityAt)
	}
	note, err := fixture.collaboration.CreateNote(ctx, &pb.CreateNoteRequest{StaffUserId: 101, CandidateUserId: 3001, Content: "strong"})
	if err != nil {
		t.Fatalf("CreateNote() error = %v", err)
	}
	if note.Code != errs.OK || note.Note == nil {
		t.Fatalf("CreateNote() = %+v, want OK note", note)
	}
	notes, err := fixture.collaboration.ListNotes(ctx, &pb.ListNotesRequest{StaffUserId: 101, CandidateUserId: 3001})
	if err != nil {
		t.Fatalf("ListNotes() error = %v", err)
	}
	if notes.Code != errs.OK || len(notes.List) != 1 {
		t.Fatalf("ListNotes() = code %d len %d, want one note", notes.Code, len(notes.List))
	}
	tag, err := fixture.collaboration.CreateTag(ctx, &pb.CreateTagRequest{StaffUserId: 101, Name: "High Potential"})
	if err != nil {
		t.Fatalf("CreateTag() error = %v", err)
	}
	if tag.Code != errs.OK || tag.Tag == nil {
		t.Fatalf("CreateTag() = %+v, want OK tag", tag)
	}
	tags, err := fixture.collaboration.ListTags(ctx, &pb.ListTagsRequest{StaffUserId: 101})
	if err != nil {
		t.Fatalf("ListTags() error = %v", err)
	}
	if tags.Code != errs.OK || len(tags.List) != 1 {
		t.Fatalf("ListTags() = code %d len %d, want one tag", tags.Code, len(tags.List))
	}
	assertCommonOK(t, func() (*pb.CommonResponse, error) {
		return fixture.collaboration.AssignTag(ctx, &pb.AssignTagRequest{StaffUserId: 101, CandidateUserId: 3001, TagId: tag.Tag.Id})
	})
	fixture.assertTagAssignmentCount(t, 1)
	candidateTags, err := fixture.collaboration.ListCandidateTags(ctx, &pb.ListCandidateTagsRequest{StaffUserId: 101, CandidateUserId: 3001})
	if err != nil {
		t.Fatalf("ListCandidateTags() error = %v", err)
	}
	if candidateTags.Code != errs.OK || len(candidateTags.List) != 1 {
		t.Fatalf("ListCandidateTags() = code %d len %d, want one tag", candidateTags.Code, len(candidateTags.List))
	}
	assertCommonOK(t, func() (*pb.CommonResponse, error) {
		return fixture.collaboration.UnassignTag(ctx, &pb.UnassignTagRequest{StaffUserId: 101, CandidateUserId: 3001, TagId: tag.Tag.Id})
	})
	fixture.assertTagAssignmentCount(t, 0)
	task, err := fixture.collaboration.CreateFollowUpTask(ctx, &pb.CreateFollowUpTaskRequest{
		StaffUserId: 101, CandidateUserId: 3001, AssigneeUserId: 101, Title: "Follow up", DueAt: fixture.now.Add(24 * time.Hour).Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("CreateFollowUpTask() error = %v", err)
	}
	if task.Code != errs.OK || task.Task == nil {
		t.Fatalf("CreateFollowUpTask() = %+v, want OK task", task)
	}
	tasks, err := fixture.collaboration.ListFollowUpTasks(ctx, &pb.ListFollowUpTasksRequest{StaffUserId: 101, CandidateUserId: 3001})
	if err != nil {
		t.Fatalf("ListFollowUpTasks() error = %v", err)
	}
	if tasks.Code != errs.OK || len(tasks.List) != 1 {
		t.Fatalf("ListFollowUpTasks() = code %d len %d, want one task", tasks.Code, len(tasks.List))
	}
	gotTask, err := fixture.collaboration.GetFollowUpTask(ctx, &pb.GetFollowUpTaskRequest{StaffUserId: 101, TaskId: task.Task.Id})
	if err != nil {
		t.Fatalf("GetFollowUpTask() error = %v", err)
	}
	if gotTask.Code != errs.OK || gotTask.Task == nil {
		t.Fatalf("GetFollowUpTask() = %+v, want OK task", gotTask)
	}
	assertCommonOK(t, func() (*pb.CommonResponse, error) {
		return fixture.collaboration.CompleteFollowUpTask(ctx, &pb.CompleteFollowUpTaskRequest{StaffUserId: 101, TaskId: task.Task.Id})
	})
	fixture.assertFollowUpStatus(t, task.Task.Id, "completed")
	timeline, err := fixture.collaboration.ListTimelineEvents(ctx, &pb.ListTimelineEventsRequest{StaffUserId: 101, CandidateUserId: 3001})
	if err != nil {
		t.Fatalf("ListTimelineEvents() error = %v", err)
	}
	if timeline.Code != errs.OK || len(timeline.Events) == 0 {
		t.Fatalf("ListTimelineEvents() = code %d len %d, want events", timeline.Code, len(timeline.Events))
	}
	eventTypes := make(map[string]bool, len(timeline.Events))
	for _, event := range timeline.Events {
		eventTypes[event.EventType] = true
	}
	for _, eventType := range []string{"note", "status_transition", "interview", "offer"} {
		if !eventTypes[eventType] {
			t.Fatalf("timeline event types = %#v, want %s", eventTypes, eventType)
		}
	}
	for i := 1; i < len(timeline.Events); i++ {
		prev, err := time.Parse(time.RFC3339, timeline.Events[i-1].Timestamp)
		if err != nil {
			t.Fatalf("parse previous timeline timestamp: %v", err)
		}
		next, err := time.Parse(time.RFC3339, timeline.Events[i].Timestamp)
		if err != nil {
			t.Fatalf("parse next timeline timestamp: %v", err)
		}
		if next.After(prev) {
			t.Fatalf("timeline not sorted desc at %d: %s before %s", i, timeline.Events[i-1].Timestamp, timeline.Events[i].Timestamp)
		}
	}
}

type scopeFixture struct {
	db            *gorm.DB
	job           *jobAdapter
	application   *applicationAdapter
	owner         *applicationOwnerAdapter
	collaboration *collaborationAdapter
	now           time.Time
	nextAppID     int64
}

func newScopeFixture(t *testing.T) *scopeFixture {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&jobRecord{},
		&candidateProfileRecord{},
		&resumeRecord{},
		&applicationRecord{},
		&applicationTransitionRecord{},
		&eventOutboxRecord{},
		&userDataScopeRecord{},
		&candidateNoteRecord{},
		&candidateTagRecord{},
		&candidateTagAssignmentRecord{},
		&followUpTaskRecord{},
		&scopeInterviewScheduleRecord{},
		&scopeInterviewFeedbackRecord{},
		&scopeOfferRecord{},
		&scopePermissionRecord{},
		&scopeRolePermissionRecord{},
		&scopeUserRoleRecord{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_candidate_tag_assignments_tag_candidate ON candidate_tag_assignments(tag_id, candidate_user_id)").Error; err != nil {
		t.Fatalf("create tag assignment index: %v", err)
	}
	now := time.Date(2026, 7, 14, 10, 0, 0, 0, time.UTC)
	store := &nativeStore{db: db, storage: scopeStorage{}, now: func() time.Time { return now }}
	application := &applicationAdapter{nativeStore: store}
	return &scopeFixture{
		db:            db,
		job:           &jobAdapter{nativeStore: store},
		application:   application,
		owner:         &applicationOwnerAdapter{applicationAdapter: application},
		collaboration: &collaborationAdapter{nativeStore: store},
		now:           now,
		nextAppID:     1,
	}
}

type scopePermissionRecord struct {
	ID            uint64 `gorm:"primaryKey"`
	PermissionKey string
}

func (scopePermissionRecord) TableName() string { return "permissions" }

type scopeRolePermissionRecord struct {
	ID           uint64 `gorm:"primaryKey"`
	RoleID       uint64
	PermissionID uint64
}

func (scopeRolePermissionRecord) TableName() string { return "role_permissions" }

type scopeUserRoleRecord struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64
	RoleID    uint64
	RevokedAt *time.Time
}

func (scopeUserRoleRecord) TableName() string { return "user_roles" }

type scopeInterviewScheduleRecord struct {
	ID            int64 `gorm:"primaryKey"`
	TenantID      int64
	ApplicationID int64
	InterviewerID int64
	RoundNo       int32
	Title         string
	Mode          string
	ScheduledAt   *time.Time
	Status        string
	CreatedBy     *int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     *time.Time
}

func (scopeInterviewScheduleRecord) TableName() string { return "interview_schedules" }

type scopeInterviewFeedbackRecord struct {
	ID                  int64 `gorm:"primaryKey"`
	TenantID            int64
	InterviewID         int64  `gorm:"column:interview_id"`
	ApplicationID       int64  `gorm:"column:application_id"`
	InterviewerID       int64  `gorm:"column:interviewer_id"`
	Recommendation      string `gorm:"column:recommendation"`
	Score               int32  `gorm:"column:score"`
	DimensionScoresJSON string `gorm:"column:dimension_scores_json"`
	Comments            string `gorm:"column:comments"`
	SubmittedAt         time.Time
	UpdatedAt           time.Time
}

func (scopeInterviewFeedbackRecord) TableName() string { return "interview_feedback" }

type scopeOfferRecord struct {
	ID              int64 `gorm:"primaryKey"`
	TenantID        int64
	ApplicationID   int64
	CandidateUserID int64
	JobID           int64
	Status          string
	Title           string
	SalaryRange     string
	Level           string
	WorkLocation    string
	StartDate       string
	CreatedBy       int64
	SentBy          *int64
	DecidedAt       *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (scopeOfferRecord) TableName() string { return "offers" }

func (f *scopeFixture) seedScope(t *testing.T, userID int64, scopeKey, resourceType string, resourceID uint64) {
	t.Helper()
	row := &userDataScopeRecord{UserID: uint64(userID), ScopeKey: scopeKey, ResourceType: resourceType, ResourceID: resourceID, AssignedAt: f.now}
	if err := f.db.Create(row).Error; err != nil {
		t.Fatalf("seed scope: %v", err)
	}
}

func (f *scopeFixture) seedJob(t *testing.T, jobID, hrID, departmentID, locationID int64) {
	t.Helper()
	row := &jobRecord{
		ID:           jobID,
		TenantID:     1,
		HrID:         hrID,
		Title:        "Job " + strconv.FormatInt(jobID, 10),
		Department:   "Department",
		DepartmentID: positivePtr(departmentID),
		Location:     "Location",
		LocationID:   positivePtr(locationID),
		Status:       1,
		CreatedAt:    f.now.Add(time.Duration(jobID) * time.Second),
		UpdatedAt:    f.now,
	}
	if err := f.db.Create(row).Error; err != nil {
		t.Fatalf("seed job: %v", err)
	}
}

func (f *scopeFixture) seedApplication(t *testing.T, jobID, userID int64, statusKey string) int64 {
	t.Helper()
	if err := f.db.FirstOrCreate(&candidateProfileRecord{UserID: userID}, candidateProfileRecord{UserID: userID, RealName: "Candidate", IsComplete: 1}).Error; err != nil {
		t.Fatalf("seed profile: %v", err)
	}
	resumeID := userID + 10000
	if err := f.db.FirstOrCreate(&resumeRecord{ID: resumeID}, resumeRecord{ID: resumeID, UserID: userID, FileName: "cv.pdf", FileType: "pdf", IsValid: 1, UploadedAt: f.now}).Error; err != nil {
		t.Fatalf("seed resume: %v", err)
	}
	appID := f.nextAppID
	f.nextAppID++
	row := &applicationRecord{
		ID:        appID,
		TenantID:  1,
		UserID:    userID,
		JobID:     jobID,
		ResumeID:  resumeID,
		Status:    domainmodel.StatusKeyToLegacy[statusKey],
		StatusKey: statusKey,
		RoundNo:   1,
		IsCurrent: 1,
		AppliedAt: f.now,
		UpdatedAt: f.now,
	}
	if err := f.db.Create(row).Error; err != nil {
		t.Fatalf("seed application: %v", err)
	}
	return appID
}

func (f *scopeFixture) seedTransition(t *testing.T, applicationID int64, from, to string, actorID int64) {
	t.Helper()
	row := &applicationTransitionRecord{TenantID: 1, ApplicationID: applicationID, FromStatus: from, ToStatus: to, ActorUserID: actorID, ActorAccountType: "staff", CreatedAt: f.now}
	if err := f.db.Create(row).Error; err != nil {
		t.Fatalf("seed transition: %v", err)
	}
}

func (f *scopeFixture) seedInterview(t *testing.T, applicationID, interviewerID int64, createdAt time.Time) int64 {
	t.Helper()
	scheduledAt := createdAt.Add(2 * time.Hour)
	creatorID := int64(101)
	row := &scopeInterviewScheduleRecord{
		TenantID:      1,
		ApplicationID: applicationID,
		InterviewerID: interviewerID,
		RoundNo:       2,
		Title:         "Technical Round",
		Mode:          "video",
		ScheduledAt:   &scheduledAt,
		Status:        "scheduled",
		CreatedBy:     &creatorID,
		CreatedAt:     createdAt,
		UpdatedAt:     createdAt.Add(10 * time.Minute),
	}
	if err := f.db.Create(row).Error; err != nil {
		t.Fatalf("seed interview: %v", err)
	}
	return row.ID
}

func (f *scopeFixture) seedInterviewFeedback(t *testing.T, interviewID, applicationID, interviewerID int64) {
	t.Helper()
	row := &scopeInterviewFeedbackRecord{
		TenantID:            1,
		InterviewID:         interviewID,
		ApplicationID:       applicationID,
		InterviewerID:       interviewerID,
		Recommendation:      "positive",
		Score:               9,
		DimensionScoresJSON: `{"technical":9}`,
		Comments:            "strong signal",
		SubmittedAt:         f.now.Add(5 * time.Hour),
		UpdatedAt:           f.now.Add(5 * time.Hour),
	}
	if err := f.db.Create(row).Error; err != nil {
		t.Fatalf("seed interview feedback: %v", err)
	}
}

func (f *scopeFixture) seedOffer(t *testing.T, applicationID, candidateUserID, jobID int64, status string, createdAt time.Time) int64 {
	t.Helper()
	row := &scopeOfferRecord{
		TenantID:        1,
		ApplicationID:   applicationID,
		CandidateUserID: candidateUserID,
		JobID:           jobID,
		Status:          status,
		Title:           "Senior Engineer Offer",
		SalaryRange:     "30k-40k",
		Level:           "P6",
		WorkLocation:    "Shanghai",
		StartDate:       "2026-08-01",
		CreatedBy:       101,
		CreatedAt:       createdAt,
		UpdatedAt:       createdAt.Add(15 * time.Minute),
	}
	if err := f.db.Create(row).Error; err != nil {
		t.Fatalf("seed offer: %v", err)
	}
	return row.ID
}

func (f *scopeFixture) seedPermissions(t *testing.T, userID int64, permissions ...string) {
	t.Helper()
	roleID := uint64(userID + 100000)
	if err := f.db.Create(&scopeUserRoleRecord{UserID: uint64(userID), RoleID: roleID}).Error; err != nil {
		t.Fatalf("seed user role: %v", err)
	}
	for _, permission := range permissions {
		perm := &scopePermissionRecord{PermissionKey: permission}
		if err := f.db.Create(perm).Error; err != nil {
			t.Fatalf("seed permission: %v", err)
		}
		if err := f.db.Create(&scopeRolePermissionRecord{RoleID: roleID, PermissionID: perm.ID}).Error; err != nil {
			t.Fatalf("seed role permission: %v", err)
		}
	}
}

func (f *scopeFixture) seedTag(t *testing.T, tagID uint64, name string) {
	t.Helper()
	row := &candidateTagRecord{ID: tagID, Name: name, Color: "#409eff", CreatedBy: positiveUintPtr(101), CreatedAt: f.now}
	if err := f.db.Create(row).Error; err != nil {
		t.Fatalf("seed tag: %v", err)
	}
}

func (f *scopeFixture) seedTagAssignment(t *testing.T, tagID, candidateUserID, createdBy uint64) {
	t.Helper()
	row := &candidateTagAssignmentRecord{TagID: tagID, CandidateUserID: candidateUserID, CreatedBy: positiveUintPtr(createdBy), CreatedAt: f.now}
	if err := f.db.Create(row).Error; err != nil {
		t.Fatalf("seed tag assignment: %v", err)
	}
}

func (f *scopeFixture) seedFollowUpTask(t *testing.T, taskID, candidateUserID, createdBy uint64, status string) {
	t.Helper()
	row := &followUpTaskRecord{ID: taskID, CandidateUserID: candidateUserID, AssigneeUserID: createdBy, CreatedBy: createdBy, Title: "Task", Status: status, CreatedAt: f.now, UpdatedAt: f.now}
	if err := f.db.Create(row).Error; err != nil {
		t.Fatalf("seed follow-up task: %v", err)
	}
}

func (f *scopeFixture) assertNoteCount(t *testing.T, want int64) {
	t.Helper()
	var got int64
	if err := f.db.Model(&candidateNoteRecord{}).Count(&got).Error; err != nil {
		t.Fatalf("count notes: %v", err)
	}
	if got != want {
		t.Fatalf("note count = %d, want %d", got, want)
	}
}

func (f *scopeFixture) assertTagAssignmentCount(t *testing.T, want int64) {
	t.Helper()
	var got int64
	if err := f.db.Model(&candidateTagAssignmentRecord{}).Count(&got).Error; err != nil {
		t.Fatalf("count tag assignments: %v", err)
	}
	if got != want {
		t.Fatalf("tag assignment count = %d, want %d", got, want)
	}
}

func (f *scopeFixture) assertFollowUpTaskCount(t *testing.T, want int64) {
	t.Helper()
	var got int64
	if err := f.db.Model(&followUpTaskRecord{}).Count(&got).Error; err != nil {
		t.Fatalf("count follow-up tasks: %v", err)
	}
	if got != want {
		t.Fatalf("follow-up task count = %d, want %d", got, want)
	}
}

func (f *scopeFixture) assertFollowUpStatus(t *testing.T, taskID uint64, want string) {
	t.Helper()
	var row followUpTaskRecord
	if err := f.db.First(&row, taskID).Error; err != nil {
		t.Fatalf("load follow-up task: %v", err)
	}
	if row.Status != want {
		t.Fatalf("follow-up task %d status = %s, want %s", taskID, row.Status, want)
	}
}

func collaborationPermissionSet() []string {
	return []string{
		domainmodel.PermissionApplicationRead,
		domainmodel.PermissionCollaborationNoteRead,
		domainmodel.PermissionCollaborationNoteCreate,
		domainmodel.PermissionCollaborationTagManage,
		domainmodel.PermissionCollaborationTaskManage,
	}
}

func (f *scopeFixture) assertJobStatus(t *testing.T, jobID int64, want int32) {
	t.Helper()
	var job jobRecord
	if err := f.db.First(&job, jobID).Error; err != nil {
		t.Fatalf("load job: %v", err)
	}
	if job.Status != want {
		t.Fatalf("job %d status = %d, want %d", jobID, job.Status, want)
	}
}

func (f *scopeFixture) assertJobTitle(t *testing.T, jobID int64, want string) {
	t.Helper()
	var job jobRecord
	if err := f.db.First(&job, jobID).Error; err != nil {
		t.Fatalf("load job: %v", err)
	}
	if job.Title != want {
		t.Fatalf("job %d title = %q, want %q", jobID, job.Title, want)
	}
}

func (f *scopeFixture) assertApplicationStatus(t *testing.T, appID int64, want string) {
	t.Helper()
	var app applicationRecord
	if err := f.db.First(&app, appID).Error; err != nil {
		t.Fatalf("load application: %v", err)
	}
	if app.StatusKey != want || app.Status != domainmodel.StatusKeyToLegacy[want] {
		t.Fatalf("application %d status = %d/%s, want %d/%s", appID, app.Status, app.StatusKey, domainmodel.StatusKeyToLegacy[want], want)
	}
}

func assertCommonOK(t *testing.T, call func() (*pb.CommonResponse, error)) {
	t.Helper()
	resp, err := call()
	if err != nil {
		t.Fatalf("response error = %v", err)
	}
	if resp.Code != errs.OK {
		t.Fatalf("response code = %d, want OK; msg=%s", resp.Code, resp.Msg)
	}
}

func assertCommonCode(t *testing.T, want int32, call func() (*pb.CommonResponse, error)) {
	t.Helper()
	resp, err := call()
	if err != nil {
		t.Fatalf("response error = %v", err)
	}
	if resp.Code != want {
		t.Fatalf("response code = %d, want %d; msg=%s", resp.Code, want, resp.Msg)
	}
}

func jobIDs(jobs []*pb.Job) []int64 {
	ids := make([]int64, 0, len(jobs))
	for _, job := range jobs {
		ids = append(ids, job.JobId)
	}
	return ids
}

type scopeStorage struct{}

func (scopeStorage) SetPresignCache(*oss.PresignCache) {}

func (scopeStorage) ProviderName() string { return "test" }

func (scopeStorage) GeneratePresignedPutURL(string, string) (string, time.Time, error) {
	return "put-url", time.Now(), nil
}

func (scopeStorage) GeneratePresignedGetURL(ossKey string) (string, error) {
	return "get://" + ossKey, nil
}

func (scopeStorage) VerifyObject(context.Context, string) error { return nil }

func (scopeStorage) VerifyObjectSize(context.Context, string, int64) error { return nil }

func (scopeStorage) DownloadObject(context.Context, string) ([]byte, error) { return nil, nil }

func (scopeStorage) CopyObject(context.Context, string, string) error { return nil }

func (scopeStorage) DeleteObject(context.Context, string) error { return nil }

func (scopeStorage) SavePresignSession(context.Context, oss.PresignSession) (string, error) {
	return "upload-id", nil
}

func (scopeStorage) SavePresignSessionWithID(context.Context, string, oss.PresignSession) error {
	return nil
}

func (scopeStorage) GetAndDeletePresignSession(context.Context, string) (*oss.PresignSession, error) {
	return nil, nil
}
