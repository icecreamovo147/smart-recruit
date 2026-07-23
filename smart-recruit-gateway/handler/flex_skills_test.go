package handler

import (
	"encoding/json"
	"testing"
)

func TestFlexSkillsUnmarshalJSON(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		raw  string
		want string
	}{
		{name: "null", raw: `null`, want: ""},
		{name: "string", raw: `"Go,Vue"`, want: "Go,Vue"},
		{name: "array", raw: `["Go"," Vue ",""]`, want: "Go,Vue"},
		{name: "empty_array", raw: `[]`, want: ""},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var got FlexSkills
			if err := json.Unmarshal([]byte(tc.raw), &got); err != nil {
				t.Fatalf("UnmarshalJSON() error = %v", err)
			}
			if got.String() != tc.want {
				t.Fatalf("got %q, want %q", got.String(), tc.want)
			}
		})
	}
}

func TestFlexSkillsUnmarshalJSONRejectsObject(t *testing.T) {
	t.Parallel()
	var got FlexSkills
	if err := json.Unmarshal([]byte(`{"a":1}`), &got); err == nil {
		t.Fatal("expected error for object skills")
	}
}
