FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/payment-service ./services/payment-service/cmd

FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /bin/payment-service /app/payment-service
USER 65534:65534
EXPOSE 50053 9090
ENTRYPOINT ["/app/payment-service"]
