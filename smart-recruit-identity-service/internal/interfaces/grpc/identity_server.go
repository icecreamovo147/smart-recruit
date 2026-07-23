package grpc

import (
	"context"
	"errors"
	"strings"
	"time"

	"smart-recruit-identity-service/internal/application/command"
	"smart-recruit-identity-service/internal/application/query"
	appservice "smart-recruit-identity-service/internal/application/service"
	securitymodel "smart-recruit-identity-service/internal/domain/model"
	"smart-recruit-identity-service/internal/domain/policy"
	"smart-recruit-identity-service/internal/domain/repository"
	"smart-recruit-platform-go/businessclock"
	"smart-recruit-platform-go/errs"
	"smart-recruit-proto/recruitment/pb"
)

type Server struct {
	auth   *appservice.AuthService
	admin  *appservice.AdminService
	tenant *appservice.TenantService
}

func NewServer(auth *appservice.AuthService, admin *appservice.AdminService, tenant *appservice.TenantService) *Server {
	return &Server{auth: auth, admin: admin, tenant: tenant}
}

func (s *Server) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	result, err := s.auth.Register(ctx, command.Register{
		Username:   req.Username,
		Password:   req.Password,
		Email:      req.Email,
		Role:       req.Role,
		InviteCode: req.InviteCode,
	})
	if err != nil {
		switch {
		case errors.Is(err, policy.ErrPasswordTooShort),
			errors.Is(err, policy.ErrPasswordTooLong),
			errors.Is(err, policy.ErrPasswordWeak),
			errors.Is(err, policy.ErrStaffInviteRequired),
			errors.Is(err, policy.ErrInvalidRegistrationRole),
			errors.Is(err, policy.ErrUsernameInvalid),
			errors.Is(err, policy.ErrUsernameBlank),
			errors.Is(err, appservice.ErrInviteInvalid),
			errors.Is(err, appservice.ErrUsernameExists),
			errors.Is(err, appservice.ErrTenantMemberQuota):
			return &pb.RegisterResponse{Code: errs.ErrBadRequest, Msg: err.Error()}, nil
		case errors.Is(err, appservice.ErrInviteServiceUnavailable),
			errors.Is(err, appservice.ErrRoleSeedMissing),
			errors.Is(err, appservice.ErrAccountCreateFailed):
			return &pb.RegisterResponse{Code: errs.ErrInternal, Msg: err.Error()}, nil
		default:
			return nil, err
		}
	}
	return &pb.RegisterResponse{Code: errs.OK, Msg: "注册成功", UserId: result.UserID, Username: result.Username, Role: result.Role}, nil
}

func (s *Server) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	result, err := s.auth.Login(ctx, command.Login{
		Username: req.Username, Password: req.Password,
		ClientApp: req.ClientApp, RequestedTenantID: req.RequestedTenantId,
	})
	if err != nil {
		if errors.Is(err, appservice.ErrInvalidCredentials) {
			return &pb.LoginResponse{Code: errs.ErrUnauthorized, Msg: "用户名或密码错误"}, nil
		}
		return nil, err
	}
	return &pb.LoginResponse{
		Code:          errs.OK,
		Msg:           "登录成功",
		Token:         result.RefreshToken,
		UserId:        result.UserID,
		Role:          result.Role,
		Username:      result.Username,
		Email:         result.Email,
		AccountType:   result.AccountType,
		Roles:         result.Roles,
		Permissions:   result.Permissions,
		TokenVersion:  result.TokenVersion,
		TenantId:      result.TenantID,
		MembershipId:  result.MembershipID,
		ClientApp:     result.ClientApp,
		AvailableApps: result.AvailableApps,
		Memberships:   membershipsResponse(result.Memberships),
	}, nil
}

func (s *Server) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	result, err := s.auth.RefreshToken(ctx, command.RefreshToken{
		RefreshToken: req.RefreshToken,
		ClientIP:     req.ClientIp,
		UserAgent:    req.UserAgent,
	})
	if err != nil {
		if errors.Is(err, appservice.ErrRefreshTokenInvalid) {
			return &pb.RefreshTokenResponse{Code: errs.ErrUnauthorized, Msg: "令牌无效或已过期，请重新登录"}, nil
		}
		if errors.Is(err, appservice.ErrRefreshTokenReused) {
			return &pb.RefreshTokenResponse{Code: errs.ErrUnauthorized, Msg: "会话异常，请重新登录"}, nil
		}
		return nil, err
	}
	return &pb.RefreshTokenResponse{
		Code:             errs.OK,
		Msg:              "刷新成功",
		UserId:           result.UserID,
		Username:         result.Username,
		Role:             result.Role,
		RefreshToken:     result.RefreshToken,
		RefreshExpiresAt: result.RefreshExpiresAt,
		AccountType:      result.AccountType,
		Roles:            result.Roles,
		Permissions:      result.Permissions,
		TokenVersion:     result.TokenVersion,
		TenantId:         result.TenantID,
		MembershipId:     result.MembershipID,
		ClientApp:        result.ClientApp,
	}, nil
}

func (s *Server) SwitchTenant(ctx context.Context, req *pb.SwitchTenantRequest) (*pb.LoginResponse, error) {
	result, err := s.auth.SwitchTenant(ctx, command.SwitchTenant{
		RefreshToken: req.RefreshToken,
		TenantID:     req.TenantId,
		ClientIP:     req.ClientIp,
		UserAgent:    req.UserAgent,
	})
	if err != nil {
		if errors.Is(err, appservice.ErrRefreshTokenInvalid) || errors.Is(err, appservice.ErrRefreshTokenReused) || errors.Is(err, appservice.ErrInvalidCredentials) {
			return &pb.LoginResponse{Code: errs.ErrUnauthorized, Msg: "无权切换到该企业或会话已失效"}, nil
		}
		return nil, err
	}
	return &pb.LoginResponse{
		Code: errs.OK, Msg: "企业切换成功", Token: result.RefreshToken,
		UserId: result.UserID, Username: result.Username, Role: result.Role, AccountType: result.AccountType,
		Roles: result.Roles, Permissions: result.Permissions, TokenVersion: result.TokenVersion,
		TenantId: result.TenantID, MembershipId: result.MembershipID, ClientApp: result.ClientApp,
		AvailableApps: result.AvailableApps, Memberships: membershipsResponse(result.Memberships),
	}, nil
}

