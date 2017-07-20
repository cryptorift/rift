FROM ubuntu:xenial

RUN \
  apt-get update && apt-get upgrade -q -y && \
  apt-get install -y --no-install-recommends golang git make gcc libc-dev ca-certificates && \
  git clone --depth 1 --branch release/1.5 https://github.com/cryptorift/riftcore && \
  (cd riftcore && make riftcmd) && \
  cp riftcore/build/bin/riftcmd /riftcmd && \
  apt-get remove -y golang git make gcc libc-dev && apt autoremove -y && apt-get clean && \
  rm -rf /riftcore

EXPOSE 8545
EXPOSE 30303

ENTRYPOINT ["/riftcmd"]
