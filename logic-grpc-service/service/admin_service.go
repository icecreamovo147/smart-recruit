package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"logic-grpc-service/model"
	"logic-grpc-service/pkg/authz"
	"logic-grpc-service/pkg/errs"
	"logic-grpc-service/pkg/jwt"
	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/pkg/metadata"
	"logic-grpc-service/recruitment/pb"
	"logic-grpc-service/repository"
)

type AdminService struct {
	inviteCodes *repository.InviteCodeRepo
	usageLogs   *repository.UsageLogRepo
	users       *repository.UserRepo
	authz       *repository.AuthzRepo
	redisClient *redis.Client // optional: syncs token_version to Redis after permission changes
	serviceAuth *ServiceAuthorizer
}

// getServiceAuth returns the ServiceAuthorizer, creating a nil-safe one if not configured.
func (s *AdminService) getServiceAuth() *ServiceAuthorizer {
	if s.serviceAuth != nil {
		return s.serviceAuth
	}
	return NewServiceAuthorizer(nil, nil)
}

// verifyAdminPermission extracts the authenticated actor from the gRPC context
// and verifies they hold the required permission.
//
// FAIL-CLOSED: if the authenticated actor is not present in the context,
// the request is REJECTED. Every admin-facing read must carry the
// authenticated user via gRPC metadata.
func (s *AdminService) verifyAdminPermission(ctx context.Context, permKey string) error {
	actorID := metadata.GetAuthUserID(ctx)
	if actorID == 0 {
		return fmt.Errorf("authenticated user not found in context — gRPC metadata x-authenticated-user-id is required for admin operations")
	}
	return s.getServiceAuth().AuthorizePermission(ctx, uint64(actorID), permKey)
}

func NewAdminService(inviteCodes *repository.InviteCodeRepo, usageLogs *repository.UsageLogRepo, users *repository.UserRepo, authz *repository.AuthzRepo, redisClient *redis.Client, serviceAuth *ServiceAuthorizer) *AdminService {
	return &AdminService{inviteCodes: inviteCodes, usageLogs: usageLogs, users: users, authz: authz, redisClient: redisClient, serviceAuth: serviceAuth}
}

// setTokenVersionCache writes the current token_version to Redis so the
// JWT middleware can detect stale tokens after permission changes.
//
// Fail-safe strategy:
//  1. SET the new version → success, done.
//  2. If SET fails, DELETE the key → success means middleware will reject on
//     cache miss (fail-closed), so the stale token is still invalidated.
//  3. If both SET and DEL fail, return an error — the caller should fail the
//     mutation because we cannot guarantee token invalidation.
//
// This is a no-op when Redis is not configured.
func (s *AdminService) setTokenVersionCache(ctx context.Context, userID uint64, version int32) error {
	if s.redisClient == nil {
		return nil
	}
	key := fmt.Sprintf("token_version:%d", userID)
	if err := s.redisClient.Set(ctx, key, version, jwt.AccessTokenTTL).Err(); err != nil {
		logger.L().Warn("failed to SET token_version to redis, attempting DELETE as fail-safe",
			zap.Uint64("user_id", userID), zap.Error(err))
		// DELETE the stale key so the JWT middleware (fail-closed on cache miss)
		// will reject the old token instead of allowing it through.
		if delErr := s.redisClient.Del(ctx, key).Err(); delErr != nil {
			logger.L().Error("failed to DELETE token_version from redis, token invalidation not guaranteed",
				zap.Uint64("user_id", userID), zap.Error(delErr))
			return fmt.Errorf("token_version sync failed: SET error=%w, DEL error=%w", err, delErr)
		}
		logger.L().Info("token_version key deleted from redis as fail-safe, old tokens will be rejected",
			zap.Uint64("user_id", userID))
	}
	return nil
}

// auditAdminAction writes an authorization audit log for admin mutations.
func (s *AdminService) auditAdminAction(ctx context.Context, actorID uint64, action string, targetUserID uint64, decision, detail, requestID, clientIP string) {
	if s.authz == nil {
		return
	}
	_ = s.authz.RecordAuthDecision(ctx, actorID, "", "admin."+action, "user", targetUserID, decision, detail, requestID, clientIP)
}

const inviteCodeChars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func generateInviteCode() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = inviteCodeChars[int(b[i])%len(inviteCodeChars)]
	}
	return string(b), nil
}

