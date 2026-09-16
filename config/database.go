package config

import (
	"fmt"
	"net"
)

type Driver string

const (
	PostgresDriver   Driver = "postgres"
	ClickhouseDriver Driver = "clickhouse"
)

func (driver Driver) Values() []Driver {
	return []Driver{
		PostgresDriver,
		ClickhouseDriver,
	}
}

func (driver Driver) String() string {
	return string(driver)
}

type Database struct {
	MigrationsTable string
	Driver          Driver
	Connection      map[Driver]DatabaseConnection
	Redis           Redis
}

type DatabaseConnection struct {
	Driver   Driver
	Host     string
	Port     string
	Username string
	Password string
	Database string
	SSLMode  DatabaseSSLMode
}

type DatabaseSSLMode string

const (
	SSLModeDisable    DatabaseSSLMode = "disable"
	SSLModeRequire    DatabaseSSLMode = "require"
	SSLModeVerifyCA   DatabaseSSLMode = "verify-ca"
	SSLModeVerifyFull DatabaseSSLMode = "verify-full"
)

func (s DatabaseSSLMode) Values() (values []DatabaseSSLMode) {
	return []DatabaseSSLMode{
		SSLModeDisable,
		SSLModeRequire,
		SSLModeVerifyCA,
		SSLModeVerifyFull,
	}
}

func (s DatabaseSSLMode) String() string {
	return string(s)
}

// DataSourceName can be upgraded/changed.
func (config Database) DataSourceName() string {
	connection := config.Connection[config.Driver]

	switch connection.Driver {
	case PostgresDriver:
		return fmt.Sprint(
			"postgres://", connection.Username, ":", connection.Password,
			"@", net.JoinHostPort(connection.Host, connection.Port), "/", connection.Database,
			"?sslmode=", connection.SSLMode,
		)
	default:
		panic(fmt.Sprintf("unsupported database driver: %s", connection.Driver))
	}
}

func (config Database) Info() DatabaseConnection {
	return config.Connection[config.Driver]
}
