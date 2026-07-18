package grpc

import (
	"testing"

	"smart-recruit-proto/recruitment/pb"
)

func TestBuiltinCapabilitiesUseRuntimeToolCatalogs(t *testing.T) {
	hr := builtinCapabilities("hr_recruiting_agent")
	if len(hr) != 19 {
		t.Fatalf("hr capabilities count = %d, want 19", len(hr))
	}
	assertCapabilityKeys(t, hr,
		"query_total_applications",
		"search_candidates",
		"parse_resume_profile",
		"evaluate_candidate_match",
		"compare_candidates_for_job",
	)
	assertCapabilityKeyAbsent(t, hr, "candidate_search")
	assertCapabilityKeyAbsent(t, hr, "resume_intelligence")
	assertCapabilityKeyAbsent(t, hr, "interview_context")

	candidate := builtinCapabilities("candidate_assistant")
	if len(candidate) != 6 {
		t.Fatalf("candidate capabilities count = %d, want 6", len(candidate))
	}
	assertCapabilityKeys(t, candidate,
		"list_my_applications",
		"get_my_application_detail",
		"get_my_resume_text",
		"list_jobs_for_recommendation",
		"get_job_detail_for_candidate",
		"recommend_jobs_by_resume",
	)
}

func assertCapabilityKeys(t *testing.T, caps []*pb.CapabilityInfo, keys ...string) {
	t.Helper()
	present := capabilityKeySet(caps)
	for _, key := range keys {
		if !present[key] {
			t.Fatalf("capability key %q missing from %v", key, present)
		}
	}
}

func assertCapabilityKeyAbsent(t *testing.T, caps []*pb.CapabilityInfo, key string) {
	t.Helper()
	if capabilityKeySet(caps)[key] {
		t.Fatalf("capability key %q should not be present", key)
	}
}

func capabilityKeySet(caps []*pb.CapabilityInfo) map[string]bool {
	out := make(map[string]bool, len(caps))
	for _, cap := range caps {
		out[cap.GetKey()] = true
	}
	return out
}
