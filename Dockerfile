# Build stage
FROM golang:1.27.4-alpine AS build

WORKDIR /app

# Download dependencies first (cached layer)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and build. The version is baked in, so /api/version reports
# the tag this image was cut from.
ARG VERSION=0.1.0
COPY . .
RUN go build -ldflags="-s -w -X jsfraz/shept-server/utils.Version=${VERSION}" -o shept-server

# Stage
FROM alpine

# Install only runtime CA certificates
RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy the compiled binary from the build stage
COPY --from=build /app/shept-server .

# Create a directory for data persistence
RUN mkdir -p /app/data

# Expose the port that the application is listening on
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Command to run the application
CMD ["./shept-server"]
