// Package domain defines the Notification domain skeleton.
//
// Responsibilities:
//   - Own notification records, unread state, delivery state, and delivery idempotency invariants.
//   - Define notification events without owning source-domain business decisions.
//
// TASK-BDME-009 intentionally adds no runtime behavior.
package domain
