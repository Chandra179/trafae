FROM docker.io/library/golang:1.27.1 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/trafae-server ./server/cmd/example

FROM docker.io/library/alpine:3.22

RUN addgroup -S trafae \
    && adduser -S -G trafae trafae \
    && mkdir -p /data \
    && chown trafae:trafae /data

WORKDIR /app
COPY --from=build --chown=trafae:trafae /out/trafae-server /app/trafae-server
COPY --from=build --chown=trafae:trafae /src/server/config /app/server/config

ENV APP_ENVIRONMENT=prd \
    SQLITE_DSN=/data/trafae.db \
    BADGER_DIR=/data/trafae

EXPOSE 8080
VOLUME ["/data"]
USER trafae
ENTRYPOINT ["/app/trafae-server"]
