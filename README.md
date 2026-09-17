# core

The framework for Go APIs: boot, routing, queueing, scheduling, eventing,
cache, console, migrations, seeders, factories, authentication and
authorization facades, and OpenTelemetry. Laravel-shaped, Go-native.

Early. Packages can still change shape between commits; pin a commit and
expect to follow changes.

## Layout

One package per concern, all versioned together:

| Package | What it is |
|---|---|
| `app` | Assembles an application from providers, database, routing, queueing, scheduling and eventing; runs console commands |
| `routing` | Gin-based HTTP engine: request parsing and validation, resources, responses, exceptions, cache control |
| `queueing` | Jobs, queues, middleware, redis and sync resolvers |
| `scheduling` | Cron-style command schedules |
| `eventing` | Typed events and subscribers |
| `cache`, `redis`, `ratelimit` | Cache engine with resolvers, redis configuration, rate limiting |
| `console` | Command registry and console output |
| `config` | Configuration types; values live in the application |
| `database/migration` | goose migrations with an environment guard |
| `facades/*` | Package-level access with an adapter behind it: authentication, authorization, database factories and seeders |
| `otel` | Tracing, logging, masking, HTTP capture, middleware and transports |

Nothing else lives here. Standalone concerns are their own modules:
[hyper](https://github.com/gonstruct/hyper) for HTTP clients,
[social](https://github.com/gonstruct/social) for signing in with another
account, [providers](https://github.com/gonstruct/providers) for storage,
encryption and mail, [validation](https://github.com/gonstruct/validation)
for env and payload validation, [sluggable](https://github.com/gonstruct/sluggable)
for slugs.

## Using it

```go
import "github.com/gonstruct/core/routing"
```
