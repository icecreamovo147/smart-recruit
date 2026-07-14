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
	"smart-recruit-platform-go/errs"
	"smart-recruit-proto/recruitment/pb"
)

type Server struct {
	auth  *appservice.AuthService
	admin *appservice.AdminService
}

func NewServer(auth *appservice.AuthService, admin *appservice.AdminService) *Server {
	return &Server{auth: auth, admin: admin}
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
			errors.Is(err, appservice.ErrUsernameExists):
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
	result, err := s.auth.Login(ctx, command.Login{Username: req.Username, Password: req.Password})
	if err != nil {
		if errors.Is(err, appservice.ErrInvalidCredentials) {
			return &pb.LoginResponse{Code: errs.ErrUnauthorized, Msg: "用户名或密码错误"}, nil
		}
		return nil, err
	}
	return &pb.LoginResponse{
		Code:         errs.OK,
		Msg:          "登录成功",
		Token:        result.RefreshToken,
		UserId:       result.UserID,
		Role:         result.Role,
		Username:     result.Username,
		Email:        result.Email,
		AccountType:  result.AccountType,
		Roles:        result.Roles,
		Permissions:  result.Permissions,
		TokenVersion: result.TokenVersion,
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
	}); err != nil {
		return nil, err
	}
	return &pb.CommonResponse{Code: errs.OK, Msg: "audited"}, nil
}

func (s *Server) GetPrincipal(ctx context.Context, req *pb.GetPrincipalRequest) (*pb.GetPrincipalResponse, error) {
	principal, err := s.auth.GetPrincipal(ctx, query.GetPrincipal{UserID: req.UserId})
	if err != nil {
		if errors.Is(err, appservice.ErrAuthzRepoUnavailable) {
			return &pb.GetPrincipalResponse{Code: errs.ErrInternal, Msg: "authz repo not configured"}, nil
		}
		return nil, err
	}
	return principalResponse(principal), nil
}

func (s *Server) AuthorizeInternal(ctx context.Context, req *pb.AuthorizeInternalRequest) (*pb.AuthorizeInternalResponse, error) {
	principal, err := s.auth.GetPrincipal(ctx, query.GetPrincipal{UserID: req.ActorUserId})
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
		Code:         errs.OK,
		Msg:          "success",
		UserId:       principal.UserID,
		Username:     principal.Username,
		AccountType:  principal.AccountType,
		Role:         principal.LegacyRole,
		Roles:        principal.Roles,
		Permissions:  principal.Permissions,
		TokenVersion: principal.TokenVersion,
		DataScopes:   scopes,
		Email:        principal.Email,
	}
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
			CreatedAt:   role.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   role.UpdatedAt.Format(time.RFC3339),
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
			AssignedAt:   scope.AssignedAt.Format(time.RFC3339),
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
			CreatedAt:     log.CreatedAt.Format(time.RFC3339),
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
