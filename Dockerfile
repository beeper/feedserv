FROM golang:1.20-alpine3.17 AS builder

RUN apk add --no-cache ca-certificates git
COPY . /build
WORKDIR /build
ARG COMMIT_HASH
ENV COMMIT_HASH=${COMMIT_HASH}
RUN CGO_ENABLED=0 go build -o /usr/bin/feedserv -ldflags "-X main.Commit=$COMMIT_HASH -X 'main.BuildTime=`date '+%b %_d %Y, %H:%M:%S'`'"

FROM alpine:3.17

RUN apk add --no-cache ca-certificates

ENV FEEDSERV_CONFIG_PATH=/data/config.yaml
VOLUME /data

CMD ["/usr/bin/feedserv"]

COPY --from=builder /usr/bin/feedserv /usr/bin/feedserv
