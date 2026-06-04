package service

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"logic-grpc-service/model"
	"logic-grpc-service/pkg/authz"
	"logic-grpc-service/pkg/errs"
	"logic-grpc-service/pkg/jwt"
	"logic-grpc-service/pkg/metadata"
	"logic-grpc-service/recruitment/pb"
	"logic-grpc-service/repository"
)

// setupRBACTestDB creates an in-memory SQLite DB with tables needed for RBAC tests.
func setupRBACTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	tables := []interface{}{
		&model.User{},
		&model.Role{},
		&model.UserRole{},
		&model.Permission{},
		&model.RolePermission{},
		&model.UserDataScope{},
		&model.Job{},
		&model.Application{},
		&model.Department{},
		&model.JobLocation{},
		&model.DepartmentLocation{},
		&model.AuthorizationAuditLog{},
	}
	for _, table := range tables {
		if err := db.AutoMigrate(table); err != nil {
			t.Fatalf("migrate %T: %v", table, err)
		}
	}
	return db
}

// seedRBACData inserts minimal RBAC seed data for test scenarios.
func seedRBACData(t *testing.T, db *gorm.DB) {
	t.Helper()

	// Create test users
	user := model.User{Username: "test_hr", Password: "hash", Role: 2, AccountType: "staff", TokenVersion: 1}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	admin := model.User{Username: "test_admin", Password: "hash", Role: 3, AccountType: "staff", TokenVersion: 1}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}

	// Create roles
	recruiterRole := model.Role{RoleKey: "recruiter", Name: "Recruiter", IsSystem: 1}
	adminRole := model.Role{RoleKey: "recruiting_admin", Name: "Recruiting Admin", IsSystem: 1}
	if err := db.Create(&recruiterRole).Error; err != nil {
		t.Fatalf("create recruiter role: %v", err)
	}
	if err := db.Create(&adminRole).Error; err != nil {
		t.Fatalf("create admin role: %v", err)
	}

	// Assign roles
	adminIDA := uint64(admin.ID)
	ur1 := model.UserRole{UserID: uint64(user.ID), RoleID: recruiterRole.ID, AssignedBy: &adminIDA}
	ur2 := model.UserRole{UserID: uint64(admin.ID), RoleID: adminRole.ID, AssignedBy: &adminIDA}
	if err := db.Create(&ur1).Error; err != nil {
		t.Fatalf("assign recruiter: %v", err)
	}
	if err := db.Create(&ur2).Error; err != nil {
		t.Fatalf("assign admin: %v", err)
	}

	// Create permissions
	perms := []model.Permission{
		{PermissionKey: authz.PermJobCreate, Resource: "job", Action: "create", Description: "Create Jobs"},
		{PermissionKey: authz.PermJobPublish, Resource: "job", Action: "publish", Description: "Publish Jobs"},
		{PermissionKey: authz.PermApplicationStatusUpdate, Resource: "application", Action: "update_status", Description: "Update Application Status"},
	}
	for i := range perms {
		if err := db.Create(&perms[i]).Error; err != nil {
			t.Fatalf("create perm %s: %v", perms[i].PermissionKey, err)
		}
	}
	// Assign perms to recruiter role
	for _, p := range perms {
		rp := model.RolePermission{RoleID: recruiterRole.ID, PermissionID: p.ID}
		if err := db.Create(&rp).Error; err != nil {
			t.Fatalf("assign perm %s: %v", p.PermissionKey, err)
		}
	}

	// Create departments and locations
	dept := model.Department{Name: "Engineering", FullName: "Engineering", IsActive: 1}
	loc := model.JobLocation{Name: "Beijing", IsActive: 1}
	if err := db.Create(&dept).Error; err != nil {
		t.Fatalf("create dept: %v", err)
	}
	if err := db.Create(&loc).Error; err != nil {
		t.Fatalf("create loc: %v", err)
	}

	// Create jobs
	deptID := int64(dept.ID)
	locID := int64(loc.ID)
	job1 := model.Job{
		HrID:         user.ID,
		Title:        "Software Engineer",
		DepartmentID: &deptID,
		LocationID:   &locID,
		Status:       1,
	}
	job2 := model.Job{
		HrID:         admin.ID,
		Title:        "Product Manager",
		DepartmentID: &deptID,
		LocationID:   &locID,
		Status:       1,
	}
	if err := db.Create(&job1).Error; err != nil {
		t.Fatalf("create job1: %v", err)
	}
	if err := db.Create(&job2).Error; err != nil {
		t.Fatalf("create job2: %v", err)
	}
}

