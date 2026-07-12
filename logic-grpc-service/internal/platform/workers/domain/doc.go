// Package domain defines the Platform Worker domain skeleton.
//
// Responsibilities:
//   - Own worker lifecycle concepts, Outbox/Inbox envelopes, idempotency, retry, dead-letter, and heartbeat invariants.
//   - Keep worker operational rules separate from source-domain business decisions.
//
// TASK-BDME-009 intentionally adds no runtime behavior.
package domain
