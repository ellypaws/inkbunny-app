# Build stage
FROM golang:1.24.1 AS builder

WORKDIR /
COPY . .

WORKDIR /cmd/server
RUN go build -o inkbunny-app

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /cmd/server/inkbunny-app .
COPY --from=builder /cmd/server/audits.sqlite* /app/ 2>/dev/null || true

EXPOSE 1373

CMD ["./inkbunny-app"]