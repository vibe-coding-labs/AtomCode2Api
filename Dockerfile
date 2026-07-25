# Build stage
FROM golang:1.25-alpine AS builder
RUN apk add --no-cache gcc musl-dev nodejs npm
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Build frontend
RUN cd web && npm ci && npm run build
# Build Go binary
RUN CGO_ENABLED=1 go build -o /atomcode-2api ./cmd/atomcode-2api/

# Runtime stage
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /atomcode-2api /usr/local/bin/atomcode-2api
EXPOSE 45678
VOLUME ["/data"]
ENV ATOMCODE_DAEMON_URL=http://host.docker.internal:13456
CMD ["atomcode-2api", "serve"]