// ── P0: Token Version Invalidation ──────────────────────────────────────

func TestIncrementTokenVersion_ReturnsNewVersion(t *testing.T) {
	db := setupRBACTestDB(t)
	seedRBACData(t, db)

	authzRepo := repository.NewAuthzRepo(db)
	ctx := context.Background()

	// Get initial version
	principal, err := authzRepo.LoadPrincipal(ctx, 1) // user ID 1
	if err != nil {
		t.Fatalf("get principal: %v", err)
	}
	initialVersion := principal.TokenVersion
	if initialVersion < 1 {
		t.Fatalf("expected token_version >= 1, got %d", initialVersion)
	}

	// Increment
	newVersion, err := authzRepo.IncrementTokenVersion(ctx, 1)
	if err != nil {
		t.Fatalf("increment token version: %v", err)
	}
	if newVersion != initialVersion+1 {
		t.Errorf("expected version %d, got %d", initialVersion+1, newVersion)
	}

	// Verify DB reflects the new version
	principal2, err := authzRepo.LoadPrincipal(ctx, 1)
	if err != nil {
		t.Fatalf("get principal after increment: %v", err)
	}
	if principal2.TokenVersion != newVersion {
		t.Errorf("DB token_version = %d, expected %d", principal2.TokenVersion, newVersion)
	}

	// Increment again to verify correct sequential behavior
	v3, err := authzRepo.IncrementTokenVersion(ctx, 1)
	if err != nil {
		t.Fatalf("second increment: %v", err)
	}
	if v3 != newVersion+1 {
		t.Errorf("expected version %d after second increment, got %d", newVersion+1, v3)
	}
}

// ── P1: Department/Location Scope Check ─────────────────────────────────

func TestCheckJobScope_DepartmentScopeReturnsScopeDepartmentOrLocation(t *testing.T) {
	db := setupRBACTestDB(t)
	seedRBACData(t, db)

	authzRepo := repository.NewAuthzRepo(db)
	jobRepo := repository.NewJobRepo(db)

	svc := NewJobService(jobRepo, nil, authzRepo, nil, &scopeEvaluator{authzRepo: authzRepo})

	ctx := context.Background()

	// Assign department scope to user 1 for department 1 (Engineering)
	adminID := uint64(2)
	if err := authzRepo.AssignDataScope(ctx, 1, authz.ScopeDepartment, "department", 1, &adminID); err != nil {
		t.Fatalf("assign dept scope: %v", err)
	}

	// User 1 checks scope for job 2 (admin's job, same department)
	scopeLevel, err := svc.checkJobScope(ctx, 1, authz.PermJobPublish, 2)
	if err != nil {
		t.Fatalf("checkJobScope: %v", err)
	}
	if scopeLevel != scopeDepartmentOrLocation {
		t.Errorf("expected scopeDepartmentOrLocation (%d), got %d", scopeDepartmentOrLocation, scopeLevel)
	}
}

