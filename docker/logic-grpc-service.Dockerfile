# Stage 1: Build
FROM golang:1.24-alpine AS builder

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

RUN apk add --no-cache ca-certificates tzdata

ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /build

COPY logic-grpc-service/go.mod logic-grpc-service/go.sum ./
RUN go mod download

COPY logic-grpc-service/ ./
RUN GOFLAGS="-p=1" CGO_ENABLED=0 go build -ldflags="-s -w" -o /logic-grpc-service .

# Stage 2: Runtime
FROM alpine:3.21

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories

RUN apk add --no-cache ca-certificates tzdata libffi mupdf && \
    ln -s /usr/lib/libmupdf.so.24.10 /usr/lib/libmupdf.so

COPY --from=builder /logic-grpc-service /usr/local/bin/logic-grpc-service
COPY logic-grpc-service/config/config.example.yaml /app/config/config.example.yaml

WORKDIR /app
EXPOSE 50051

ENTRYPOINT ["/usr/local/bin/logic-grpc-service"]
