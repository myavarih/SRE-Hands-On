FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/auth-service ./services/auth-service/cmd

FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /bin/auth-service /app/auth-service
USER 65534:65534
EXPOSE 50051 9090
ENTRYPOINT ["/app/auth-service"]
