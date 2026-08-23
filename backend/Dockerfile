# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS build
ENV GOPROXY=https://goproxy.cn,direct \
    CGO_ENABLED=0 \
    GOOS=linux
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk add --no-cache git ca-certificates
WORKDIR /src
COPY go.mod ./
COPY go.sum* ./
RUN go mod download
COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /out/gorag ./cmd/server

FROM alpine:3.21
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk add --no-cache ca-certificates tzdata wget \
    && adduser -D -u 10001 gorag
ENV TZ=Asia/Shanghai
WORKDIR /app
COPY --from=build /out/gorag /app/gorag
RUN mkdir -p /data && chown -R gorag:gorag /app /data
USER gorag
EXPOSE 8080
HEALTHCHECK --interval=5s --timeout=3s --retries=8 CMD wget -qO- http://127.0.0.1:8080/healthz || exit 1
ENTRYPOINT ["/app/gorag"]
