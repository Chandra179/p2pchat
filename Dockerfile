# Use Go as builder
FROM golang:1.24.3 AS builder

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .

RUN go build -o p2pchat main.go

# Final runtime image
FROM debian:bullseye-slim

WORKDIR /app

COPY --from=builder /app/p2pchat ./

# Needed for stdin and terminal I/O
ENTRYPOINT ["./p2pchat"]
