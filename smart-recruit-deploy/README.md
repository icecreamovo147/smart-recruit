# smart-recruit-deploy

Deployment source root for the extracted Smart Recruit microservice runtime.

## Responsibility

- Own local microservice Compose overlays, runtime configuration examples, observability configuration, smoke tests, and export manifests.
- Keep the database strategy as a single shared MySQL instance for this feature.
- Avoid storing real `.env` files, secrets, tokens, private keys, or production credentials.

## Startup

Later TASKs add Compose files, profiles, Nacos/observability configuration, service build targets, and smoke scripts.

## Monolith Relationship

Deployment assets must keep monolith fallback available until the monolith retirement gate passes.
