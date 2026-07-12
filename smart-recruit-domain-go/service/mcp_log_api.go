package service

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"smart-recruit-domain-go/model"
	"smart-recruit-platform-go/logger"
	"smart-recruit-proto/recruitment/pb"
)

func (s *MCPService) ListMCPToolLogs(ctx context.Context, req *pb.ListMCPToolLogsRequest) (*pb.ListMCPToolLogsResponse, error) {
	log := logger.GetRequestLogger(ctx)
	serverID := req.GetServerId()
	log.Info("[domain][mcp] ListMCPToolLogs started", zap.Int64("server_id", serverID))
	if serverID <= 0 {
		log.Warn("[domain][mcp] ListMCPToolLogs invalid server_id")
		return nil, status.Error(codes.InvalidArgument, "server_id is required")
	}
	page, pageSize := normalizeManagementPage(req.GetPage(), req.GetPageSize())
	logs, total, err := s.mcpRepo.ListToolLogsByServer(ctx, serverID, page, pageSize)
	if err != nil {
		log.Error("[domain][mcp] list MCP tool logs failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "list MCP tool logs failed")
	}
	list := make([]*pb.MCPToolLogInfo, 0, len(logs))
	for i := range logs {
		list = append(list, mcpToolLogToInfo(&logs[i]))
	}
	log.Info("[domain][mcp] ListMCPToolLogs succeeded", zap.Int64("server_id", serverID), zap.Int("total", int(total)), zap.Int("returned", len(list)))
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
