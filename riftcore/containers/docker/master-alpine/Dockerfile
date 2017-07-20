FROM alpine:3.5

RUN \
  apk add --update go git make gcc musl-dev linux-headers ca-certificates && \
  git clone --depth 1 --branch release/1.6 https://github.com/cryptorift/riftcore && \
  (cd riftcore && make riftcmd) && \
  cp riftcore/build/bin/riftcmd /riftcmd && \
  apk del go git make gcc musl-dev linux-headers && \
  rm -rf /riftcore && rm -rf /var/cache/apk/*

EXPOSE 8545
EXPOSE 30303

ENTRYPOINT ["/riftcmd"]
