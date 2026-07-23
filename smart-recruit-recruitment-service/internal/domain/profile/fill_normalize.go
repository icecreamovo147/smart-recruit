package profile

import (
	"encoding/json"
	"regexp"
	"strings"
)

var (
	mainlandPhoneRe = regexp.MustCompile(`^1\d{10}$`)
	regionPathRe    = regexp.MustCompile(`^([^/]+)/([^/]+)(?:/([^/]+))?$`)
	cnRegionRe      = regexp.MustCompile(`(?:(?P<province>[^省市区县/]{2,10}省))?` +
		`(?:(?P<city>[^省市区县/]{1,12}市))?` +
		`(?:(?P<district>[^省市区县/]{1,12}(?:区|县|旗)))?`)
)

var municipalityAliases = map[string]string{
	"北京": "北京市", "北京市": "北京市",
	"上海": "上海市", "上海市": "上海市",
	"天津": "天津市", "天津市": "天津市",
	"重庆": "重庆市", "重庆市": "重庆市",
}

var degreeCanonical = []struct {
	canonical string
	aliases   []string
}{
	{"博士", []string{"博士", "phd", "ph.d", "ph.d.", "doctor", "doctorate", "doctoral"}},
	{"硕士", []string{"硕士", "研究生", "master", "masters", "m.s", "m.s.", "msc", "mba", "meng"}},
	{"本科", []string{"本科", "学士", "bachelor", "bachelors", "b.s", "b.s.", "bsc", "ba", "beng"}},
	{"大专", []string{"大专", "专科", "高职", "associate", "college diploma"}},
	{"高中", []string{"高中", "中专", "技校", "high school", "secondary"}},
}

// NormalizeDegree maps common degree labels onto the candidate form enums.
// ok=false means the value is empty; unmatched non-empty values are returned as-is with matched=false.
func NormalizeDegree(raw string) (value string, matched bool, empty bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", false, true
	}
	lower := strings.ToLower(trimmed)
	for _, item := range degreeCanonical {
		for _, alias := range item.aliases {
			if lower == alias {
				return item.canonical, true, false
			}
		}
	}
	for _, item := range degreeCanonical {
		for _, alias := range item.aliases {
			if len(alias) >= 2 && strings.Contains(lower, alias) {
				return item.canonical, true, false
			}
		}
	}
	return trimmed, false, false
}

// NormalizePhone keeps mainland mobile numbers only.
func NormalizePhone(raw string) (string, bool) {
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, strings.TrimSpace(raw))
	if mainlandPhoneRe.MatchString(digits) {
		return digits, true
	}
	return "", false
}

// NormalizeRegionForProfile converts free-text locations into slash-separated
// 省/市[/区] form when recognizable. ok=true means a usable cascader value was
// produced; the value may still lack a district (incomplete for final save).
func NormalizeRegionForProfile(raw string) (normalized string, ok bool) {
	raw = strings.TrimSpace(strings.ReplaceAll(raw, " ", ""))
	if raw == "" {
		return "", false
	}
	raw = strings.ReplaceAll(raw, "／", "/")
	if parts := splitRegionParts(raw); len(parts) > 0 {
		if value, recognized := normalizeRegionParts(parts); recognized {
			return value, true
		}
	}
	if value, recognized := parseChineseRegionText(raw); recognized {
		return value, true
	}
	return "", false
}

// IsRegionPathComplete reports whether a stored city value satisfies 省/市/区
// (or 直辖市/区) selection completeness.
func IsRegionPathComplete(value string) bool {
	parts := splitRegionParts(strings.TrimSpace(strings.ReplaceAll(value, "／", "/")))
	if len(parts) == 0 {
		// Also accept compact Chinese text that fully parses.
		if normalized, ok := NormalizeRegionForProfile(value); ok {
			parts = splitRegionParts(normalized)
		}
	}
	if len(parts) == 0 {
		return false
	}
	province := parts[0]
	if isMunicipalityLabel(province) || municipalityAliases[province] != "" {
		if len(parts) >= 2 && parts[1] != "" && parts[1] != "市辖区" {
			return true
		}
		return len(parts) >= 3 && parts[2] != ""
	}
	return len(parts) >= 3 && parts[0] != "" && parts[1] != "" && parts[2] != ""
}

