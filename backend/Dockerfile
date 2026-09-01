# Multi-stage Docker build for Go Backend
FROM golang:alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the backend source code
COPY . .

# Build a compact, statically compiled Go binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o backend-server main.go

# Production runner image using minimal Alpine
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Kolkata

WORKDIR /app

# Copy compiled binary from builder
COPY --from=builder /app/backend-server ./

# Expose backend port
EXPOSE 8080

CMD ["./backend-server"]
