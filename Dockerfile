FROM golang:1.26-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build -o url-shortener ./cmd/server

FROM alpine:3.21

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /app/url-shortener .
COPY --from=builder /app/templates ./templates

EXPOSE 8080

ENV PORT=8080
ENV DB_PATH=/app/data/urls.db

RUN mkdir -p /app/data

CMD ["./url-shortener"]
