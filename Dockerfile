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
COPY --from=builder /app/docs ./docs
RUN mkdir -p /app/uploads /app/templates

EXPOSE 8088

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO- http://localhost:8089/swagger/index.html > /dev/null || exit 1

CMD ["./main"]
