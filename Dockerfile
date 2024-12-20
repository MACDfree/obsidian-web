FROM golang:1.23.1-alpine3.19 AS builder

WORKDIR /build

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.tuna.tsinghua.edu.cn/g' /etc/apk/repositories
RUN apk add --no-cache build-base upx

RUN go env -w GOPROXY="https://goproxy.cn,direct"
ENV CGO_ENABLED=1

# cache deps before building
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN go build -ldflags="-s -w"
RUN upx ./obsidian-web

FROM alpine:3.19

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.tuna.tsinghua.edu.cn/g' /etc/apk/repositories \
&& apk update && apk add --no-cache tzdata \
&& cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
&& echo "Shanghai/Asia" > /etc/timezone \
&& apk del tzdata

RUN apk add --no-cache git ca-certificates openssh

WORKDIR /app

COPY --from=builder /build/obsidian-web /app/obsidian-web

ENV GIN_MODE=release
ENTRYPOINT ["./obsidian-web"]
