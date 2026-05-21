FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o rule-service ./cmd

FROM alpine:latest

RUN apk add --no-cache wget

WORKDIR /app

COPY --from=builder /app/rule-service .

EXPOSE 5004

CMD ["./rule-service"]