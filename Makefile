APP_ENV ?= dev
STACK    = dilbert-feed-$(APP_ENV)
CDK     ?= yarn --silent cdk
CARGO   ?= cargo

ifeq ("$(origin V)", "command line")
  VERBOSE = $(V)
endif
ifneq ($(VERBOSE),1)
.SILENT:
endif

dev: APP_ENV=dev
dev: HOTSWAP=1
dev: deploy

prod: APP_ENV=prod
prod: deploy

deploy: lint test build node_modules
	$(CDK) $@ -e $(STACK) $(if $(HOTSWAP),--hotswap,)

diff synth: build node_modules
	$(CDK) $@ -e $(STACK)

destroy: build node_modules
	$(CDK) destroy --force $(STACK)

bootstrap: build node_modules
	$(CDK) bootstrap --cloudformation-execution-policies arn:aws:iam::aws:policy/AdministratorAccess

node_modules:
	yarn install

TARGET := aarch64-unknown-linux-gnu

LAMBDA_FUNCS := $(notdir $(realpath $(dir $(wildcard src/bin/*/main.rs))))

build:
	RUSTFLAGS="-C link-arg=-s" $(CARGO) build --release --target $(TARGET) --bins $(if $(VERBOSE),--verbose,)
	for func in $(LAMBDA_FUNCS); do mkdir -p bin/$$func; cp -f target/$(TARGET)/release/$$func bin/$$func/bootstrap; done

lint:
	$(CARGO) clippy --workspace $(if $(VERBOSE),--verbose,)

test:
	$(CARGO) test --workspace $(if $(VERBOSE),--verbose,)
