// Package infrastructure defines the Analytics infrastructure layer skeleton.
//
// Responsibilities:
//   - Adapt read-model storage, projection checkpoints, backfill inputs, and reporting caches.
//   - Avoid final direct repository dependencies on source-domain transactional tables.
//
// TASK-BDME-009 intentionally adds no runtime behavior.
package infrastructure
