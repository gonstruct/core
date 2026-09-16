package routing

import "github.com/gonstruct/core/routing/response"

type Option func(*Routing)

func WithRouters(routers ...func(*Engine)) Option {
	return func(r *Routing) {
		r.Routers = append(r.Routers, routers...)
	}
}

func WithHealth(endpoint string) Option {
	return func(r *Routing) {
		r.HealthEndpoint = endpoint
	}
}

func WithTrustedPlatform(platform string) Option {
	return func(r *Routing) {
		r.TrustedPlatform = platform
	}
}

func WithRecovery(handler RecoveryHandler) Option {
	return func(r *Routing) {
		r.Recovery = handler
	}
}

type middlewareOption func() (string, response.HandlerChain)

func WithMiddlewares(middlewares ...middlewareOption) Option {
	return func(r *Routing) {
		if r.Middlewares.Named == nil {
			r.Middlewares.Named = make(map[string]response.HandlerChain)
		}

		for _, middlewareOption := range middlewares {
			name, handlers := middlewareOption()
			if name == "global" {
				r.Middlewares.Global = append(r.Middlewares.Global, handlers...)
				continue
			}

			if _, exists := r.Middlewares.Named[name]; !exists {
				r.Middlewares.Named[name] = make(response.HandlerChain, 0)
			}

			r.Middlewares.Named[name] = append(r.Middlewares.Named[name], handlers...)
		}
	}
}

func WithMiddleware(name string, middlewares ...Middleware) middlewareOption {
	return func() (string, response.HandlerChain) {
		var handlerChain response.HandlerChain
		for _, middleware := range middlewares {
			handlerChain = append(handlerChain, middleware.Handle)
		}

		return name, handlerChain
	}
}
