package exception

// Exception is the raw error and neutral context passed to registered
// renderers. Renderers decide the HTTP status and response shape; context only
// adds details that make their response more useful.
type Exception struct {
	Err     error
	Subject string
	Meta    map[string]any
}

func (exception Exception) SubjectOr(fallback string) string {
	if exception.Subject != "" {
		return exception.Subject
	}

	return fallback
}

type Option func(*Exception)

func WithSubject(subject string) Option {
	return func(exception *Exception) {
		exception.Subject = subject
	}
}

func WithMeta(key string, value any) Option {
	return func(exception *Exception) {
		if exception.Meta == nil {
			exception.Meta = map[string]any{}
		}

		exception.Meta[key] = value
	}
}

func New(err error, options ...Option) Exception {
	exception := Exception{Err: err}
	for _, option := range options {
		option(&exception)
	}

	return exception
}
