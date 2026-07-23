FROM node:22-alpine AS builder

ENV PNPM_HOME=/pnpm
ENV PATH=$PNPM_HOME:$PATH
RUN corepack enable && corepack prepare pnpm@10.19.0 --activate
WORKDIR /build
RUN echo "registry=https://registry.npmmirror.com" > .npmrc
COPY pnpm-lock.yaml pnpm-workspace.yaml ./
COPY platform-frontend/package.json ./platform-frontend/
WORKDIR /build/platform-frontend
RUN pnpm install --frozen-lockfile
WORKDIR /build
COPY platform-frontend/ ./platform-frontend/
COPY packages/ ./packages/
WORKDIR /build/platform-frontend
ARG VITE_API_BASE_URL=""
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}
RUN pnpm build

FROM nginx:alpine
COPY docker/nginx-platform.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /build/platform-frontend/dist /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
