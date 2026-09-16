package config

import (
	"log"
	"os"
	"path/filepath"

	"github.com/gonstruct/validation/env"
	"github.com/joho/godotenv"
)

func init() {
	root := new(envHelpers).root()

	appEnv := os.Getenv("APP_ENV")

	loaded := func() bool {
		if appEnv == "" {
			log.Printf("[config] APP_ENV not set, default loading enabled")
			return new(envHelpers).load(root, ".env.production", ".env.staging", ".env.local", ".env")
		}

		if new(envHelpers).load(root, ".env."+appEnv, ".env") {
			return true
		}

		log.Printf("[config] unknown environment %q, skipping environment variable loading from .env file(s)", env.Enum("APP_ENV", LocalEnvironment))
		return false
	}()

	if loaded {
		log.Printf("[config] loaded environment variables from .env file(s) for environment %q", appEnv)
	}

	if !loaded {
		log.Println("[config] no .env file found in project root, skipping environment variable loading")
	}
}

type envHelpers struct{}

func (envHelpers) root() string {
	dir, _ := os.Getwd()

	for dir != filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		dir = filepath.Dir(dir)
	}

	return "."
}

func (envHelpers) load(root string, names ...string) bool {
	var paths []string
	for _, name := range names {
		path := filepath.Join(root, name)
		if _, err := os.Stat(path); err != nil {
			log.Printf("[config] environment file %q not found, skipping", path)
			continue
		}

		paths = append(paths, path)
	}

	if len(paths) == 0 {
		return false
	}

	if err := godotenv.Load(paths...); err != nil {
		return false
	}

	return true
}