func (s *AdminService) CreateInviteCode(ctx context.Context, req *pb.CreateInviteCodeRequest) (*pb.CreateInviteCodeResponse, error) {
	code, err := generateInviteCode()
	if err != nil {
		return nil, err
	}
	ic := &model.InviteCode{
		Code:      code,
		CreatedBy: req.CreatedBy,
		IsActive:  1,
	}
	if req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			return &pb.CreateInviteCodeResponse{Code: errs.ErrBadRequest, Msg: "过期时间格式无效，请使用 RFC 3339 格式"}, nil
		}
		ic.ExpiresAt = &t
	}
	if err := s.inviteCodes.Create(ctx, ic); err != nil {
		logger.L().Error("create invite code failed", zap.Error(err))
		return nil, err
	}
	logger.L().Info("invite code created", zap.Int64("created_by", req.CreatedBy), zap.String("code", code))
	return &pb.CreateInviteCodeResponse{Code: errs.OK, Msg: "success", InviteCode: toPBInviteCode(ic)}, nil
}

func (s *AdminService) ListInviteCodes(ctx context.Context, req *pb.ListInviteCodesRequest) (*pb.ListInviteCodesResponse, error) {
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	rows, total, err := s.inviteCodes.ListByCreator(ctx, req.CreatedBy, page, pageSize)
	if err != nil {
		logger.L().Error("list invite codes failed", zap.Error(err))
		return nil, err
	}
	list := make([]*pb.InviteCodeInfo, len(rows))
	for i, r := range rows {
		list[i] = toPBInviteCode(&r)
	}
	return &pb.ListInviteCodesResponse{Code: errs.OK, Msg: "success", Total: total, List: list}, nil
}

func (s *AdminService) ExtendInviteCode(ctx context.Context, req *pb.ExtendInviteCodeRequest) (*pb.CommonResponse, error) {
	ic, err := s.inviteCodes.GetByID(ctx, req.Id)
	if err != nil {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "邀请码不存在"}, nil
	}
	if ic.CreatedBy != req.AdminId {
		return &pb.CommonResponse{Code: errs.ErrUnauthorized, Msg: "无权操作该邀请码"}, nil
	}
	t, err := time.Parse(time.RFC3339, req.NewExpiresAt)
	if err != nil {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "过期时间格式无效"}, nil
	}
	if err := s.inviteCodes.Extend(ctx, req.Id, &t); err != nil {
		return nil, err
	}
	logger.L().Info("invite code extended", zap.Int64("id", req.Id), zap.Time("new_expires_at", t))
	return &pb.CommonResponse{Code: errs.OK, Msg: "success"}, nil
}

func (s *AdminService) RevokeInviteCode(ctx context.Context, req *pb.RevokeInviteCodeRequest) (*pb.CommonResponse, error) {
	ic, err := s.inviteCodes.GetByID(ctx, req.Id)
	if err != nil {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "邀请码不存在"}, nil
	}
	if ic.CreatedBy != req.AdminId {
		return &pb.CommonResponse{Code: errs.ErrUnauthorized, Msg: "无权操作该邀请码"}, nil
	}
	if err := s.inviteCodes.Revoke(ctx, req.Id); err != nil {
		return nil, err
	}
	logger.L().Info("invite code revoked", zap.Int64("id", req.Id))
	return &pb.CommonResponse{Code: errs.OK, Msg: "success"}, nil
}

func (s *AdminService) ReactivateInviteCode(ctx context.Context, req *pb.ReactivateInviteCodeRequest) (*pb.CommonResponse, error) {
	ic, err := s.inviteCodes.GetByID(ctx, req.Id)
	if err != nil {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "邀请码不存在"}, nil
	}
	if ic.CreatedBy != req.AdminId {
		return &pb.CommonResponse{Code: errs.ErrUnauthorized, Msg: "无权操作该邀请码"}, nil
	}
	if err := s.inviteCodes.Reactivate(ctx, req.Id); err != nil {
		return nil, err
	}
	logger.L().Info("invite code reactivated", zap.Int64("id", req.Id))
	return &pb.CommonResponse{Code: errs.OK, Msg: "success"}, nil
}

