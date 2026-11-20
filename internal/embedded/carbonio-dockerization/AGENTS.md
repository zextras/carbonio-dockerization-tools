# carbonio-dockerization — agent notes

## Overview
Docker Compose environment for running the full Carbonio stack locally. Entry point: `docker-compose.yaml`.

## Key commands
- Start a single service: `docker compose up -d <service-name>`
- Check logs: `docker compose logs -f <service-name>`
- Override image: `CARBONIO_USER_MANAGEMENT_IMAGE=<image> docker compose up -d`

## carbonio-user-management — Quarkus image config
The `quarkus-refactor-and-cache` tag (and future Quarkus-based tags) uses GraalVM native binary, NOT a JAR.

Config via SmallRye (CarbonioBootstrapFactory): env vars use the property path with dots and hyphens replaced by underscores, uppercased.
- Property `networking-config.carbonio.service-discover.host` → env var `NETWORKING_CONFIG_CARBONIO_SERVICE_DISCOVER_HOST`
- The old `CARBONIO_*` env vars (used by the `devel` tag's `entrypoint.sh`) do NOT work with Quarkus images.

Required env vars for Quarkus UM in Docker:
- `NETWORKING_CONFIG_CARBONIO_SERVICE_DISCOVER_HOST=consul`
- `NETWORKING_CONFIG_CARBONIO_SERVICE_DISCOVER_PORT=8500`
- `NETWORKING_CONFIG_CARBONIO_SERVICE_HOST=0.0.0.0`
- `NETWORKING_CONFIG_CARBONIO_MAILBOX_PORT=8080` (not 20000 — Docker uses HTTP port, not internal port)

## Consul dependency
`carbonio-user-management` must declare `consul: condition: service_healthy` in `depends_on`, or it crashes on startup trying to reach `127.0.0.1:8500`.
