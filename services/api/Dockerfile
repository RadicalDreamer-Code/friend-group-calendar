FROM golang:1.27-alpine AS base
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

# Development: rebuilds and restarts on every file change (source is mounted).
FROM base AS dev
RUN go install github.com/air-verse/air@v1.67.4
CMD ["air", "-c", ".air.toml"]

FROM base AS build
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# Production: a small image with only the binary.
FROM gcr.io/distroless/static-debian12:nonroot AS prod
COPY --from=build /out/api /api
EXPOSE 8080
ENTRYPOINT ["/api"]
