ENV   ?= dev
STACK := dilbert-feed-cdk-$(ENV)
FUNCS := $(subst /,,$(dir $(wildcard */main.go)))

#
# deploy & destroy
#

dev: ENV=dev
dev: deploy

prod: ENV=prod
prod: deploy

venv:
	python3 -m venv .venv
	.venv/bin/pip install -r requirements.txt

deploy: zip
	cdk deploy $(STACK)

diff: zip
	cdk diff $(STACK)

destroy:
	cdk destroy --force $(STACK)

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
