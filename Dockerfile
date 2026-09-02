FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
ENV GOPROXY=https://goproxy.cn,direct
RUN go mod download

COPY . .
RUN go build -o main cmd/main.go

FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/main .
COPY --from=builder /app/.env .
COPY --from=builder /app/docs ./docs
COPY --from=builder /app/migrations ./migrations

EXPOSE 8088

CMD ["./main"]