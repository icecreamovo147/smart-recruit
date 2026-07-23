// Package mysqltime standardizes MySQL connection timezone semantics.
package mysqltime

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	mysqldriver "github.com/go-sql-driver/mysql"

	"smart-recruit-platform-go/businessclock"
)

const SessionTimezone = "+08:00"

// NormalizeDSN forces every physical connection created by database/sql to
// parse DATETIME values in Asia/Shanghai and initialize its MySQL session to
// +08:00. The driver sends Params as SET SESSION statements on connection.
func NormalizeDSN(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("MySQL DSN is required")
	}
	config, err := mysqldriver.ParseDSN(raw)
	if err != nil {
		return "", fmt.Errorf("parse MySQL DSN: %w", err)
	}
	config.ParseTime = true
	config.Loc = businessclock.Location
	if config.Params == nil {
		config.Params = make(map[string]string)
	}
	config.Params["time_zone"] = "'" + SessionTimezone + "'"
	return config.FormatDSN(), nil
}

type queryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// ValidateSession fails closed when the active physical connection is not
// using the platform timezone contract. NormalizeDSN makes this invariant hold
// for all other connections subsequently opened by the pool.
func ValidateSession(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("MySQL database is required")
	}
	connection, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("acquire MySQL connection: %w", err)
	}
	defer connection.Close()
	return validateSession(ctx, connection)
}

func validateSession(ctx context.Context, db queryer) error {
	var timezone string
	var offsetSeconds int
	if err := db.QueryRowContext(ctx, `SELECT @@session.time_zone, TIMESTAMPDIFF(SECOND, UTC_TIMESTAMP(), NOW())`).Scan(&timezone, &offsetSeconds); err != nil {
		return fmt.Errorf("query MySQL session timezone: %w", err)
	}
	if timezone != SessionTimezone || offsetSeconds != 8*60*60 {
		return fmt.Errorf("MySQL session timezone mismatch: time_zone=%q offset_seconds=%d, want %q/28800", timezone, offsetSeconds, SessionTimezone)
	}
	return nil
}
