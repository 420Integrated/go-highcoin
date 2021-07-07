# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: highcoin android ios highcoin-cross evm all test clean
.PHONY: highcoin-linux highcoin-linux-386 highcoin-linux-amd64 highcoin-linux-mips64 highcoin-linux-mips64le
.PHONY: highcoin-linux-arm highcoin-linux-arm-5 highcoin-linux-arm-6 highcoin-linux-arm-7 highcoin-linux-arm64
.PHONY: highcoin-darwin highcoin-darwin-386 highcoin-darwin-amd64
.PHONY: highcoin-windows highcoin-windows-386 highcoin-windows-amd64

GOBIN = ./build/bin
GO ?= latest
GORUN = env GO111MODULE=on go run

highcoin:
	$(GORUN) build/ci.go install ./cmd/highcoin
	@echo "Done building."
	@echo "Run \"$(GOBIN)/highcoin\" to launch highcoin."

all:
	$(GORUN) build/ci.go install

android:
	$(GORUN) build/ci.go aar --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/highcoin.aar\" to use the library."
	@echo "Import \"$(GOBIN)/highcoin-sources.jar\" to add javadocs"
	@echo "For more info see https://stackoverflow.com/questions/20994336/android-studio-how-to-attach-javadoc"
	
ios:
	$(GORUN) build/ci.go xcode --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/Highcoin.framework\" to use the library."

test: all
	$(GORUN) build/ci.go test

lint: ## Run linters.
	$(GORUN) build/ci.go lint

clean:
	env GO111MODULE=on go clean -cache
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

highcoin-cross: highcoin-linux highcoin-darwin highcoin-windows highcoin-android highcoin-ios
	@echo "Full cross compilation done:"
	@ls -ld $(GOBIN)/highcoin-*

highcoin-linux: highcoin-linux-386 highcoin-linux-amd64 highcoin-linux-arm highcoin-linux-mips64 highcoin-linux-mips64le
	@echo "Linux cross compilation done:"
	@ls -ld $(GOBIN)/highcoin-linux-*

highcoin-linux-386:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/386 -v ./cmd/highcoin
	@echo "Linux 386 cross compilation done:"
	@ls -ld $(GOBIN)/highcoin-linux-* | grep 386

highcoin-linux-amd64:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/amd64 -v ./cmd/highcoin
	@echo "Linux amd64 cross compilation done:"
	@ls -ld $(GOBIN)/highcoin-linux-* | grep amd64

highcoin-linux-arm: highcoin-linux-arm-5 highcoin-linux-arm-6 highcoin-linux-arm-7 highcoin-linux-arm64
	@echo "Linux ARM cross compilation done:"
	@ls -ld $(GOBIN)/highcoin-linux-* | grep arm

highcoin-linux-arm-5:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/arm-5 -v ./cmd/highcoin
	@echo "Linux ARMv5 cross compilation done:"
	@ls -ld $(GOBIN)/highcoin-linux-* | grep arm-5

highcoin-linux-arm-6:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/arm-6 -v ./cmd/highcoin
	@echo "Linux ARMv6 cross compilation done:"
	@ls -ld $(GOBIN)/highcoin-linux-* | grep arm-6

highcoin-linux-arm-7:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/arm-7 -v ./cmd/highcoin
	@echo "Linux ARMv7 cross compilation done:"
	@ls -ld $(GOBIN)/highcoin-linux-* | grep arm-7

highcoin-linux-arm64:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/arm64 -v ./cmd/highcoin
	@echo "Linux ARM64 cross compilation done:"
	@ls -ld $(GOBIN)/highcoin-linux-* | grep arm64

highcoin-linux-mips:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/mips --ldflags '-extldflags "-static"' -v ./cmd/highcoin
	@echo "Linux MIPS cross compilation done:"
	@ls -ld $(GOBIN)/highcoin-linux-* | grep mips

highcoin-linux-mipsle:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/mipsle --ldflags '-extldflags "-static"' -v ./cmd/highcoin
	@echo "Linux MIPSle cross compilation done:"
	@ls -ld $(GOBIN)/highcoin-linux-* | grep mipsle

highcoin-linux-mips64:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/mips64 --ldflags '-extldflags "-static"' -v ./cmd/highcoin
	@echo "Linux MIPS64 cross compilation done:"
	@ls -ld $(GOBIN)/highcoin-linux-* | grep mips64

highcoin-linux-mips64le:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/mips64le --ldflags '-extldflags "-static"' -v ./cmd/highcoin
	@echo "Linux MIPS64le cross compilation done:"
	@ls -ld $(GOBIN)/highcoin-linux-* | grep mips64le

highcoin-darwin: highcoin-darwin-386 highcoin-darwin-amd64
	@echo "Darwin cross compilation done:"
	@ls -ld $(GOBIN)/highcoin-darwin-*

highcoin-darwin-386:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=darwin/386 -v ./cmd/highcoin
	@echo "Darwin 386 cross compilation done:"
	@ls -ld $(GOBIN)/highcoin-darwin-* | grep 386

highcoin-darwin-amd64:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=darwin/amd64 -v ./cmd/highcoin
	@echo "Darwin amd64 cross compilation done:"
	@ls -ld $(GOBIN)/highcoin-darwin-* | grep amd64

highcoin-windows: highcoin-windows-386 highcoin-windows-amd64
	@echo "Windows cross compilation done:"
	@ls -ld $(GOBIN)/highcoin-windows-*

highcoin-windows-386:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=windows/386 -v ./cmd/highcoin
	@echo "Windows 386 cross compilation done:"
	@ls -ld $(GOBIN)/highcoin-windows-* | grep 386

highcoin-windows-amd64:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=windows/amd64 -v ./cmd/highcoin
	@echo "Windows amd64 cross compilation done:"
	@ls -ld $(GOBIN)/highcoin-windows-* | grep amd64
