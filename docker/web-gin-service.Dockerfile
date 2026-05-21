# Stage 1: Build
FROM golang:1.25-alpine AS builder

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

RUN apk add --no-cache ca-certificates tzdata

ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /build

COPY web-gin-service/go.mod web-gin-service/go.sum ./
RUN go mod download

COPY web-gin-service/ ./
RUN GOFLAGS="-p=1" CGO_ENABLED=0 go build -ldflags="-s -w" -gcflags="-d=checkptr=0" -o /web-gin-service .

# Stage 2: Runtime
FROM alpine:3.21

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /web-gin-service /usr/local/bin/web-gin-service

WORKDIR /app
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/web-gin-service"]
