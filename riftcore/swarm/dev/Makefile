.PHONY: build cluster test

default: build

build:
	go build -o bin/swarm    github.com/cryptorift/riftcore/cmd/swarm
	go build -o bin/riftcmd     github.com/cryptorift/riftcore/cmd/riftcmd
	go build -o bin/bootnode github.com/cryptorift/riftcore/cmd/bootnode

cluster: build
	scripts/boot-cluster.sh

test:
	go test -v github.com/cryptorift/riftcore/swarm/...
