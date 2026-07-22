# Build stage
FROM golang:1.23-alpine AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o /atomcode-2api ./cmd/atomcode-2api/

# Runtime stage
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /atomcode-2api /usr/local/bin/atomcode-2api
EXPOSE 13457
CMD ["atomcode-2api", "serve"]