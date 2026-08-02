package profile

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	FillActionFill        = "fill"
	FillActionOverwrite   = "overwrite"
	FillActionSkip        = "skip"
	FillActionUnsupported = "unsupported"
)

// ProfileFillDraft is the transport-independent profile data proposed by resume parsing.
type ProfileFillDraft struct {
	RealName          string
	Phone             string
	City              string
	ExpectedPosition  string
	Summary           string
	YearsOfExperience float64
	Skills            []string
	Educations        []EducationInput
	Experiences       []ExperienceInput
}

// ProfileFillFieldDiff describes one merge decision without binding the domain
// package to a transport schema.
type ProfileFillFieldDiff struct {
	Field  string
	Label  string
	Action string
	Before string
	After  string
}

// BuildProfileFillDiffs compares existing profile with draft under merge rules.
func BuildProfileFillDiffs(existing Bundle, draft ProfileFillDraft, overwrite bool, projectCount int) []ProfileFillFieldDiff {
	diffs := make([]ProfileFillFieldDiff, 0, 16)
	addString := func(field, label, before, after string) {
		after = strings.TrimSpace(after)
		before = strings.TrimSpace(before)
		if after == "" {
			return
		}
		action := FillActionSkip
		switch {
		case before == "":
			action = FillActionFill
		case field == "city" && !IsRegionPathComplete(before) && after != before:
			action = FillActionFill
		case overwrite && before != after:
			action = FillActionOverwrite
		case before == after:
			action = FillActionSkip
		default:
			action = FillActionSkip
		}
		diffs = append(diffs, ProfileFillFieldDiff{
			Field: field, Label: label, Action: action, Before: before, After: after,
		})
	}

	addString("real_name", "真实姓名", existing.Profile.RealName, draft.RealName)
	addString("phone", "联系电话", existing.Profile.Phone, draft.Phone)
	addString("city", "所在城市", existing.Profile.City, draft.City)
	addString("expected_position", "期望岗位", existing.Profile.ExpectedPosition, draft.ExpectedPosition)
	addString("summary", "个人简介", existing.Profile.Summary, draft.Summary)

	if draft.YearsOfExperience > 0 {
		before := ""
		if existing.Profile.YearsOfExperience > 0 {
			before = trimFloat(existing.Profile.YearsOfExperience)
		}
		after := trimFloat(draft.YearsOfExperience)
		action := FillActionSkip
		if existing.Profile.YearsOfExperience <= 0 {
			action = FillActionFill
		} else if overwrite && before != after {
			action = FillActionOverwrite
		}
		diffs = append(diffs, ProfileFillFieldDiff{
			Field: "years_of_experience", Label: "工作年限", Action: action, Before: before, After: after,
		})
	}

	skillsAfter := strings.Join(draft.Skills, ",")
	addString("skills", "核心技能", existing.Profile.Skills, skillsAfter)

	eduMerged := MergeEducations(existing.Educations, draft.Educations, overwrite)
	eduBefore := summarizeEducations(existing.Educations)
	eduAfter := summarizeEducations(eduMerged)
	if eduAfter != "" && eduAfter != eduBefore {
		action := FillActionFill
		if overwrite && len(existing.Educations) > 0 {
			action = FillActionOverwrite
		}
		diffs = append(diffs, ProfileFillFieldDiff{
			Field: "educations", Label: "教育经历", Action: action, Before: eduBefore, After: eduAfter,
		})
	}

	expMerged := MergeExperiences(existing.Experiences, draft.Experiences, overwrite)
	expBefore := summarizeExperiences(existing.Experiences)
	expAfter := summarizeExperiences(expMerged)
	if expAfter != "" && expAfter != expBefore {
		action := FillActionFill
		if overwrite && len(existing.Experiences) > 0 {
			action = FillActionOverwrite
		}
		diffs = append(diffs, ProfileFillFieldDiff{
			Field: "experiences", Label: "工作经历", Action: action, Before: expBefore, After: expAfter,
		})
	}

	if projectCount > 0 {
		diffs = append(diffs, ProfileFillFieldDiff{
			Field:  "projects",
			Label:  "项目经历",
			Action: FillActionUnsupported,
			Before: "",
			After:  fmt.Sprintf("简历含 %d 个项目，暂不写入个人资料", projectCount),
		})
	}
	return diffs
}

func summarizeEducations(items []EducationInput) string {
	if len(items) == 0 {
		return ""
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		chunks := make([]string, 0, 4)
		if school := strings.TrimSpace(item.School); school != "" {
			chunks = append(chunks, school)
		}
		if degree := strings.TrimSpace(item.Degree); degree != "" {
			chunks = append(chunks, degree)
		}
		if major := strings.TrimSpace(item.Major); major != "" {
			chunks = append(chunks, major)
		}
		if period := formatDateRange(item.StartDate, item.EndDate, false); period != "" {
			chunks = append(chunks, period)
		}
		if len(chunks) > 0 {
			parts = append(parts, strings.Join(chunks, " · "))
		}
	}
	return strings.Join(parts, "；")
}

func summarizeExperiences(items []ExperienceInput) string {
	if len(items) == 0 {
		return ""
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		chunks := make([]string, 0, 3)
		if company := strings.TrimSpace(item.Company); company != "" {
			chunks = append(chunks, company)
		}
		if title := strings.TrimSpace(item.Title); title != "" {
			chunks = append(chunks, title)
		}
		if period := formatDateRange(item.StartDate, item.EndDate, item.IsCurrent == 1); period != "" {
			chunks = append(chunks, period)
		}
		if len(chunks) > 0 {
			parts = append(parts, strings.Join(chunks, " · "))
		}
	}
	return strings.Join(parts, "；")
}

func trimFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
