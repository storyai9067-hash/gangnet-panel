FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod .
COPY main.go .
RUN go build -o gangnet-panel .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/gangnet-panel .
EXPOSE 8080
CMD ["./gangnet-panel"]
