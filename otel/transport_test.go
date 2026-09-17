package otel_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	coreotel "github.com/gonstruct/core/otel"

	otelapi "go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// recording gives the transport a live span to write to. Without one
// IsRecording is false and the whole capture path is skipped — which is
// exactly how an earlier version of these tests passed while testing nothing.
func recording(t *testing.T) (context.Context, *tracetest.SpanRecorder) {
	t.Helper()

	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	otelapi.SetTracerProvider(provider)

	coreotel.Bind(coreotel.New(
		coreotel.WithEnabled(true),
		coreotel.WithEndpoint("localhost:4318"),
		coreotel.WithService("example", "api"),
	))

	ctx, span := provider.Tracer("test").Start(context.Background(), "caller")
	t.Cleanup(func() { span.End() })

	return ctx, recorder
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func respond(status int, body string) http.RoundTripper {
	return roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: status,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{"Content-Type": {"application/json"}, "Authorization": {"Bearer hunter2"}},
			Request:    r,
		}, nil
	})
}

// The caller must read exactly what it would have without the transport in the
// way — recording a prefix must not consume it.
func TestTransportRestoresTheBody(t *testing.T) {
	body := `{"errors":{"ids[0]":"must be a valid UUIDv4"}}`

	client := &http.Client{Transport: coreotel.NewTransport(respond(http.StatusBadRequest, body))}

	request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.test/x", nil)

	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer response.Body.Close()

	read, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}

	if string(read) != body {
		t.Errorf("body was altered\n got: %s\nwant: %s", read, body)
	}
}

// A successful call must cost nothing: the outgoing body is never re-read and
// no attribute is recorded. This is the guarantee that makes it safe to wrap
// every client rather than only the ones being debugged.
func TestTransportReadsNothingOnSuccess(t *testing.T) {
	reread := 0

	ctx, _ := recording(t)

	request, _ := http.NewRequestWithContext(ctx, http.MethodPost, "http://example.test/x", strings.NewReader(`{"a":1}`))
	original := request.GetBody
	request.GetBody = func() (io.ReadCloser, error) {
		reread++

		return original()
	}

	response, err := (&http.Client{Transport: coreotel.NewTransport(respond(http.StatusOK, `{"ok":true}`))}).Do(request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer response.Body.Close()

	if reread != 0 {
		t.Errorf("the outgoing body was re-read %d times on a successful call", reread)
	}
}

// And on failure it is read exactly once.
func TestTransportReadsTheRequestOnceOnFailure(t *testing.T) {
	reread := 0

	ctx, _ := recording(t)

	request, _ := http.NewRequestWithContext(ctx, http.MethodPost, "http://example.test/x", strings.NewReader(`{"a":1}`))
	original := request.GetBody
	request.GetBody = func() (io.ReadCloser, error) {
		reread++

		return original()
	}

	response, err := (&http.Client{Transport: coreotel.NewTransport(respond(http.StatusBadRequest, `{"error":"no"}`))}).Do(request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer response.Body.Close()

	if reread != 1 {
		t.Errorf("expected the outgoing body to be re-read once, got %d", reread)
	}
}

// A successful response is never buffered, and must still arrive intact.
func TestTransportLeavesSuccessAlone(t *testing.T) {
	body := `{"data":{"id":"1"}}`

	client := &http.Client{Transport: coreotel.NewTransport(respond(http.StatusOK, body))}

	request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.test/x", nil)

	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer response.Body.Close()

	read, _ := io.ReadAll(response.Body)
	if string(read) != body {
		t.Errorf("body was altered\n got: %s\nwant: %s", read, body)
	}
}

// The outgoing body must reach the server untouched: it is re-read through
// GetBody, never by consuming request.Body.
func TestTransportDoesNotConsumeTheRequestBody(t *testing.T) {
	sent := ""

	inner := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		read, _ := io.ReadAll(r.Body)
		sent = string(read)

		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(strings.NewReader(`{"error":"nope"}`)),
			Header:     http.Header{},
			Request:    r,
		}, nil
	})

	payload := `{"prompt":"hello","password":"hunter2"}`
	request, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "http://example.test/x", strings.NewReader(payload))

	response, err := (&http.Client{Transport: coreotel.NewTransport(inner)}).Do(request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer response.Body.Close()

	if sent != payload {
		t.Errorf("the server received an altered body\n got: %s\nwant: %s", sent, payload)
	}
}

// A transport error has no response, and must not panic on the nil.
func TestTransportHandlesATransportError(t *testing.T) {
	inner := roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("dial tcp: connection refused")
	})

	request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.test/x", nil)

	response, err := (&http.Client{Transport: coreotel.NewTransport(inner)}).Do(request)
	if err == nil {
		t.Error("expected the transport error to reach the caller")
	}

	if response != nil {
		_ = response.Body.Close()
	}
}

// What is captured must go through the same masker as the inbound middleware.
// A failed third-party call is exactly where a key or a password would
// otherwise be written to a span verbatim.
func TestTransportMasksWhatItCaptures(t *testing.T) {
	ctx, recorder := recording(t)

	inner := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(strings.NewReader(`{"error":"bad","access_token":"leaked-token"}`)),
			Header:     http.Header{"Authorization": {"Bearer hunter2"}},
			Request:    r,
		}, nil
	})

	request, _ := http.NewRequestWithContext(ctx, http.MethodPost, "http://example.test/x",
		strings.NewReader(`{"username":"arjen","password":"hunter2"}`))
	request.Header.Set("X-Api-Key", "secret-key")

	response, err := (&http.Client{Transport: coreotel.NewTransport(inner)}).Do(request)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = response.Body.Close()

	trace.SpanFromContext(ctx).End()

	spans := recorder.Ended()
	if len(spans) == 0 {
		t.Fatal("expected the caller's span to be recorded")
	}

	builder := strings.Builder{}
	for _, attribute := range spans[0].Attributes() {
		builder.WriteString(string(attribute.Key) + "=" + attribute.Value.AsString() + "\n")
	}

	recorded := builder.String()

	for _, secret := range []string{"hunter2", "leaked-token", "secret-key"} {
		if strings.Contains(recorded, secret) {
			t.Errorf("%q was written to the span verbatim:\n%s", secret, recorded)
		}
	}

	if !strings.Contains(recorded, "REDACTED") {
		t.Errorf("expected redaction to have happened, got:\n%s", recorded)
	}
}
