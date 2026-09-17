package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/gonstruct/core/console"

	"github.com/rs/zerolog/log"
)

func Providers(provs ...provider) providers {
	return provs
}

type provider interface {
	Register(context context.Context) error
}

type providerWithBoot interface {
	Boot(context context.Context) error
}

type providerWithShutdown interface {
	Shutdown(context context.Context) error
}

type providers []provider

func (providers providers) Register(context context.Context) error {
	console.Section("Registering providers")
	for _, provider := range providers {
		normalizedName := strings.TrimSuffix(strings.TrimPrefix(fmt.Sprintf("%T", provider), "*providers."), "Provider")

		if err := console.Task(fmt.Sprintf("Registering provider: %v", normalizedName), func() error {
			return provider.Register(context)
		}); err != nil {
			return err
		}
	}
	return nil
}

func (providers providers) Boot(context context.Context) error {
	var bootableProviders []providerWithBoot
	for _, provider := range providers {
		if bootable, ok := provider.(providerWithBoot); ok {
			bootableProviders = append(bootableProviders, bootable)
		}
	}

	if len(bootableProviders) == 0 {
		log.Info().Msg("No providers to boot")
		return nil
	}

	console.Section("Booting providers")
	for _, provider := range bootableProviders {
		normalizedName := strings.TrimSuffix(strings.TrimPrefix(fmt.Sprintf("%T", provider), "*providers."), "Provider")

		if err := console.Task(fmt.Sprintf("Booting provider: %v", normalizedName), func() error {
			return provider.Boot(context)
		}); err != nil {
			return err
		}
	}
	return nil
}

func (providers providers) Shutdown(context context.Context) {
	var shutdownableProviders []providerWithShutdown
	for _, provider := range providers {
		if shutdownable, ok := provider.(providerWithShutdown); ok {
			shutdownableProviders = append(shutdownableProviders, shutdownable)
		}
	}

	if len(shutdownableProviders) == 0 {
		log.Info().Msg("No providers to shutdown")
		return
	}

	console.Section("Shutting down providers")
	for _, provider := range shutdownableProviders {
		normalizedName := strings.TrimSuffix(strings.TrimPrefix(fmt.Sprintf("%T", provider), "*providers."), "Provider")

		if err := console.Task(fmt.Sprintf("Shutting down provider: %v", normalizedName), func() error {
			return provider.Shutdown(context)
		}); err != nil {
			log.Error().Err(err).Msgf("Failed to shutdown provider %v", normalizedName)
		}
	}
}
