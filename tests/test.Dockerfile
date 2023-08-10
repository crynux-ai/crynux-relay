FROM golang:alpine3.18
RUN apk add --no-cache --update gcc g++

WORKDIR /h_relay

COPY go.* .

RUN CGO_ENABLED=1 go mod download

COPY . .
CMD go test ./...
