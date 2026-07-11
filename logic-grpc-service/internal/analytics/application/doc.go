// Package application defines the Analytics application layer skeleton.
//
// Responsibilities:
//   - Coordinate reporting queries, projection consumers, replay, backfill, and dashboard use cases.
//   - Expose ports for projection stores and event sources without mutating source domains.
//
// TASK-BDME-009 intentionally adds no runtime behavior.
package application
