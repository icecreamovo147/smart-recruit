package repository

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/go-sql-driver/mysql"
)

func TestMySQLGeneratedColumnConstraints(t *testing.T) {
	dsn := os.Getenv("MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("MYSQL_TEST_DSN is not set; skipping MySQL generated-column constraint test")
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open mysql failed: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatalf("get mysql conn failed: %v", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, `
CREATE TEMPORARY TABLE stage1_applications (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  job_id BIGINT UNSIGNED NOT NULL,
  user_id BIGINT UNSIGNED NOT NULL,
  status TINYINT NOT NULL DEFAULT 0,
  is_current TINYINT NOT NULL DEFAULT 1,
  applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  active_key TINYINT GENERATED ALWAYS AS (
    CASE WHEN is_current = 1 AND status <> 3 THEN 1 ELSE NULL END
  ) STORED,
  PRIMARY KEY (id),
  UNIQUE KEY uk_active_application (job_id, user_id, active_key)
) ENGINE=InnoDB`); err != nil {
		t.Fatalf("create temporary applications table failed: %v", err)
	}

	if _, err := conn.ExecContext(ctx, `
CREATE TEMPORARY TABLE stage1_resumes (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  user_id BIGINT UNSIGNED NOT NULL,
  is_valid TINYINT NOT NULL DEFAULT 1,
  uploaded_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  valid_key TINYINT GENERATED ALWAYS AS (
    CASE WHEN is_valid = 1 THEN 1 ELSE NULL END
  ) STORED,
  PRIMARY KEY (id),
  UNIQUE KEY uk_user_valid_resume (user_id, valid_key)
) ENGINE=InnoDB`); err != nil {
		t.Fatalf("create temporary resumes table failed: %v", err)
	}

	if _, err := conn.ExecContext(ctx, `INSERT INTO stage1_applications (job_id, user_id, status, is_current) VALUES (1, 1, 0, 1)`); err != nil {
		t.Fatalf("insert first active application failed: %v", err)
	}
	if _, err := conn.ExecContext(ctx, `INSERT INTO stage1_applications (job_id, user_id, status, is_current) VALUES (1, 1, 1, 1)`); err == nil {
		t.Fatalf("expected duplicate active application to fail")
	}
	if _, err := conn.ExecContext(ctx, `INSERT INTO stage1_applications (job_id, user_id, status, is_current) VALUES (1, 1, 3, 1)`); err != nil {
		t.Fatalf("rejected current application should be allowed by generated key: %v", err)
	}

	if _, err := conn.ExecContext(ctx, `INSERT INTO stage1_resumes (user_id, is_valid) VALUES (1, 1)`); err != nil {
		t.Fatalf("insert first valid resume failed: %v", err)
	}
	if _, err := conn.ExecContext(ctx, `INSERT INTO stage1_resumes (user_id, is_valid) VALUES (1, 1)`); err == nil {
		t.Fatalf("expected duplicate valid resume to fail")
	}
	if _, err := conn.ExecContext(ctx, `INSERT INTO stage1_resumes (user_id, is_valid) VALUES (1, 0)`); err != nil {
		t.Fatalf("invalid historical resume should be allowed by generated key: %v", err)
	}
}
