# Stage 1: Build Frontend
FROM oven/bun:1 AS frontend-builder

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
FROM golang:1.25-alpine AS backend-builder

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

# Build the binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o cubik .

# Stage 3: Final Runtime Image
FROM alpine:latest

WORKDIR /app

# Install curl for healthcheck
RUN apk --no-cache add curl

COPY --from=backend-builder /app/cubik .

# Expose the server port
EXPOSE 9080

# Healthcheck - request index page
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:9080/ || exit 1

# Run the application in server mode
CMD ["./cubik"]
