// Package model defines Identity domain models.
//
// Domain models must not be aliases for protobuf messages or GORM records.
// Later migration tasks will introduce explicit models for users, sessions,
// refresh-token families, principals, roles, permissions, scopes, invite codes,
// and audit records only when behavior is moved into this service.
package model