func (s *AdminService) ValidateInviteCode(ctx context.Context, req *pb.ValidateInviteCodeRequest) (*pb.ValidateInviteCodeResponse, error) {
	if req.InviteCode == "" {
		return &pb.ValidateInviteCodeResponse{Code: errs.OK, Msg: "success", Valid: false}, nil
	}
	_, err := s.inviteCodes.GetByCode(ctx, req.InviteCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &pb.ValidateInviteCodeResponse{Code: errs.OK, Msg: "success", Valid: false}, nil
		}
		logger.L().Error("validate invite code: db error", zap.Error(err))
		return nil, err
	}
	return &pb.ValidateInviteCodeResponse{Code: errs.OK, Msg: "success", Valid: true}, nil
}

func toPBInviteCode(ic *model.InviteCode) *pb.InviteCodeInfo {
	info := &pb.InviteCodeInfo{
		Id:        ic.ID,
		Code:      ic.Code,
		CreatedBy: ic.CreatedBy,
		IsActive:  ic.IsActive,
		CreatedAt: ic.CreatedAt.Format(time.RFC3339),
	}
	if ic.ExpiresAt != nil {
		info.ExpiresAt = ic.ExpiresAt.Format(time.RFC3339)
	}
	return info
}

func (s *AdminService) QueryUsageLogs(ctx context.Context, req *pb.QueryUsageLogsRequest) (*pb.QueryUsageLogsResponse, error) {
	if err := s.verifyAdminPermission(ctx, authz.PermAuditUsageRead); err != nil {
		return &pb.QueryUsageLogsResponse{Code: errs.ErrForbidden, Msg: err.Error()}, nil
	}
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	var filter repository.UsageLogFilter
	if req.ServiceType != "" {
		filter.ServiceType = req.ServiceType
	}
	if req.Provider != "" {
		filter.Provider = req.Provider
	}
	if req.Status != "" {
		filter.Status = req.Status
	}
	if req.UserId != 0 {
		filter.UserID = req.UserId
	}
	if req.RequestId != "" {
		filter.RequestID = req.RequestId
	}
	if req.StartTime != "" {
		t, err := time.Parse(time.RFC3339, req.StartTime)
		if err != nil {
			return &pb.QueryUsageLogsResponse{Code: errs.ErrBadRequest, Msg: "开始时间格式无效"}, nil
		}
		filter.StartTime = &t
	}
	if req.EndTime != "" {
		t, err := time.Parse(time.RFC3339, req.EndTime)
		if err != nil {
			return &pb.QueryUsageLogsResponse{Code: errs.ErrBadRequest, Msg: "结束时间格式无效"}, nil
		}
		filter.EndTime = &t
	}

	logs, total, err := s.usageLogs.List(ctx, filter, int(page), int(pageSize))
	if err != nil {
		logger.L().Error("query usage logs failed", zap.Error(err))
		return nil, err
	}

	list := make([]*pb.UsageLogItem, len(logs))
	for i, l := range logs {
		list[i] = toPBUsageLogItem(&l)
	}
	return &pb.QueryUsageLogsResponse{Code: errs.OK, Msg: "success", Total: total, List: list}, nil
}

// ── RBAC Role & Permission Management ──────────────────────────────────

func (s *AdminService) ListRoles(ctx context.Context, req *pb.ListRolesRequest) (*pb.ListRolesResponse, error) {
	if s.authz == nil {
		return &pb.ListRolesResponse{Code: errs.ErrInternal, Msg: "authz repo not configured"}, nil
	}
	if err := s.verifyAdminPermission(ctx, authz.PermAdminRoleManage); err != nil {
		return &pb.ListRolesResponse{Code: errs.ErrForbidden, Msg: err.Error()}, nil
	}
	roles, err := s.authz.ListRoles(ctx)
	if err != nil {
		return nil, err
	}
	list := make([]*pb.RoleInfo, len(roles))
	for i, r := range roles {
		list[i] = &pb.RoleInfo{
			Id:          uint64(r.ID),
			RoleKey:     r.RoleKey,
			Name:        r.Name,
			Description: r.Description,
			IsSystem:    r.IsSystem == 1,
			CreatedAt:   r.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   r.UpdatedAt.Format(time.RFC3339),
		}
	}
	return &pb.ListRolesResponse{Code: errs.OK, Msg: "success", List: list}, nil
}

