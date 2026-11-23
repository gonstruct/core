package routing

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gonstruct/core/config"
	"github.com/gonstruct/core/console"
	"github.com/gonstruct/core/routing/requestid"
	"github.com/gonstruct/core/routing/response"

	"github.com/rs/zerolog/log"

	"github.com/gin-gonic/gin"
)

// ErrNoConfig is Run without WithConfig.
var ErrNoConfig = errors.New("[routing] no config: pass routing.WithConfig")

type Routing struct {
	Config          *config.Server
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

// Run serves until the process is asked to stop, by SIGTERM as a container
// is or by an interrupt, then takes no new connections and lets the
// requests already in flight finish, for up to the config's
// ShutdownTimeout: a deploy that starts a new container beside this one and
// then stops it drops nothing.
func (r *Routing) Run() error {
	if r.Config == nil {
		return ErrNoConfig
	}
	engine := r.Engine()

	host, port := r.Config.Host, r.Config.Port
	if host == "" {
		host = "0.0.0.0"
	}
	if port == 0 {
		port = 80
	}

	hostWithPort := net.JoinHostPort(host, fmt.Sprint(port))
	server := &http.Server{
		Addr:              hostWithPort,
		Handler:           engine.Handler(),
		ReadHeaderTimeout: 30 * time.Second,
	}

	stopping, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()

	served := make(chan error, 1)
	go func() {
		log.Info().Msgf("Server running on [http://%s].", hostWithPort)
		served <- server.ListenAndServe()
	}()

	select {
	case err := <-served:
		return err
	case <-stopping.Done():
	}

	timeout := r.Config.ShutdownTimeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	log.Info().Msgf("Stopping: finishing the requests in flight, for up to %s.", timeout)
	finishing, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return server.Shutdown(finishing)
}
