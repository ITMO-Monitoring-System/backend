FROM golang:1.25-alpine AS builder

WORKDIR /app

# RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go install github.com/swaggo/swag/cmd/swag@v1.8.12
RUN swag init -g ./cmd/api/main.go

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o app ./cmd/api

FROM alpine:3.20

# RUN apk add --no-cache ca-certificates

RUN adduser -D -g '' appuser

WORKDIR /app

# бинарь
COPY --from=builder /app/app /app/app

COPY --from=builder /app/docs ./docs

COPY migrations /migrations

USER appuser

EXPOSE 8080

ENV MIGRATIONS_PATH=/migrations

ENTRYPOINT ["/app/app"]