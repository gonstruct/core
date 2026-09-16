package otel

import (
	"context"
	"encoding/json"
	"time"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/attribute"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/trace"
)

// LogLevel is the lowest level forwarded to the backend. Everything below it
// stays on stdout only.
//
// Warn and above is deliberate: info and debug are the bulk of the volume and
// almost none of the value once traces cover the request path.
const LogLevel = zerolog.WarnLevel

// zerolog fields that map onto the record itself rather than an attribute.
const (
	fieldMessage = "message"
	fieldLevel   = "level"
	fieldTime    = "time"
	fieldTraceID = "trace_id"
	fieldSpanID  = "span_id"
)

// TraceHook writes the active span onto the event so the record can be put back
// on its trace.
//
// The hand-off has to go through the event because the two halves cannot talk
// directly. A hook can see the context but not the fields; a writer sees the
// fields but never the context, because zerolog hands it finished bytes. So the
// hook writes the ids as ordinary fields and LogWriter reads them back.
//
// The span only reaches the hook when the call site passes it:
//
//	log.Warn().Ctx(context).Err(err).Msg("...")
//
// Without .Ctx the record still ships, just uncorrelated, which is correct for
// anything logged outside a request.
type TraceHook struct{}

func (TraceHook) Run(event *zerolog.Event, level zerolog.Level, _ string) {
	if level < LogLevel {
		return
	}

	span := trace.SpanContextFromContext(event.GetCtx())
	if !span.IsValid() {
		return
	}

	event.Str(fieldTraceID, span.TraceID().String())
	event.Str(fieldSpanID, span.SpanID().String())
}

// LogWriter forwards zerolog records to the collector as OTLP logs.
//
// It is a zerolog.LevelWriter rather than a zerolog.Hook on purpose. A Hook
// cannot read the event's fields (rs/zerolog#493), so the official bridge
// ships the message and nothing else. A writer receives the finished JSON,
// fields included, so everything survives.
type LogWriter struct{}

// NewLogWriter builds the writer. Wrap it around the console writer with
// zerolog.MultiLevelWriter so records go to both.
//
// It resolves the instance per record rather than holding one, for the same
// reason TracingMiddleware does: the log provider runs before the otel
// provider, so there is nothing to hold yet.
func NewLogWriter() *LogWriter {
	return &LogWriter{}
}

// Write handles records written without a level. zerolog only calls this for
// output that bypassed the level system, which is nothing we want to ship.
func (self *LogWriter) Write(payload []byte) (int, error) {
	return len(payload), nil
}

func (self *LogWriter) WriteLevel(level zerolog.Level, payload []byte) (int, error) {
	if level < LogLevel {
		return len(payload), nil
	}

	instance := Instance()
	if !instance.Enabled() {
		return len(payload), nil
	}

	var fields map[string]any
	if err := json.Unmarshal(payload, &fields); err != nil {
		// A record we cannot parse is not worth failing the write for: the
		// console writer has already printed it.
		return len(payload), nil //nolint:nilerr // A record we cannot parse must not fail the write.
	}

	record := otellog.Record{}
	record.SetTimestamp(timestampOf(fields))
	record.SetSeverity(severityOf(level))
	record.SetSeverityText(level.String())

	if message, ok := fields[fieldMessage].(string); ok {
		record.SetBody(attribute.StringValue(message))
	}

	record.AddAttributes(attributesOf(fields)...)

	global.GetLoggerProvider().Logger(instance.Service()).Emit(spanContextOf(fields), record)

	return len(payload), nil
}

// spanContextOf rebuilds the context TraceHook took the ids from, so the SDK
// stamps the record with the span it belongs to.
//
// Sampled is set because an unsampled record would be dropped, and a record
// only carries ids at all when its span was recording.
func spanContextOf(fields map[string]any) context.Context {
	rawTrace, hasTrace := fields[fieldTraceID].(string)
	rawSpan, hasSpan := fields[fieldSpanID].(string)
	if !hasTrace || !hasSpan {
		return context.Background()
	}

	traceID, err := trace.TraceIDFromHex(rawTrace)
	if err != nil {
		return context.Background()
	}

	spanID, err := trace.SpanIDFromHex(rawSpan)
	if err != nil {
		return context.Background()
	}

	return trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	}))
}

// attributesOf carries every remaining zerolog field onto the record. This is
// the whole reason for writing our own bridge: err, job name, user id and the
// rest are the context that makes a warning actionable.
func attributesOf(fields map[string]any) []attribute.KeyValue {
	attributes := make([]attribute.KeyValue, 0, len(fields))

	for key, value := range fields {
		switch key {
		case fieldMessage, fieldLevel, fieldTime:
			continue
		// The ids are on the record itself; repeating them as attributes would
		// duplicate them in the backend.
		case fieldTraceID, fieldSpanID:
			continue
		}

		attributes = append(attributes, attribute.KeyValue{
			Key:   attribute.Key(key),
			Value: logValueOf(value),
		})
	}

	return attributes
}

func logValueOf(value any) attribute.Value {
	switch typed := value.(type) {
	case string:
		return attribute.StringValue(typed)
	case bool:
		return attribute.BoolValue(typed)
	case float64:
		// All JSON numbers decode as float64. Keep whole numbers as integers so
		// they are not rendered as 1.409e+03 in the backend.
		if typed == float64(int64(typed)) {
			return attribute.Int64Value(int64(typed))
		}

		return attribute.Float64Value(typed)
	default:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return attribute.StringValue("")
		}

		return attribute.StringValue(string(encoded))
	}
}

func timestampOf(fields map[string]any) time.Time {
	raw, ok := fields[fieldTime].(string)
	if !ok {
		return time.Now()
	}

	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Now()
	}

	return parsed
}

func severityOf(level zerolog.Level) otellog.Severity {
	switch level {
	case zerolog.WarnLevel:
		return otellog.SeverityWarn
	case zerolog.ErrorLevel:
		return otellog.SeverityError
	case zerolog.FatalLevel:
		return otellog.SeverityFatal
	case zerolog.PanicLevel:
		return otellog.SeverityFatal4
	default:
		return otellog.SeverityInfo
	}
}
