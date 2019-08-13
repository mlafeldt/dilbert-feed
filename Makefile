ENV        ?= dev
FUNCS      := $(subst /,,$(dir $(wildcard */main.go)))
SERVICE    := $(shell awk '/^service:/ {print $$2}' serverless.yml)
S3_BUCKET  := dilbert-feed-sam
STACK_NAME := dilbert-feed-sam-$(ENV)

dev: ENV=dev
dev: deploy

prod: ENV=prod
prod: deploy

package: zip
	sam package --s3-bucket $(S3_BUCKET) \
		--template-file infrastructure.yaml \
		--output-template-file build/packaged.yaml

deploy: package
	sam deploy --template-file build/packaged.yaml \
		--stack-name $(STACK_NAME) \
		--force-upload \
		--capabilities CAPABILITY_IAM

destroy:
	aws cloudformation delete-stack --stack-name $(STACK_NAME)

build_funcs = $(FUNCS:%=build-%)

build: $(build_funcs)

$(build_funcs):
	GOOS=linux GOARCH=amd64 go build -o build/$(@:build-%=%) ./$(@:build-%=%)

zip_funcs = $(FUNCS:%=zip-%)

zip: $(zip_funcs)

$(zip_funcs): zip-%: build-%
	(cd build; zip $(@:zip-%=%).zip $(@:zip-%=%))

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

.PHONY: build
