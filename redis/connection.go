package redis

import (
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	goredis "github.com/redis/go-redis/v9"
)

const (
	DefaultConnection = "default"

	ModeSingle   = "single"
	ModeSentinel = "sentinel"
	ModeCluster  = "cluster"
)

type Configuration struct {
	Mode             string
	Addresses        []string
	MasterName       string
	Username         string
	Password         string
	SentinelUsername string
	SentinelPassword string
	Database         int
	TLS              bool
}

func Load(connection string) (Configuration, error) {
	prefix, err := connectionPrefix(connection)
	if err != nil {
		return Configuration{}, err
	}

	configuration := Configuration{
		Mode:             strings.ToLower(setting(prefix, "MODE", ModeSingle)),
		MasterName:       setting(prefix, "MASTER_NAME", ""),
		Username:         setting(prefix, "USERNAME", ""),
		Password:         setting(prefix, "PASSWORD", ""),
		SentinelUsername: setting(prefix, "SENTINEL_USERNAME", ""),
		SentinelPassword: setting(prefix, "SENTINEL_PASSWORD", ""),
	}

	addresses, err := loadAddresses(prefix)
	if err != nil {
		return Configuration{}, err
	}
	configuration.Addresses = addresses

	database, err := strconv.Atoi(setting(prefix, "DATABASE", "0"))
	if err != nil || database < 0 || database > 15 {
		return Configuration{}, fmt.Errorf("%s_DATABASE must be between 0 and 15", prefix)
	}
	configuration.Database = database

	tlsEnabled, err := strconv.ParseBool(setting(prefix, "TLS", "false"))
	if err != nil {
		return Configuration{}, fmt.Errorf("%s_TLS must be true or false", prefix)
	}
	configuration.TLS = tlsEnabled

	if err := configuration.validate(prefix); err != nil {
		return Configuration{}, err
	}

	return configuration, nil
}

func NewClient(connection string) (goredis.UniversalClient, error) {
	configuration, err := Load(connection)
	if err != nil {
		return nil, err
	}

	return goredis.NewUniversalClient(configuration.Options()), nil
}

func (configuration Configuration) Options() *goredis.UniversalOptions {
	options := &goredis.UniversalOptions{
		Addrs:            configuration.Addresses,
		Username:         configuration.Username,
		Password:         configuration.Password,
		SentinelUsername: configuration.SentinelUsername,
		SentinelPassword: configuration.SentinelPassword,
		DB:               configuration.Database,
	}

	switch configuration.Mode {
	case ModeSentinel:
		options.MasterName = configuration.MasterName
	case ModeCluster:
		options.IsClusterMode = true
	}

	if configuration.TLS {
		options.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}

	return options
}

func (configuration Configuration) validate(prefix string) error {
	switch configuration.Mode {
	case ModeSingle:
		if len(configuration.Addresses) != 1 {
			return fmt.Errorf("%s_ADDRESSES must contain one address in single mode", prefix)
		}
	case ModeSentinel:
		if configuration.MasterName == "" {
			return fmt.Errorf("%s_MASTER_NAME is required in sentinel mode", prefix)
		}
	case ModeCluster:
		if configuration.Database != 0 {
			return fmt.Errorf("%s_DATABASE must be 0 in cluster mode", prefix)
		}
	default:
		return fmt.Errorf("%s_MODE must be single, sentinel, or cluster", prefix)
	}

	return nil
}

func loadAddresses(prefix string) ([]string, error) {
	configured := setting(prefix, "ADDRESSES", "")
	if configured == "" {
		host := setting(prefix, "HOST", "")
		port := setting(prefix, "PORT", "")
		if host == "" || port == "" {
			return nil, fmt.Errorf("%s_ADDRESSES or %s_HOST and %s_PORT are required", prefix, prefix, prefix)
		}
		configured = net.JoinHostPort(host, port)
	}

	addresses := make([]string, 0)
	for configuredAddress := range strings.SplitSeq(configured, ",") {
		address := strings.TrimSpace(configuredAddress)
		if address == "" {
			continue
		}
		if _, _, err := net.SplitHostPort(address); err != nil {
			return nil, fmt.Errorf("%s_ADDRESSES contains invalid address %q", prefix, address)
		}
		addresses = append(addresses, address)
	}
	if len(addresses) == 0 {
		return nil, fmt.Errorf("%s_ADDRESSES must contain an address", prefix)
	}

	return addresses, nil
}

func setting(prefix, name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(prefix + "_" + name)); value != "" {
		return value
	}
	return fallback
}

func connectionPrefix(connection string) (string, error) {
	connection = strings.TrimSpace(connection)
	if connection == "" || connection == DefaultConnection {
		return "REDIS", nil
	}

	for _, character := range connection {
		if character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' ||
			character >= '0' && character <= '9' || character == '_' {
			continue
		}
		return "", fmt.Errorf("invalid Redis connection name %q", connection)
	}

	return "REDIS_" + strings.ToUpper(connection), nil
}
