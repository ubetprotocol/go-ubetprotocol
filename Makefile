# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: gubet android ios gubet-cross evm all test clean
.PHONY: gubet-linux gubet-linux-386 gubet-linux-amd64 gubet-linux-mips64 gubet-linux-mips64le
.PHONY: gubet-linux-arm gubet-linux-arm-5 gubet-linux-arm-6 gubet-linux-arm-7 gubet-linux-arm64
.PHONY: gubet-darwin gubet-darwin-386 gubet-darwin-amd64
.PHONY: gubet-windows gubet-windows-386 gubet-windows-amd64

GOBIN = $(shell pwd)/build/bin
GO ?= latest

gubet:
	build/env.sh go run build/ci.go install ./cmd/gubet
	cd ./build/bin/
	./build/bin/gubet init ./build/bin/genesis.json
	@echo "Done building."
	@echo "Run \"$(GOBIN)/gubet\" to launch gubet."

all:
	build/env.sh go run build/ci.go install
	cd ./build/bin/
	./build/bin/gubet init ./build/bin/genesis.json

android:
	build/env.sh go run build/ci.go aar --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/gubet.aar\" to use the library."

ios:
	build/env.sh go run build/ci.go xcode --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/Gubet.framework\" to use the library."

test: all
	build/env.sh go run build/ci.go test

lint: ## Run linters.
	build/env.sh go run build/ci.go lint

clean:
	./build/clean_go_build_cache.sh
	rm -fr build/_workspace/pkg/ $(GOBIN)/*

# The devtools target installs tools required for 'go generate'.
# You need to put $GOBIN (or $GOPATH/bin) in your PATH to use 'go generate'.

devtools:
	env GOBIN= go get -u golang.org/x/tools/cmd/stringer
	env GOBIN= go get -u github.com/kevinburke/go-bindata/go-bindata
	env GOBIN= go get -u github.com/fjl/gencodec
	env GOBIN= go get -u github.com/golang/protobuf/protoc-gen-go
	env GOBIN= go install ./cmd/abigen
	@type "npm" 2> /dev/null || echo 'Please install node.js and npm'
	@type "solc" 2> /dev/null || echo 'Please install solc'
	@type "protoc" 2> /dev/null || echo 'Please install protoc'

# Cross Compilation Targets (xgo)

gubet-cross: gubet-linux gubet-darwin gubet-windows gubet-android gubet-ios
	@echo "Full cross compilation done:"
	@ls -ld $(GOBIN)/gubet-*

gubet-linux: gubet-linux-386 gubet-linux-amd64 gubet-linux-arm gubet-linux-mips64 gubet-linux-mips64le
	@echo "Linux cross compilation done:"
	@ls -ld $(GOBIN)/gubet-linux-*

gubet-linux-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/386 -v ./cmd/gubet
	@echo "Linux 386 cross compilation done:"
	@ls -ld $(GOBIN)/gubet-linux-* | grep 386

gubet-linux-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/amd64 -v ./cmd/gubet
	@echo "Linux amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gubet-linux-* | grep amd64

gubet-linux-arm: gubet-linux-arm-5 gubet-linux-arm-6 gubet-linux-arm-7 gubet-linux-arm64
	@echo "Linux ARM cross compilation done:"
	@ls -ld $(GOBIN)/gubet-linux-* | grep arm

gubet-linux-arm-5:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-5 -v ./cmd/gubet
	@echo "Linux ARMv5 cross compilation done:"
	@ls -ld $(GOBIN)/gubet-linux-* | grep arm-5

gubet-linux-arm-6:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-6 -v ./cmd/gubet
	@echo "Linux ARMv6 cross compilation done:"
	@ls -ld $(GOBIN)/gubet-linux-* | grep arm-6

gubet-linux-arm-7:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-7 -v ./cmd/gubet
	@echo "Linux ARMv7 cross compilation done:"
	@ls -ld $(GOBIN)/gubet-linux-* | grep arm-7

gubet-linux-arm64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm64 -v ./cmd/gubet
	@echo "Linux ARM64 cross compilation done:"
	@ls -ld $(GOBIN)/gubet-linux-* | grep arm64

gubet-linux-mips:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips --ldflags '-extldflags "-static"' -v ./cmd/gubet
	@echo "Linux MIPS cross compilation done:"
	@ls -ld $(GOBIN)/gubet-linux-* | grep mips

gubet-linux-mipsle:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mipsle --ldflags '-extldflags "-static"' -v ./cmd/gubet
	@echo "Linux MIPSle cross compilation done:"
	@ls -ld $(GOBIN)/gubet-linux-* | grep mipsle

gubet-linux-mips64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64 --ldflags '-extldflags "-static"' -v ./cmd/gubet
	@echo "Linux MIPS64 cross compilation done:"
	@ls -ld $(GOBIN)/gubet-linux-* | grep mips64

gubet-linux-mips64le:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64le --ldflags '-extldflags "-static"' -v ./cmd/gubet
	@echo "Linux MIPS64le cross compilation done:"
	@ls -ld $(GOBIN)/gubet-linux-* | grep mips64le

gubet-darwin: gubet-darwin-386 gubet-darwin-amd64
	@echo "Darwin cross compilation done:"
	@ls -ld $(GOBIN)/gubet-darwin-*

gubet-darwin-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/386 -v ./cmd/gubet
	@echo "Darwin 386 cross compilation done:"
	@ls -ld $(GOBIN)/gubet-darwin-* | grep 386

gubet-darwin-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/amd64 -v ./cmd/gubet
	@echo "Darwin amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gubet-darwin-* | grep amd64

gubet-windows: gubet-windows-386 gubet-windows-amd64
	@echo "Windows cross compilation done:"
	@ls -ld $(GOBIN)/gubet-windows-*

gubet-windows-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/386 -v ./cmd/gubet
	@echo "Windows 386 cross compilation done:"
	@ls -ld $(GOBIN)/gubet-windows-* | grep 386

gubet-windows-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/amd64 -v ./cmd/gubet
	@echo "Windows amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gubet-windows-* | grep amd64