func (s *AdminService) ListPermissions(ctx context.Context, req *pb.ListPermissionsRequest) (*pb.ListPermissionsResponse, error) {
	if s.authz == nil {
		return &pb.ListPermissionsResponse{Code: errs.ErrInternal, Msg: "authz repo not configured"}, nil
	}
	if err := s.verifyAdminPermission(ctx, authz.PermAdminRoleManage); err != nil {
		return &pb.ListPermissionsResponse{Code: errs.ErrForbidden, Msg: err.Error()}, nil
	}
	perms, err := s.authz.ListPermissions(ctx)
	if err != nil {
		return nil, err
	}
	list := make([]*pb.PermissionInfo, len(perms))
	for i, p := range perms {
		list[i] = &pb.PermissionInfo{
			Id:            uint64(p.ID),
			PermissionKey: p.PermissionKey,
			Resource:      p.Resource,
			Action:        p.Action,
			Description:   p.Description,
		}
	}
	return &pb.ListPermissionsResponse{Code: errs.OK, Msg: "success", List: list}, nil
}

func (s *AdminService) GetUserRoles(ctx context.Context, req *pb.GetUserRolesRequest) (*pb.GetUserRolesResponse, error) {
	if s.authz == nil {
		return &pb.GetUserRolesResponse{Code: errs.ErrInternal, Msg: "authz repo not configured"}, nil
	}
	if err := s.verifyAdminPermission(ctx, authz.PermAdminRoleManage); err != nil {
		return &pb.GetUserRolesResponse{Code: errs.ErrForbidden, Msg: err.Error()}, nil
	}
	roleKeys, err := s.authz.GetUserRoles(ctx, uint64(req.UserId))
	if err != nil {
		return nil, err
	}
	permKeys, err := s.authz.GetUserPermissions(ctx, uint64(req.UserId))
	if err != nil {
		return nil, err
	}
	scopes, err := s.authz.GetUserDataScopes(ctx, uint64(req.UserId))
	if err != nil {
		return nil, err
	}
	scopeInfos := make([]*pb.DataScopeInfo, 0, len(scopes))
	for _, ds := range scopes {
		scopeInfos = append(scopeInfos, &pb.DataScopeInfo{
			Id:           uint64(ds.ID),
			ScopeKey:     ds.ScopeKey,
			ResourceType: ds.ResourceType,
			ResourceId:   ds.ResourceID,
			AssignedAt:   ds.AssignedAt.Format(time.RFC3339),
		})
	}
	return &pb.GetUserRolesResponse{
		Code:           errs.OK,
		Msg:            "success",
		RoleKeys:       roleKeys,
		PermissionKeys: permKeys,
		DataScopes:     scopeInfos,
	}, nil
}

func (s *AdminService) AssignUserRole(ctx context.Context, req *pb.AssignUserRoleRequest) (*pb.CommonResponse, error) {
	if s.authz == nil {
		return &pb.CommonResponse{Code: errs.ErrInternal, Msg: "authz repo not configured"}, nil
	}
	role, err := s.authz.GetRoleByKey(ctx, req.RoleKey)
	if err != nil || role == nil {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "角色不存在"}, nil
	}
	adminID := uint64(req.AdminId)
	if err := s.authz.AssignRole(ctx, uint64(req.UserId), uint64(role.ID), &adminID); err != nil {
		logger.L().Error("assign role failed", zap.Error(err))
		return nil, err
	}
	// Invalidate stale tokens for the affected user and sync to Redis.
	newVersion, err := s.authz.IncrementTokenVersion(ctx, uint64(req.UserId))
	if err != nil {
		logger.L().Error("increment token version failed after role assign, permission change may not be enforced",
			zap.Error(err), zap.Int64("user_id", req.UserId))
		return &pb.CommonResponse{Code: errs.ErrInternal, Msg: "权限变更成功但令牌同步失败，请重试"}, nil
	}
	if err := s.setTokenVersionCache(ctx, uint64(req.UserId), newVersion); err != nil {
		logger.L().Error("token_version redis sync failed after role assign, rejecting mutation", zap.Error(err))
		return &pb.CommonResponse{Code: errs.ErrInternal, Msg: "权限变更成功但令牌同步失败，请重试"}, nil
	}
	s.auditAdminAction(ctx, uint64(req.AdminId), "assign_role", uint64(req.UserId), "allowed",
		"assigned role "+req.RoleKey, "", "")
	logger.L().Info("user role assigned",
		zap.Int64("user_id", req.UserId),
		zap.Int64("admin_id", req.AdminId),
		zap.String("role_key", req.RoleKey),
	)
	return &pb.CommonResponse{Code: errs.OK, Msg: "success"}, nil
}

