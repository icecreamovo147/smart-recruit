# smart-recruit-deploy

Deployment source root for the extracted Smart Recruit microservice runtime.

## Responsibility

- Own local microservice Compose overlays, runtime configuration examples, observability configuration, smoke tests, and export manifests.
- Keep the database strategy as a single shared MySQL instance for this feature.
- Avoid storing real `.env` files, secrets, tokens, private keys, or production credentials.

## Startup

Later TASKs add Compose files, profiles, Nacos/observability configuration, service build targets, and smoke scripts.

The initial microservice runtime Compose file is:

```bash
docker compose -f smart-recruit-deploy/docker-compose.microservices.yml --profile infra up -d
docker compose -f smart-recruit-deploy/docker-compose.microservices.yml --profile observability up -d
```

Available profiles:

- `infra`: Nacos, MySQL, Redis, RabbitMQ.
- `observability`: Prometheus, Grafana, Jaeger, Loki.
- `compat`: legacy logic/web fallback through the existing compose file.
- `services`: placeholder profile reserved for later service runtime tasks.

## Monolith Relationship

Deployment assets must keep monolith fallback available until the monolith retirement gate passes.
