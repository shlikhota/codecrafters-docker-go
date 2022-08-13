FROM golang:1.19-alpine

RUN apk add curl

ADD . /app
WORKDIR /app

ENTRYPOINT ["/app/your_docker.sh"]
