package profile

import "strings"

// MergeEducations applies resume education drafts onto the candidate profile.
// overwrite replaces the whole list; otherwise matching schools are enriched
// (blank fields only) and unmatched draft schools are appended.
func MergeEducations(existing, draft []EducationInput, overwrite bool) []EducationInput {
	draft = normalizeEducationList(draft)
	if len(draft) == 0 {
		return existing
	}
	if overwrite || len(existing) == 0 {
		return draft
	}
	merged := make([]EducationInput, len(existing))
	copy(merged, existing)
	for _, item := range draft {
		if idx := findEducationIndex(merged, item.School); idx >= 0 {
			merged[idx] = enrichEducation(merged[idx], item)
			continue
		}
		merged = append(merged, item)
	}
	for i := range merged {
		merged[i].SortOrder = int32(i)
	}
	return merged
}

// MergeExperiences applies resume experience drafts onto the candidate profile.
// Matching uses company (+ title when both sides have a title).
func MergeExperiences(existing, draft []ExperienceInput, overwrite bool) []ExperienceInput {
	draft = normalizeExperienceList(draft)
	if len(draft) == 0 {
		return existing
	}
	if overwrite || len(existing) == 0 {
		return draft
	}
	merged := make([]ExperienceInput, len(existing))
	copy(merged, existing)
	for _, item := range draft {
		if idx := findExperienceIndex(merged, item); idx >= 0 {
			merged[idx] = enrichExperience(merged[idx], item)
			continue
		}
		merged = append(merged, item)
	}
	for i := range merged {
		merged[i].SortOrder = int32(i)
	}
	return merged
}

func normalizeEducationList(items []EducationInput) []EducationInput {
	out := make([]EducationInput, 0, len(items))
	for i, item := range items {
		item.School = strings.TrimSpace(item.School)
		item.Degree = strings.TrimSpace(item.Degree)
		item.Major = strings.TrimSpace(item.Major)
		item.StartDate = strings.TrimSpace(item.StartDate)
		item.EndDate = strings.TrimSpace(item.EndDate)
		item.Description = strings.TrimSpace(item.Description)
		if item.School == "" && item.Degree == "" && item.Major == "" {
			continue
		}
		item.SortOrder = int32(i)
		out = append(out, item)
	}
	return out
}

func normalizeExperienceList(items []ExperienceInput) []ExperienceInput {
	out := make([]ExperienceInput, 0, len(items))
	for i, item := range items {
		item.Company = strings.TrimSpace(item.Company)
		item.Title = strings.TrimSpace(item.Title)
		item.Location = strings.TrimSpace(item.Location)
		item.StartDate = strings.TrimSpace(item.StartDate)
		item.EndDate = strings.TrimSpace(item.EndDate)
		item.Description = strings.TrimSpace(item.Description)
		if item.Company == "" && item.Title == "" {
			continue
		}
		item.SortOrder = int32(i)
		out = append(out, item)
	}
	return out
}

func findEducationIndex(items []EducationInput, school string) int {
	key := normalizeMatchKey(school)
	if key == "" {
		return -1
	}
	for i, item := range items {
		if normalizeMatchKey(item.School) == key {
			return i
		}
	}
	return -1
}

func findExperienceIndex(items []ExperienceInput, draft ExperienceInput) int {
	company := normalizeMatchKey(draft.Company)
	if company == "" {
		return -1
	}
	title := normalizeMatchKey(draft.Title)
	for i, item := range items {
		if normalizeMatchKey(item.Company) != company {
			continue
		}
		existingTitle := normalizeMatchKey(item.Title)
		if title == "" || existingTitle == "" || title == existingTitle {
			return i
		}
	}
	return -1
}

func enrichEducation(existing, draft EducationInput) EducationInput {
	setIfEmpty(&existing.Degree, draft.Degree)
	setIfEmpty(&existing.Major, draft.Major)
	setIfEmpty(&existing.StartDate, draft.StartDate)
	setIfEmpty(&existing.EndDate, draft.EndDate)
	setIfEmpty(&existing.Description, draft.Description)
	if strings.TrimSpace(existing.School) == "" {
		existing.School = draft.School
	}
	return existing
}

func enrichExperience(existing, draft ExperienceInput) ExperienceInput {
	setIfEmpty(&existing.Title, draft.Title)
	setIfEmpty(&existing.Location, draft.Location)
	setIfEmpty(&existing.StartDate, draft.StartDate)
	setIfEmpty(&existing.EndDate, draft.EndDate)
	setIfEmpty(&existing.Description, draft.Description)
	if existing.IsCurrent == 0 && draft.IsCurrent != 0 {
		existing.IsCurrent = draft.IsCurrent
	}
	if strings.TrimSpace(existing.Company) == "" {
		existing.Company = draft.Company
	}
	return existing
}

func setIfEmpty(current *string, incoming string) {
	incoming = strings.TrimSpace(incoming)
	if incoming == "" || strings.TrimSpace(*current) != "" {
		return
	}
	*current = incoming
}

func normalizeMatchKey(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, " ", "")
	value = strings.ReplaceAll(value, "（", "(")
	value = strings.ReplaceAll(value, "）", ")")
	return value
}
