package memory

import "regexp"

var (
	phonePattern    = regexp.MustCompile(`(?:(?:\+|00)86[\s-]?)?1[3-9]\d{9}`)
	emailPattern    = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
	idCardPattern   = regexp.MustCompile(`(?:\d{17}[\dXx]|\d{15})`)
	salaryPattern   = regexp.MustCompile(`(?i)(?:薪资|工资|薪水|年薪|月薪|薪酬|package|salary)[^\d]{0,12}\d+(?:\.\d+)?(?:\s*(?:万|k|K|元|人民币|rmb|USD|\$))?|\d+(?:\.\d+)?(?:\s*(?:万|k|K))(?:\s*(?:元|人民币|rmb|USD|\$))?[^\d]{0,12}(?:薪资|工资|薪水|年薪|月薪|薪酬|salary)`)
	bankCardPattern = regexp.MustCompile(`\b(?:\d{4}[\s-]?){3,4}\d{1,7}\b`)
)

func ClassifyPIILevel(content string) PIILevel {
	if phonePattern.MatchString(content) ||
		emailPattern.MatchString(content) ||
		idCardPattern.MatchString(content) ||
		salaryPattern.MatchString(content) ||
		bankCardPattern.MatchString(content) {
		return PIILevelHigh
	}
	return PIILevelNone
}

func WriteAllowedPIILevel(level PIILevel, confirmHighPII bool) bool {
	if level != PIILevelHigh {
		return true
	}
	return confirmHighPII
}

func FilterInjectables(items []RankedMemory) []RankedMemory {
	if len(items) == 0 {
		return nil
	}
	out := make([]RankedMemory, 0, len(items))
	for _, item := range items {
		if item.Memory.PIILevel == PIILevelHigh {
			continue
		}
		out = append(out, item)
	}
	return out
}
