package registry

import (
	"github.com/noso-dev/noso/internal/evidence"
	"github.com/noso-dev/noso/internal/safety"
	"github.com/noso-dev/noso/pkg/models"
)

func postgresVersionIntent(collector evidence.Collector) (models.Response, error) {
	return simpleDatabaseIntent(
		collector.Lookup("psql"),
		"inspect_postgres_version",
		"psql --version",
		"Shows the installed PostgreSQL client version.",
		"PostgreSQL client version information for the local host.",
		"psql",
	)
}

func postgresDatabasesIntent(collector evidence.Collector) (models.Response, error) {
	return simpleDatabaseIntent(
		collector.Lookup("psql"),
		"inspect_postgres_databases",
		"psql -l",
		"Lists PostgreSQL databases visible to the current connection settings.",
		"A table of PostgreSQL databases with owner, encoding, locale, and access privilege details.",
		"psql",
	)
}

func mysqlVersionIntent(collector evidence.Collector) (models.Response, error) {
	return simpleDatabaseIntent(
		collector.Lookup("mysql"),
		"inspect_mysql_version",
		"mysql --version",
		"Shows the installed MySQL client version.",
		"MySQL client version information for the local host.",
		"mysql",
	)
}

func mysqlDatabasesIntent(collector evidence.Collector) (models.Response, error) {
	return simpleDatabaseIntent(
		collector.Lookup("mysql"),
		"inspect_mysql_databases",
		`mysql --execute="SHOW DATABASES;"`,
		"Lists MySQL databases visible to the current login context without changing schema state.",
		"A result set of database names returned by the MySQL server.",
		"mysql",
	)
}

func redisVersionIntent(collector evidence.Collector) (models.Response, error) {
	return simpleDatabaseIntent(
		collector.Lookup("redis-cli"),
		"inspect_redis_version",
		"redis-cli --version",
		"Shows the installed Redis CLI version.",
		"Redis CLI version information for the local host.",
		"redis-cli",
	)
}

func redisPingIntent(collector evidence.Collector) (models.Response, error) {
	return simpleDatabaseIntent(
		collector.Lookup("redis-cli"),
		"inspect_redis_health",
		"redis-cli ping",
		"Sends a lightweight Redis ping to confirm a reachable Redis endpoint with the current client settings.",
		"A PONG response when Redis is reachable, or a connection or authentication error if it is not.",
		"redis-cli",
	)
}

func simpleDatabaseIntent(ev evidence.Evidence, intentID, command, explanation, expected, evidenceName string) (models.Response, error) {
	response := models.Response{
		IntentID:       intentID,
		Command:        command,
		Explanation:    explanation,
		ExpectedOutput: expected,
		Risk:           safety.Classify(command),
		Confidence:     confidenceFor(ev),
		VerifiedFrom:   append([]string{}, ev.VerificationSources...),
	}
	addHelpEvidence(&response, ev, evidenceName)
	return response, nil
}
