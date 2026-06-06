# Multi-stage build for kt9s
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git make

# Copy source
COPY . .

# Build
RUN make build

# Final stage
FROM alpine:latest

RUN apk add --no-cache ca-certificates kubectl

WORKDIR /root

# Copy binary from builder
COPY --from=builder /app/kt9s .

# Copy kubeconfig mount point
VOLUME /root/.kube

ENTRYPOINT ["./kt9s"]
