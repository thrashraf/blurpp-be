# Use the latest Alpine Linux image as the base
FROM alpine:latest

# Install necessary packages to build and run the Go application
RUN apk add --no-cache build-base go ca-certificates unzip openssh

# Set the working directory to /app
WORKDIR /app

# Copy go.mod and go.sum to the working directory
COPY go.mod go.sum ./

# Download Go module dependencies
RUN go mod download

# Copy all the Go source files to the working directory
COPY *.go ./

# Build the Go application and name the binary as "blurpp"
RUN go build -o blurpp

# Expose port 8080 (not strictly required in the Dockerfile, but good for documentation purposes)
EXPOSE 8080

# Start the Go application
CMD ["./blurpp", "serve", "--http=0.0.0.0:8080"]
