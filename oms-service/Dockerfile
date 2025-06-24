# syntax=docker/dockerfile:1.4
# Build stage
FROM golang:1.22-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git openssh

# Set up SSH for private repos
RUN mkdir -p ~/.ssh && chmod 700 ~/.ssh
RUN ssh-keyscan github.com >> ~/.ssh/known_hosts
RUN git config --global url."git@github.com:".insteadOf "https://github.com/"

# Copy codebase
COPY . .

# Debug SSH connection
RUN --mount=type=ssh ssh -T git@github.com || true
# Download dependencies with SSH agent forwarding
RUN --mount=type=ssh go mod download
# Make sure we use the go_commons repo
RUN --mount=type=ssh go mod why github.com/omniful/go_commons
# Build the service
RUN go build -o oms-service ./cmd/server

# Run stage
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/oms-service .
CMD ["./oms-service"]
