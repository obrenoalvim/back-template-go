# --- build ---
FROM golang:1.25-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# CGO_ENABLED=0: static binary, no libc dependency — required for the distroless runtime below.
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/api ./cmd/api
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/healthcheck ./cmd/healthcheck

# --- runtime ---
FROM gcr.io/distroless/static:nonroot AS runtime
COPY --from=build /out/api /api
COPY --from=build /out/healthcheck /healthcheck

USER nonroot:nonroot
EXPOSE 8083

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
	CMD ["/healthcheck"]

ENTRYPOINT ["/api"]
