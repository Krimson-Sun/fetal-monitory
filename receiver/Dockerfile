FROM golang:1.23-alpine AS builder

# Устанавливаем зависимости для сборки
RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY ../go.mod ../go.sum ./

RUN go mod download

COPY .. .

WORKDIR /app/receiver
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/receiver ./cmd/receiver

FROM alpine:latest

RUN apk --no-cache add ca-certificates

RUN adduser -D -s /bin/sh receiver

WORKDIR /app

COPY --from=builder /app/receiver/bin/receiver .

RUN chown receiver:receiver /app/receiver

USER receiver

EXPOSE 50051

ENV GRPC_PORT=50051
ENV BATCH_MAX_SAMPLES=250
ENV BATCH_MAX_SPAN_MS=30000
ENV FLUSH_INTERVAL_MS=500
ENV ACK_EVERY_N=50
ENV OUT_OF_ORDER_TOLERANCE_MS=1000
ENV DROP_TOO_OLD_MS=30000

CMD ["./receiver"]