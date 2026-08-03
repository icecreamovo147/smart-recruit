package migration

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestAgentSkillPackageV2MigrationContract(t *testing.T) {
	t.Parallel()

	up := readMigrationFixture(t, "000090_agent_skill_package_v2.sql")
	down := readMigrationFixture(t, "000090_agent_skill_package_v2.down.sql")
	dbSQLBytes, err := os.ReadFile(filepath.Join("..", "..", "db.sql"))
	if err != nil {
		t.Fatalf("read db.sql: %v", err)
	}
	dbSQL := string(dbSQLBytes)

	retireCurrentRelease := sqlStatementContaining(t, up, "SET version.`status` = 'retired'")
	requireSQLFragments(t, retireCurrentRelease, []string{
		"ON capability.`current_published_version_id` = version.`id`",
		"WHERE capability.`audience` = 'tenant_hr'",
		"AND capability.`capability_key` IN ('ai.chat', 'ai.agent_run')",
	})
	requireExactCapabilityTarget(
		t,
		retireCurrentRelease,
		`capability\.\x60audience\x60`,
		`capability\.\x60capability_key\x60`,
	)

	clearCurrentRelease := sqlStatementContaining(t, up, "SET `current_published_version_id` = NULL")
	requireSQLFragments(t, clearCurrentRelease, []string{
		"UPDATE `platform_ai_capabilities`",
		"WHERE `audience` = 'tenant_hr'",
		"AND `capability_key` IN ('ai.chat', 'ai.agent_run')",
	})
	requireExactCapabilityTarget(t, clearCurrentRelease, `\x60audience\x60`, `\x60capability_key\x60`)

	requiredUpFragments := []string{
		"`object_type` IN ('agent_skill', 'agent_skill_version', 'agent_skill_section')",
		"SET `agent_skill_ids_json` = NULL",
		"CHANGE COLUMN `agent_skill_ids_json` `agent_skill_version_ids_json`",
		"DELETE FROM `agent_skill_versions`",
		"DELETE FROM `agent_skills`",
		"CREATE TABLE `agent_skill_version_sections`",
		"ON DELETE CASCADE",
		"CHECK (`core_estimated_tokens` BETWEEN 1 AND 800)",
		"CHECK (REGEXP_LIKE(`section_key`, '^[a-z][a-z0-9_-]{1,127}$', 'c'))",
		"CHECK (`estimated_tokens` BETWEEN 1 AND 1200)",
	}
	requireSQLFragments(t, up, requiredUpFragments)

	destructivePlatformHistorySQL := regexp.MustCompile(
		`(?is)\b(?:DELETE|TRUNCATE|DROP)\b[^;]*\b` +
			`(?:platform_ai_capability_versions|platform_ai_config_audit_logs)\b`,
	)
	if match := destructivePlatformHistorySQL.FindString(up); match != "" {
		t.Errorf("up migration must preserve release/audit history; found %q", match)
	}

	assertFragmentsInOrder(t, up, []string{
		"SET version.`status` = 'retired'",
		"SET `current_published_version_id` = NULL",
		"DELETE FROM `ai_embeddings`",
		"SET `agent_skill_ids_json` = NULL",
		"CHANGE COLUMN `agent_skill_ids_json` `agent_skill_version_ids_json`",
		"UPDATE `agent_skills`\nSET `current_version_id` = NULL",
		"DROP FOREIGN KEY `fk_agent_skills_current_version`",
		"DELETE FROM `agent_skill_versions`",
		"DELETE FROM `agent_skills`",
		"DROP TABLE `agent_skill_versions`",
		"DROP TABLE `agent_skills`",
		"CREATE TABLE `agent_skills`",
		"CREATE TABLE `agent_skill_versions`",
		"CREATE TABLE `agent_skill_version_sections`",
		"ADD CONSTRAINT `fk_agent_skills_current_version`",
	})

	if !strings.Contains(down, "CHANGE COLUMN `agent_skill_version_ids_json` `agent_skill_ids_json`") {
		t.Error("down migration does not restore the legacy chat column name")
	}
	if !strings.Contains(down, "restoring deleted data or retired capability releases") {
		t.Error("down migration must document its non-restorative data semantics")
	}

	requiredSnapshotFragments := []string{
		"`agent_skill_version_ids_json` TEXT NULL",
		"CREATE TABLE IF NOT EXISTS `agent_skill_version_sections`",
		"`manifest_json` JSON NOT NULL",
		"`core_markdown` MEDIUMTEXT NOT NULL",
		"`compiled_markdown` MEDIUMTEXT NOT NULL",
		"`authoring_json` JSON NULL",
		"`compiled_hash` CHAR(64) NOT NULL",
		"`core_estimated_tokens` INT UNSIGNED NOT NULL",
		"CHECK (REGEXP_LIKE(`section_key`, '^[a-z][a-z0-9_-]{1,127}$', 'c'))",
	}
	requireSQLFragments(t, dbSQL, requiredSnapshotFragments)

	agentSkillsStart := strings.Index(dbSQL, "CREATE TABLE IF NOT EXISTS `agent_skills`")
	agentVersionsStart := strings.Index(dbSQL, "CREATE TABLE IF NOT EXISTS `agent_skill_versions`")
	if agentSkillsStart < 0 || agentVersionsStart <= agentSkillsStart {
		t.Fatal("cannot isolate agent_skills definition in db.sql")
	}
	registryDDL := dbSQL[agentSkillsStart:agentVersionsStart]
	for _, legacyColumn := range []string{
		"`trigger_keywords`",
		"`agent_type`",
		"`category`",
		"`scenario`",
		"`risk_level`",
		"`required_capabilities`",
		"`output_schema`",
		"`evaluation_criteria`",
		"`semantic_tags`",
	} {
		if strings.Contains(registryDDL, legacyColumn) {
			t.Errorf("registry table still contains legacy runtime column %s", legacyColumn)
		}
	}

	coldStartReleaseSeed := sqlStatementContaining(
		t,
		dbSQL,
		"INSERT INTO `platform_ai_capability_versions`\n  (`capability_id`, `version`, `status`",
	)
	requireSQLFragments(t, coldStartReleaseSeed, []string{
		"WHEN capability.audience = 'tenant_hr'",
		"AND capability.capability_key IN ('ai.chat', 'ai.agent_run')",
		"THEN 'retired'",
		"ELSE 'published'",
	})

	coldStartCurrentPointer := sqlStatementContaining(
		t,
		dbSQL,
		"SET capability.current_published_version_id = version.id",
	)
	requireSQLFragments(t, coldStartCurrentPointer, []string{
		"ON version.capability_id = capability.id",
		"AND version.version = 1",
		"AND version.status = 'published'",
	})
	if strings.Contains(coldStartCurrentPointer, "version.status = 'retired'") {
		t.Error("db.sql cold start must not point capabilities at retired chat/run releases")
	}
}

func readMigrationFixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "migrations", name))
	if err != nil {
		t.Fatalf("read migration %s: %v", name, err)
	}
	return string(data)
}

func sqlStatementContaining(t *testing.T, sql, marker string) string {
	t.Helper()
	for _, statement := range strings.Split(sql, ";") {
		if strings.Contains(statement, marker) {
			return statement
		}
	}
	t.Fatalf("SQL statement containing %q not found", marker)
	return ""
}

func requireSQLFragments(t *testing.T, sql string, fragments []string) {
	t.Helper()
	for _, fragment := range fragments {
		if !strings.Contains(sql, fragment) {
			t.Errorf("SQL missing contract fragment %q", fragment)
		}
	}
}

func assertFragmentsInOrder(t *testing.T, sql string, fragments []string) {
	t.Helper()
	cursor := 0
	for _, fragment := range fragments {
		offset := strings.Index(sql[cursor:], fragment)
		if offset < 0 {
			t.Fatalf("SQL fragment %q missing or out of order", fragment)
		}
		cursor += offset + len(fragment)
	}
}

func requireExactCapabilityTarget(t *testing.T, statement, audienceColumn, capabilityColumn string) {
	t.Helper()
	target := regexp.MustCompile(
		`(?is)\bWHERE\s+` + audienceColumn + `\s*=\s*'tenant_hr'\s+` +
			`AND\s+` + capabilityColumn + `\s+IN\s*\(\s*'ai\.chat'\s*,\s*'ai\.agent_run'\s*\)\s*$`,
	)
	if !target.MatchString(statement) {
		t.Errorf("capability cutover must target exactly tenant_hr ai.chat/ai.agent_run; statement:\n%s", statement)
	}
}
