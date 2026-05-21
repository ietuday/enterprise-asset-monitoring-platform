FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o alert-service ./cmd

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/alert-service .

EXPOSE 5003

CMD ["./alert-service"]