func (s *Server) RevokeRefreshToken(ctx context.Context, req *pb.RevokeRefreshTokenRequest) (*pb.CommonResponse, error) {
	err := s.auth.RevokeRefreshToken(ctx, command.RevokeRefreshToken{RefreshToken: req.RefreshToken})
	if err != nil {
		if errors.Is(err, appservice.ErrRefreshTokenRequired) {
			return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "令牌不能为空"}, nil
		}
		return nil, err
	}
	return &pb.CommonResponse{Code: errs.OK, Msg: "已撤销"}, nil
}

func (s *Server) RecordAuthDecision(ctx context.Context, req *pb.AuthAuditRequest) (*pb.CommonResponse, error) {
	if err := s.auth.RecordAuthDecision(ctx, command.RecordAuthDecision{
		ActorUserID:   uint64(req.ActorUserId),
		ActorRoles:    req.ActorRoles,
		PermissionKey: req.PermissionKey,
		ResourceType:  req.ResourceType,
		ResourceID:    req.ResourceId,
		Decision:      req.Decision,
		Reason:        req.Reason,
		RequestID:     req.RequestId,
		ClientIP:      req.ClientIp,
		TenantID:      req.TenantId,
		MembershipID:  req.MembershipId,
	}); err != nil {
		return nil, err
	}
	return &pb.CommonResponse{Code: errs.OK, Msg: "audited"}, nil
}

func (s *Server) GetPrincipal(ctx context.Context, req *pb.GetPrincipalRequest) (*pb.GetPrincipalResponse, error) {
	principal, err := s.auth.GetPrincipal(ctx, query.GetPrincipal{
		UserID: req.UserId, TenantID: req.TenantId, MembershipID: req.MembershipId, ClientApp: req.ClientApp,
	})
	if err != nil {
		if errors.Is(err, appservice.ErrAuthzRepoUnavailable) {
			return &pb.GetPrincipalResponse{Code: errs.ErrInternal, Msg: "authz repo not configured"}, nil
		}
		return nil, err
	}
	return principalResponse(principal), nil
}

func (s *Server) AuthorizeInternal(ctx context.Context, req *pb.AuthorizeInternalRequest) (*pb.AuthorizeInternalResponse, error) {
	principal, err := s.auth.GetPrincipal(ctx, query.GetPrincipal{
		UserID: req.ActorUserId, TenantID: req.TenantId, MembershipID: req.MembershipId, ClientApp: req.ClientApp,
	})
	if err != nil {
		if errors.Is(err, appservice.ErrAuthzRepoUnavailable) {
			return &pb.AuthorizeInternalResponse{Code: errs.ErrInternal, Msg: "authz repo not configured", Allowed: false, Reason: "authz repo not configured"}, nil
		}
		return nil, err
	}
	resp := &pb.AuthorizeInternalResponse{
		Code:      errs.OK,
		Msg:       "success",
		Allowed:   true,
		Principal: principalResponse(principal),
	}
	if principal == nil {
		resp.Allowed = false
		resp.Reason = "principal not found"
	} else if req.PermissionKey != "" && !principal.HasPermission(req.PermissionKey) {
		resp.Allowed = false
		resp.Reason = "permission denied"
	} else if req.RequiredScopeKey != "" {
		matched := matchingScopes(principal, req.RequiredScopeKey, req.ResourceType, req.ResourceId)
		if len(matched) == 0 {
			resp.Allowed = false
			resp.Reason = "scope denied"
		}
		resp.MatchedScopes = matched
	}
	if resp.Allowed {
		resp.Reason = "allowed"
	}
	decision := "deny"
	if resp.Allowed {
		decision = "allow"
	}
	if err := s.auth.RecordAuthDecision(ctx, command.RecordAuthDecision{
		ActorUserID:   uint64(req.ActorUserId),
		ActorRoles:    strings.Join(resp.Principal.GetRoles(), ","),
		PermissionKey: req.PermissionKey,
		ResourceType:  req.ResourceType,
		ResourceID:    uint64(req.ResourceId),
		Decision:      decision,
		Reason:        resp.Reason,
		RequestID:     req.RequestId,
		ClientIP:      req.ClientIp,
		TenantID:      req.TenantId,
		MembershipID:  req.MembershipId,
	}); err != nil {
		return nil, err
	}
	return resp, nil
}

func principalResponse(principal *securitymodel.Principal) *pb.GetPrincipalResponse {
	if principal == nil {
		return &pb.GetPrincipalResponse{Code: errs.ErrBadRequest, Msg: "principal not found"}
	}
	scopes := make([]*pb.ScopeAssignment, 0, len(principal.DataScopes))
	for _, scope := range principal.DataScopes {
		scopes = append(scopes, &pb.ScopeAssignment{
			ScopeKey:     scope.ScopeKey,
			ResourceType: scope.ResourceType,
			ResourceId:   scope.ResourceID,
		})
	}
	return &pb.GetPrincipalResponse{
		Code:          errs.OK,
		Msg:           "success",
		UserId:        principal.UserID,
		Username:      principal.Username,
		AccountType:   principal.AccountType,
		Role:          principal.LegacyRole,
		Roles:         principal.Roles,
		Permissions:   principal.Permissions,
		TokenVersion:  principal.TokenVersion,
		DataScopes:    scopes,
		Email:         principal.Email,
		TenantId:      principal.TenantID,
		MembershipId:  principal.MembershipID,
		ClientApp:     principal.ClientApp,
		AvailableApps: principal.AvailableApps,
		Memberships:   membershipsResponse(principal.Memberships),
	}
}

func membershipsResponse(memberships []securitymodel.TenantMembership) []*pb.TenantMembershipInfo {
	result := make([]*pb.TenantMembershipInfo, 0, len(memberships))
	for _, membership := range memberships {
		result = append(result, &pb.TenantMembershipInfo{
			MembershipId:     membership.ID,
			TenantId:         membership.TenantID,
			TenantKey:        membership.Tenant.TenantKey,
			Slug:             membership.Tenant.Slug,
			Name:             membership.Tenant.Name,
			TenantStatus:     membership.Tenant.Status,
			MembershipStatus: membership.Status,
			Roles:            membership.Roles,
			IsDefault:        membership.Tenant.IsDefault,
			UserId:           membership.UserID,
			Username:         membership.Username,
			Email:            membership.Email,
			JoinedAt:         formatOptionalTime(membership.JoinedAt),
		})
	}
	return result
}

