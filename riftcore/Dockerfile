FROM alpine:3.5

ADD . /riftcore
RUN \
  apk add --update git go make gcc musl-dev linux-headers && \
  (cd riftcore && make riftcmd)                           && \
  cp riftcore/build/bin/riftcmd /usr/local/bin/           && \
  apk del git go make gcc musl-dev linux-headers          && \
  rm -rf /riftcore && rm -rf /var/cache/apk/*

EXPOSE 8545
EXPOSE 30303
EXPOSE 30303/udp

ENTRYPOINT ["riftcmd"]
