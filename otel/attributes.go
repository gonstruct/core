package otel

// Attribute names recorded by this service.
//
// These constants are the contract. `packages/otel/src/attributes.ts` mirrors
// them for the Next.js side, so a single trace crossing both halves uses one
// spelling for each concept. Change one, change the other.
//
// Names follow OpenTelemetry semantic conventions where one exists. Where none
// exists the name is ours and is marked as such.

// Resource attributes are set once per process and attached to every span.
const (
	AttributeServiceName         = "service.name"
	AttributeServiceNamespace    = "service.namespace"
	AttributeServiceVersion      = "service.version"
	AttributeServiceInstanceID   = "service.instance.id"
	AttributeDeploymentEnv       = "deployment.environment.name"
	AttributeDeploymentCommitSHA = "deployment.commit_sha" // ours
)

// Request attributes describe one inbound request. Bodies are only ever set on
// a failed response.
const (
	AttributeURLQuery              = "url.query"
	AttributeRequestReferer        = "http.request.header.referer"
	AttributeRequestContentType    = "http.request.header.content_type"
	AttributeRequestAcceptLanguage = "http.request.header.accept_language"
	AttributeRequestID             = "request.id" // ours — correlates spans with logs
	AttributeClientAddress         = "client.address"
	AttributeRequestBody           = "http.request.body"
	AttributeRequestHeaders        = "http.request.headers"
	AttributeResponseBody          = "http.response.body"
	AttributeResponseHeaders       = "http.response.headers"
)

// Actor attributes are supplied by the application through WithAttributes.
// Core never derives them: it has no idea what a session looks like.
const (
	AttributeUserID    = "user.id"
	AttributeUserEmail = "user.email"
)

// Background attributes describe work that no request asked for. Both are
// root spans: nothing upstream started a trace for them.
const (
	AttributeJobName     = "job.name"
	AttributeCommandName = "command.name"
)