func (s *Server) GetTenant(ctx context.Context, req *pb.GetTenantRequest) (*pb.TenantResponse, error) {
	tenant, err := s.tenant.Get(ctx, req.TenantId)
	if err != nil {
		if errors.Is(err, appservice.ErrTenantInvalid) {
			return &pb.TenantResponse{Code: errs.ErrBadRequest, Msg: "企业不存在或参数无效"}, nil
		}
		if errors.Is(err, appservice.ErrInvalidCredentials) {
			return &pb.TenantResponse{Code: errs.ErrForbidden, Msg: "无平台查看权限"}, nil
		}
		return nil, err
	}
	return &pb.TenantResponse{Code: errs.OK, Msg: "success", Tenant: tenantResponse(tenant)}, nil
}

func (s *Server) CreateTenant(ctx context.Context, req *pb.CreateTenantRequest) (*pb.TenantResponse, error) {
	tenant, err := s.tenant.Create(ctx, req.Slug, req.Name, req.Timezone, req.Locale)
	if err != nil {
		if errors.Is(err, appservice.ErrTenantInvalid) {
			return &pb.TenantResponse{Code: errs.ErrBadRequest, Msg: err.Error()}, nil
		}
		if errors.Is(err, appservice.ErrInvalidCredentials) {
			return &pb.TenantResponse{Code: errs.ErrForbidden, Msg: "无平台管理权限"}, nil
		}
		return nil, err
	}
	return &pb.TenantResponse{Code: errs.OK, Msg: "企业创建成功", Tenant: tenantResponse(tenant)}, nil
}

func (s *Server) ListTenants(ctx context.Context, req *pb.ListTenantsRequest) (*pb.ListTenantsResponse, error) {
	rows, total, err := s.tenant.List(ctx, req.Page, req.PageSize, req.Keyword, req.Status)
	if err != nil {
		if errors.Is(err, appservice.ErrTenantInvalid) {
			return &pb.ListTenantsResponse{Code: errs.ErrBadRequest, Msg: err.Error()}, nil
		}
		if errors.Is(err, appservice.ErrInvalidCredentials) {
			return &pb.ListTenantsResponse{Code: errs.ErrForbidden, Msg: "无平台管理权限"}, nil
		}
		return nil, err
	}
	list := make([]*pb.TenantInfo, len(rows))
	for i := range rows {
		list[i] = tenantResponse(&rows[i])
	}
	return &pb.ListTenantsResponse{Code: errs.OK, Msg: "success", Total: total, List: list}, nil
}

func (s *Server) UpdateTenantStatus(ctx context.Context, req *pb.UpdateTenantStatusRequest) (*pb.TenantResponse, error) {
	tenant, err := s.tenant.UpdateStatus(ctx, req.TenantId, req.Status, req.Reason)
	if err != nil {
		if errors.Is(err, appservice.ErrTenantInvalid) {
			return &pb.TenantResponse{Code: errs.ErrBadRequest, Msg: err.Error()}, nil
		}
		if errors.Is(err, appservice.ErrInvalidCredentials) {
			return &pb.TenantResponse{Code: errs.ErrForbidden, Msg: "无平台管理权限"}, nil
		}
		return nil, err
	}
	return &pb.TenantResponse{Code: errs.OK, Msg: "企业状态已更新", Tenant: tenantResponse(tenant)}, nil
}

func (s *Server) ListTenantMemberships(ctx context.Context, req *pb.ListTenantMembershipsRequest) (*pb.ListTenantMembershipsResponse, error) {
	rows, total, err := s.tenant.ListMemberships(ctx, req.TenantId, req.Page, req.PageSize)
	if err != nil {
		if errors.Is(err, appservice.ErrTenantInvalid) {
			return &pb.ListTenantMembershipsResponse{Code: errs.ErrBadRequest, Msg: err.Error()}, nil
		}
		if errors.Is(err, appservice.ErrInvalidCredentials) {
			return &pb.ListTenantMembershipsResponse{Code: errs.ErrForbidden, Msg: "无平台管理权限"}, nil
		}
		return nil, err
	}
	return &pb.ListTenantMembershipsResponse{Code: errs.OK, Msg: "success", Total: total, List: membershipsResponse(rows)}, nil
}

func (s *Server) UpdateTenantMembershipStatus(ctx context.Context, req *pb.UpdateTenantMembershipStatusRequest) (*pb.TenantMembershipResponse, error) {
	membership, err := s.tenant.UpdateMembershipStatus(ctx, req.TenantId, req.MembershipId, req.Status, req.Reason)
	if err != nil {
		switch {
		case errors.Is(err, appservice.ErrTenantInvalid):
			return &pb.TenantMembershipResponse{Code: errs.ErrBadRequest, Msg: "成员不存在或参数无效"}, nil
		case errors.Is(err, appservice.ErrLastTenantAdmin):
			return &pb.TenantMembershipResponse{Code: errs.ErrBadRequest, Msg: err.Error()}, nil
		case errors.Is(err, appservice.ErrInvalidCredentials):
			return &pb.TenantMembershipResponse{Code: errs.ErrForbidden, Msg: "无平台成员管理权限"}, nil
		default:
			return nil, err
		}
	}
	items := membershipsResponse([]securitymodel.TenantMembership{*membership})
	return &pb.TenantMembershipResponse{Code: errs.OK, Msg: "成员状态已更新", Membership: items[0]}, nil
}

func (s *Server) GetPlatformDashboard(ctx context.Context, _ *pb.GetPlatformDashboardRequest) (*pb.GetPlatformDashboardResponse, error) {
	dashboard, err := s.tenant.Dashboard(ctx)
	if err != nil {
		if errors.Is(err, appservice.ErrInvalidCredentials) {
			return &pb.GetPlatformDashboardResponse{Code: errs.ErrForbidden, Msg: "无平台总览权限"}, nil
		}
		return nil, err
	}
	distribution := []*pb.PlatformTenantStatusCount{
		{Status: "active", Count: dashboard.ActiveTenants},
		{Status: "suspended", Count: dashboard.SuspendedTenants},
		{Status: "disabled", Count: dashboard.DisabledTenants},
	}
	return &pb.GetPlatformDashboardResponse{
		Code: errs.OK, Msg: "success", TotalTenants: dashboard.TotalTenants,
		ActiveTenants: dashboard.ActiveTenants, SuspendedTenants: dashboard.SuspendedTenants,
		DisabledTenants: dashboard.DisabledTenants, NewTenants_30D: dashboard.NewTenants30D,
		TotalMemberships: dashboard.TotalMemberships, ActiveMemberships: dashboard.ActiveMemberships,
		TenantsWithoutAdmin: dashboard.TenantsWithoutAdmin, StatusDistribution: distribution,
	}, nil
}

