ENV        ?= dev
FUNCS      := $(subst /,,$(dir $(wildcard */main.go)))
SERVICE    := $(shell awk '/^service:/ {print $$2}' serverless.yml)
SERVERLESS := node_modules/.bin/serverless

dev: ENV=dev
dev: deploy

prod: ENV=prod
prod: deploy

deploy: test build $(SERVERLESS)
	$(SERVERLESS) deploy --stage $(ENV) --verbose

deploy_funcs = $(FUNCS:%=deploy-%)

$(deploy_funcs): deploy-%: test-% build-% $(SERVERLESS)
	$(SERVERLESS) deploy function --function $(@:deploy-%=%) --stage $(ENV) --verbose

destroy: $(SERVERLESS)
	$(SERVERLESS) remove --stage $(ENV) --verbose

logs_funcs = $(FUNCS:%=logs-%)

$(logs_funcs): $(SERVERLESS)
	$(SERVERLESS) logs --function $(@:logs-%=%) --stage $(ENV) --tail --no-color

$(SERVERLESS): node_modules

node_modules:
	npm install

#
# zip
#

zip_funcs := $(FUNCS:%=zip-%)

zip: $(zip_funcs)

$(zip_funcs): zip-%: build-%
	(cd build; zip $(@:zip-%=%).zip $(@:zip-%=%))

#
# build
#

build_funcs := $(FUNCS:%=build-%)

build: $(build_funcs)

$(build_funcs):
	GOOS=linux GOARCH=amd64 go build -o build/$(@:build-%=%) ./$(@:build-%=%)

.PHONY: build

#
# test
#

test:
	go vet ./...
	go test -v -cover -count=1 ./...

test_funcs = $(FUNCS:%=test-%)

$(test_funcs):
	go vet ./$(@:test-%=%)
	go test -v -cover ./$(@:test-%=%)

update-deps:
	go get -u ./...
	go mod tidy
