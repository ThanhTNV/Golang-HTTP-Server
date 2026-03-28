FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /daemonset main.go

FROM alpine:3.21

EXPOSE 3000

COPY --from=builder /daemonset /daemonset

ENTRYPOINT ["/daemonset"]