func (s *Server) QueryPlatformAuditLogs(ctx context.Context, req *pb.QueryPlatformAuditLogsRequest) (*pb.QueryPlatformAuditLogsResponse, error) {
	startTime, err := parseOptionalTime(req.StartTime)
	if err != nil {
		return &pb.QueryPlatformAuditLogsResponse{Code: errs.ErrBadRequest, Msg: "开始时间格式无效"}, nil
	}
	endTime, err := parseOptionalTime(req.EndTime)
	if err != nil {
		return &pb.QueryPlatformAuditLogsResponse{Code: errs.ErrBadRequest, Msg: "结束时间格式无效"}, nil
	}
	rows, total, err := s.tenant.QueryAuditLogs(ctx, securitymodel.PlatformAuditFilter{
		TenantID: req.TenantId, ActorUserID: req.ActorUserId, Action: strings.TrimSpace(req.Action),
		RequestID: strings.TrimSpace(req.RequestId), StartTime: startTime, EndTime: endTime,
	}, req.Page, req.PageSize)
	if err != nil {
		if errors.Is(err, appservice.ErrTenantInvalid) {
			return &pb.QueryPlatformAuditLogsResponse{Code: errs.ErrBadRequest, Msg: "审计查询参数无效"}, nil
		}
		if errors.Is(err, appservice.ErrInvalidCredentials) {
			return &pb.QueryPlatformAuditLogsResponse{Code: errs.ErrForbidden, Msg: "无平台审计查看权限"}, nil
		}
		return nil, err
	}
	list := make([]*pb.PlatformAuditLogItem, len(rows))
	for i, row := range rows {
		list[i] = &pb.PlatformAuditLogItem{
			Id: row.ID, ActorUserId: row.ActorUserID, ActorUsername: row.ActorUsername,
			Action: row.Action, ResourceType: row.ResourceType, ResourceId: row.ResourceID,
			TargetTenantId: row.TargetTenantID, TargetTenantName: row.TargetTenantName,
			BeforeJson: row.BeforeJSON, AfterJson: row.AfterJSON, RequestId: row.RequestID,
			ClientIp: row.ClientIP, CreatedAt: businessclock.FormatRFC3339(row.CreatedAt),
		}
	}
	return &pb.QueryPlatformAuditLogsResponse{Code: errs.OK, Msg: "success", Total: total, List: list}, nil
}

func (s *Server) ListPlatformPlans(ctx context.Context, req *pb.ListPlatformPlansRequest) (*pb.ListPlatformPlansResponse, error) {
	rows, err := s.tenant.ListPlans(ctx, req.Status)
	if err != nil {
		return &pb.ListPlatformPlansResponse{Code: platformErrorCode(err), Msg: err.Error()}, nil
	}
	list := make([]*pb.PlatformPlanInfo, len(rows))
	for i := range rows {
		list[i] = platformPlanResponse(rows[i])
	}
	return &pb.ListPlatformPlansResponse{Code: errs.OK, Msg: "success", List: list}, nil
}

func (s *Server) SavePlatformPlanVersion(ctx context.Context, req *pb.SavePlatformPlanVersionRequest) (*pb.PlatformPlanVersionResponse, error) {
	version, err := s.tenant.SavePlanVersion(ctx, req.PlanId, req.VersionId, req.ChangeNote, entitlementModels(req.Entitlements))
	if err != nil {
		return &pb.PlatformPlanVersionResponse{Code: platformErrorCode(err), Msg: err.Error()}, nil
	}
	return &pb.PlatformPlanVersionResponse{Code: errs.OK, Msg: "套餐草稿已保存", Version: platformPlanVersionResponse(*version)}, nil
}

func (s *Server) PublishPlatformPlanVersion(ctx context.Context, req *pb.PublishPlatformPlanVersionRequest) (*pb.PlatformPlanVersionResponse, error) {
	effectiveAt, err := businessclock.Parse(req.EffectiveAt)
	if err != nil {
		return &pb.PlatformPlanVersionResponse{Code: errs.ErrBadRequest, Msg: "生效时间格式无效"}, nil
	}
	version, err := s.tenant.PublishPlanVersion(ctx, req.PlanId, req.VersionId, effectiveAt, req.Reason)
	if err != nil {
		return &pb.PlatformPlanVersionResponse{Code: platformErrorCode(err), Msg: err.Error()}, nil
	}
	return &pb.PlatformPlanVersionResponse{Code: errs.OK, Msg: "套餐版本已发布", Version: platformPlanVersionResponse(*version)}, nil
}

func (s *Server) GetTenantSubscription(ctx context.Context, req *pb.GetTenantSubscriptionRequest) (*pb.TenantSubscriptionResponse, error) {
	subscription, err := s.tenant.GetSubscription(ctx, req.TenantId)
	if err != nil {
		return &pb.TenantSubscriptionResponse{Code: platformErrorCode(err), Msg: err.Error()}, nil
	}
	return &pb.TenantSubscriptionResponse{Code: errs.OK, Msg: "success", Subscription: tenantSubscriptionResponse(subscription)}, nil
}

func (s *Server) UpdateTenantSubscription(ctx context.Context, req *pb.UpdateTenantSubscriptionRequest) (*pb.TenantSubscriptionResponse, error) {
	startsAt, err := businessclock.Parse(req.StartsAt)
	if err != nil {
		return &pb.TenantSubscriptionResponse{Code: errs.ErrBadRequest, Msg: "开始时间格式无效"}, nil
	}
	endsAt, err := parseOptionalTime(req.EndsAt)
	if err != nil {
		return &pb.TenantSubscriptionResponse{Code: errs.ErrBadRequest, Msg: "结束时间格式无效"}, nil
	}
	subscription, err := s.tenant.UpdateSubscription(ctx, req.TenantId, req.PlanVersionId, startsAt, endsAt, req.Reason)
	if err != nil {
		return &pb.TenantSubscriptionResponse{Code: platformErrorCode(err), Msg: err.Error()}, nil
	}
	return &pb.TenantSubscriptionResponse{Code: errs.OK, Msg: "租户订阅已更新", Subscription: tenantSubscriptionResponse(subscription)}, nil
}

