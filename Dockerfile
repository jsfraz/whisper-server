# Build stage
FROM golang:1.26.4-alpine AS build

WORKDIR /app

# Install build dependencies for CGO (SQLCipher)
RUN apk add --no-cache gcc musl-dev

# Download dependencies first (cached layer)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and build
COPY . .
ENV CGO_ENABLED=1
RUN CGO_CFLAGS="-Doff64_t=off_t -Dpread64=pread -Dpwrite64=pwrite" go build -ldflags="-s -w" -o whisper-server

# Stage
FROM alpine

# Install only runtime CA certificates
RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy the compiled binary from the build stage
COPY --from=build /app/whisper-server .
COPY --from=build /app/static ./static

# Create a directory for data persistence
RUN mkdir -p /app/data

# Expose the port that the application will run on
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Command to run the application
CMD ["./whisper-server"]