# Build stage
FROM golang:1.23-alpine AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o /atomcode-proxy ./cmd/atomcode-proxy/

# Runtime stage
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /atomcode-proxy /usr/local/bin/atomcode-proxy
EXPOSE 13457
CMD ["atomcode-proxy", "serve"]