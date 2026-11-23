package routing_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gonstruct/core/config"
	"github.com/gonstruct/core/routing"
	"github.com/gonstruct/core/routing/response"

	"github.com/gin-gonic/gin"
)

func TestFailedResponsesShowTheirErrorOnlyWhenAsked(t *testing.T) {
	for _, show := range []bool{true, false} {
		engine := routing.New(routing.WithConfig(config.Server{ShowErrors: show}), routing.WithRouters(func(engine *routing.Engine) {
			engine.GET("/broken", func(*gin.Context) response.Response {
				return response.Error(errors.New("the database is down"))
			})
		})).Engine()
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/broken", nil))

		if said := strings.Contains(recorder.Body.String(), "the database is down"); said != show {
			t.Errorf("showing errors %v: the response said it %v: %s", show, said, recorder.Body.String())
		}
	}
}
