# Build Stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install git for go mod download if needed
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o server cmd/server/main.go

# Run Stage
FROM alpine:latest

WORKDIR /app

# Install tzdata for timezone
RUN apk add --no-cache tzdata

COPY --from=builder /app/server .
COPY --from=builder /app/configs ./configs

EXPOSE 8080

CMD ["./server"]
