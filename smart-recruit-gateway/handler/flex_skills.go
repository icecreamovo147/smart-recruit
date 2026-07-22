package handler

import (
	"encoding/json"
	"fmt"
	"strings"
)

// FlexSkills accepts either a comma-separated string or a JSON string array.
// CandidateProfile.skills is repeated string (array) in fill drafts, while
// UpdateProfileRequest.skills is a single string — clients may send either shape.
type FlexSkills string

func (s *FlexSkills) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*s = ""
		return nil
	}
	if len(data) > 0 && data[0] == '"' {
		var str string
		if err := json.Unmarshal(data, &str); err != nil {
			return err
		}
		*s = FlexSkills(str)
		return nil
	}
	if len(data) > 0 && data[0] == '[' {
		var arr []string
		if err := json.Unmarshal(data, &arr); err != nil {
			return err
		}
		parts := make([]string, 0, len(arr))
		for _, item := range arr {
			if trimmed := strings.TrimSpace(item); trimmed != "" {
				parts = append(parts, trimmed)
			}
		}
		*s = FlexSkills(strings.Join(parts, ","))
		return nil
	}
	return fmt.Errorf("skills must be a string or string array")
}

func (s FlexSkills) String() string {
	return string(s)
}
