FROM golang:1.23-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /bin/proxygate ./cmd/proxygate

FROM alpine:latest
LABEL authors="Rebel028"
WORKDIR /app
COPY --from=builder /bin/proxygate /app/proxygate
RUN chmod +x /app/proxygate
ENTRYPOINT ["/app/proxygate"]