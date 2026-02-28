FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /payment-api ./cmd/api

# ---

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /payment-api /payment-api

EXPOSE 8080

ENTRYPOINT ["/payment-api"]
