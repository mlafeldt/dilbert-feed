ENV   ?= dev
STACK  = dilbert-feed-$(ENV)
CDK   ?= yarn --silent cdk
CARGO ?= cargo

ifeq ("$(origin V)", "command line")
  VERBOSE = $(V)
endif
ifneq ($(VERBOSE),1)
.SILENT:
endif

dev: ENV=dev
dev: deploy

prod: ENV=prod
prod: deploy

deploy: lint test build node_modules
	$(CDK) $@ -e $(STACK)

diff synth: build node_modules
	$(CDK) $@ -e $(STACK)

destroy: build node_modules
	$(CDK) destroy --force $(STACK)

bootstrap: build node_modules
	$(CDK) bootstrap --cloudformation-execution-policies arn:aws:iam::aws:policy/AdministratorAccess

node_modules:
	yarn install

TARGET := x86_64-unknown-linux-gnu
export CC_x86_64_unknown_linux_gnu  = $(TARGET)-gcc
export CXX_x86_64_unknown_linux_gnu = $(TARGET)-g++
export AR_x86_64_unknown_linux_gnu  = $(TARGET)-ar
export CARGO_TARGET_X86_64_UNKNOWN_LINUX_GNU_LINKER = $(TARGET)-gcc

RUST_FUNCS := $(subst src/bin/,,$(dir $(wildcard src/bin/*/main.rs)))
rust_funcs := $(RUST_FUNCS:%=rust-%)

build: $(rust_funcs)

$(rust_funcs):
	RUSTFLAGS="-C link-arg=-s" $(CARGO) build --release --target $(TARGET) --bin $(@:rust-%=%) $(if $(VERBOSE),--verbose,)
	mkdir -p bin/$(@:rust-%=%)
	cp -f target/$(TARGET)/release/$(@:rust-%=%) bin/$(@:rust-%=%)/bootstrap

lint:
	$(CARGO) clippy --workspace $(if $(VERBOSE),--verbose,)

test:
	$(CARGO) test --workspace $(if $(VERBOSE),--verbose,)
