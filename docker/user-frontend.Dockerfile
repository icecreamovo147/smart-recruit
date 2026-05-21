# Stage 1: Build
FROM node:22-alpine AS builder

ENV PNPM_HOME=/pnpm
ENV PATH=$PNPM_HOME:$PATH

RUN corepack enable && corepack prepare pnpm@10.19.0 --activate

WORKDIR /build

RUN echo "registry=https://registry.npmmirror.com" > .npmrc

COPY user-frontend/package.json user-frontend/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

COPY user-frontend/ ./
RUN pnpm build

# Stage 2: Runtime
FROM nginx:alpine

COPY docker/nginx-user.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /build/dist /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
