package contextbudget

import (
	"testing"

	domaintokenbudget "smart-recruit-ai-agent-service/internal/domain/tokenbudget"
)

func TestEstimateTokensConservativeMatchesDomainEstimator(t *testing.T) {
	values := []string{
		"",
		" \t\r\n",
		"a",
		"abcd",
		"abcde",
		"internal spaces remain counted",
		"招聘助手🙂",
		"abcd招聘🙂",
		string([]byte{0xff, 'a'}),
	}
	for _, value := range values {
		want := domaintokenbudget.EstimateConservative(value)
		if got := EstimateTokensConservative(value); got != want {
			t.Fatalf("EstimateTokensConservative(%q) = %d, want shared estimate %d", value, got, want)
		}
	}
}
