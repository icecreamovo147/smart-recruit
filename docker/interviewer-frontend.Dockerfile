# Stage 1: Build
FROM node:22-alpine AS builder

ENV PNPM_HOME=/pnpm
ENV PATH=$PNPM_HOME:$PATH

RUN corepack enable && corepack prepare pnpm@10.19.0 --activate

WORKDIR /build

RUN echo "registry=https://registry.npmmirror.com" > .npmrc

# Preserve monorepo structure so Vite @shared alias resolves correctly.
COPY interviewer-frontend/package.json interviewer-frontend/pnpm-lock.yaml ./interviewer-frontend/

WORKDIR /build/interviewer-frontend
RUN pnpm install --frozen-lockfile

WORKDIR /build
COPY interviewer-frontend/ ./interviewer-frontend/
COPY packages/ ./packages/

WORKDIR /build/interviewer-frontend

ARG VITE_API_BASE_URL=""
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}
RUN pnpm build

# Stage 2: Runtime
FROM nginx:alpine

COPY docker/nginx-interviewer.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /build/interviewer-frontend/dist /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
