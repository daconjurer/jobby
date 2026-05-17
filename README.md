Jobby
======

I once worked on a system that had to interact heavily with a job queueing application.
It was terrible.

This project is inspired by the lessons from those days.

## Configuration

Application settings are read from the process environment with [`internal/config`](./internal/config/README.md). After parsing, configs are **validated** (pool bounds, timeouts, listen port).

## Quick Start

```bash
# Run tests
make test

# Start MongoDB
make mongo-up

# Run application
make run
```

## Documentation

All documentation lives in the [docs](./docs) folder.
The [development setup](./docs/dev/setup.md) is a good starting point and the
[architecture overview](./docs/architecture/intro.md) provides technical details.
