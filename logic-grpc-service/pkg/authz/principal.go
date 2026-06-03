package authz

// Principal represents the authenticated identity and its authorization context.
// It is populated server-side from the JWT and database and must never be
// derived from client-provided values.
type Principal struct {
	UserID       int64    `json:"user_id"`
	Username     string   `json:"username"`
	AccountType  string   `json:"account_type"`
	Roles        []string `json:"roles"`
	Permissions  []string `json:"permissions"`
	DataScopes   []ScopeAssignment `json:"data_scopes,omitempty"`
	TokenVersion int32    `json:"token_version"`
	// Deprecated: kept for compatibility during migration window.
	LegacyRole int32 `json:"role"`
}

// ScopeAssignment pairs a scope key with optional resource identifiers.
type ScopeAssignment struct {
	ScopeKey     string `json:"scope_key"`
	ResourceType string `json:"resource_type,omitempty"`
	ResourceID   int64  `json:"resource_id,omitempty"`
}

// HasRole returns true if the principal has the given role key.
func (p *Principal) HasRole(roleKey string) bool {
	for _, r := range p.Roles {
		if r == roleKey {
			return true
		}
	}
	return false
}

// HasAnyRole returns true if the principal has any of the given role keys.
func (p *Principal) HasAnyRole(roleKeys ...string) bool {
	for _, r := range p.Roles {
		for _, target := range roleKeys {
			if r == target {
				return true
			}
		}
	}
	return false
}

// HasPermission returns true if the principal has the given permission key.
func (p *Principal) HasPermission(permKey string) bool {
	for _, pm := range p.Permissions {
		if pm == permKey {
			return true
		}
	}
	return false
}

// HasAnyPermission returns true if the principal has any of the given permission keys.
func (p *Principal) HasAnyPermission(permKeys ...string) bool {
	for _, pm := range p.Permissions {
		for _, target := range permKeys {
			if pm == target {
				return true
			}
		}
	}
	return false
}

// HasScope returns true if the principal has the given scope key (with or without resource qualifiers).
func (p *Principal) HasScope(scopeKey string) bool {
	for _, s := range p.DataScopes {
		if s.ScopeKey == scopeKey {
			return true
		}
	}
	return false
}

// IsStaff returns true if the principal has a staff account type.
func (p *Principal) IsStaff() bool {
	return p.AccountType == "staff"
}

// IsCandidate returns true if the principal has a candidate account type.
func (p *Principal) IsCandidate() bool {
	return p.AccountType == "candidate"
}
