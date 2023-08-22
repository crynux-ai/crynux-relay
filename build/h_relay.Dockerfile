FROM golang:alpine3.18 AS builder
RUN apk add --no-cache --update gcc g++

WORKDIR /h_relay

COPY go.* .

RUN CGO_ENABLED=1 go mod download

COPY . .

# Private key is only used in automated testing
COPY ./config/private_key.go.example ./config/private_key.go
COPY ./config/test_private_key.go.example ./config/test_private_key.go

RUN CGO_ENABLED=1 go build

FROM alpine:3.18

RUN apk add --no-cache tzdata
ENV TZ=Asia/Tokyo

WORKDIR /app

COPY --from=builder /h_relay/h_relay .
COPY static ./static

CMD ["/app/h_relay"]
