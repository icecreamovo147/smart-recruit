package tokenbudget

import "testing"

func TestEstimateConservativePreservesTokenBoundarySemantics(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  int
	}{
		{name: "empty", value: "", want: 0},
		{name: "whitespace only", value: " \t\r\n", want: 0},
		{name: "trimmed single ASCII", value: "  a  ", want: 1},
		{name: "four ASCII", value: "abcd", want: 1},
		{name: "five ASCII", value: "abcde", want: 2},
		{name: "internal ASCII whitespace", value: "ab cd", want: 2},
		{name: "non ASCII by rune", value: "招聘助手🙂", want: 5},
		{name: "mixed", value: "abcd招聘🙂", want: 4},
		{name: "invalid UTF-8 remains conservative", value: string([]byte{0xff, 'a'}), want: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EstimateConservative(tt.value); got != tt.want {
				t.Fatalf("EstimateConservative(%q) = %d, want %d", tt.value, got, tt.want)
			}
		})
	}
}
