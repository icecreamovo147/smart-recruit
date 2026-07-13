// Package grpc will hold Identity protobuf service adapters.
//
// The existing runtime facade remains active during TASK-012. Later migration
// tasks must keep AuthService and the Identity-owned AdminService subset
// behavior compatible when gRPC adapters move into this package.
package grpc