func (s *AdminService) RevokeUserRole(ctx context.Context, req *pb.RevokeUserRoleRequest) (*pb.CommonResponse, error) {
	if s.authz == nil {
		return &pb.CommonResponse{Code: errs.ErrInternal, Msg: "authz repo not configured"}, nil
	}
	role, err := s.authz.GetRoleByKey(ctx, req.RoleKey)
	if err != nil || role == nil {
		return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "角色不存在"}, nil
	}
	adminID := uint64(req.AdminId)

	// Self-revoke safety: if admin is removing their own system_admin and they have no
	// other admin role, block it (UX guardrail — they should let another admin do it).
	// Done before the DB write so we don't have to roll back.
	if role.RoleKey == authz.RoleSystemAdmin && uint64(req.UserId) == adminID {
		principal, perr := s.authz.LoadPrincipal(ctx, adminID)
		if perr != nil {
			logger.L().Error("cannot verify admin role safety, blocking self-revoke",
				zap.Error(perr), zap.Int64("user_id", req.UserId))
			return &pb.CommonResponse{Code: errs.ErrInternal, Msg: "无法验证管理员角色，请稍后重试"}, nil
		}
		if principal != nil && !principal.HasRole(authz.RoleRecruitingAdmin) {
			s.auditAdminAction(ctx, adminID, "revoke_role", uint64(req.UserId), "denied",
				"self-revoke of own system_admin blocked", "", "")
			return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: "不能移除自己的系统管理员角色，请让其他系统管理员操作"}, nil
		}
	}

	// Atomic revoke + last-admin guard. The repo does the COUNT and UPDATE in a
	// single SQL statement so concurrent revokes cannot both pass count=2.
	revoked, err := s.authz.RevokeRoleWithLastAdminGuard(ctx, uint64(req.UserId), uint64(role.ID), role.RoleKey, &adminID)
	if err != nil {
		if errors.Is(err, repository.ErrLastAdmin) {
			s.auditAdminAction(ctx, adminID, "revoke_role", uint64(req.UserId), "denied",
				"last system_admin revoke blocked", "", "")
			return &pb.CommonResponse{Code: errs.ErrForbidden, Msg: "无法移除最后一个系统管理员，至少需要保留一个系统管理员"}, nil
		}
		if errors.Is(err, repository.ErrUserRoleNotFound) {
			return &pb.CommonResponse{Code: errs.ErrBadRequest, Msg: "该用户未持有此角色"}, nil
		}
		logger.L().Error("revoke role failed", zap.Error(err))
		return nil, err
	}
	if !revoked {
		// Shouldn't happen — repo returns explicit errors when revoked is false.
		return &pb.CommonResponse{Code: errs.ErrInternal, Msg: "撤销角色失败，请稍后重试"}, nil
	}

	// Bump token version + sync Redis. On failure, roll back the revoke so the
	// DB and the in-flight token state stay consistent.
	newVersion, err := s.authz.IncrementTokenVersion(ctx, uint64(req.UserId))
	if err != nil {
		logger.L().Error("increment token version failed after role revoke, rolling back",
			zap.Error(err), zap.Int64("user_id", req.UserId))
		if rerr := s.authz.RestoreRole(ctx, uint64(req.UserId), uint64(role.ID)); rerr != nil {
			logger.L().Error("rollback restore role failed; DB now inconsistent",
				zap.Error(rerr), zap.Int64("user_id", req.UserId), zap.String("role_key", role.RoleKey))
		}
		return &pb.CommonResponse{Code: errs.ErrInternal, Msg: "权限变更失败，请稍后重试"}, nil
	}
	if err := s.setTokenVersionCache(ctx, uint64(req.UserId), newVersion); err != nil {
		logger.L().Error("token_version redis sync failed after role revoke, rolling back",
			zap.Error(err), zap.Int64("user_id", req.UserId))
		if rerr := s.authz.RestoreRole(ctx, uint64(req.UserId), uint64(role.ID)); rerr != nil {
			logger.L().Error("rollback restore role failed; DB now inconsistent",
				zap.Error(rerr), zap.Int64("user_id", req.UserId), zap.String("role_key", role.RoleKey))
		}
		return &pb.CommonResponse{Code: errs.ErrInternal, Msg: "权限变更失败，请稍后重试"}, nil
	}

	s.auditAdminAction(ctx, uint64(req.AdminId), "revoke_role", uint64(req.UserId), "allowed",
		"revoked role "+req.RoleKey, "", "")
	logger.L().Info("user role revoked",
		zap.Int64("user_id", req.UserId),
		zap.Int64("admin_id", req.AdminId),
		zap.String("role_key", req.RoleKey),
	)
	return &pb.CommonResponse{Code: errs.OK, Msg: "success"}, nil
}

