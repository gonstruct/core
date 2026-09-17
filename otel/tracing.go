package otel

import (
	"net/http"
	"slices"
	"strings"

	"github.com/gonstruct/core/otel/httpcapture"
	"github.com/gonstruct/core/routing/requestid"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Tracing returns the gin handler that records a server span for each request.
//
// It is a no-op when no endpoint is configured, so routes can register it
// unconditionally.
func (self *Otel) Tracing() gin.HandlerFunc {
	if !self.Enabled() {
		return func(context *gin.Context) { context.Next() }
	}

	return func(context *gin.Context) {
		var request *httpcapture.Request
		var response *httpcapture.Response

		if self.captureBody {
			request = httpcapture.NewRequest(context, self.maxBodySize)
			response = httpcapture.NewResponse(context.Writer, self.maxBodySize)
			context.Writer = response
		}

		otelgin.Middleware(self.Service(),
			otelgin.WithSpanNameFormatter(spanName),
			otelgin.WithFilter(traceable),

			// otelgin exposes no hook for span attributes, only metric ones.
			// This callback runs after the handler and before the span ends,
			// and fires whenever it is non-nil — it does not depend on a meter
			// provider existing. It is the only point where both the response
			// status and a live span are available.
			otelgin.WithGinMetricAttributeFn(func(context *gin.Context) []attribute.KeyValue {
				span := trace.SpanFromContext(context.Request.Context())
				if !span.IsRecording() {
					return nil
				}

				span.SetAttributes(self.attributes(context, request, response)...)

				return nil
			}),
		)(context)
	}
}

func (self *Otel) attributes(
	context *gin.Context,
	request *httpcapture.Request,
	response *httpcapture.Response,
) []attribute.KeyValue {
	attributes := []attribute.KeyValue{
		attribute.String(AttributeURLQuery, context.Request.URL.RawQuery),
		attribute.String(AttributeRequestReferer, context.Request.Referer()),
		attribute.String(AttributeRequestContentType, context.Request.Header.Get("Content-Type")),
		attribute.String(AttributeRequestAcceptLanguage, context.Request.Header.Get("Accept-Language")),
		attribute.String(AttributeRequestID, requestid.Ensure(context)),
		attribute.String(AttributeClientAddress, clientAddress(context)),
	}

	if context.Writer.Status() >= http.StatusBadRequest {
		if request != nil {
			attributes = self.appendBody(attributes, AttributeRequestBody, request.Body())
		}

		if response != nil {
			attributes = self.appendBody(attributes, AttributeResponseBody, response.Body())
		}
	}

	if self.attributeFn != nil {
		attributes = append(attributes, self.attributeFn(context)...)
	}

	return attributes
}

// clientAddress is whoever actually made the request.
//
// otelgin records the socket peer, which behind Cloudflare is the edge and on
// the internal network is the calling container — neither is the visitor. The
// headers are tried in order of trust: Cloudflare's own first, then the
// standard forwarding chain, whose first entry is the original client.
func clientAddress(context *gin.Context) string {
	if address := context.GetHeader("CF-Connecting-IP"); address != "" {
		return address
	}

	if forwarded := context.GetHeader("X-Forwarded-For"); forwarded != "" {
		if address := strings.TrimSpace(strings.SplitN(forwarded, ",", 2)[0]); address != "" {
			return address
		}
	}

	if address := context.GetHeader("X-Real-IP"); address != "" {
		return address
	}

	return context.ClientIP()
}

func (self *Otel) appendBody(attributes []attribute.KeyValue, key string, body []byte) []attribute.KeyValue {
	if len(body) == 0 {
		return attributes
	}

	masked := self.masker.Mask(body)
	if masked == "" {
		return attributes
	}

	return append(attributes, attribute.String(key, masked))
}

// traceable skips requests that would only add noise: the health probe fires
// constantly and preflight requests carry no application context.
func traceable(request *http.Request) bool {
	return request.URL.Path != "/health" && request.Method != http.MethodOptions
}

// spanName uses the route template rather than the resolved path, so
// /users/1 and /users/2 group into one operation instead of thousands.
func spanName(context *gin.Context) string {
	method := strings.ToUpper(context.Request.Method)

	if !slices.Contains([]string{
		http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut,
		http.MethodPatch, http.MethodDelete, http.MethodConnect,
		http.MethodOptions, http.MethodTrace,
	}, method) {
		method = "HTTP"
	}

	if path := context.FullPath(); path != "" {
		return method + " " + path
	}

	return method + " <unmatched>"
}
