# Build stage
FROM golang:1.24.1 AS builder

WORKDIR /app
COPY ../.. .

# Install git to handle submodules
RUN apt-get update && apt-get install -y git

# Initialize and update submodules
RUN git submodule update --init --recursive

# Build the application
RUN cd cmd/server && go build -o server .

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=builder /app/cmd/server/server .

EXPOSE 1373

CMD ["./server"]