func (s *AdminService) AssignDataScope(ctx context.Context, req *pb.AssignDataScopeRequest) (*pb.CommonResponse, error) {
	if s.authz == nil {
		return &pb.CommonResponse{Code: errs.ErrInternal, Msg: "authz repo not configured"}, nil
	}
	adminID := uint64(req.AdminId)
	if err := s.authz.AssignDataScope(ctx, uint64(req.UserId), req.ScopeKey, req.ResourceType, req.ResourceId, &adminID); err != nil {
		logger.L().Error("assign data scope failed", zap.Error(err))
		return nil, err
	}
	newVersion, err := s.authz.IncrementTokenVersion(ctx, uint64(req.UserId))
	if err != nil {
		logger.L().Error("increment token version failed after scope assign, permission change may not be enforced",
			zap.Error(err), zap.Int64("user_id", req.UserId))
		return &pb.CommonResponse{Code: errs.ErrInternal, Msg: "权限变更成功但令牌同步失败，请重试"}, nil
	}
	if err := s.setTokenVersionCache(ctx, uint64(req.UserId), newVersion); err != nil {
		logger.L().Error("token_version redis sync failed after scope assign, rejecting mutation", zap.Error(err))
		return &pb.CommonResponse{Code: errs.ErrInternal, Msg: "权限变更成功但令牌同步失败，请重试"}, nil
	}
	s.auditAdminAction(ctx, uint64(req.AdminId), "assign_scope", uint64(req.UserId), "allowed",
		"assigned scope "+req.ScopeKey, "", "")
	logger.L().Info("data scope assigned",
		zap.Int64("user_id", req.UserId),
		zap.String("scope_key", req.ScopeKey),
	)
	return &pb.CommonResponse{Code: errs.OK, Msg: "success"}, nil
}

func (s *AdminService) RevokeDataScope(ctx context.Context, req *pb.RevokeDataScopeRequest) (*pb.CommonResponse, error) {
	if s.authz == nil {
		return &pb.CommonResponse{Code: errs.ErrInternal, Msg: "authz repo not configured"}, nil
	}
	// Look up the scope owner before revoking.
	scopeUserID, err := s.authz.GetScopeOwnerID(ctx, req.ScopeId)
	if err != nil {
		logger.L().Error("lookup scope owner failed", zap.Error(err))
		return nil, err
	}
	if err := s.authz.RevokeDataScope(ctx, req.ScopeId); err != nil {
		logger.L().Error("revoke data scope failed", zap.Error(err))
		return nil, err
	}
	if scopeUserID > 0 {
		newVersion, err := s.authz.IncrementTokenVersion(ctx, scopeUserID)
		if err != nil {
			logger.L().Error("increment token version failed after scope revoke, permission change may not be enforced",
				zap.Error(err), zap.Uint64("user_id", scopeUserID))
			return &pb.CommonResponse{Code: errs.ErrInternal, Msg: "权限变更成功但令牌同步失败，请重试"}, nil
		}
		if err := s.setTokenVersionCache(ctx, scopeUserID, newVersion); err != nil {
			logger.L().Error("token_version redis sync failed after scope revoke, rejecting mutation", zap.Error(err))
			return &pb.CommonResponse{Code: errs.ErrInternal, Msg: "权限变更成功但令牌同步失败，请重试"}, nil
		}
	}
	s.auditAdminAction(ctx, uint64(req.AdminId), "revoke_scope", scopeUserID, "allowed",
		"revoked scope id "+formatUint(req.ScopeId), "", "")
	return &pb.CommonResponse{Code: errs.OK, Msg: "success"}, nil
}

