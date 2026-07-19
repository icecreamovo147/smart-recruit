package service

import (
	"context"
	"errors"
	"testing"
	"time"

	commonsquota "smart-recruit-commons/quota"
	"smart-recruit-identity-service/internal/application/command"
	"smart-recruit-identity-service/internal/application/port"
	"smart-recruit-identity-service/internal/domain/model"
	"smart-recruit-identity-service/internal/domain/repository"
)

func TestAuthServiceRegisterStaffAssignsRecruiterAndOwnJobs(t *testing.T) {
	users := newMemoryUsers()
	authz := newMemoryAuthz()
	auth := newTestAuthService(t, AuthDeps{
		Users:   users,
		Authz:   authz,
		Invites: staticInvites{invite: &model.InviteCode{ID: 1, Code: "abc123", CreatedBy: 99, IsActive: true}},
	})

	result, err := auth.Register(context.Background(), command.Register{
		Username:   " recruiter ",
		Password:   "Password1",
		Email:      "hr@example.com",
		Role:       model.LegacyRoleHR,
		InviteCode: "abc123",
	})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	if result.UserID == 0 || result.Role != model.LegacyRoleHR {
		t.Fatalf("unexpected register result: %#v", result)
	}
	user := users.byID[result.UserID]
	if user.Username != "recruiter" || user.AccountType != model.AccountTypeStaff || user.TokenVersion != 1 {
		t.Fatalf("unexpected created user: %#v", user)
	}
	if got := authz.assignedRoles[result.UserID]; len(got) != 1 || got[0] != model.RoleRecruiter {
		t.Fatalf("assigned roles = %#v, want recruiter", got)
	}
	if got := authz.assignedScopes[result.UserID]; len(got) != 1 || got[0] != model.ScopeOwnJobs {
		t.Fatalf("assigned scopes = %#v, want own_jobs", got)
	}
}

func TestAuthServiceRegisterStaffEnforcesTenantMemberQuotaBeforeCreatingUser(t *testing.T) {
	users := newMemoryUsers()
	tenantRepo := &tenantRepositoryFake{quotaErr: commonsquota.ErrLimitExceeded}
	auth := newTestAuthService(t, AuthDeps{
		Users: users, Authz: newMemoryAuthz(), Tenants: tenantRepo,
		Invites: staticInvites{invite: &model.InviteCode{ID: 1, Code: "abc123", TenantID: 7, CreatedBy: 99, IsActive: true}},
	})
	_, err := auth.Register(context.Background(), command.Register{Username: "quota-user", Password: "Password1", Role: model.LegacyRoleHR, InviteCode: "abc123"})
	if !errors.Is(err, ErrTenantMemberQuota) {
		t.Fatalf("Register error = %v, want ErrTenantMemberQuota", err)
	}
	if len(users.byID) != 0 {
		t.Fatal("quota denial must happen before creating the global user")
	}
}

func TestAuthServiceLoginCreatesOpaqueRefreshToken(t *testing.T) {
	users := newMemoryUsers()
	users.seed(&model.User{
		ID:           7,
		Username:     "alice",
		PasswordHash: "hash:Secret123",
		Role:         model.LegacyRoleCandidate,
		AccountType:  model.AccountTypeCandidate,
		TokenVersion: 3,
	})
	tokens := &recordingTokens{}
	authz := newMemoryAuthz()
	authz.userRoles[7] = []string{model.RoleCandidate}
	authz.userPerms[7] = []string{"auth.session.read"}
	auth := newTestAuthService(t, AuthDeps{Users: users, Tokens: tokens, Authz: authz})

	result, err := auth.Login(context.Background(), command.Login{Username: "alice", Password: "Secret123"})
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if result.RefreshToken != "refresh-1" || result.TokenVersion != 3 {
		t.Fatalf("unexpected auth result: %#v", result)
	}
	if tokens.created.familyID != "family-1" {
		t.Fatalf("family id = %q, want family-1", tokens.created.familyID)
	}
	wantExpiry := fixedNow.Add(port.RefreshTokenTTL).Unix()
	if tokens.created.expiresAt.Unix() != wantExpiry {
		t.Fatalf("refresh expiry = %v, want unix %d", tokens.created.expiresAt, wantExpiry)
	}
}

