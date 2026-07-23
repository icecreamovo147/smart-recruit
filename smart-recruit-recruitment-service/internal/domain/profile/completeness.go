package profile

import (
	"fmt"
	"sort"
	"strings"

	"smart-recruit-recruitment-service/internal/domain/model"
)

const (
	JobStatusEmployed       = "employed"
	JobStatusResigned       = "resigned"
	JobStatusFreshGraduate  = "fresh_graduate"
	JobStatusStudent        = "student"
)

var degreeRank = map[string]int{
	"博士": 5, "phd": 5, "doctor": 5,
	"硕士": 4, "master": 4, "研究生": 4,
	"本科": 3, "bachelor": 3, "学士": 3,
	"大专": 2, "associate": 2, "专科": 2,
	"高中": 1, "high school": 1,
}

type EducationInput struct {
	School      string
	Degree      string
	Major       string
	StartDate   string
	EndDate     string
	Description string
	SortOrder   int32
}

type ExperienceInput struct {
	Company     string
	Title       string
	Location    string
	StartDate   string
	EndDate     string
	IsCurrent   int32
	Description string
	SortOrder   int32
}

type Bundle struct {
	Profile     model.CandidateProfile
	Educations  []EducationInput
	Experiences []ExperienceInput
}

func IsStudentLike(jobStatus string) bool {
	switch strings.TrimSpace(strings.ToLower(jobStatus)) {
	case JobStatusFreshGraduate, JobStatusStudent:
		return true
	default:
		return false
	}
}

func Complete(bundle *Bundle) {
	if bundle == nil {
		return
	}
	profile := &bundle.Profile
	if !allNotEmpty(profile.RealName, profile.Phone, profile.Skills, profile.City, profile.JobStatus, profile.ExpectedPosition, profile.Summary) {
		profile.IsComplete = 0
		return
	}
	if !IsStudentLike(profile.JobStatus) && profile.YearsOfExperience < 0 {
		profile.IsComplete = 0
		return
	}
	if len(bundle.Educations) == 0 {
		profile.IsComplete = 0
		return
	}
	for _, edu := range bundle.Educations {
		if strings.TrimSpace(edu.School) == "" {
			profile.IsComplete = 0
			return
		}
	}
	// Work experience is optional (interns / students may have none).
	for _, exp := range bundle.Experiences {
		if strings.TrimSpace(exp.Company) == "" {
			profile.IsComplete = 0
			return
		}
	}
	profile.IsComplete = 1
}

func DenormalizeSummary(bundle *Bundle) {
	if bundle == nil {
		return
	}
	profile := &bundle.Profile
	if top := highestEducation(bundle.Educations); top != nil {
		profile.Education = strings.TrimSpace(top.Degree)
		profile.School = strings.TrimSpace(top.School)
	}
	profile.WorkExperience = buildWorkExperienceSummary(bundle.Experiences)
}

func highestEducation(items []EducationInput) *EducationInput {
	if len(items) == 0 {
		return nil
	}
	sorted := append([]EducationInput(nil), items...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return degreeScore(sorted[i].Degree) > degreeScore(sorted[j].Degree)
	})
	return &sorted[0]
}

func degreeScore(degree string) int {
	key := strings.ToLower(strings.TrimSpace(degree))
	if rank, ok := degreeRank[key]; ok {
		return rank
	}
	for label, rank := range degreeRank {
		if strings.Contains(key, strings.ToLower(label)) {
			return rank
		}
	}
	return 0
}

func buildWorkExperienceSummary(items []ExperienceInput) string {
	if len(items) == 0 {
		return ""
	}
	sorted := append([]ExperienceInput(nil), items...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].SortOrder < sorted[j].SortOrder
	})
	parts := make([]string, 0, len(sorted))
	for _, item := range sorted {
		title := strings.TrimSpace(item.Title)
		company := strings.TrimSpace(item.Company)
		if title == "" && company == "" {
			continue
		}
		line := company
		if title != "" {
			if line != "" {
				line += " · "
			}
			line += title
		}
		if date := formatDateRange(item.StartDate, item.EndDate, item.IsCurrent == 1); date != "" {
			line += "（" + date + "）"
		}
		if desc := strings.TrimSpace(item.Description); desc != "" {
			line += "\n" + desc
		}
		parts = append(parts, line)
	}
	return strings.Join(parts, "\n\n")
}

func formatDateRange(start, end string, isCurrent bool) string {
	start = strings.TrimSpace(start)
	end = strings.TrimSpace(end)
	switch {
	case start != "" && end != "":
		return start + " - " + end
	case start != "" && isCurrent:
		return start + " - 至今"
	case start != "":
		return start
	case end != "":
		return end
	default:
		return ""
	}
}

func allNotEmpty(values ...string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return false
		}
	}
	return true
}

func JoinSkills(skills []string) string {
	out := make([]string, 0, len(skills))
	for _, skill := range skills {
		if trimmed := strings.TrimSpace(skill); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return strings.Join(out, ",")
}

func FormatSalaryRange(min, max int32) string {
	switch {
	case min > 0 && max > 0:
		return fmt.Sprintf("%d-%d", min, max)
	case min > 0:
		return fmt.Sprintf("%d+", min)
	case max > 0:
		return fmt.Sprintf("≤%d", max)
	default:
		return ""
	}
}
