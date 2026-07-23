// Package application contains Identity use-case orchestration.
//
// TASK-012 only establishes the target package boundary. Auth, refresh-token,
// principal, RBAC, data-scope, invite-code, staff-user, and audit behavior
// still execute through the current runtime adapters until later migration
// tasks move commands, queries, ports, and services into this layer.
package application
