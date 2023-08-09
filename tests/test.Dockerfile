FROM golang:alpine3.18
RUN apk add --no-cache --update gcc g++

WORKDIR /app

COPY . .

RUN CGO_ENABLED=1 go install h_relay

CMD go test ./...
