FROM golang:1.26.1-alpine3.23 AS builder

WORKDIR /build

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
RUN apk add --no-cache build-base upx

RUN go env -w GOPROXY="https://goproxy.cn,direct"
ENV CGO_ENABLED=1

# cache deps before building
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
# 缓存 Go 编译产物：仅改样式/模板时触发整层重建，也能命中已编译的依赖
RUN --mount=type=cache,target=/root/.cache/go-build go build -tags sqlite_fts5 -ldflags="-s -w"
RUN upx ./obsidian-web

FROM alpine:3.23

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
&& apk update && apk add --no-cache tzdata \
&& cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
&& echo "Shanghai/Asia" > /etc/timezone \
&& apk del tzdata

WORKDIR /app

COPY --from=builder /build/obsidian-web /app/obsidian-web

# 模板与静态资源随镜像一起发布，运行时只读；可变内容走 volume
COPY templates /app/templates
COPY static /app/static
COPY assets /app/assets

ENV GIN_MODE=release
ENTRYPOINT ["./obsidian-web"]
