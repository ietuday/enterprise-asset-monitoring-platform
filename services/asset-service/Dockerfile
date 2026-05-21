FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o asset-service ./cmd

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/asset-service .

EXPOSE 5001

CMD ["./asset-service"]