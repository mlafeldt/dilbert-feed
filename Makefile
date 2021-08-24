ENV   ?= dev
STACK  = dilbert-feed-$(ENV)
CDK   ?= yarn cdk
GOX   ?= gox

dev: ENV=dev
dev: deploy

prod: ENV=prod
prod: deploy

deploy diff synth: build node_modules
	@$(CDK) $@ $(STACK)

deploy: test

destroy: build node_modules
	@$(CDK) destroy --force $(STACK)

bootstrap: build node_modules
	@$(CDK) bootstrap --cloudformation-execution-policies arn:aws:iam::aws:policy/AdministratorAccess

node_modules:
	yarn install

build: rust
	@GOFLAGS=-trimpath $(GOX) -os=linux -arch=amd64 -ldflags=-s -output="bin/{{.Dir}}/handler" ./...

lint:
	go vet ./...
	golint -set_exit_status $$(go list ./...)

test:
	go test -v -cover ./...

# https://github.com/messense/homebrew-macos-cross-toolchains
TARGET := x86_64-unknown-linux-gnu
export CC_x86_64_unknown_linux_gnu  = $(TARGET)-gcc
export CXX_x86_64_unknown_linux_gnu = $(TARGET)-g++
export AR_x86_64_unknown_linux_gnu  = $(TARGET)-ar
export CARGO_TARGET_X86_64_UNKNOWN_LINUX_GNU_LINKER = $(TARGET)-gcc
export RUSTFLAGS = -C link-arg=-s

RUST_FUNCS := $(subst src/bin/,,$(dir $(wildcard src/bin/*/main.rs)))

rust_funcs := $(RUST_FUNCS:%=rust-%)

rust: $(rust_funcs)

$(rust_funcs):
	cargo build --release --target $(TARGET) --bin $(@:rust-%=%)
	mkdir -p bin/$(@:rust-%=%)
	cp -f target/$(TARGET)/release/$(@:rust-%=%) bin/$(@:rust-%=%)/bootstrap
