# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: riftcmd android ios riftcmd-cross swarm evm all test clean
.PHONY: riftcmd-linux riftcmd-linux-386 riftcmd-linux-amd64 riftcmd-linux-mips64 riftcmd-linux-mips64le
.PHONY: riftcmd-linux-arm riftcmd-linux-arm-5 riftcmd-linux-arm-6 riftcmd-linux-arm-7 riftcmd-linux-arm64
.PHONY: riftcmd-darwin riftcmd-darwin-386 riftcmd-darwin-amd64
.PHONY: riftcmd-windows riftcmd-windows-386 riftcmd-windows-amd64

GOBIN = build/bin
GO ?= latest

riftcmd:
	build/env.sh go run build/ci.go install ./cmd/riftcmd
	@echo "Done building."
	@echo "Run \"$(GOBIN)/riftcmd\" to launch riftcmd."

swarm:
	build/env.sh go run build/ci.go install ./cmd/swarm
	@echo "Done building."
	@echo "Run \"$(GOBIN)/swarm\" to launch swarm."

evm:
	build/env.sh go run build/ci.go install ./cmd/evm
	@echo "Done building."
	@echo "Run \"$(GOBIN)/evm\" to start the evm."

all:
	build/env.sh go run build/ci.go install

android:
	build/env.sh go run build/ci.go aar --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/riftcmd.aar\" to use the library."

ios:
	build/env.sh go run build/ci.go xcode --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/Riftcmd.framework\" to use the library."

test: all
	build/env.sh go run build/ci.go test

clean:
	rm -fr build/_workspace/pkg/ $(GOBIN)/*

# The devtools target installs tools required for 'go generate'.
# You need to put $GOBIN (or $GOPATH/bin) in your PATH to use 'go generate'.

devtools:
	env GOBIN= go get -u golang.org/x/tools/cmd/stringer
	env GOBIN= go get -u github.com/jteeuwen/go-bindata/go-bindata
	env GOBIN= go get -u github.com/fjl/gencodec
	env GOBIN= go install ./cmd/abigen

# Cross Compilation Targets (xgo)

riftcmd-cross: riftcmd-linux riftcmd-darwin riftcmd-windows riftcmd-android riftcmd-ios
	@echo "Full cross compilation done:"
	@ls -ld $(GOBIN)/riftcmd-*

riftcmd-linux: riftcmd-linux-386 riftcmd-linux-amd64 riftcmd-linux-arm riftcmd-linux-mips64 riftcmd-linux-mips64le
	@echo "Linux cross compilation done:"
	@ls -ld $(GOBIN)/riftcmd-linux-*

riftcmd-linux-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/386 -v ./cmd/riftcmd
	@echo "Linux 386 cross compilation done:"
	@ls -ld $(GOBIN)/riftcmd-linux-* | grep 386

riftcmd-linux-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/amd64 -v ./cmd/riftcmd
	@echo "Linux amd64 cross compilation done:"
	@ls -ld $(GOBIN)/riftcmd-linux-* | grep amd64

riftcmd-linux-arm: riftcmd-linux-arm-5 riftcmd-linux-arm-6 riftcmd-linux-arm-7 riftcmd-linux-arm64
	@echo "Linux ARM cross compilation done:"
	@ls -ld $(GOBIN)/riftcmd-linux-* | grep arm

riftcmd-linux-arm-5:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-5 -v ./cmd/riftcmd
	@echo "Linux ARMv5 cross compilation done:"
	@ls -ld $(GOBIN)/riftcmd-linux-* | grep arm-5

riftcmd-linux-arm-6:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-6 -v ./cmd/riftcmd
	@echo "Linux ARMv6 cross compilation done:"
	@ls -ld $(GOBIN)/riftcmd-linux-* | grep arm-6

riftcmd-linux-arm-7:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-7 -v ./cmd/riftcmd
	@echo "Linux ARMv7 cross compilation done:"
	@ls -ld $(GOBIN)/riftcmd-linux-* | grep arm-7

riftcmd-linux-arm64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm64 -v ./cmd/riftcmd
	@echo "Linux ARM64 cross compilation done:"
	@ls -ld $(GOBIN)/riftcmd-linux-* | grep arm64

riftcmd-linux-mips:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips --ldflags '-extldflags "-static"' -v ./cmd/riftcmd
	@echo "Linux MIPS cross compilation done:"
	@ls -ld $(GOBIN)/riftcmd-linux-* | grep mips

riftcmd-linux-mipsle:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mipsle --ldflags '-extldflags "-static"' -v ./cmd/riftcmd
	@echo "Linux MIPSle cross compilation done:"
	@ls -ld $(GOBIN)/riftcmd-linux-* | grep mipsle

riftcmd-linux-mips64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64 --ldflags '-extldflags "-static"' -v ./cmd/riftcmd
	@echo "Linux MIPS64 cross compilation done:"
	@ls -ld $(GOBIN)/riftcmd-linux-* | grep mips64

riftcmd-linux-mips64le:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64le --ldflags '-extldflags "-static"' -v ./cmd/riftcmd
	@echo "Linux MIPS64le cross compilation done:"
	@ls -ld $(GOBIN)/riftcmd-linux-* | grep mips64le

riftcmd-darwin: riftcmd-darwin-386 riftcmd-darwin-amd64
	@echo "Darwin cross compilation done:"
	@ls -ld $(GOBIN)/riftcmd-darwin-*

riftcmd-darwin-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/386 -v ./cmd/riftcmd
	@echo "Darwin 386 cross compilation done:"
	@ls -ld $(GOBIN)/riftcmd-darwin-* | grep 386

riftcmd-darwin-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/amd64 -v ./cmd/riftcmd
	@echo "Darwin amd64 cross compilation done:"
	@ls -ld $(GOBIN)/riftcmd-darwin-* | grep amd64

riftcmd-windows: riftcmd-windows-386 riftcmd-windows-amd64
	@echo "Windows cross compilation done:"
	@ls -ld $(GOBIN)/riftcmd-windows-*

riftcmd-windows-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/386 -v ./cmd/riftcmd
	@echo "Windows 386 cross compilation done:"
	@ls -ld $(GOBIN)/riftcmd-windows-* | grep 386

riftcmd-windows-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/amd64 -v ./cmd/riftcmd
	@echo "Windows amd64 cross compilation done:"
	@ls -ld $(GOBIN)/riftcmd-windows-* | grep amd64
