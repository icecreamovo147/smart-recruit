# Stage 1: Build
FROM node:22-alpine AS builder

ENV PNPM_HOME=/pnpm
ENV PATH=$PNPM_HOME:$PATH

RUN corepack enable && corepack prepare pnpm@10.19.0 --activate

WORKDIR /build

RUN echo "registry=https://registry.npmmirror.com" > .npmrc

# Preserve monorepo structure so Vite @shared alias resolves correctly.
# vite.config.ts maps @shared → ../packages/shared/src, which needs
# sibling directories /build/user-frontend/ and /build/packages/shared/.
COPY user-frontend/package.json user-frontend/pnpm-lock.yaml ./user-frontend/

WORKDIR /build/user-frontend
RUN pnpm install --frozen-lockfile

WORKDIR /build
COPY user-frontend/ ./user-frontend/
COPY packages/ ./packages/

WORKDIR /build/user-frontend

ARG VITE_API_BASE_URL=""
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}
RUN pnpm build

# Stage 2: Runtime
FROM nginx:alpine

COPY docker/nginx-user.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /build/user-frontend/dist /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