func TestCheckJobScope_OwnJobsScopeReturnsScopeOwned(t *testing.T) {
	db := setupRBACTestDB(t)
	seedRBACData(t, db)

	authzRepo := repository.NewAuthzRepo(db)
	jobRepo := repository.NewJobRepo(db)

	svc := NewJobService(jobRepo, nil, authzRepo, nil, &scopeEvaluator{authzRepo: authzRepo})

	ctx := context.Background()

	// Assign own_jobs scope to user 1
	adminID := uint64(2)
	if err := authzRepo.AssignDataScope(ctx, 1, authz.ScopeOwnJobs, "global", 0, &adminID); err != nil {
		t.Fatalf("assign own_jobs scope: %v", err)
	}

	// User 1 checks scope for their own job (job 1)
	scopeLevel, err := svc.checkJobScope(ctx, 1, authz.PermJobPublish, 1)
	if err != nil {
		t.Fatalf("checkJobScope for own job: %v", err)
	}
	if scopeLevel != scopeOwned {
		t.Errorf("expected scopeOwned (%d), got %d", scopeOwned, scopeLevel)
	}

	// User 1 checks scope for admin's job (job 2) — should be denied
	scopeLevel, err = svc.checkJobScope(ctx, 1, authz.PermJobPublish, 2)
	if err == nil {
		t.Errorf("expected scope denied error for admin's job, got scopeLevel=%d", scopeLevel)
	}
}

func TestCheckJobScope_NoScopeReturnsDenied(t *testing.T) {
	db := setupRBACTestDB(t)
	seedRBACData(t, db)

	authzRepo := repository.NewAuthzRepo(db)
	jobRepo := repository.NewJobRepo(db)

	svc := NewJobService(jobRepo, nil, authzRepo, nil, &scopeEvaluator{authzRepo: authzRepo})

	ctx := context.Background()

	// User 1 has roles but no data scopes assigned — should be denied
	scopeLevel, err := svc.checkJobScope(ctx, 1, authz.PermJobPublish, 1)
	if err == nil {
		t.Errorf("expected scope denied error, got scopeLevel=%d", scopeLevel)
	}
}

// ── P1: Notification Account Type (No Numeric Role Fallback) ────────────

func TestNotifAccountType_UsesReceiverAccountType(t *testing.T) {
	n := &model.Notification{
		ReceiverID:          1,
		ReceiverAccountType: "staff",
		ReceiverRole:        2,
	}
	got := notifAccountType(n)
	if got != "staff" {
		t.Errorf("expected 'staff', got %q", got)
	}
}

func TestNotifAccountType_FallsBackToCandidateWhenEmpty(t *testing.T) {
	// When ReceiverAccountType is empty (shouldn't happen after fix),
	// the function defaults to "candidate", NOT deriving from ReceiverRole.
	n := &model.Notification{
		ReceiverID:          1,
		ReceiverAccountType: "",
		ReceiverRole:        2, // legacy role=2 (staff) — must NOT be used
	}
	got := notifAccountType(n)
	if got != "candidate" {
		t.Errorf("expected fallback to 'candidate', got %q (numeric role fallback is removed)", got)
	}
}

func TestNotifAccountType_HandlesCandidateFallback(t *testing.T) {
	n := &model.Notification{
		ReceiverID:          1,
		ReceiverAccountType: "",
		ReceiverRole:        1,
	}
	got := notifAccountType(n)
	if got != "candidate" {
		t.Errorf("expected 'candidate', got %q", got)
	}
}

// ── P1: RBAC Scope Levels ───────────────────────────────────────────────

func TestScopeLevel_Ordering(t *testing.T) {
	// Verify the scope level ordering for access control logic
	if scopeDenied >= scopeOwned {
		t.Error("scopeDenied should be < scopeOwned")
	}
	if scopeOwned >= scopeDepartmentOrLocation {
		t.Error("scopeOwned should be < scopeDepartmentOrLocation")
	}
	if scopeDepartmentOrLocation >= scopeFull {
		t.Error("scopeDepartmentOrLocation should be < scopeFull")
	}

	// scopeFull and above grant full access in service methods (no hr_id filter)
	if scopeDepartmentOrLocation >= scopeFull {
		t.Error("scopeDepartmentOrLocation should NOT grant full access")
	}
	if scopeOwned >= scopeFull {
		t.Error("scopeOwned should NOT grant full access")
	}
}

// ── P0: Token Version Redis Sync ─────────────────────────────────────────

func TestSetTokenVersionCache_NilRedisReturnsNil(t *testing.T) {
	svc := &AdminService{redisClient: nil}
	err := svc.setTokenVersionCache(context.Background(), 1, 5)
	if err != nil {
		t.Errorf("expected nil error for nil redis client, got: %v", err)
	}
}

