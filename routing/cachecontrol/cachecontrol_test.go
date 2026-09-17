package cachecontrol_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gonstruct/core/routing/cachecontrol"

	"github.com/gin-gonic/gin"
)

func request(t *testing.T, handlers ...gin.HandlerFunc) http.Header {
	t.Helper()
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.GET("/", handlers...)

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	return recorder.Header()
}

func TestPrivateStopsSharedCaches(t *testing.T) {
	header := request(t, func(context *gin.Context) {
		cachecontrol.Private(context)
		context.Status(http.StatusOK)
	})

	if directive := header.Get("Cache-Control"); directive != "private, no-store" {
		t.Errorf("expected private, no-store, got %q", directive)
	}

	if vary := header.Get("Vary"); vary != "Cookie" {
		t.Errorf("expected Vary: Cookie, got %q", vary)
	}
}

// CORS writes Vary: Origin before the route runs. Replacing the header instead
// of appending would drop it and let one origin's response answer another.
func TestPrivateKeepsAnExistingVary(t *testing.T) {
	header := request(t, func(context *gin.Context) {
		context.Header("Vary", "Origin")
		cachecontrol.Private(context)
		context.Status(http.StatusOK)
	})

	values := header.Values("Vary")
	if len(values) != 2 || values[0] != "Origin" || values[1] != "Cookie" {
		t.Errorf("expected Origin and Cookie, got %v", values)
	}
}

func TestAddVaryIgnoresAFieldAlreadyPresent(t *testing.T) {
	header := request(t, func(context *gin.Context) {
		context.Header("Vary", "Origin, Cookie")
		cachecontrol.AddVary(context, "cookie")
		context.Status(http.StatusOK)
	})

	if values := header.Values("Vary"); len(values) != 1 {
		t.Errorf("expected the header to be left alone, got %v", values)
	}
}

// The media proxy serves signed URLs that every visitor may share. It sets its
// own Cache-Control after the middleware, and that has to win.
func TestARouteCanOptBackIntoCaching(t *testing.T) {
	header := request(t,
		func(context *gin.Context) {
			cachecontrol.Private(context)
			context.Next()
		},
		func(context *gin.Context) {
			context.Header("Cache-Control", "public, max-age=86400, immutable")
			context.Status(http.StatusOK)
		},
	)

	if directive := header.Get("Cache-Control"); directive != "public, max-age=86400, immutable" {
		t.Errorf("expected the route directive to replace the default, got %q", directive)
	}
}
