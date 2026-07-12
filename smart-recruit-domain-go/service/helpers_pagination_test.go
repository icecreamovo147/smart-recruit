package service

import "testing"

func TestNormalizeManagementPage(t *testing.T) {
	page, pageSize := normalizeManagementPage(0, 0)
	if page != 1 || pageSize != 20 {
		t.Fatalf("expected 1/20 defaults, got %d/%d", page, pageSize)
	}

	page, pageSize = normalizeManagementPage(2, 250)
	if page != 2 || pageSize != 100 {
		t.Fatalf("expected clamp to 100, got %d/%d", page, pageSize)
	}
}
