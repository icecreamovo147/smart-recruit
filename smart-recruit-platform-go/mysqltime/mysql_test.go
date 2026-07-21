package mysqltime

import (
	"net/url"
	"strings"
	"testing"

	mysqldriver "github.com/go-sql-driver/mysql"
)

func TestNormalizeDSNForcesUTC8Contract(t *testing.T) {
	normalized, err := NormalizeDSN("user:pass@tcp(localhost:3306)/recruitment?parseTime=false&loc=Local&time_zone=%27%2B00%3A00%27")
	if err != nil {
		t.Fatal(err)
	}
	config, err := mysqldriver.ParseDSN(normalized)
	if err != nil {
		t.Fatal(err)
	}
	if !config.ParseTime || config.Loc.String() != "Asia/Shanghai" {
		t.Fatalf("parseTime=%v loc=%s", config.ParseTime, config.Loc)
	}
	if config.Params["time_zone"] != "'+08:00'" {
		t.Fatalf("time_zone=%q in %s", config.Params["time_zone"], normalized)
	}
	if strings.Contains(normalized, "loc=Local") {
		t.Fatalf("normalized DSN retained loc=Local: %s", normalized)
	}
	if _, err := url.QueryUnescape(normalized); err != nil {
		t.Fatalf("invalid escaping: %v", err)
	}
}

func TestNormalizeDSNRejectsEmptyAndInvalid(t *testing.T) {
	for _, value := range []string{"", "://not-a-dsn"} {
		if _, err := NormalizeDSN(value); err == nil {
			t.Fatalf("NormalizeDSN(%q) unexpectedly succeeded", value)
		}
	}
}
