# syntax=docker/dockerfile:1
# Una sola imagen de build y dos targets de ejecución (api, worker).
# El mismo Dockerfile se reutiliza en E2 (VM), E3 (plantilla del MIG) y E4 (Cloud Run).

FROM golang:1.26-alpine AS build
WORKDIR /src
COPY . .
# Las cachés de BuildKit evitan descargar módulos y recompilar en cada build;
# go build solo descarga los módulos que realmente usa (no las herramientas como mockery).
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker

FROM alpine:3.22 AS runtime
RUN apk add --no-cache ca-certificates tzdata \
 && adduser -D -u 10001 app \
 && mkdir -p /var/lib/app/files && chown app:app /var/lib/app/files
USER app
WORKDIR /app

FROM runtime AS api
COPY --from=build /out/api /app/api
EXPOSE 8080
ENTRYPOINT ["/app/api"]

FROM runtime AS worker
COPY --from=build /out/worker /app/worker
ENTRYPOINT ["/app/worker"]
