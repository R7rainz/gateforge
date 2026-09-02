FROM golang:1.27-alpine AS builder

WORKDIR /backend

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /gateforge ./cmd/gateforge


FROM alpine:latest

WORKDIR /app

COPY --from=builder /gateforge ./gateforge
COPY config.json ./config.json

EXPOSE 8080

CMD ["./gateforge"]
