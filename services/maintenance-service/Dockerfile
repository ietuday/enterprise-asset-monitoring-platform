FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY main.go ./
COPY internal ./internal

RUN go build -o maintenance-service .

FROM alpine:3.20

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /app/maintenance-service .

RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8087

CMD ["./maintenance-service"]
