ENV   ?= dev
STACK  = dilbert-feed-$(ENV)-ts
FUNCS := $(subst /,,$(dir $(wildcard */main.go)))
CDK   ?= ./node_modules/.bin/cdk

#
# deploy & destroy
#

dev: ENV=dev
dev: deploy

prod: ENV=prod
prod: deploy

deploy diff synth: build transpile
	@$(CDK) $@ $(STACK)

deploy: test

destroy: build transpile
	@$(CDK) destroy --force $(STACK)

bootstrap: build transpile
	@$(CDK) bootstrap

transpile: node_modules
	@npm run build

node_modules:
	npm install

#
# build
#

build_funcs := $(FUNCS:%=build-%)

build: $(build_funcs)

$(build_funcs):
	mkdir -p bin/$(@:build-%=%)
	GOOS=linux GOARCH=amd64 go build -trimpath -ldflags=-buildid= -o bin/$(@:build-%=%)/handler ./$(@:build-%=%)

#
# lint
#

lint:
	go vet ./...
	golint -set_exit_status $$(go list ./...)

lint_funcs := $(FUNCS:%=lint-%)

$(lint_funcs):
	go vet ./$(@:lint-%=%)
	golint -set_exit_status ./$(@:lint-%=%)

#
# test
#

test:
	go test -v -cover -count=1 ./...

test_funcs := $(FUNCS:%=test-%)

$(test_funcs):
	go test -v -cover ./$(@:test-%=%)
