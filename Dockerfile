FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /service ./cmd/service/

FROM alpine:3.19
WORKDIR /

COPY --from=builder /service /service
COPY config.yaml /config.yaml

ENTRYPOINT ["/service"]
