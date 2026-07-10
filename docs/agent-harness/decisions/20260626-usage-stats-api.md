# ADR: AI Usage Statistics API (GetUsageStats / GetUsageTrend)

## 日期
2026-06-26

## 决策
在已有 `AdminService` gRPC 中追加 `GetUsageStats` 和 `GetUsageTrend` 两个聚合查询方法，新增 `UsageStatsService` 服务层和 `UsageStatsRepo` 仓库层，前端通过 HTTP 代理调用聚合接口。

## 背景
「第三方服务审计」页面仅支持逐条查看调用日志，无法从全局把握 Token 消耗趋势、模型使用分布和成本估算。为满足缺陷 7（无独立 Token/成本统计与调试面板）的成本统计部分，需要新增多维度聚合查询能力。

## 方案

### Proto 变更
在 `AdminService` 中追加：
- `GetUsageStats(GetUsageStatsRequest) returns (GetUsageStatsResponse)` — 按维度的聚合统计
- `GetUsageTrend(GetUsageTrendRequest) returns (GetUsageTrendResponse)` — 时间序列趋势

### 聚合维度
- **model**: 按模型名聚合（默认），返回带估算花费的分布
- **user**: 按 user_id 聚合，限 Top 10
- **session**: 按 request_id 聚合，限 Top 10

### 趋势粒度
- **day**（默认）: `DATE(created_at)`
- **week**: `DATE_FORMAT(created_at, '%x-W%v')`
- **month**: `DATE_FORMAT(created_at, '%Y-%m')`

### 成本估算
采用统一公式 `SUM(estimated_tokens) * 0.002 / 1000`，未区分模型定价差异（作为已知限制记录）。

### 数据库
复用已有 `third_party_usage_logs` 表，无需 DDL 变更。

### 权限
复用 `PermAuditUsageRead`（`AUDIT_USAGE_READ`）权限，与已有日志列表接口一致。

## 影响范围
- `recruitment.proto`：新增 2 个 RPC + 消息类型
- `logic-grpc-service/repository/usage_stats_repo.go`：新增聚合查询仓库
- `logic-grpc-service/service/usage_stats_service.go`：新增统计查询服务
- `logic-grpc-service/service/services.go`：注册 UsageStatsService
- `logic-grpc-service/server/server.go`：新增 gRPC 委托方法
- `web-gin-service/handler/hr/admin.go`：新增 HTTP handler
- `web-gin-service/router/router.go`：新增路由
- `hr-frontend/src/api/admin.ts`：新增前端 API 调用
- `hr-frontend/src/views/hr/UsageAuditView.vue`：统计卡片、趋势图、维度切换

## 回退方案
- 前端：从 `UsageAuditView.vue` 移除统计面板，保留日志列表
- 后端：移除 HTTP handler 和路由，gRPC 方法向后兼容不影响已有功能