func (s *AdminService) ListStaffUsers(ctx context.Context, req *pb.ListStaffUsersRequest) (*pb.ListStaffUsersResponse, error) {
	if s.users == nil {
		return &pb.ListStaffUsersResponse{Code: errs.ErrInternal, Msg: "user repo not configured"}, nil
	}
	if err := s.verifyAdminPermission(ctx, authz.PermAdminUserManage); err != nil {
		return &pb.ListStaffUsersResponse{Code: errs.ErrForbidden, Msg: err.Error()}, nil
	}
	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	users, total, err := s.users.ListStaff(ctx, page, pageSize, req.Status)
	if err != nil {
		return nil, err
	}
	list := make([]*pb.StaffUserInfo, len(users))
	for i, u := range users {
		roleKeys, _ := s.authz.GetUserRoles(ctx, uint64(u.ID))
		if roleKeys == nil {
			roleKeys = []string{}
		}
		list[i] = &pb.StaffUserInfo{
			UserId:      int64(u.ID),
			Username:    u.Username,
			Email:       u.Email,
			Status:      u.Status,
			AccountType: u.AccountType,
			Roles:       roleKeys,
			TokenVersion: u.TokenVersion,
			CreatedAt:   u.CreatedAt.Format(time.RFC3339),
		}
	}
	return &pb.ListStaffUsersResponse{Code: errs.OK, Msg: "success", Total: total, List: list}, nil
}

func (s *AdminService) CreateStaffUser(ctx context.Context, req *pb.CreateStaffUserRequest) (*pb.CreateStaffUserResponse, error) {
	if s.users == nil {
		return &pb.CreateStaffUserResponse{Code: errs.ErrInternal, Msg: "user repo not configured"}, nil
	}
	// Validate password — use the same complexity rules as registration.
	if err := validatePassword(req.Password); err != nil {
		return &pb.CreateStaffUserResponse{Code: errs.ErrBadRequest, Msg: err.Error()}, nil
	}
	// Check username uniqueness
	existing, err := s.users.GetByUsername(ctx, req.Username)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return &pb.CreateStaffUserResponse{Code: errs.ErrBadRequest, Msg: "用户名已存在"}, nil
	}
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user := &model.User{
		Username:    req.Username,
		Password:    string(hashedBytes),
		Email:       req.Email,
		Role:        3, // deprecated legacy role
		AccountType: "staff",
		Status:      "active",
		TokenVersion: 1,
	}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}
	// Assign RBAC roles
	if s.authz != nil && len(req.RoleKeys) > 0 {
		adminID := uint64(req.AdminId)
		for _, rk := range req.RoleKeys {
			role, err := s.authz.GetRoleByKey(ctx, rk)
			if err != nil || role == nil {
				logger.L().Warn("role not found during staff creation", zap.String("role_key", rk))
				continue
			}
			if err := s.authz.AssignRole(ctx, uint64(user.ID), uint64(role.ID), &adminID); err != nil {
				logger.L().Error("assign role failed during staff creation",
					zap.Error(err), zap.String("role_key", rk))
			}
		}
	}
	s.auditAdminAction(ctx, uint64(req.AdminId), "create_staff", uint64(user.ID), "allowed",
		"created staff user "+req.Username, "", "")
	logger.L().Info("staff user created",
		zap.Int64("user_id", user.ID),
		zap.String("username", req.Username),
	)
	return &pb.CreateStaffUserResponse{Code: errs.OK, Msg: "success", UserId: int64(user.ID)}, nil
}

func formatUint(v uint64) string {
	if v == 0 {
		return "0"
	}
	return fmt.Sprintf("%d", v)
}

func toPBUsageLogItem(l *model.ThirdPartyUsageLog) *pb.UsageLogItem {
	return &pb.UsageLogItem{
		Id:              l.ID,
		UserId:          l.UserID,
		Role:            l.Role,
		ServiceType:     l.ServiceType,
		Endpoint:        l.Endpoint,
		Provider:        l.Provider,
		Model:           l.Model,
		RequestChars:    int32(l.RequestChars),
		ResponseChars:   int32(l.ResponseChars),
		EstimatedTokens: int32(l.EstimatedTokens),
		ObjectKey:       l.ObjectKey,
		ObjectSize:      l.ObjectSize,
		Status:          l.Status,
		ErrorCode:       l.ErrorCode,
		CostMs:          int32(l.CostMs),
		RequestId:       l.RequestID,
		Ip:              l.IP,
		CreatedAt:       l.CreatedAt.Format(time.RFC3339),
	}
}