func TestSetTokenVersionCache_SetsKeyInRedis(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	svc := &AdminService{redisClient: rdb}
	ctx := context.Background()

	err := svc.setTokenVersionCache(ctx, 42, 3)
	if err != nil {
		t.Fatalf("setTokenVersionCache failed: %v", err)
	}

	// Verify the key is in Redis with the correct value
	key := "token_version:42"
	val, err := rdb.Get(ctx, key).Int()
	if err != nil {
		t.Fatalf("key not found in redis: %v", err)
	}
	if val != 3 {
		t.Errorf("expected value 3, got %d", val)
	}

	// Verify TTL is set (AccessTokenTTL = 24h, should be > 0)
	ttl := rdb.TTL(ctx, key).Val()
	if ttl <= 0 {
		t.Errorf("expected positive TTL, got %v", ttl)
	}
	if ttl > jwt.AccessTokenTTL+time.Second {
		t.Errorf("TTL %v exceeds AccessTokenTTL %v", ttl, jwt.AccessTokenTTL)
	}
}

func TestSetTokenVersionCache_DelFallbackOnSetFailure(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	svc := &AdminService{redisClient: rdb}
	ctx := context.Background()
	key := "token_version:99"

	// Pre-set a value so we can verify it gets deleted
	rdb.Set(ctx, key, 1, time.Hour)

	// Shut down miniredis to simulate Redis being unreachable
	mr.Close()

	// Now SET should fail, triggering the DEL fallback
	// Since miniredis is closed, DEL will also fail → should return error
	err := svc.setTokenVersionCache(ctx, 99, 2)
	if err == nil {
		t.Error("expected error when both SET and DEL fail, got nil")
	}
}

// ── P1-7: Last System Admin Safety ──────────────────────────────────────

func TestCountActiveUsersWithRole_ReturnsCorrectCount(t *testing.T) {
	db := setupRBACTestDB(t)
	seedRBACData(t, db)

	authzRepo := repository.NewAuthzRepo(db)
	ctx := context.Background()

	// Both test users have recruiter role assigned
	count, err := authzRepo.CountActiveUsersWithRole(ctx, "recruiter")
	if err != nil {
		t.Fatalf("count recruiter: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 active recruiter, got %d", count)
	}

	// No one has system_admin yet (not seeded in test data)
	count, err = authzRepo.CountActiveUsersWithRole(ctx, "system_admin")
	if err != nil {
		t.Fatalf("count system_admin: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 system_admin, got %d", count)
	}
}

