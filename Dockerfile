# Stage 1: Builder
FROM golang:1.23-alpine AS builder
ARG SERVICE
# Set the working directory to the root of the workspace
WORKDIR /workspace
# Copy the entire workspace (to utilize go.work for dependency management)
COPY . .
# Navigate to the project directory and build the binary
WORKDIR /workspace/${SERVICE}
RUN go build -o /${SERVICE} -tags timetzdata

# Stage 2: Runtime
FROM alpine:latest
LABEL authors="Rebel028"
ARG SERVICE
ENV SERVICE=${SERVICE}
# Set the working directory for the runtime container
WORKDIR /app
# Copy the built binary from the builder stage
COPY --from=builder /${SERVICE} /app/${SERVICE}
# Set permissions and expose the entry point
RUN chmod +x /app/${SERVICE}
ENTRYPOINT "/app/${SERVICE}"