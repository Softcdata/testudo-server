# Build the disaster-server binary
# Use --platform=${BUILDPLATFORM} to run build stage on host architecture (faster cross-compilation)
FROM --platform=${BUILDPLATFORM} golang:1.24.5-alpine AS builder
WORKDIR /src

# Build variables - declare early for use in build stage
ARG TARGETOS=linux
ARG TARGETARCH
ARG VERSION=dev

# go.mod replaces github.com/softcdata/testudo-operator with ../disaster-operator.
# Buildx supplies this named context from the sibling repository.
COPY --from=operator_source . /disaster-operator

# Install git and root CAs for module downloads.
RUN apk add --no-cache git ca-certificates

# Configure Go module proxy. Override GOPROXY at build time if needed.
ARG GOPROXY=https://proxy.golang.org,direct
RUN go env -w GOPROXY=${GOPROXY}

# Enable Go modules and caching - download dependencies first
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the binary for target platform using cross-compilation
# CGO_ENABLED=0 ensures static linking for cross-compilation
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -mod=mod -trimpath -ldflags "-s -w -X github.com/softcdata/testudo-server/cmd/app.Version=${VERSION}" \
    -o /out/disaster ./

# --- Runtime image ---
# distroless/static supports multi-arch (amd64, arm64, etc.)
FROM gcr.io/distroless/static:nonroot

# Set workdir so config path configs/config.yaml resolves
WORKDIR /app

# Copy binary
COPY --from=builder /out/disaster /usr/local/bin/disaster

# Copy default config (can be overridden by ConfigMap)
COPY configs/config.yaml /app/configs/config.yaml

# Copy Swagger/OpenAPI spec used by /openapi.yaml and /openapi.json.
COPY openspec/specs/disaster-server-openapi.yaml /app/openspec/specs/disaster-server-openapi.yaml

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/disaster", "server"]
