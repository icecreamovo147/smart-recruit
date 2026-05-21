package cache

import "testing"

func TestPublicFirstPageKeyIncludesPageSize(t *testing.T) {
	if publicFirstPageKey(10) == publicFirstPageKey(20) {
		t.Fatalf("expected public job cache key to vary by page size")
	}
	if got := publicFirstPageKey(0); got != publicFirstPageKey(10) {
		t.Fatalf("expected invalid page size to use default key, got %q", got)
	}
}
