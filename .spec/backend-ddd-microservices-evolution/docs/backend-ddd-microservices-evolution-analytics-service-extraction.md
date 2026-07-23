# Analytics Service Extraction

## Scope

TASK-BDME-044 creates the compile-safe `analytics-service` command and extracts the Analytics-owned reporting surface without introducing transactional service-read adapters. This TASK does not route gateway traffic, change protobuf contracts, change database schema, or alter public HTTP behavior.

## Extracted APIs

The Analytics runtime can explicitly register the Analytics-owned `AdminService` reporting subset:

- `AdminService.GetDashboardReport`
- `AdminService.GetFunnelReport`
- `AdminService.GetTimeInStageReport`
- `AdminService.GetInterviewOfferMetrics`

`AdminService.QueryAuthAuditLogs` remains Identity-owned and is intentionally not claimed by the Analytics runtime.

## Projection Contract

Analytics extraction uses `domain-event-projection-read-models` mode:

- event projection ingestion stays under `internal/analytics/application`;
- projection persistence stays behind Analytics-owned ports and infrastructure adapters;
- reporting queries use the Analytics read-model repository port;
- no `service_read` adapter or transitional transactional service read API is introduced.

## Runtime Behavior

`logic-grpc-service/cmd/analytics-service` supports:

```bash
go run ./cmd/analytics-service --check
go run ./cmd/analytics-service --describe
```

Default execution exits non-zero and does not bind a listener. Runtime registration requires an explicit reporting API dependency and a gRPC registrar. The runtime descriptor keeps `TrafficEnabled=false` and `CutoverMode=none`.

## Compatibility

Current dashboard, funnel, time-in-stage, interview/offer metrics, and audit-log routing remain on the monolith path. This TASK only prepares Analytics runtime extraction and records that the extracted runtime must be backed by event projections/read models.

## Verification

Run:

```bash
cd logic-grpc-service && go test ./internal/analytics/interfaces ./internal/analytics/runtime ./cmd/analytics-service
cd logic-grpc-service && go run ./cmd/analytics-service --check
cd logic-grpc-service && go run ./cmd/analytics-service --describe
cd logic-grpc-service && go test ./...
cd web-gin-service && go test ./...
```
