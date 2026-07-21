package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"smart-recruit-commons/migration"
	"smart-recruit-platform-go/businessclock"
	"smart-recruit-platform-go/mysqltime"
)

const defaultMySQLDSN = "root:Aa123456@tcp(127.0.0.1:3306)/recruitment?charset=utf8mb4&parseTime=true&loc=Asia%2FShanghai&time_zone=%27%2B08%3A00%27"

func main() {
	businessclock.Configure()
	statusFlag := flag.Bool("status", false, "show migration status and exit")
	downFlag := flag.Int("down", -1, "rollback migrations to specified version and exit")
	baselineFlag := flag.Int("baseline", -1, "mark v1-N as applied without executing and exit")
	dsnFlag := flag.String("dsn", "", "MySQL DSN; defaults to MYSQL_DSN or local dev DSN")
	migrationsDirFlag := flag.String("migrations-dir", "", "migration SQL directory; defaults to ./migrations or smart-recruit-commons/migrations")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	dsn := strings.TrimSpace(*dsnFlag)
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("MYSQL_DSN"))
	}
	if dsn == "" {
		dsn = defaultMySQLDSN
	}
	dsn, err := mysqltime.NormalizeDSN(dsn)
	if err != nil {
		exitf("normalize mysql dsn: %v", err)
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		exitf("connect mysql: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		exitf("get sql db: %v", err)
	}
	defer sqlDB.Close()
	if err := mysqltime.ValidateSession(ctx, sqlDB); err != nil {
		exitf("validate mysql timezone: %v", err)
	}

	migrationsFS, subDir, err := resolveMigrationsFS(*migrationsDirFlag)
	if err != nil {
		exitf("resolve migrations: %v", err)
	}
	runner, err := migration.NewRunner(db, migrationsFS, subDir)
	if err != nil {
		exitf("init migration runner: %v", err)
	}

	switch {
	case *baselineFlag >= 0:
		if err := runner.Baseline(ctx, *baselineFlag); err != nil {
			exitf("migration baseline: %v", err)
		}
		fmt.Println("migration baseline completed")
	case *statusFlag:
		entries, err := runner.Status(ctx)
		if err != nil {
			exitf("migration status: %v", err)
		}
		migration.PrintStatus(entries)
	case *downFlag >= 0:
		if err := runner.Down(ctx, *downFlag); err != nil {
			exitf("migration down: %v", err)
		}
		fmt.Println("migration rollback completed")
	default:
		if err := runner.Up(ctx); err != nil {
			exitf("run migrations: %v", err)
		}
		fmt.Println("migration up completed")
	}
}

func resolveMigrationsFS(value string) (fs.FS, string, error) {
	if dir := strings.TrimSpace(value); dir != "" {
		if _, err := os.Stat(dir); err != nil {
			return nil, "", err
		}
		return os.DirFS(dir), ".", nil
	}
	candidates := []string{
		"migrations",
		filepath.Join("smart-recruit-commons", "migrations"),
		filepath.Join("..", "..", "migrations"),
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return os.DirFS(candidate), ".", nil
		}
	}
	return nil, "", fmt.Errorf("migrations directory not found")
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
