package op

import (
	"strings"
	"testing"

	"gorm.io/gorm/clause"
)

func exprSQL(t *testing.T, updates map[string]interface{}, key string) string {
	t.Helper()
	v, ok := updates[key]
	if !ok {
		t.Fatalf("missing assignment %q", key)
	}
	expr, ok := v.(clause.Expr)
	if !ok {
		t.Fatalf("assignment %q is %T, want clause.Expr", key, v)
	}
	return expr.SQL
}

func TestSiteModelHourlyConflictUpdatesPostgres(t *testing.T) {
	updates := siteModelHourlyConflictUpdatesForDialect("postgres")

	if got := exprSQL(t, updates, "date"); got != "EXCLUDED.date" {
		t.Fatalf("postgres date: got %q, want EXCLUDED.date", got)
	}
	if got := exprSQL(t, updates, "last_request_at"); !strings.Contains(got, "GREATEST(") {
		t.Fatalf("postgres last_request_at should use GREATEST, got %q", got)
	}
	if got := exprSQL(t, updates, "input_token"); !strings.Contains(got, "EXCLUDED.input_token") {
		t.Fatalf("postgres input_token should use EXCLUDED, got %q", got)
	}
}

func TestSiteModelHourlyConflictUpdatesMySQL(t *testing.T) {
	updates := siteModelHourlyConflictUpdatesForDialect("mysql")

	if got := exprSQL(t, updates, "date"); got != "VALUES(date)" {
		t.Fatalf("mysql date: got %q, want VALUES(date)", got)
	}
	if got := exprSQL(t, updates, "last_request_at"); !strings.Contains(got, "GREATEST(") {
		t.Fatalf("mysql last_request_at should use GREATEST, got %q", got)
	}
}

func TestSiteModelHourlyConflictUpdatesSQLite(t *testing.T) {
	updates := siteModelHourlyConflictUpdatesForDialect("sqlite")

	if got := exprSQL(t, updates, "date"); got != "EXCLUDED.date" {
		t.Fatalf("sqlite date: got %q, want EXCLUDED.date", got)
	}
	if got := exprSQL(t, updates, "last_request_at"); !strings.Contains(got, "MAX(") || strings.Contains(got, "GREATEST(") {
		t.Fatalf("sqlite last_request_at should use MAX, got %q", got)
	}
}
