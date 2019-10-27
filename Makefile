ENV   ?= dev
STACK  = dilbert-feed-$(ENV)
FUNCS := $(subst /,,$(dir $(wildcard */main.go)))

#
# deploy & destroy
#

dev: ENV=dev
dev: deploy

prod: ENV=prod
prod: deploy

deploy diff synth: venv build
	@cdk $@ $(STACK)

deploy: test

destroy: venv build
	@cdk destroy --force $(STACK)

bootstrap: venv build
	@cdk bootstrap

venv:
	python3 -m venv $@
	venv/bin/pip install -r requirements.txt

#
# build
#

build_funcs := $(FUNCS:%=build-%)

build: $(build_funcs)

$(build_funcs):
	mkdir -p bin/$(@:build-%=%)
	GOOS=linux GOARCH=amd64 go build -o bin/$(@:build-%=%)/handler ./$(@:build-%=%)

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
