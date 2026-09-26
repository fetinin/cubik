# Build stages run on the native build platform ($BUILDPLATFORM) so multi-arch
# builds don't compile under QEMU; Go cross-compiles to $TARGETARCH instead.

# Stage 1: Build Frontend
FROM --platform=$BUILDPLATFORM oven/bun:1 AS frontend-builder

ENV PUBLIC_API_BASE_PATH=""

WORKDIR /app/front

# Copy frontend package files
COPY front/package.json front/bun.lock ./

# Install dependencies
RUN bun install --frozen-lockfile

# Copy frontend source code
COPY front/ ./

# Generate SvelteKit files
RUN bun run prepare

# Build the frontend SPA
RUN bun run build

# Stage 2: Build Backend
FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS backend-builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

# Copy go module files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

COPY *.go ./
COPY migrations/ ./migrations/
COPY api/ ./api/

# Copy frontend build from stage 1
COPY --from=frontend-builder /app/front/build ./front/build

# Build the binary with optimizations. The build cache mount speeds up local
# rebuilds; in CI the layer cache (type=gha) does the heavy lifting.
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-w -s" -o cubik .

# Stage 3: Final Runtime Image
FROM alpine:latest

WORKDIR /app

COPY --from=backend-builder /app/cubik .

# Expose the server port
EXPOSE 9080

# Healthcheck - request index page. Uses busybox wget so the runtime stage has
# no RUN steps and never needs emulation for non-native platforms.
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO /dev/null http://localhost:9080/ || exit 1

# Run the application in server mode
CMD ["./cubik"]
