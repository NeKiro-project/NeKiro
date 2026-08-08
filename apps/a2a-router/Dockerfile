FROM golang:1.26.4-bookworm@sha256:b305420a68d0f229d91eb3b3ed9e519fcf2cf5461da4bef997bf927e8c0bfd2b AS build

WORKDIR /src
ARG GOPROXY=https://proxy.golang.org
ARG GOSUMDB=sum.golang.org
COPY go.mod go.sum ./
RUN GOPROXY="$GOPROXY" GOSUMDB="$GOSUMDB" go mod download
COPY contracts ./contracts
COPY config_center ./config_center
COPY registry ./registry
COPY apps/a2a-router ./apps/a2a-router
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/a2a-router ./apps/a2a-router/cmd/a2a-router

FROM alpine:3.22

RUN apk add --no-cache ca-certificates wget \
    && adduser -D -H -u 10001 nekiro
COPY --from=build /out/a2a-router /usr/local/bin/a2a-router
USER nekiro
EXPOSE 8081
ENTRYPOINT ["/usr/local/bin/a2a-router"]
