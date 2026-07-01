# Jobby

Distributed job orchestration in Go: HTTP/CLI enqueue → MongoDB metadata → Pulsar dispatch → executor workers. Built from hard-won lessons with job queues.

I once worked on a system that had to interact heavily with a job queueing application. It was terrible. This project is inspired by the lessons from those days.

## Quick start

Requires [Task](https://taskfile.dev/installation/).

```bash
cp .env.example .env
task docker-up
curl http://localhost:3001/health
```

See [Getting started](docs/dev/getting-started.md) for host-native workflows and [Testing](docs/dev/testing.md) for running tests.

## Documentation

> Note: These docs are written for AI-assisted software engineering as well as humans, so some pages are intentionally detailed or verbose.

- [Documentation index](docs/README.md)
- [Architecture overview](docs/architecture/intro.md)
- [Development guides](docs/dev/getting-started.md)