func TestLastAdminSafety_RevokeLastSystemAdminIsBlocked(t *testing.T) {
	db := setupRBACTestDB(t)
	seedRBACData(t, db)

	authzRepo := repository.NewAuthzRepo(db)
	ctx := context.Background()

	// Create system_admin role
	sysAdminRole := model.Role{RoleKey: "system_admin", Name: "System Admin", IsSystem: 1}
	if err := db.Create(&sysAdminRole).Error; err != nil {
		t.Fatalf("create system_admin role: %v", err)
	}

	// Assign system_admin to admin user (ID 2)
	adminIDA := uint64(2)
	ur := model.UserRole{UserID: 2, RoleID: sysAdminRole.ID, AssignedBy: &adminIDA}
	if err := db.Create(&ur).Error; err != nil {
		t.Fatalf("assign system_admin: %v", err)
	}

	// Verify count is 1
	count, err := authzRepo.CountActiveUsersWithRole(ctx, "system_admin")
	if err != nil {
		t.Fatalf("count system_admin: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 system_admin, got %d", count)
	}

	// Create AdminService and attempt to revoke the last system_admin from user 2
	svc := NewAdminService(nil, nil, nil, authzRepo, nil, NewServiceAuthorizer(nil, nil))

	// Revoke system_admin from user 2 (the only one) by admin 2 (self)
	req := &pb.RevokeUserRoleRequest{
		AdminId: 2,
		UserId:  2,
		RoleKey: "system_admin",
	}
	resp, err := svc.RevokeUserRole(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Code == 0 || resp.Code == errs.OK {
		t.Errorf("expected Forbidden when revoking last system_admin, got code=%d msg=%s", resp.Code, resp.Msg)
	}
}

// ── Service-layer authorization fail-closed tests ────────────────────────

// TestVerifyActorMatchMissingMetadata verifies that ListNotifications rejects
// requests when the authenticated actor is not in the gRPC context.
func TestVerifyActorMatchMissingMetadata(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.Notification{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	svc := NewNotificationService(repository.NewNotificationRepo(db), nil, NewServiceAuthorizer(nil, nil))
	// No actor in context — VerifyActorMatch must reject.
	resp, err := svc.ListNotifications(context.Background(), &pb.ListNotificationsRequest{
		UserId:      10,
		AccountType: "candidate",
		Page:        0,
		PageSize:    20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Code != errs.ErrForbidden {
		t.Errorf("expected ErrForbidden (403) when actor is missing from context, got code=%d msg=%s", resp.Code, resp.Msg)
	}
}

// TestVerifyActorMatchRejectsMismatchedUser verifies that ListNotifications
// rejects a request whose UserId does not match the authenticated actor.
func TestVerifyActorMatchRejectsMismatchedUser(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.Notification{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	svc := NewNotificationService(repository.NewNotificationRepo(db), nil, NewServiceAuthorizer(nil, nil))
	// Auth user is 10, but request asks for user 20 — must reject.
	ctx := metadata.WithAuthActor(context.Background(), 10, "candidate")
	resp, err := svc.ListNotifications(ctx, &pb.ListNotificationsRequest{
		UserId:      20,
		AccountType: "candidate",
		Page:        0,
		PageSize:    20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Code != errs.ErrForbidden {
		t.Errorf("expected ErrForbidden (403) for mismatched actor, got code=%d msg=%s", resp.Code, resp.Msg)
	}
}

// TestVerifyAdminPermissionMissingMetadata verifies that admin read methods
// reject requests when the authenticated actor is not in the gRPC context.
func TestVerifyAdminPermissionMissingMetadata(t *testing.T) {
	db := setupRBACTestDB(t)
	authzRepo := repository.NewAuthzRepo(db)

	svc := NewAdminService(nil, nil, nil, authzRepo, nil, NewServiceAuthorizer(authzRepo, nil))
	// No actor in context — ListRoles must reject.
	resp, err := svc.ListRoles(context.Background(), &pb.ListRolesRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Code != errs.ErrForbidden {
		t.Errorf("expected ErrForbidden (403) when actor is missing from context, got code=%d msg=%s", resp.Code, resp.Msg)
	}
}

// TestVerifyAdminPermissionRejectsInsufficientPermission verifies that
// admin read methods reject a user who lacks the required permission.
func TestVerifyAdminPermissionRejectsInsufficientPermission(t *testing.T) {
	db := setupRBACTestDB(t)
	authzRepo := repository.NewAuthzRepo(db)

	ctx := context.Background()
	// Create a staff user with no admin permissions (only a basic role).
	user := model.User{
		Username:    "basic_user",
		Password:    "hashed",
		AccountType: "staff",
		Status:      "active",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	// Give them a role with no admin permissions.
	role := model.Role{RoleKey: "recruiter", Name: "Recruiter", IsSystem: int32(1)}
	if err := db.Create(&role).Error; err != nil {
		t.Fatalf("create role: %v", err)
	}
	if err := db.Exec("INSERT INTO user_roles (user_id, role_id, assigned_by) VALUES (?, ?, ?)", user.ID, role.ID, user.ID).Error; err != nil {
		t.Fatalf("assign role: %v", err)
	}

	svc := NewAdminService(nil, nil, nil, authzRepo, nil, NewServiceAuthorizer(authzRepo, nil))
	// Auth as basic_user who lacks admin.role.manage
	ctx = metadata.WithAuthActor(ctx, int64(user.ID), "staff")
	resp, err := svc.ListRoles(ctx, &pb.ListRolesRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Code != errs.ErrForbidden {
		t.Errorf("expected ErrForbidden (403) for user without admin.role.manage, got code=%d msg=%s", resp.Code, resp.Msg)
	}
}

