# Build stage
FROM golang:1.24.1 AS builder

WORKDIR /
COPY . .

WORKDIR /cmd/server
RUN CGO_ENABLED=0 go build -o inkbunny-app

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /data
COPY --from=builder /cmd/server/inkbunny-app /usr/local/bin/inkbunny-app
EXPOSE 1323

CMD ["/usr/local/bin/inkbunny-app"]