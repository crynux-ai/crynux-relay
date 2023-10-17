FROM golang:alpine3.18
RUN apk add --no-cache --update gcc g++

WORKDIR /app

COPY go.* .

RUN CGO_ENABLED=1 go mod download

COPY . .

COPY ./tests/config.yml /app/config/config.yml

CMD go test -p 1 ./api/v1/inference_tasks
