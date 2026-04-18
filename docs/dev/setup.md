Dev setup
==========

The whole setup uses [make](https://www.gnu.org/software/make/) so *make* (lol) sure you have
it installed. This is the main [Makefile](../Makefile), so checkout the targets there.

Then get started with:

```sh
make build
```

The [compose.yml](../compose.yml) file defines the **docker compose** stack used for development
and integration testing.

# Project structure

This project is a monorepo with multiple microservices.

For this version everything is in one Go module, but I expect this will change as the dependency
trees get more complex or need to be narrowed down.
