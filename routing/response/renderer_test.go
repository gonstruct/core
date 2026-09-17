package response_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gonstruct/core/routing/response"
	"github.com/gonstruct/core/routing/response/exception"

	"github.com/gin-gonic/gin"
)

var registerNotFoundRenderer sync.Once

func registerNotFoundExceptionRenderer() {
	registerNotFoundRenderer.Do(func() {
		response.RenderExceptionIs(sql.ErrNoRows, func(input exception.Exception) response.Response {
			subject := input.SubjectOr("resource")

			return response.NotFound(
				response.WithMessageF("The %s was not found", subject),
				response.WithCode(subject+"_not_found"),
			)
		})
	})
}

func render(t *testing.T, handler response.Handler) (int, map[string]any) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.GET("/", response.Handle(handler))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	engine.ServeHTTP(recorder, request)

	body := map[string]any{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	return recorder.Code, body
}

func TestExceptionUsesMatchingRendererAndCallSiteContext(t *testing.T) {
	registerNotFoundExceptionRenderer()

	status, body := render(t, func(context *gin.Context) response.Response {
		return response.Exception(fmt.Errorf("load user: %w", sql.ErrNoRows), exception.WithSubject("user"))
	})

	if status != http.StatusNotFound {
		t.Fatalf("expected %d, got %d", http.StatusNotFound, status)
	}
	if message := body["message"]; message != "The user was not found" {
		t.Fatalf("unexpected message: %q", message)
	}
	if code := body["code"]; code != "user_not_found" {
		t.Fatalf("unexpected code: %q", code)
	}
}

func TestErrorPreservesExplicitResponseOptionsAfterRendering(t *testing.T) {
	registerNotFoundExceptionRenderer()

	status, body := render(t, func(context *gin.Context) response.Response {
		return response.Error(sql.ErrNoRows, response.WithMessage("The session is no longer valid"))
	})

	if status != http.StatusNotFound {
		t.Fatalf("expected %d, got %d", http.StatusNotFound, status)
	}
	if message := body["message"]; message != "The session is no longer valid" {
		t.Fatalf("expected explicit message to be retained, got %q", message)
	}
}

func TestErrorCodeIsIncludedInTheJSONResponse(t *testing.T) {
	status, body := render(t, func(context *gin.Context) response.Response {
		return response.Conflict(
			response.WithMessage("An account with this email already exists."),
			response.WithCode("email_exists"),
		)
	})

	if status != http.StatusConflict {
		t.Fatalf("expected %d, got %d", http.StatusConflict, status)
	}
	if code := body["code"]; code != "email_exists" {
		t.Fatalf("unexpected error code: %#v", body["code"])
	}
}

func TestErrorsSummarizesTheFirstFieldMessage(t *testing.T) {
	_, body := render(t, func(context *gin.Context) response.Response {
		return response.Errors(response.ErrorsMap{
			"password": "The password field is required",
			"email":    "The email field must be a valid email address",
		})
	})

	message, _ := body["message"].(string)
	expected := "The email field must be a valid email address and 1 more errors"
	if message != expected {
		t.Fatalf("expected summary %q, got %q", expected, message)
	}
}

func TestErrorsDoesNotOverrideAnExplicitMessage(t *testing.T) {
	_, body := render(t, func(context *gin.Context) response.Response {
		return response.Errors(
			response.ErrorsMap{"email": "The email field must be a valid email address"},
			response.WithMessage("Please correct the highlighted fields"),
		)
	})

	if message := body["message"]; message != "Please correct the highlighted fields" {
		t.Fatalf("expected explicit message to be retained, got %q", message)
	}
}
