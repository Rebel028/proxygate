# Stage 1: Builder
FROM golang:1.23-alpine AS builder
# Set the working directory to the root of the workspace
WORKDIR /workspace
# Copy the entire workspace (to utilize go.work for dependency management)
COPY . .
RUN go build -o /proxygate

# Stage 2: Runtime
FROM alpine:latest
LABEL authors="Rebel028"
# Set the working directory for the runtime container
WORKDIR /app
# Copy the built binary from the builder stage
COPY --from=builder /workspace/proxygate /app/proxygate
# Set permissions and expose the entry point
RUN chmod +x /app/proxygate
ENTRYPOINT "/app/proxygate"