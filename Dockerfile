# Use the official Golang image as the base image (Debian based)
FROM golang:1.25.1 AS build

# Set the working directory inside the container
WORKDIR /app

# Installing the necessary tools and libraries for CGO and SQLite
RUN apt-get update && apt-get install -y gcc libc6-dev

# Copy the Go mod and sum files to the working directory
COPY go.mod go.sum ./

# Download and install dependencies
RUN go mod download

# Copy the source code into the container
COPY . .

# Create a directory for data persistence
RUN mkdir -p /app/data

# Build the application with CGO
ENV CGO_ENABLED=1
RUN go build -o whisper-server

# Expose the port that the application will run on
EXPOSE 8080

# Command to run the application
CMD ["./whisper-server"]