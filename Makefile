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

RUST_FUNCS := $(subst /,,$(dir $(wildcard */lambda.rs)))

rust_funcs := $(RUST_FUNCS:%=rust-%)

rust: $(rust_funcs)

$(rust_funcs):
	RUSTFLAGS="-C link-arg=-s" cargo build --release --target x86_64-unknown-linux-musl --bin $(@:rust-%=%)
	mkdir -p bin/$(@:rust-%=%)
	cp -f target/x86_64-unknown-linux-musl/release/$(@:rust-%=%) bin/$(@:rust-%=%)/bootstrap
