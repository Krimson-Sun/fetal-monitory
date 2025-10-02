FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/offline-service ./cmd/server

FROM alpine:latest

RUN apk --no-cache add ca-certificates


RUN adduser -D -s /bin/sh offline

WORKDIR /app


COPY --from=builder /app/bin/offline-service .


RUN chown offline:offline /app/offline-service

USER offline


EXPOSE 8081

# Переменные окружения по умолчанию
ENV HTTP_PORT=8081
ENV REDIS_ADDR=redis:6379
ENV REDIS_PASSWORD=
ENV REDIS_DB=0
ENV REDIS_TTL=24h
ENV POSTGRES_CONN_STR="host=postgres port=5432 user=fetal_user password=fetal_pass dbname=fetal_monitor sslmode=disable"
ENV FILTER_SERVICE_ADDR=feature-extractor:50052
ENV ML_SERVICE_ADDR=feature-extractor:50052

CMD ["./offline-service"]

