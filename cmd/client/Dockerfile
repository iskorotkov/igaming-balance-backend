# Build
FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -o main cmd/client/main.go

# Run
FROM alpine:3.22
EXPOSE 8080

COPY --from=builder /app/main /main

ENTRYPOINT ["/main"]
