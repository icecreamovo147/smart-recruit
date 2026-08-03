package hr

import (
	"testing"

	"smart-recruit-proto/recruitment/pb"
)

func TestAvailableCapabilitiesKeepsAvailableRuntimeCapabilities(t *testing.T) {
	items := []*pb.CapabilityInfo{
		{Source: "builtin", Key: "search_jobs", IsAvailable: true},
		{Source: "mcp", Key: "7:search", IsAvailable: true},
		{Source: "skill", Key: "legacy", IsAvailable: false},
		nil,
	}

	got := availableCapabilities(items)
	if len(got) != 2 || got[0].GetKey() != "search_jobs" || got[1].GetKey() != "7:search" {
		t.Fatalf("available capabilities = %#v, want builtin and MCP entries", got)
	}
}
