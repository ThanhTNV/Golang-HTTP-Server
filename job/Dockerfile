FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /job main.go

FROM alpine:3.21

COPY --from=builder /job /job

ENTRYPOINT ["/job"]
