FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o maintenance-service .

FROM alpine:3.20

WORKDIR /app

COPY --from=builder /app/maintenance-service .

EXPOSE 8087

CMD ["./maintenance-service"]
