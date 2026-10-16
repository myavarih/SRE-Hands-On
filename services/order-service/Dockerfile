FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/order-service ./services/order-service/cmd

FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /bin/order-service /app/order-service
USER 65534:65534
EXPOSE 50052 9090
ENTRYPOINT ["/app/order-service"]
