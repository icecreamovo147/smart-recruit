package profile

import (
	"strings"
	"testing"
)

func TestEvaluateResumeFillRefresh(t *testing.T) {
	t.Parallel()
	hash := HashResumeParsedText("hello resume")
	cases := []struct {
		name   string
		input  ResumeFillRefreshInput
		need   bool
		reason string
	}{
		{name: "no resume", input: ResumeFillRefreshInput{}, need: false, reason: RefreshReasonNoResume},
		{
			name:   "no parsed text",
			input:  ResumeFillRefreshInput{HasResume: true, ParsedText: "  "},
			need:   false,
			reason: RefreshReasonNoParsedText,
		},
		{
			name: "force",
			input: ResumeFillRefreshInput{
				HasResume: true, ParsedText: "x", ForceRefresh: true, HasProfile: true, StoredInputHash: hash,
			},
			need: true, reason: RefreshReasonForced,
		},
		{
			name:   "missing profile",
			input:  ResumeFillRefreshInput{HasResume: true, ParsedText: "x"},
			need:   true,
			reason: RefreshReasonMissing,
		},
		{
			name: "heuristic",
			input: ResumeFillRefreshInput{
				HasResume: true, ParsedText: "hello resume", HasProfile: true,
				ParserVersion: "resume-profile-heuristic-v1", StoredInputHash: hash,
			},
			need: true, reason: RefreshReasonHeuristic,
		},
		{
			name: "input changed",
			input: ResumeFillRefreshInput{
				HasResume: true, ParsedText: "hello resume", HasProfile: true,
				ParserVersion: "resume-profile-llm-v1", StoredInputHash: "deadbeef",
			},
			need: true, reason: RefreshReasonInputChanged,
		},
		{
			name: "reuse",
			input: ResumeFillRefreshInput{
				HasResume: true, ParsedText: "hello resume", HasProfile: true,
				ParserVersion: "resume-profile-llm-v1", StoredInputHash: hash,
			},
			need: false, reason: RefreshReasonReused,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			need, reason := EvaluateResumeFillRefresh(tc.input)
			if need != tc.need || reason != tc.reason {
				t.Fatalf("got (%v, %q), want (%v, %q)", need, reason, tc.need, tc.reason)
			}
		})
	}
}

func TestNormalizeDegreePhoneRegion(t *testing.T) {
	t.Parallel()
	if got, matched, empty := NormalizeDegree("Master of Science"); empty || !matched || got != "硕士" {
		t.Fatalf("degree = %q matched=%v empty=%v", got, matched, empty)
	}
	if phone, ok := NormalizePhone("13800138000"); !ok || phone != "13800138000" {
		t.Fatalf("phone = %q ok=%v", phone, ok)
	}
	if phone, ok := NormalizePhone("not-a-phone"); ok || phone != "" {
		t.Fatalf("invalid phone should fail")
	}
	if region, ok := NormalizeRegionForProfile("北京市/朝阳区"); !ok || region != "北京市/朝阳区" {
		t.Fatalf("municipality path = %q ok=%v", region, ok)
	}
	if region, ok := NormalizeRegionForProfile("广东省深圳市南山区"); !ok || region != "广东省/深圳市/南山区" {
		t.Fatalf("cn text = %q ok=%v", region, ok)
	}
	if region, ok := NormalizeRegionForProfile("湖北省黄冈市"); !ok || region != "湖北省/黄冈市" {
		t.Fatalf("province+city = %q ok=%v", region, ok)
	}
	if IsRegionPathComplete("湖北省/黄冈市") {
		t.Fatal("province+city should be incomplete without district")
	}
	if !IsRegionPathComplete("广东省/深圳市/南山区") {
		t.Fatal("full region should be complete")
	}
	if _, ok := NormalizeRegionForProfile("深圳"); ok {
		t.Fatal("incomplete city should not normalize")
	}
}

func TestJoinAchievementsIntoDescription(t *testing.T) {
	t.Parallel()
	got := JoinAchievementsIntoDescription("负责后端", `["提升性能","完成迁移"]`)
	for _, part := range []string{"负责后端", "提升性能", "完成迁移"} {
		if !strings.Contains(got, part) {
			t.Fatalf("description %q missing %q", got, part)
		}
	}
}

func TestMergeEducationsEnrichesAndAppends(t *testing.T) {
	t.Parallel()
	existing := []EducationInput{{
		School: "中国地质大学（武汉）", Degree: "硕士", SortOrder: 0,
	}}
	draft := []EducationInput{
		{
			School: "中国地质大学(武汉)", Degree: "硕士", Major: "电子信息",
			StartDate: "2024-09-01", EndDate: "2027-06-01",
		},
		{
			School: "三峡大学", Degree: "本科", Major: "计算机科学与技术",
			StartDate: "2020-09-01", EndDate: "2024-06-01",
		},
	}
	merged := MergeEducations(existing, draft, false)
	if len(merged) != 2 {
		t.Fatalf("len = %d, want 2: %+v", len(merged), merged)
	}
	if merged[0].Major != "电子信息" || merged[0].StartDate != "2024-09-01" || merged[0].EndDate != "2027-06-01" {
		t.Fatalf("enriched first = %+v", merged[0])
	}
	if merged[1].School != "三峡大学" || merged[1].Degree != "本科" || merged[1].Major != "计算机科学与技术" {
		t.Fatalf("appended second = %+v", merged[1])
	}

	overwritten := MergeEducations(existing, draft, true)
	if len(overwritten) != 2 || overwritten[0].School != "中国地质大学(武汉)" {
		t.Fatalf("overwrite = %+v", overwritten)
	}
}

func TestBuildProfileFillDiffsEducationsEnrich(t *testing.T) {
	t.Parallel()
	bundle := Bundle{
		Educations: []EducationInput{{School: "三峡大学", Degree: "本科"}},
	}
	draft := ProfileFillDraft{
		Educations: []EducationInput{
			{School: "三峡大学", Degree: "本科", Major: "计算机", StartDate: "2020-09-01", EndDate: "2024-06-01"},
			{School: "中国地质大学（武汉）", Degree: "硕士", Major: "电子信息", StartDate: "2024-09-01", EndDate: "2027-06-01"},
		},
	}
	diffs := BuildProfileFillDiffs(bundle, draft, false, 0)
	assertDiffAction(t, diffs, "educations", FillActionFill)
}

func assertDiffAction(t *testing.T, diffs []ProfileFillFieldDiff, field, action string) {
	t.Helper()
	for _, diff := range diffs {
		if diff.Field == field {
			if diff.Action != action {
				t.Fatalf("field %s action = %q, want %q", field, diff.Action, action)
			}
			return
		}
	}
	t.Fatalf("field %s not found in diffs", field)
}