func (s *Server) UpdateTenantEntitlementOverride(ctx context.Context, req *pb.UpdateTenantEntitlementOverrideRequest) (*pb.TenantSubscriptionResponse, error) {
	expiresAt, err := parseOptionalTime(req.ExpiresAt)
	if err != nil {
		return &pb.TenantSubscriptionResponse{Code: errs.ErrBadRequest, Msg: "过期时间格式无效"}, nil
	}
	subscription, err := s.tenant.UpdateEntitlementOverride(ctx, req.TenantId, securitymodel.PlatformEntitlement{Key: req.EntitlementKey, ValueType: req.ValueType, ValueJSON: req.ValueJson, EnforcementMode: "hard", Source: "override"}, expiresAt, req.Reason)
	if err != nil {
		return &pb.TenantSubscriptionResponse{Code: platformErrorCode(err), Msg: err.Error()}, nil
	}
	return &pb.TenantSubscriptionResponse{Code: errs.OK, Msg: "租户权益覆盖已更新", Subscription: tenantSubscriptionResponse(subscription)}, nil
}

func (s *Server) GetTenantUsage(ctx context.Context, req *pb.GetTenantUsageRequest) (*pb.GetTenantUsageResponse, error) {
	metrics, err := s.tenant.GetUsage(ctx, req.TenantId)
	if err != nil {
		return &pb.GetTenantUsageResponse{Code: platformErrorCode(err), Msg: err.Error()}, nil
	}
	items := make([]*pb.TenantUsageMetric, len(metrics))
	for i, metric := range metrics {
		items[i] = &pb.TenantUsageMetric{Key: metric.Key, UsageValue: metric.UsageValue, QuotaValue: metric.QuotaValue, UsagePercent: metric.UsagePercent, EnforcementMode: metric.EnforcementMode, MeasuredAt: formatTime(metric.MeasuredAt)}
	}
	return &pb.GetTenantUsageResponse{Code: errs.OK, Msg: "success", TenantId: req.TenantId, Metrics: items}, nil
}

func (s *Server) ListQuotaAlerts(ctx context.Context, req *pb.ListQuotaAlertsRequest) (*pb.ListQuotaAlertsResponse, error) {
	rows, total, err := s.tenant.ListAlerts(ctx, securitymodel.QuotaAlertFilter{TenantID: req.TenantId, Status: req.Status, MetricKey: req.MetricKey}, req.Page, req.PageSize)
	if err != nil {
		return &pb.ListQuotaAlertsResponse{Code: platformErrorCode(err), Msg: err.Error()}, nil
	}
	list := make([]*pb.QuotaAlertInfo, len(rows))
	for i := range rows {
		list[i] = quotaAlertResponse(&rows[i])
	}
	return &pb.ListQuotaAlertsResponse{Code: errs.OK, Msg: "success", Total: total, List: list}, nil
}

func (s *Server) UpdateQuotaAlert(ctx context.Context, req *pb.UpdateQuotaAlertRequest) (*pb.QuotaAlertResponse, error) {
	alert, err := s.tenant.UpdateAlert(ctx, req.AlertId, req.Status, req.AssigneeUserId, req.ResolutionNote)
	if err != nil {
		return &pb.QuotaAlertResponse{Code: platformErrorCode(err), Msg: err.Error()}, nil
	}
	return &pb.QuotaAlertResponse{Code: errs.OK, Msg: "告警状态已更新", Alert: quotaAlertResponse(alert)}, nil
}

func platformErrorCode(err error) int32 {
	if errors.Is(err, appservice.ErrInvalidCredentials) {
		return errs.ErrForbidden
	}
	if errors.Is(err, appservice.ErrTenantInvalid) || errors.Is(err, appservice.ErrTenantMemberQuota) {
		return errs.ErrBadRequest
	}
	return errs.ErrInternal
}

func entitlementModels(items []*pb.PlatformEntitlementItem) []securitymodel.PlatformEntitlement {
	result := make([]securitymodel.PlatformEntitlement, len(items))
	for i, item := range items {
		result[i] = securitymodel.PlatformEntitlement{Key: item.Key, ValueType: item.ValueType, ValueJSON: item.ValueJson, EnforcementMode: item.EnforcementMode, Source: item.Source}
	}
	return result
}

func entitlementResponses(items []securitymodel.PlatformEntitlement) []*pb.PlatformEntitlementItem {
	result := make([]*pb.PlatformEntitlementItem, len(items))
	for i, item := range items {
		result[i] = &pb.PlatformEntitlementItem{Key: item.Key, ValueType: item.ValueType, ValueJson: item.ValueJSON, EnforcementMode: item.EnforcementMode, Source: item.Source}
	}
	return result
}

func platformPlanResponse(plan securitymodel.PlatformPlan) *pb.PlatformPlanInfo {
	versions := make([]*pb.PlatformPlanVersionInfo, len(plan.Versions))
	for i := range plan.Versions {
		versions[i] = platformPlanVersionResponse(plan.Versions[i])
	}
	return &pb.PlatformPlanInfo{Id: plan.ID, PlanKey: plan.PlanKey, Name: plan.Name, Description: plan.Description, Status: plan.Status, Versions: versions, CreatedAt: formatTime(plan.CreatedAt), UpdatedAt: formatTime(plan.UpdatedAt)}
}

func platformPlanVersionResponse(version securitymodel.PlatformPlanVersion) *pb.PlatformPlanVersionInfo {
	return &pb.PlatformPlanVersionInfo{Id: version.ID, PlanId: version.PlanID, Version: version.Version, Status: version.Status, EffectiveAt: formatOptionalTime(version.EffectiveAt), RetiredAt: formatOptionalTime(version.RetiredAt), ChangeNote: version.ChangeNote, Entitlements: entitlementResponses(version.Entitlements), CreatedAt: formatTime(version.CreatedAt), UpdatedAt: formatTime(version.UpdatedAt)}
}

