# Getting started

Jobby orchestrates jobs through MongoDB metadata, Pulsar dispatch, and executor workers. This page gets a full stack running in Docker.

## Prerequisites

Install [Task](https://taskfile.dev/installation/) (`go install` or a distro package). Tasks are declared in [Taskfile.yml](../../Taskfile.yml) at the repo root (`task --list` to discover names).

## Run the stack

```sh
cp .env.example .env
task build
task docker-up
curl http://localhost:3001/health
```

`task docker-up` starts MongoDB (replica set), runs migrations, then brings up Pulsar, the HTTP API, dispatch worker, and executor. See [Compose services](./compose.md) for what each service does.

## Next steps

| Goal | Doc |
|------|-----|
| Run binaries on the host instead of in Docker | [Local development](./local-development.md) |
| Understand env vars and MongoDB URIs | [Environment](./environment.md) |
| Run tests | [Testing](./testing.md) |
| Learn the system design | [Architecture overview](../architecture/intro.md) |
