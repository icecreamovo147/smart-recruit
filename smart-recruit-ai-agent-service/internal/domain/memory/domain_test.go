package memory

import "testing"

func TestValidateScope(t *testing.T) {
	tests := []struct {
		name      string
		ownerRole OwnerRole
		scope     Scope
		wantErr   bool
	}{
		{name: "hr scope ok", ownerRole: OwnerRoleHR, scope: Scope{Type: ScopeHR, ID: 0}},
		{name: "hr rejects user scope", ownerRole: OwnerRoleHR, scope: Scope{Type: ScopeUser, ID: 1}, wantErr: true},
		{name: "candidate user scope ok", ownerRole: OwnerRoleCandidate, scope: Scope{Type: ScopeUser, ID: 9}},
		{name: "candidate rejects candidate scope", ownerRole: OwnerRoleCandidate, scope: Scope{Type: ScopeCandidate, ID: 1}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateScope(tt.ownerRole, tt.scope)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateScope() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClassifyPIILevel(t *testing.T) {
	if got := ClassifyPIILevel("联系我 13800138000"); got != PIILevelHigh {
		t.Fatalf("phone PII = %q, want high", got)
	}
	if got := ClassifyPIILevel("邮箱 test@example.com"); got != PIILevelHigh {
		t.Fatalf("email PII = %q, want high", got)
	}
	if got := ClassifyPIILevel("身份证号 110101199001011234"); got != PIILevelHigh {
		t.Fatalf("id card PII = %q, want high", got)
	}
	if got := ClassifyPIILevel("期望月薪 25000 元"); got != PIILevelHigh {
		t.Fatalf("salary PII = %q, want high", got)
	}
	if got := ClassifyPIILevel("银行卡 6222 0212 3456 7890 123"); got != PIILevelHigh {
		t.Fatalf("bank card PII = %q, want high", got)
	}
	if got := ClassifyPIILevel("偏好远程办公"); got != PIILevelNone {
		t.Fatalf("benign text PII = %q, want none", got)
	}
}

func TestWriteAllowedPIILevel(t *testing.T) {
	if WriteAllowedPIILevel(PIILevelHigh, false) {
		t.Fatal("high PII should be rejected without confirmation")
	}
	if !WriteAllowedPIILevel(PIILevelHigh, true) {
		t.Fatal("high PII should be allowed with confirmation")
	}
}

func TestContentHashDedupStable(t *testing.T) {
	a := ContentHash("  请记住   远程办公 ")
	b := ContentHash("请记住 远程办公")
	if a != b {
		t.Fatalf("hash mismatch: %q vs %q", a, b)
	}
}

func TestFilterInjectablesRemovesHighPII(t *testing.T) {
	items := []RankedMemory{
		{Memory: Memory{ID: 1, PIILevel: PIILevelNone, Content: "safe"}},
		{Memory: Memory{ID: 2, PIILevel: PIILevelHigh, Content: "13800138000"}},
	}
	filtered := FilterInjectables(items)
	if len(filtered) != 1 || filtered[0].Memory.ID != 1 {
		t.Fatalf("filtered = %+v, want only non-PII memory", filtered)
	}
}

func TestStatusTransition(t *testing.T) {
	if err := ValidateStatusTransition(StatusActive, StatusRevoked); err != nil {
		t.Fatalf("active->revoked should be allowed: %v", err)
	}
	if err := ValidateStatusTransition(StatusRevoked, StatusActive); err == nil {
		t.Fatal("revoked->active should be rejected")
	}
}