func tenantSubscriptionResponse(subscription *securitymodel.TenantSubscription) *pb.TenantSubscriptionInfo {
	if subscription == nil {
		return nil
	}
	return &pb.TenantSubscriptionInfo{Id: subscription.ID, TenantId: subscription.TenantID, PlanVersionId: subscription.PlanVersionID, PlanKey: subscription.PlanKey, PlanName: subscription.PlanName, PlanVersion: subscription.PlanVersion, Status: subscription.Status, StartsAt: formatTime(subscription.StartsAt), EndsAt: formatOptionalTime(subscription.EndsAt), Reason: subscription.Reason, EffectiveEntitlements: entitlementResponses(subscription.EffectiveEntitlements)}
}

func quotaAlertResponse(alert *securitymodel.QuotaAlert) *pb.QuotaAlertInfo {
	if alert == nil {
		return nil
	}
	return &pb.QuotaAlertInfo{Id: alert.ID, TenantId: alert.TenantID, TenantName: alert.TenantName, MetricKey: alert.MetricKey, ThresholdPercent: alert.ThresholdPercent, UsageValue: alert.UsageValue, QuotaValue: alert.QuotaValue, Status: alert.Status, AssigneeUserId: alert.AssigneeUserID, AcknowledgedAt: formatOptionalTime(alert.AcknowledgedAt), ResolvedAt: formatOptionalTime(alert.ResolvedAt), ResolutionNote: alert.ResolutionNote, FirstTriggeredAt: formatTime(alert.FirstTriggeredAt), LastTriggeredAt: formatTime(alert.LastTriggeredAt)}
}

func tenantResponse(tenant *securitymodel.Tenant) *pb.TenantInfo {
	if tenant == nil {
		return nil
	}
	return &pb.TenantInfo{Id: tenant.ID, TenantKey: tenant.TenantKey, Slug: tenant.Slug, Name: tenant.Name, Status: tenant.Status, Timezone: tenant.Timezone, Locale: tenant.Locale, IsDefault: tenant.IsDefault, MembershipCount: tenant.MembershipCount, CreatedAt: formatTime(tenant.CreatedAt), UpdatedAt: formatTime(tenant.UpdatedAt)}
}

func formatOptionalTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return businessclock.FormatRFC3339(*value)
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return businessclock.FormatRFC3339(value)
}

func parseOptionalTime(value string) (*time.Time, error) {
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

func matchingScopes(principal *securitymodel.Principal, requiredScopeKey, resourceType string, resourceID int64) []*pb.ScopeAssignment {
	if principal == nil {
		return nil
	}
	matches := make([]*pb.ScopeAssignment, 0, len(principal.DataScopes))
	for _, scope := range principal.DataScopes {
		if scope.ScopeKey != requiredScopeKey && scope.ScopeKey != securitymodel.ScopeRecruitingAll && scope.ScopeKey != securitymodel.ScopeSystemAll {
			continue
		}
		if resourceType != "" && scope.ResourceType != "" && scope.ResourceType != resourceType {
			continue
		}
		if resourceID > 0 && scope.ResourceID > 0 && scope.ResourceID != resourceID {
			continue
		}
		matches = append(matches, &pb.ScopeAssignment{
			ScopeKey:     scope.ScopeKey,
			ResourceType: scope.ResourceType,
			ResourceId:   scope.ResourceID,
		})
	}
	return matches
}

func (s *Server) UpdateEmail(ctx context.Context, req *pb.UpdateEmailRequest) (*pb.CommonResponse, error) {
	err := s.auth.UpdateEmail(ctx, command.UpdateEmail{UserID: req.UserId, Email: req.Email})
	if err != nil {
		switch {
		case errors.Is(err, policy.ErrEmailTooLong), errors.Is(err, policy.ErrEmailInvalid), errors.Is(err, appservice.ErrUserNotFound):
			return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: err.Error()}, nil
		default:
			return nil, err
		}
	}
	return &pb.CommonResponse{Code: errs.OK, Msg: "邮箱更新成功"}, nil
}

func (s *Server) ListRoles(ctx context.Context, req *pb.ListRolesRequest) (*pb.ListRolesResponse, error) {
	roles, err := s.admin.ListRoles(ctx)
	if err != nil {
		return &pb.ListRolesResponse{Code: errs.ErrForbidden, Msg: err.Error()}, nil
	}
	list := make([]*pb.RoleInfo, len(roles))
	for i, role := range roles {
		list[i] = &pb.RoleInfo{
			Id:          role.ID,
			RoleKey:     role.RoleKey,
			Name:        role.Name,
			Description: role.Description,
			IsSystem:    role.IsSystem,
			CreatedAt:   businessclock.FormatRFC3339(role.CreatedAt),
			UpdatedAt:   businessclock.FormatRFC3339(role.UpdatedAt),
		}
	}
	return &pb.ListRolesResponse{Code: errs.OK, Msg: "success", List: list}, nil
}

func (s *Server) ListPermissions(ctx context.Context, req *pb.ListPermissionsRequest) (*pb.ListPermissionsResponse, error) {
	permissions, err := s.admin.ListPermissions(ctx)
	if err != nil {
		return &pb.ListPermissionsResponse{Code: errs.ErrForbidden, Msg: err.Error()}, nil
	}
	list := make([]*pb.PermissionInfo, len(permissions))
	for i, permission := range permissions {
		list[i] = &pb.PermissionInfo{
			Id:            permission.ID,
			PermissionKey: permission.PermissionKey,
			Resource:      permission.Resource,
			Action:        permission.Action,
			Description:   permission.Description,
		}
	}
	return &pb.ListPermissionsResponse{Code: errs.OK, Msg: "success", List: list}, nil
}

func (s *Server) GetUserRoles(ctx context.Context, req *pb.GetUserRolesRequest) (*pb.GetUserRolesResponse, error) {
	result, err := s.admin.GetUserRoles(ctx, query.GetUserRoles{UserID: req.UserId})
	if err != nil {
		return &pb.GetUserRolesResponse{Code: errs.ErrForbidden, Msg: err.Error()}, nil
	}
	scopes := make([]*pb.DataScopeInfo, 0, len(result.DataScopes))
	for _, scope := range result.DataScopes {
		scopes = append(scopes, &pb.DataScopeInfo{
			Id:           scope.ID,
			ScopeKey:     scope.ScopeKey,
			ResourceType: scope.ResourceType,
			ResourceId:   scope.ResourceID,
			AssignedAt:   businessclock.FormatRFC3339(scope.AssignedAt),
		})
	}
	return &pb.GetUserRolesResponse{Code: errs.OK, Msg: "success", RoleKeys: result.RoleKeys, PermissionKeys: result.PermissionKeys, DataScopes: scopes}, nil
}

