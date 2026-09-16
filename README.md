# gonstruct/core

The framework behind the terrabyte-studio Go APIs: boot, routing, queueing,
scheduling, eventing, cache, console, migrations, seeders, factories,
authentication and authorization facades, OpenTelemetry, and a few
standalone packages. Laravel-shaped, Go-native.

> **Not stable. Not versioned in any meaningful way yet.**
>
> This is the `apps/api/core` directory of the apps that use it, lifted out
> so it is maintained in one place instead of copied per project. Every
> package can change shape between commits, packages will move and merge,
> and nothing here promises backwards compatibility. Pin a commit, expect to
> follow changes, and do not build on it outside terrabyte-studio yet.

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
| `config` | Configuration types; values live in the app |
| `database/migration` | goose migrations with an environment guard |
| `facades/*` | Package-level access with an adapter behind it: authentication, authorization, database factories and seeders |
| `otel` | Tracing, logging, masking, HTTP capture, middleware and transports |
| `packages/*` | Standalone libraries that happen to live here for now: `socialite`. HTTP clients use [gonstruct/hyper](https://github.com/gonstruct/hyper) |

Storage, encryption and mail are in [gonstruct/providers](https://github.com/gonstruct/providers).
Env and payload validation is [gonstruct/validation](https://github.com/gonstruct/validation).

## Using it

```go
import "github.com/gonstruct/core/routing"
```

The repository is private for now, so builds need `GOPRIVATE=github.com/gonstruct`
and git access to the organisation.

## Where it comes from

Extracted from `terrabyte-studio/contentpad`, which carried the most recent
copy, with the packages only `terrabyte-studio/foundation` had added on top.
Both apps are expected to delete their `core` directory and depend on this
module. Until they do, this repository is the source of truth and the copies
are downstream.
