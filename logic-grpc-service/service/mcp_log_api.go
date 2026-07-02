package service

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"logic-grpc-service/model"
	"logic-grpc-service/pkg/logger"
	"logic-grpc-service/recruitment/pb"
)

func (s *MCPService) ListMCPToolLogs(ctx context.Context, req *pb.ListMCPToolLogsRequest) (*pb.ListMCPToolLogsResponse, error) {
	serverID := req.GetServerId()
	if serverID <= 0 {
		return nil, status.Error(codes.InvalidArgument, "server_id is required")
	}
	page := req.GetPage()
	if page <= 0 {
		page = 1
	}
	pageSize := req.GetPageSize()
	if pageSize <= 0 {
		pageSize = 20
	}
	logs, total, err := s.mcpRepo.ListToolLogsByServer(ctx, serverID, page, pageSize)
	if err != nil {
		logger.L().Error("list MCP tool logs failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "list MCP tool logs failed")
	}
	list := make([]*pb.MCPToolLogInfo, 0, len(logs))
	for i := range logs {
		list = append(list, mcpToolLogToInfo(&logs[i]))
	}
	return &pb.ListMCPToolLogsResponse{Code: 0, Msg: "ok", Total: total, List: list}, nil
}

func mcpToolLogToInfo(log *model.MCPToolLog) *pb.MCPToolLogInfo {
	if log == nil {
		return nil
	}
	return &pb.MCPToolLogInfo{
		Id:             log.ID,
		ServerId:       log.ServerID,
		ToolName:       log.ToolName,
		ArgsJson:       valueOrEmpty(log.ArgsJSON),
		ResultContent:  valueOrEmpty(log.ResultContent),
		DurationMs:     log.DurationMs,
		ErrorMsg:       valueOrEmpty(log.ErrorMsg),
		CalledByHrId:   int64Value(log.CalledByHRID),
		SessionId:      int64Value(log.SessionID),
		PolicyId:       int64Value(log.PolicyID),
		PolicyDecision: log.PolicyDecision,
		PolicyReason:   valueOrEmpty(log.PolicyReason),
		CreatedAt:      log.CreatedAt.Format(time.RFC3339),
	}
}
