FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

WORKDIR /app/emulator
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/client ./cmd/client

FROM alpine:latest

RUN apk --no-cache add ca-certificates

RUN adduser -D -s /bin/sh emulator

WORKDIR /app

COPY --from=builder /app/emulator/bin/client .

RUN chown emulator:emulator /app/client

USER emulator

USER root
RUN mkdir -p /data && chown emulator:emulator /data
USER emulator

ENV TARGET_ADDR="data-receiver:50051"
ENV SESSION_ID=""

CMD sh -c "./client -server=$TARGET_ADDR -session=$SESSION_ID -fhr=/data/fhr.csv -uc=/data/uc.csv"
