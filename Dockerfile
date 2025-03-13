# Use a multi-stage build to keep the final image small
# Stage 1: Build the Go application
FROM golang:1.24-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum to cache dependencies
COPY go.mod go.sum ./

# Download all dependencies. Caching is leveraged where possible.
RUN go mod download

# Copy the source from the current directory to the working directory
COPY . .

# Build the Go application
RUN go build -o main .

# Stage 2: Create the final image
FROM alpine:latest

# Set the working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/main .

# Expose the port that your application listens on
EXPOSE 8080

# Command to run the executable
CMD ["./main"]
