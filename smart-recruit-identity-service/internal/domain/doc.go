// Package domain contains Identity business concepts and invariants.
//
// The domain layer must stay free of transport, persistence, cache, and runtime
// dependencies. Later tasks will move only behavior that can be expressed as
// Identity-owned auth, principal, RBAC, data-scope, invite, staff-user, and
// audit rules.
package domain
