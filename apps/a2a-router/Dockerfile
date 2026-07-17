FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /out/a2a-router ./apps/a2a-router/cmd/a2a-router

FROM alpine:3.22
RUN adduser -D -H -u 10001 nekiro
USER nekiro
COPY --from=build /out/a2a-router /usr/local/bin/a2a-router
ENTRYPOINT ["/usr/local/bin/a2a-router"]
