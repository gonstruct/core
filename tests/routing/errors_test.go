package routing_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gonstruct/core/routing"
	"github.com/gonstruct/core/routing/response"

	"github.com/gin-gonic/gin"
)

// A failed response says the error behind it only when the app shows
// errors, as it does outside production.
func TestFailedResponsesShowTheirErrorOnlyWhenAsked(t *testing.T) {
	t.Cleanup(func() { response.ShowErrors(false) })
	engine := routing.New(routing.WithRouters(func(engine *routing.Engine) {
		engine.GET("/broken", func(*gin.Context) response.Response {
			return response.Error(errors.New("the database is down"))
		})
	})).Engine()

	for _, show := range []bool{true, false} {
		response.ShowErrors(show)
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/broken", nil))

		if said := strings.Contains(recorder.Body.String(), "the database is down"); said != show {
			t.Errorf("showing errors %v: the response said it %v: %s", show, said, recorder.Body.String())
		}
	}
}
