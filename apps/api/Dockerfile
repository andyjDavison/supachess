# Build context is apps/ (not apps/api/) so db/migrations is reachable:
#   docker build -f api/Dockerfile .
FROM golang:1.26-alpine AS build
WORKDIR /src

COPY api/go.mod api/go.sum ./
RUN go mod download

COPY api/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app

COPY --from=build /out/api /app/api
COPY db/migrations /app/migrations

# main.go applies migrations on boot from this path.
ENV MIGRATIONS_PATH=/app/migrations \
    ADDR=:8080

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/api"]
