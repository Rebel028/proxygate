FROM golang:1.23-alpine AS builder
WORKDIR /workspace
COPY . .
RUN go build -o ./proxygate

FROM alpine:latest
LABEL authors="Rebel028"
WORKDIR /app
COPY --from=builder /workspace/proxygate /app/proxygate
RUN chmod +x /app/proxygate
ENTRYPOINT ["/app/proxygate"]