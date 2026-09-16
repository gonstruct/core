package otel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gonstruct/core/otel/masking"
	"io"
	"net/http"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// NewTransport records what a failed outbound call actually said.
//
// otelhttp gives the method, url, status and duration of a client call and
// deliberately stops there — the HTTP semantic conventions leave bodies out
// because they are unbounded and routinely carry credentials. That is the right
// default for every call that succeeds and useless for the one that does not: a
// 400 from a third party is only actionable if you can see the reason it gave.
//
// So this mirrors the inbound middleware rather than inventing a second
// convention. Nothing is buffered unless the response failed, the recorded copy
// is capped at the same limit and run through the same masker, and the body is
// put back so the caller reads it exactly as it would have.
//
// It goes *inside* otelhttp, which is what puts the client span in the context
// this reads from:
//
//	&http.Client{Transport: otelhttp.NewTransport(otel.NewTransport(nil))}
//
// The instance is resolved per request, not here. These clients are package
// level, so they are built while the binary is still initialising — before the
// provider binds telemetry. Capturing the instance at construction would take
// the disabled fallback and never record anything.
func NewTransport(next http.RoundTripper) http.RoundTripper {
	if next == nil {
		next = http.DefaultTransport
	}

	return &transport{next: next}
}

type transport struct {
	next http.RoundTripper
}

func (self *transport) RoundTrip(request *http.Request) (*http.Response, error) {
	response, err := self.next.RoundTrip(request)

	failed := err != nil || (response != nil && response.StatusCode >= http.StatusBadRequest)
	if !failed {
		return response, err
	}

	span := trace.SpanFromContext(request.Context())
	if !span.IsRecording() {
		return response, err
	}

	instance := Instance()
	attributes := make([]attribute.KeyValue, 0, 4)

	if headers := self.headers(instance, request.Header); headers != "" {
		attributes = append(attributes, attribute.String(AttributeRequestHeaders, headers))
	}

	if body := instance.masker.Mask(self.requestBody(instance, request)); body != "" {
		attributes = append(attributes, attribute.String(AttributeRequestBody, body))
	}

	// A transport error has no response to read — the span still says so.
	if err != nil {
		attributes = append(attributes, attribute.String(AttributeResponseBody, fmt.Sprintf("transport error: %v", err)))
		span.SetAttributes(attributes...)

		return response, err
	}

	if headers := self.headers(instance, response.Header); headers != "" {
		attributes = append(attributes, attribute.String(AttributeResponseHeaders, headers))
	}

	// Only the first maxBodySize bytes are held. The caller still gets the
	// whole body: what was read is handed back in front of whatever is left, so
	// a large response is never buffered in full just to record a prefix.
	if response.Body != nil {
		buffered, _ := io.ReadAll(io.LimitReader(response.Body, int64(instance.maxBodySize)))
		response.Body = restore(buffered, response.Body)

		if body := instance.masker.Mask(buffered); body != "" {
			attributes = append(attributes, attribute.String(AttributeResponseBody, body))
		}
	}

	span.SetAttributes(attributes...)

	return response, nil
}

// requestBody re-reads the outgoing body through GetBody, which hands back a
// fresh reader over the same bytes. Reading request.Body directly would consume
// what the transport is about to send.
func (self *transport) requestBody(instance *Otel, request *http.Request) []byte {
	if request.Body == nil || request.GetBody == nil {
		return nil
	}

	reader, err := request.GetBody()
	if err != nil {
		return nil
	}
	defer reader.Close()

	body, _ := io.ReadAll(io.LimitReader(reader, int64(instance.maxBodySize)))

	return body
}

// headers go through the same masker as bodies, so Authorization and the
// various api-key headers are redacted by the one key list.
func (self *transport) headers(instance *Otel, header http.Header) string {
	if len(header) == 0 {
		return ""
	}

	flattened := make(map[string]any, len(header))
	for key, values := range header {
		if instance.masker.IsSensitive(key) {
			flattened[key] = masking.Redacted

			continue
		}

		flattened[key] = strings.Join(values, ", ")
	}

	raw, err := json.Marshal(flattened)
	if err != nil {
		return ""
	}

	return instance.masker.Mask(raw)
}

type restored struct {
	io.Reader
	io.Closer
}

func restore(buffered []byte, rest io.ReadCloser) io.ReadCloser {
	return restored{Reader: io.MultiReader(bytes.NewReader(buffered), rest), Closer: rest}
}
