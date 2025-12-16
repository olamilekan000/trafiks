FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download && go mod verify

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags="-w -s" \
    -o /app/trafiks \
    ./cmd/server/main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/trafiks .

EXPOSE 8080

CMD ["./trafiks"]
