FROM docker.io/library/golang:1.27 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/lux-server ./server/cmd/example

FROM docker.io/library/alpine:3.22

RUN addgroup -S lux \
    && adduser -S -G lux lux \
    && mkdir -p /data \
    && chown lux:lux /data

WORKDIR /app
COPY --from=build --chown=lux:lux /out/lux-server /app/lux-server
COPY --from=build --chown=lux:lux /src/server/config /app/server/config

ENV APP_ENVIRONMENT=prd \
    SQLITE_DSN=/data/lux.db \
    BADGER_DIR=/data/lux

EXPOSE 8080
VOLUME ["/data"]
USER lux
ENTRYPOINT ["/app/lux-server"]
