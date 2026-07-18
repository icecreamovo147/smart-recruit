// Package repository defines Identity repository ports.
//
// Concrete persistence stays in infrastructure. Repository ports here must
// model Identity-owned storage contracts for users, refresh tokens, RBAC,
// data scopes, invite codes, and authorization audit records.
package repository
