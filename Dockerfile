FROM golang:1.25-alpine AS builder

ARG BUILD_TARGET=api

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app-binary ./cmd/${BUILD_TARGET}

# ---

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app-binary /app-binary

EXPOSE 8080

ENTRYPOINT ["/app-binary"]
