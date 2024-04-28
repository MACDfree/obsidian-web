FROM golang:1.22.2-alpine3.18 AS builder

WORKDIR /build

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.tuna.tsinghua.edu.cn/g' /etc/apk/repositories
RUN apk add --no-cache build-base upx

RUN go env -w GOPROXY="https://goproxy.cn,direct"
ENV CGO_ENABLED 1

COPY . .
RUN go build -ldflags="-s -w"
RUN upx ./obsidian-web

FROM alpine:3.18

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.tuna.tsinghua.edu.cn/g' /etc/apk/repositories
RUN apk add --no-cache git

WORKDIR /app
COPY --from=builder /build/obsidian-web /app/obsidian-web

ENV GIN_MODE release
ENTRYPOINT ["./obsidian-web"]
