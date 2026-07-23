// Package port defines outbound Recruitment application dependencies.
//
// Cross-context dependencies such as identity authorization, interview/offer
// snapshots, notification outbox, OSS, and usage logging must be represented as
// ports before concrete adapters are introduced.
package port
