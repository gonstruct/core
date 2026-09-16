package routing

import (
	"fmt"
	"github.com/gonstruct/core/console"
	"github.com/gonstruct/core/routing/requestid"
	"github.com/gonstruct/core/routing/response"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/gin-gonic/gin"
)

type Routing struct {
	Routers         []func(*Engine)
	Middlewares     MiddlewareEngine
	Recovery        RecoveryHandler
	HealthEndpoint  string
	TrustedPlatform string
}

type RecoveryHandler func(context *gin.Context, recovered any) response.Response

func New(options ...Option) *Routing {
	routing := new(Routing)

	for _, option := range options {
		option(routing)
	}

	return routing
}

func (r *Routing) Engine() *Engine {
	gin.SetMode(gin.ReleaseMode)

	router := NewEngine()

	router.TrustedPlatform = r.TrustedPlatform
	router.ContextWithFallback = true
	router.RemoteIPHeaders = []string{
		"CF-Connecting-IP",
		"X-Forwarded-For", "X-Real-IP",
	}

	if r.Recovery == nil {
		router.Engine.Use(gin.Recovery())
	} else {
		router.Engine.Use(gin.CustomRecovery(func(context *gin.Context, recovered any) {
			response.Handle(func(context *gin.Context) response.Response {
				return r.Recovery(context, recovered)
			})(context)
		}))
	}
	router.Engine.Use(func(context *gin.Context) {
		start := time.Now()
		context.Next()

		for _, contextError := range context.Errors {
			log.Error().
				Ctx(context.Request.Context()).
				Str("request_id", requestid.Get(context)).
				Str("method", context.Request.Method).
				Str("path", context.Request.URL.Path).
				Err(contextError.Err).
				Msg("Request failed")
		}

		console.Request(
			context.Request.Context(),
			context.Request.Method,
			context.Request.URL.Path,
			context.Writer.Status(),
			time.Since(start),
			requestid.Get(context),
		)
	})

	router.Engine.Use(r.Middlewares.Global.Handle()...)
	router.middlewares = r.Middlewares.Named

	router.NoMethod(func(c *gin.Context) response.Response {
		return response.JSON(
			response.WithStatusCode(http.StatusMethodNotAllowed),
			response.WithMessage("Method not allowed"),
		)
	})

	router.NoRoute(func(c *gin.Context) response.Response {
		return response.NotFound(response.WithMessage("Route not found"))
	})

	if r.HealthEndpoint != "" {
		startedAt := time.Now()
		router.GET(r.HealthEndpoint, func(c *gin.Context) response.Response {
			return response.Success(response.WithRaw(map[string]any{
				"status": "healthy",
				"uptime": time.Since(startedAt),
			}))
		})
	}

	for _, f := range r.Routers {
		f(router)
	}

	return router
}

func (r *Routing) Run() error {
	engine := r.Engine()

	host := "0.0.0.0"
	port := 80

	if h := os.Getenv("HOST"); h != "" {
		host = h
	}

	if p := os.Getenv("PORT"); p != "" {
		if p, err := strconv.Atoi(p); err == nil {
			port = p
		}
	}

	hostWithPort := net.JoinHostPort(host, fmt.Sprint(port))
	log.Info().Msgf("Server running on [http://%s].", hostWithPort)
	return engine.Run(hostWithPort)
}