func (s *Server) AssignUserRole(ctx context.Context, req *pb.AssignUserRoleRequest) (*pb.CommonResponse, error) {
	err := s.admin.AssignUserRole(ctx, command.AssignUserRole{UserID: req.UserId, AdminID: req.AdminId, RoleKey: req.RoleKey})
	if err != nil {
		if errors.Is(err, appservice.ErrRoleNotFound) {
			return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "角色不存在"}, nil
		}
		if errors.Is(err, appservice.ErrPermissionTokenSyncFailed) {
			return &pb.CommonResponse{Code: errs.ErrInternal, Msg: "权限变更成功但令牌同步失败，请重试"}, nil
		}
		return nil, err
	}
	return &pb.CommonResponse{Code: errs.OK, Msg: "success"}, nil
}

func (s *Server) RevokeUserRole(ctx context.Context, req *pb.RevokeUserRoleRequest) (*pb.CommonResponse, error) {
	err := s.admin.RevokeUserRole(ctx, command.RevokeUserRole{UserID: req.UserId, AdminID: req.AdminId, RoleKey: req.RoleKey})
	if err != nil {
		switch {
		case errors.Is(err, appservice.ErrRoleNotFound):
			return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "角色不存在"}, nil
		case errors.Is(err, appservice.ErrRoleNotHeld):
			return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "该用户未持有此角色"}, nil
		case errors.Is(err, policy.ErrSelfSystemAdminRevoke):
			return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: policy.ErrSelfSystemAdminRevoke.Error()}, nil
		case errors.Is(err, policy.ErrLastSystemAdmin):
			return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: policy.ErrLastSystemAdmin.Error()}, nil
		case errors.Is(err, appservice.ErrPermissionChangeFailed):
			return &pb.CommonResponse{Code: errs.ErrInternal, Msg: "权限变更失败，请稍后重试"}, nil
		default:
			return nil, err
		}
	}
	return &pb.CommonResponse{Code: errs.OK, Msg: "success"}, nil
}

func (s *Server) AssignDataScope(ctx context.Context, req *pb.AssignDataScopeRequest) (*pb.CommonResponse, error) {
	err := s.admin.AssignDataScope(ctx, command.AssignDataScope{
		UserID:       req.UserId,
		AdminID:      req.AdminId,
		ScopeKey:     req.ScopeKey,
		ResourceType: req.ResourceType,
		ResourceID:   req.ResourceId,
	})
	if err != nil {
		if errors.Is(err, appservice.ErrPermissionTokenSyncFailed) {
			return &pb.CommonResponse{Code: errs.ErrInternal, Msg: "权限变更成功但令牌同步失败，请重试"}, nil
		}
		return nil, err
	}
	return &pb.CommonResponse{Code: errs.OK, Msg: "success"}, nil
}

func (s *Server) RevokeDataScope(ctx context.Context, req *pb.RevokeDataScopeRequest) (*pb.CommonResponse, error) {
	err := s.admin.RevokeDataScope(ctx, command.RevokeDataScope{ScopeID: req.ScopeId, AdminID: req.AdminId})
	if err != nil {
		if errors.Is(err, appservice.ErrPermissionTokenSyncFailed) {
			return &pb.CommonResponse{Code: errs.ErrInternal, Msg: "权限变更成功但令牌同步失败，请重试"}, nil
		}
		return nil, err
	}
	return &pb.CommonResponse{Code: errs.OK, Msg: "success"}, nil
}

func (s *Server) ListStaffUsers(ctx context.Context, req *pb.ListStaffUsersRequest) (*pb.ListStaffUsersResponse, error) {
	result, err := s.admin.ListStaffUsers(ctx, query.ListStaffUsers{Page: req.Page, PageSize: req.PageSize, Status: req.Status})
	if err != nil {
		return &pb.ListStaffUsersResponse{Code: errs.ErrForbidden, Msg: err.Error()}, nil
	}
	list := make([]*pb.StaffUserInfo, len(result.List))
	for i, user := range result.List {
		list[i] = &pb.StaffUserInfo{
			UserId:       user.UserID,
			Username:     user.Username,
			Email:        user.Email,
			Status:       user.Status,
			AccountType:  user.AccountType,
			Roles:        user.Roles,
			TokenVersion: user.TokenVersion,
			CreatedAt:    user.CreatedAt,
		}
	}
	return &pb.ListStaffUsersResponse{Code: errs.OK, Msg: "success", Total: result.Total, List: list}, nil
}

func (s *Server) CreateStaffUser(ctx context.Context, req *pb.CreateStaffUserRequest) (*pb.CreateStaffUserResponse, error) {
	userID, err := s.admin.CreateStaffUser(ctx, command.CreateStaffUser{
		Username: req.Username,
		Password: req.Password,
		Email:    req.Email,
		AdminID:  req.AdminId,
		RoleKeys: req.RoleKeys,
	})
	if err != nil {
		switch {
		case errors.Is(err, policy.ErrPasswordTooShort),
			errors.Is(err, policy.ErrPasswordTooLong),
			errors.Is(err, policy.ErrPasswordWeak),
			errors.Is(err, policy.ErrUsernameInvalid),
			errors.Is(err, policy.ErrUsernameBlank),
			errors.Is(err, appservice.ErrUsernameExists):
			return &pb.CreateStaffUserResponse{Code: errs.ErrBadRequest, Msg: err.Error()}, nil
		case errors.Is(err, appservice.ErrAccountCreateFailed):
			return &pb.CreateStaffUserResponse{Code: errs.ErrInternal, Msg: "账号创建失败，请稍后重试"}, nil
		default:
			return nil, err
		}
	}
	return &pb.CreateStaffUserResponse{Code: errs.OK, Msg: "success", UserId: userID}, nil
}

func (s *Server) ListPlatformUsers(ctx context.Context, req *pb.ListPlatformUsersRequest) (*pb.ListPlatformUsersResponse, error) {
	rows, total, err := s.admin.ListPlatformAccounts(ctx, req.Page, req.PageSize, req.Status)
	if err != nil {
		return &pb.ListPlatformUsersResponse{Code: errs.ErrForbidden, Msg: err.Error()}, nil
	}
	list := make([]*pb.PlatformUserInfo, len(rows))
	for i := range rows {
		list[i] = platformUserResponse(&rows[i])
	}
	return &pb.ListPlatformUsersResponse{Code: errs.OK, Msg: "success", Total: total, List: list}, nil
}