func splitRegionParts(raw string) []string {
	if !strings.Contains(raw, "/") {
		return nil
	}
	match := regionPathRe.FindStringSubmatch(raw)
	if match == nil {
		parts := strings.Split(raw, "/")
		out := make([]string, 0, len(parts))
		for _, part := range parts {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				out = append(out, trimmed)
			}
		}
		return out
	}
	out := make([]string, 0, 3)
	for _, part := range match[1:] {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func normalizeRegionParts(parts []string) (string, bool) {
	if len(parts) == 0 {
		return "", false
	}
	province := parts[0]
	if canonical, isMuni := municipalityAliases[province]; isMuni || isMunicipalityLabel(province) {
		if canonical == "" {
			canonical = ensureMunicipalitySuffix(province)
		}
		district := ""
		if len(parts) >= 3 {
			district = parts[2]
			if parts[1] != "市辖区" && parts[1] != canonical {
				// 北京市/朝阳区
				district = parts[1]
			}
		} else if len(parts) == 2 {
			if parts[1] == "市辖区" {
				return "", false
			}
			district = parts[1]
		}
		if district == "" {
			return canonical, true
		}
		return canonical + "/" + district, true
	}
	if len(parts) >= 3 {
		return strings.Join(parts[:3], "/"), true
	}
	if len(parts) == 2 {
		return strings.Join(parts, "/"), true
	}
	if len(parts) == 1 {
		return parts[0], true
	}
	return "", false
}

func parseChineseRegionText(raw string) (string, bool) {
	for alias, canonical := range municipalityAliases {
		if strings.HasPrefix(raw, alias) || strings.HasPrefix(raw, canonical) {
			rest := strings.TrimPrefix(raw, canonical)
			rest = strings.TrimPrefix(rest, alias)
			rest = strings.TrimPrefix(rest, "市")
			district := extractTrailingDistrict(rest)
			if district == "" {
				district = extractTrailingDistrict(raw)
			}
			if district == "" {
				return canonical, true
			}
			return canonical + "/" + district, true
		}
	}
	match := cnRegionRe.FindStringSubmatch(raw)
	if match == nil {
		return "", false
	}
	groups := make(map[string]string, 3)
	for i, name := range cnRegionRe.SubexpNames() {
		if i == 0 || name == "" {
			continue
		}
		groups[name] = match[i]
	}
	province := groups["province"]
	city := groups["city"]
	district := groups["district"]
	if province == "" && city == "" && district == "" {
		return "", false
	}
	if canonical, ok := municipalityAliases[strings.TrimSuffix(city, "市")]; ok || isMunicipalityLabel(city) {
		if canonical == "" {
			canonical = ensureMunicipalitySuffix(city)
		}
		if district == "" {
			return canonical, true
		}
		return canonical + "/" + district, true
	}
	if province != "" && city != "" && district != "" {
		return province + "/" + city + "/" + district, true
	}
	if province != "" && city != "" {
		return province + "/" + city, true
	}
	if province != "" {
		return province, true
	}
	if city != "" && district != "" {
		// City + district without province is ambiguous for the cascader.
		return "", false
	}
	return "", false
}

func extractTrailingDistrict(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	runes := []rune(raw)
	for i := 0; i < len(runes); i++ {
		chunk := string(runes[i:])
		if strings.HasSuffix(chunk, "区") || strings.HasSuffix(chunk, "县") || strings.HasSuffix(chunk, "旗") {
			if len([]rune(chunk)) >= 2 && len([]rune(chunk)) <= 12 {
				return chunk
			}
		}
	}
	return ""
}

func isMunicipalityLabel(label string) bool {
	_, ok := municipalityAliases[label]
	if ok {
		return true
	}
	_, ok = municipalityAliases[strings.TrimSuffix(label, "市")]
	return ok
}

func ensureMunicipalitySuffix(label string) string {
	label = strings.TrimSpace(label)
	if strings.HasSuffix(label, "市") {
		return label
	}
	return label + "市"
}

// TruncateRunes truncates a string to max runes.
func TruncateRunes(value string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= max {
		return string(runes)
	}
	return string(runes[:max])
}

// JoinAchievementsIntoDescription appends achievement bullets into a description.
func JoinAchievementsIntoDescription(description string, achievementsJSON string) string {
	description = strings.TrimSpace(description)
	achievements := parseJSONStringArray(achievementsJSON)
	if len(achievements) == 0 {
		return description
	}
	var b strings.Builder
	if description != "" {
		b.WriteString(description)
		b.WriteString("\n")
	}
	for _, item := range achievements {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		b.WriteString("• ")
		b.WriteString(item)
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

func parseJSONStringArray(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" {
		return nil
	}
	var items []string
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