func TestAuthServiceRefreshTokenMapsReuseDetection(t *testing.T) {
	auth := newTestAuthService(t, AuthDeps{
		Users:  newMemoryUsers(),
		Tokens: &recordingTokens{rotateErr: repository.ErrTokenReuseDetected},
	})
	_, err := auth.RefreshToken(context.Background(), command.RefreshToken{RefreshToken: "old"})
	if !errors.Is(err, ErrRefreshTokenReused) {
		t.Fatalf("RefreshToken error = %v, want %v", err, ErrRefreshTokenReused)
	}
}

func newTestAuthService(t *testing.T, deps AuthDeps) *AuthService {
	t.Helper()
	if deps.Passwords == nil {
		deps.Passwords = fakePasswords{}
	}
	if deps.TokenFactory == nil {
		deps.TokenFactory = &sequenceTokens{}
	}
	if deps.Clock == nil {
		deps.Clock = fixedClock{}
	}
	svc, err := NewAuthService(deps)
	if err != nil {
		t.Fatalf("NewAuthService returned error: %v", err)
	}
	return svc
}

type fakePasswords struct{}

func (fakePasswords) HashPassword(password string) (string, error) {
	return "hash:" + password, nil
}

func (fakePasswords) VerifyPassword(hash, password string) bool {
	return hash == "hash:"+password
}

var fixedNow = time.Date(2026, 7, 13, 8, 30, 0, 0, time.UTC)

type fixedClock struct{}

func (fixedClock) Now() time.Time {
	return fixedNow
}

type sequenceTokens struct {
	refreshCount int
	familyCount  int
}

func (g *sequenceTokens) NewRefreshToken() (string, error) {
	g.refreshCount++
	return "refresh-1", nil
}

func (g *sequenceTokens) NewFamilyID() (string, error) {
	g.familyCount++
	return "family-1", nil
}

type staticInvites struct {
	invite *model.InviteCode
	err    error
}

func (s staticInvites) GetByCode(context.Context, string) (*model.InviteCode, error) {
	return s.invite, s.err
}

type recordingTokens struct {
	created struct {
		userID       int64
		token        string
		familyID     string
		clientApp    string
		tenantID     int64
		membershipID int64
		expiresAt    time.Time
	}
	rotateErr error
}

func (r *recordingTokens) GetActive(context.Context, string) (*model.RefreshSession, error) {
	if r.rotateErr != nil {
		return nil, r.rotateErr
	}
	return &model.RefreshSession{UserID: 7, Username: "alice", AccountType: model.AccountTypeCandidate}, nil
}

func (r *recordingTokens) Create(_ context.Context, userID int64, plainToken, familyID, clientApp string, tenantID, membershipID int64, expiresAt time.Time, _, _ string) error {
	r.created.userID = userID
	r.created.token = plainToken
	r.created.familyID = familyID
	r.created.clientApp = clientApp
	r.created.tenantID = tenantID
	r.created.membershipID = membershipID
	r.created.expiresAt = expiresAt
	return nil
}

func (r *recordingTokens) Rotate(context.Context, string, string, time.Time, string, string) (*model.RefreshSession, error) {
	if r.rotateErr != nil {
		return nil, r.rotateErr
	}
	return &model.RefreshSession{UserID: 7, Username: "alice", Role: model.LegacyRoleCandidate, AccountType: model.AccountTypeCandidate, TokenVersion: 3}, nil
}

func (r *recordingTokens) RotateToTenant(context.Context, string, string, int64, int64, time.Time, string, string) (*model.RefreshSession, error) {
	if r.rotateErr != nil {
		return nil, r.rotateErr
	}
	return &model.RefreshSession{UserID: 7, Username: "alice", Role: model.LegacyRoleHR, AccountType: model.AccountTypeStaff, TokenVersion: 3, ClientApp: "staff"}, nil
}

func (r *recordingTokens) Revoke(context.Context, string) error {
	return nil
}
