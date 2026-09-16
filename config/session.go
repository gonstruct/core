package config

import (
	nativeHttp "net/http"
	"time"
)

type sessionDriver string

const (
	SessionDriverMemory   sessionDriver = "memory"
	SessionDriverDatabase sessionDriver = "database"
)

func (sessionDriver) Values() []sessionDriver {
	return []sessionDriver{
		SessionDriverMemory,
		SessionDriverDatabase,
	}
}

func (driver sessionDriver) String() string {
	return string(driver)
}

type sameSite string

const (
	SameSiteLax    sameSite = "lax"
	SameSiteStrict sameSite = "strict"
	SameSiteNone   sameSite = "none"
)

func (sameSite) Values() []sameSite {
	return []sameSite{
		SameSiteLax,
		SameSiteStrict,
		SameSiteNone,
	}
}

func (site sameSite) String() string {
	return string(site)
}

func (site sameSite) Native() nativeHttp.SameSite {
	switch site {
	case SameSiteLax:
		return nativeHttp.SameSiteLaxMode
	case SameSiteStrict:
		return nativeHttp.SameSiteStrictMode
	case SameSiteNone:
		return nativeHttp.SameSiteNoneMode
	default:
		return nativeHttp.SameSiteLaxMode // Default to Lax if not specified
	}
}

type Session struct {
	Driver          sessionDriver
	Lifetime        time.Duration
	RefreshLifetime time.Duration
	Path            string
	RefreshPath     string
	Domain          string
	Secure          bool
	HttpOnly        bool
	SameSite        sameSite
}
