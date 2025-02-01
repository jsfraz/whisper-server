# Use the official Golang image as the base image
FROM golang:1.23.5-alpine AS build

# Set the working directory inside the container
WORKDIR /app

# Copy the Go mod and sum files to the working directory
COPY go.mod go.sum ./

# Download and install dependencies
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the application
RUN go build -o whisper-server

# Expose the port that the application will run on
EXPOSE 8080

# Command to run the application
CMD ["./whisper-server"]