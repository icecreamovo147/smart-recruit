package provider

// Package provider will adapt AI and embedding provider SDKs. It must protect
// provider credentials, timeouts, circuit breakers, retry budgets, and fallback
// policy from leaking into domain logic.