func (s *Server) CreatePlatformUser(ctx context.Context, req *pb.CreatePlatformUserRequest) (*pb.CreatePlatformUserResponse, error) {
	userID, err := s.admin.CreatePlatformAccount(ctx, req.AdminId, req.Username, req.Email, req.Password, req.RoleKey)
	if err != nil {
		code := int32(errs.ErrBadRequest)
		if !errors.Is(err, appservice.ErrUsernameExists) && !errors.Is(err, appservice.ErrAccountCreateFailed) && !errors.Is(err, appservice.ErrRoleNotFound) {
			code = errs.ErrForbidden
		}
		return &pb.CreatePlatformUserResponse{Code: code, Msg: err.Error()}, nil
	}
	return &pb.CreatePlatformUserResponse{Code: errs.OK, Msg: "平台账号已创建", UserId: userID}, nil
}

func (s *Server) UpdatePlatformUser(ctx context.Context, req *pb.UpdatePlatformUserRequest) (*pb.PlatformUserResponse, error) {
	user, err := s.admin.UpdatePlatformAccount(ctx, req.AdminId, req.UserId, req.RoleKey, req.Status, req.Reason)
	if err != nil {
		if errors.Is(err, repository.ErrLastAdmin) {
			return &pb.PlatformUserResponse{Code: errs.ErrBadRequest, Msg: "平台必须至少保留一名有效平台管理员"}, nil
		}
		return &pb.PlatformUserResponse{Code: errs.ErrBadRequest, Msg: err.Error()}, nil
	}
	return &pb.PlatformUserResponse{Code: errs.OK, Msg: "平台账号已更新", User: platformUserResponse(user)}, nil
}

func platformUserResponse(user *securitymodel.PlatformAccount) *pb.PlatformUserInfo {
	if user == nil {
		return nil
	}
	return &pb.PlatformUserInfo{UserId: user.ID, Username: user.Username, Email: user.Email, Status: user.Status, Roles: user.Roles, TokenVersion: user.TokenVersion, CreatedAt: formatTime(user.CreatedAt)}
}

func (s *Server) QueryAuthAuditLogs(ctx context.Context, req *pb.QueryAuthAuditLogsRequest) (*pb.QueryAuthAuditLogsResponse, error) {
	var actorUserID *uint64
	if req.ActorUserId != 0 {
		uid := uint64(req.ActorUserId)
		actorUserID = &uid
	}
	result, err := s.admin.QueryAuthAuditLogs(ctx, query.QueryAuthAuditLogs{
		ActorUserID:   actorUserID,
		PermissionKey: req.PermissionKey,
		Decision:      req.Decision,
		Page:          req.Page,
		PageSize:      req.PageSize,
	})
	if err != nil {
		if errors.Is(err, appservice.ErrAuditQueryFailed) {
			return &pb.QueryAuthAuditLogsResponse{Code: errs.ErrInternal, Msg: "查询安全审计日志失败"}, nil
		}
		return &pb.QueryAuthAuditLogsResponse{Code: errs.ErrForbidden, Msg: err.Error()}, nil
	}
	list := make([]*pb.AuthAuditLogItem, len(result.List))
	for i, log := range result.List {
		list[i] = &pb.AuthAuditLogItem{
			Id:            log.ID,
			ActorUserId:   int64(log.ActorUserID),
			ActorRoles:    log.ActorRoles,
			PermissionKey: log.PermissionKey,
			ResourceType:  log.ResourceType,
			ResourceId:    log.ResourceID,
			Decision:      log.Decision,
			Reason:        log.Reason,
			RequestId:     log.RequestID,
			ClientIp:      log.ClientIP,
			CreatedAt:     businessclock.FormatRFC3339(log.CreatedAt),
		}
	}
	return &pb.QueryAuthAuditLogsResponse{Code: errs.OK, Msg: "success", Total: result.Total, List: list}, nil
}

var (
	_ interface {
		Register(context.Context, *pb.RegisterRequest) (*pb.RegisterResponse, error)
		Login(context.Context, *pb.LoginRequest) (*pb.LoginResponse, error)
		RefreshToken(context.Context, *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error)
		RevokeRefreshToken(context.Context, *pb.RevokeRefreshTokenRequest) (*pb.CommonResponse, error)
		RecordAuthDecision(context.Context, *pb.AuthAuditRequest) (*pb.CommonResponse, error)
		GetPrincipal(context.Context, *pb.GetPrincipalRequest) (*pb.GetPrincipalResponse, error)
		AuthorizeInternal(context.Context, *pb.AuthorizeInternalRequest) (*pb.AuthorizeInternalResponse, error)
		UpdateEmail(context.Context, *pb.UpdateEmailRequest) (*pb.CommonResponse, error)
	} = (*Server)(nil)

	_ interface {
		ListRoles(context.Context, *pb.ListRolesRequest) (*pb.ListRolesResponse, error)
		ListPermissions(context.Context, *pb.ListPermissionsRequest) (*pb.ListPermissionsResponse, error)
		GetUserRoles(context.Context, *pb.GetUserRolesRequest) (*pb.GetUserRolesResponse, error)
		AssignUserRole(context.Context, *pb.AssignUserRoleRequest) (*pb.CommonResponse, error)
		RevokeUserRole(context.Context, *pb.RevokeUserRoleRequest) (*pb.CommonResponse, error)
		AssignDataScope(context.Context, *pb.AssignDataScopeRequest) (*pb.CommonResponse, error)
		RevokeDataScope(context.Context, *pb.RevokeDataScopeRequest) (*pb.CommonResponse, error)
		ListStaffUsers(context.Context, *pb.ListStaffUsersRequest) (*pb.ListStaffUsersResponse, error)
		CreateStaffUser(context.Context, *pb.CreateStaffUserRequest) (*pb.CreateStaffUserResponse, error)
	} = (*Server)(nil)

	_ interface {
		QueryAuthAuditLogs(context.Context, *pb.QueryAuthAuditLogsRequest) (*pb.QueryAuthAuditLogsResponse, error)
	} = (*Server)(nil)
)
