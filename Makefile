ENV   ?= dev
STACK  = dilbert-feed-$(ENV)
CDK   ?= yarn cdk
GOX   ?= gox

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
	@$(CDK) bootstrap --cloudformation-execution-policies arn:aws:iam::aws:policy/AdministratorAccess

transpile: node_modules
	@yarn run build

node_modules:
	yarn install

build:
	@GOFLAGS=-trimpath $(GOX) -os=linux -arch=amd64 -ldflags=-s -output="bin/{{.Dir}}/handler" ./functions/...

lint:
	go vet ./functions/...
	golint -set_exit_status $$(go list ./functions/...)

test:
	go test -v -cover ./functions/...
