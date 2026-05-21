package service

import (
	"context"
	"crypto/rand"
	"time"

	"go.uber.org/zap"

	"logic-grpc-service/model"
	"logic-grpc-service/pkg/errs"
	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/recruitment/pb"
	"logic-grpc-service/repository"
)

type AdminService struct {
	inviteCodes *repository.InviteCodeRepo
}

func NewAdminService(inviteCodes *repository.InviteCodeRepo) *AdminService {
	return &AdminService{inviteCodes: inviteCodes}
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
		return &pb.ValidateInviteCodeResponse{Code: errs.OK, Msg: "success", Valid: false}, nil
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
