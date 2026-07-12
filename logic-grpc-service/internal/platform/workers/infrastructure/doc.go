// Package infrastructure defines the Platform Worker infrastructure layer skeleton.
//
// Responsibilities:
//   - Adapt RabbitMQ, Outbox/Inbox persistence, retry stores, health checks, logging, and operational clients.
//   - Keep queue and process details behind worker application ports.
//
// TASK-BDME-009 intentionally adds no runtime behavior.
package infrastructure
