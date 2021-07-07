# Build Highcoin in a stock Go builder container
FROM golang:1.16-alpine as builder

RUN apk add --no-cache make gcc musl-dev linux-headers git

ADD . /go-highcoin
RUN cd /go-highcoin && make highcoin

# Pull Highcoin into a second stage deploy alpine container
FROM alpine:latest

RUN apk add --no-cache ca-certificates
COPY --from=builder /go-highcoin/build/bin/highcoin /usr/local/bin/

EXPOSE 42000 41999 30303 30303/udp
ENTRYPOINT ["highcoin"]
