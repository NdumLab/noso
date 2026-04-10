package registry

import (
	"testing"

	"github.com/noso-dev/noso/internal/evidence"
	"github.com/noso-dev/noso/pkg/models"
)

func TestPostgresVersionIntent(t *testing.T) {
	response, err := Resolve("show postgres version", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_postgres_version" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
}

func TestPostgresDatabasesIntent(t *testing.T) {
	response, err := Resolve("list postgres databases", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_postgres_databases" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
}

func TestMySQLVersionIntent(t *testing.T) {
	response, err := Resolve("show mysql version", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_mysql_version" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
}

func TestMySQLDatabasesIntent(t *testing.T) {
	response, err := Resolve("list mysql databases", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_mysql_databases" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
}

func TestRedisVersionIntent(t *testing.T) {
	response, err := Resolve("show redis version", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_redis_version" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
}

func TestRedisPingIntent(t *testing.T) {
	response, err := Resolve("check redis health", models.Environment{}, evidence.NewCollector())
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if response.IntentID != "inspect_redis_health" {
		t.Fatalf("IntentID = %q", response.IntentID)
	}
}
