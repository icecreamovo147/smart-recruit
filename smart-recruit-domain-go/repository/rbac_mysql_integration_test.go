//go:build integration
// +build integration

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"gopkg.in/yaml.v3"
)

// mysqlConfig is a minimal struct matching config.Config.MySQL for DSN extraction.
type mysqlConfig struct {
	MySQL struct {
		DSN string `yaml:"dsn"`
	} `yaml:"mysql"`
}

// loadMySQLDSN reads the backend's config.yaml and extracts the MySQL DSN.
// Returns empty string if config is not found or unreadable.
func loadMySQLDSN() string {
	// Try common config file locations
	candidates := []string{
		"config/config.yaml",
		"../config/config.yaml",
	}
	// Also check the absolute project config path relative to this file
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "config/config.yaml"))
	}

	for _, p := range candidates {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var cfg mysqlConfig
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			continue
		}
		if cfg.MySQL.DSN != "" {
			return cfg.MySQL.DSN
		}
	}
	return ""
}

// TestIntegration_RBACSchemaExists verifies that importing db.sql into a
// real MySQL database creates all required RBAC tables and seed data.
// Requires: go test -tags=integration -run TestIntegration_RBACSchemaExists
// The configured database will be modified — use only in development.
func TestIntegration_RBACSchemaExists(t *testing.T) {
	dsn := loadMySQLDSN()
	if dsn == "" {
		t.Skip("MySQL config not found; set up config/config.yaml with mysql.dsn to run integration tests")
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open mysql: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping mysql: %v", err)
	}

	// ── Verify required RBAC tables exist ──────────────────────────────
	requiredTables := []string{
		"roles",
		"permissions",
		"role_permissions",
		"user_roles",
		"user_data_scopes",
		"authorization_audit_logs",
	}
	for _, table := range requiredTables {
		var count int
		query := fmt.Sprintf("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = '%s'", table)
		if err := db.QueryRowContext(ctx, query).Scan(&count); err != nil {
			t.Errorf("check table %s: %v", table, err)
		} else if count == 0 {
			t.Errorf("required table %s does not exist", table)
		} else {
			t.Logf("✅ table %s exists", table)
		}
	}

	// ── Verify required role keys exist ────────────────────────────────
	requiredRoles := []string{"candidate", "recruiter", "recruiting_admin", "system_admin", "interviewer"}
	for _, roleKey := range requiredRoles {
		var count int
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM roles WHERE role_key = ?", roleKey).Scan(&count); err != nil {
			t.Errorf("check role %s: %v", roleKey, err)
		} else if count == 0 {
			t.Errorf("required role %s not found", roleKey)
		} else {
			t.Logf("✅ role %s exists", roleKey)
		}
	}

	// ── Verify key permissions exist ───────────────────────────────────
	requiredPermissions := []string{
		"application.status.update",
		"admin.user.manage",
		"admin.role.manage",
		"audit.security.read",
		"interview.feedback.submit",
	}
	for _, permKey := range requiredPermissions {
		var count int
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM permissions WHERE permission_key = ?", permKey).Scan(&count); err != nil {
			t.Errorf("check permission %s: %v", permKey, err)
		} else if count == 0 {
			t.Errorf("required permission %s not found", permKey)
		} else {
			t.Logf("✅ permission %s exists", permKey)
		}
	}

	// ── Verify role-permission mappings exist ──────────────────────────
	for _, roleKey := range requiredRoles {
		var count int
		query := `SELECT COUNT(*) FROM role_permissions rp
			JOIN roles r ON r.id = rp.role_id
			JOIN permissions p ON p.id = rp.permission_id
			WHERE r.role_key = ?`
		if err := db.QueryRowContext(ctx, query, roleKey).Scan(&count); err != nil {
			t.Errorf("check role_permissions for %s: %v", roleKey, err)
		} else if count == 0 {
			t.Errorf("role %s has no permission mappings", roleKey)
		} else {
			t.Logf("✅ role %s has %d permissions", roleKey, count)
		}
	}

	// ── Verify default admin exists with staff account_type ────────────
	var adminCount int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE username = 'admin' AND account_type = 'staff'").Scan(&adminCount); err != nil {
		t.Errorf("check default admin: %v", err)
	} else if adminCount == 0 {
		t.Error("default admin user not found or account_type != 'staff'")
	} else {
		t.Log("✅ default admin exists with account_type='staff'")
	}

	// ── Verify default admin has RBAC assignments ──────────────────────
	var rbacCount int
	query := `SELECT COUNT(*) FROM user_roles ur
		JOIN users u ON u.id = ur.user_id
		JOIN roles r ON r.id = ur.role_id
		WHERE u.username = 'admin' AND ur.revoked_at IS NULL`
	if err := db.QueryRowContext(ctx, query).Scan(&rbacCount); err != nil {
		t.Errorf("check admin RBAC: %v", err)
	} else if rbacCount == 0 {
		t.Error("default admin has no active RBAC role assignments")
	} else {
		t.Logf("✅ default admin has %d active RBAC role(s)", rbacCount)
	}
}

// TestIntegration_SeedIdempotency verifies that re-running seed SQL does not
// duplicate role, permission, or role_permission records.
func TestIntegration_SeedIdempotency(t *testing.T) {
	dsn := loadMySQLDSN()
	if dsn == "" {
		t.Skip("MySQL config not found")
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open mysql: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping mysql: %v", err)
	}

	// Count before
	countBefore := make(map[string]int64)
	for _, table := range []string{"roles", "permissions", "role_permissions", "user_roles"} {
		var c int64
		db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&c)
		countBefore[table] = c
	}

	// Re-run the seed SQL (UPSERT semantics should make this idempotent)
	seedSQL := `
INSERT INTO roles (role_key, name, description, is_system, created_at, updated_at) VALUES
('candidate','求职者','外部求职者',1,NOW(),NOW()),
('recruiter','招聘专员','负责岗位发布和候选人流程',1,NOW(),NOW()),
('recruiting_admin','招聘管理员','管理招聘配置和用户角色',1,NOW(),NOW()),
('system_admin','系统管理员','管理平台安全配置',1,NOW(),NOW()),
('interviewer','面试官','查看被分配的面试并提交反馈',1,NOW(),NOW())
ON DUPLICATE KEY UPDATE name=VALUES(name), description=VALUES(description), updated_at=NOW()`

	if _, err := db.ExecContext(ctx, seedSQL); err != nil {
		t.Fatalf("re-run seed SQL: %v", err)
	}

	// Count after — should be identical
	for _, table := range []string{"roles", "permissions", "role_permissions", "user_roles"} {
		var c int64
		db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&c)
		if c != countBefore[table] {
			t.Errorf("table %s: count changed from %d to %d after re-seed (not idempotent)", table, countBefore[table], c)
		} else {
			t.Logf("✅ table %s idempotent: %d rows", table, c)
		}
	}

	// Verify active user_roles uniqueness for default admin
	var duplicateUserRoles int64
	db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM (
			SELECT user_id, role_id, COUNT(*) as cnt
			FROM user_roles WHERE revoked_at IS NULL
			GROUP BY user_id, role_id HAVING cnt > 1
		) dup`).Scan(&duplicateUserRoles)
	if duplicateUserRoles > 0 {
		t.Errorf("found %d duplicate active user_role assignments", duplicateUserRoles)
	} else {
		t.Log("✅ no duplicate active user_role assignments")
	}
}
