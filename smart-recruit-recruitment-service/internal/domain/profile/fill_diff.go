package profile

import (
	"fmt"
	"strconv"
	"strings"

	"smart-recruit-proto/recruitment/pb"
)

const (
	FillActionFill        = "fill"
	FillActionOverwrite   = "overwrite"
	FillActionSkip        = "skip"
	FillActionUnsupported = "unsupported"
)

// BuildProfileFillDiffs compares existing profile with draft under merge rules.
func BuildProfileFillDiffs(existing Bundle, draft *pb.CandidateProfile, overwrite bool, projectCount int) []*pb.ProfileFillFieldDiff {
	if draft == nil {
		draft = &pb.CandidateProfile{}
	}
	diffs := make([]*pb.ProfileFillFieldDiff, 0, 16)
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
		diffs = append(diffs, &pb.ProfileFillFieldDiff{
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
		diffs = append(diffs, &pb.ProfileFillFieldDiff{
			Field: "years_of_experience", Label: "工作年限", Action: action, Before: before, After: after,
		})
	}

	skillsAfter := strings.Join(draft.Skills, ",")
	addString("skills", "核心技能", existing.Profile.Skills, skillsAfter)

	eduDraft := EducationsFromPB(draft.Educations)
	eduMerged := MergeEducations(existing.Educations, eduDraft, overwrite)
	eduBefore := summarizeEducations(existing.Educations)
	eduAfter := summarizeEducations(eduMerged)
	if eduAfter != "" && eduAfter != eduBefore {
		action := FillActionFill
		if overwrite && len(existing.Educations) > 0 {
			action = FillActionOverwrite
		}
		diffs = append(diffs, &pb.ProfileFillFieldDiff{
			Field: "educations", Label: "教育经历", Action: action, Before: eduBefore, After: eduAfter,
		})
	}

	expDraft := ExperiencesFromPB(draft.Experiences)
	expMerged := MergeExperiences(existing.Experiences, expDraft, overwrite)
	expBefore := summarizeExperiences(existing.Experiences)
	expAfter := summarizeExperiences(expMerged)
	if expAfter != "" && expAfter != expBefore {
		action := FillActionFill
		if overwrite && len(existing.Experiences) > 0 {
			action = FillActionOverwrite
		}
		diffs = append(diffs, &pb.ProfileFillFieldDiff{
			Field: "experiences", Label: "工作经历", Action: action, Before: expBefore, After: expAfter,
		})
	}

	if projectCount > 0 {
		diffs = append(diffs, &pb.ProfileFillFieldDiff{
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

func summarizeEducationsPB(items []*pb.CandidateEducationInfo) string {
	return summarizeEducations(EducationsFromPB(items))
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

func summarizeExperiencesPB(items []*pb.CandidateExperienceInfo) string {
	return summarizeExperiences(ExperiencesFromPB(items))
}

// EducationsFromPB converts protobuf education rows into domain inputs.
func EducationsFromPB(items []*pb.CandidateEducationInfo) []EducationInput {
	out := make([]EducationInput, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, EducationInput{
			School: item.School, Degree: item.Degree, Major: item.Major,
			StartDate: item.StartDate, EndDate: item.EndDate, Description: item.Description, SortOrder: item.SortOrder,
		})
	}
	return out
}

// ExperiencesFromPB converts protobuf experience rows into domain inputs.
func ExperiencesFromPB(items []*pb.CandidateExperienceInfo) []ExperienceInput {
	out := make([]ExperienceInput, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, ExperienceInput{
			Company: item.Company, Title: item.Title, Location: item.Location,
			StartDate: item.StartDate, EndDate: item.EndDate, IsCurrent: item.IsCurrent,
			Description: item.Description, SortOrder: item.SortOrder,
		})
	}
	return out
}

func trimFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
