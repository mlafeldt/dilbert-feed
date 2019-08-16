ENV        ?= dev
FUNCS      := $(subst /,,$(dir $(wildcard */main.go)))
STACK_NAME := dilbert-feed-sam-$(ENV)
SAM_BUCKET := dilbert-feed-sam-$(ENV)-sam

#
# deploy & destroy
#

dev: ENV=dev
dev: deploy

prod: ENV=prod
prod: deploy

deploy: package
	sam deploy --template-file build/packaged.yaml \
		--stack-name $(STACK_NAME) \
		--force-upload \
		--capabilities CAPABILITY_IAM

package: bucket zip
	sam package --s3-bucket $(SAM_BUCKET) \
		--template-file infrastructure.yaml \
		--output-template-file build/packaged.yaml

bucket:
	@aws s3api head-bucket --bucket $(SAM_BUCKET) || \
		aws s3api create-bucket --bucket $(SAM_BUCKET) \
			--create-bucket-configuration LocationConstraint=$(AWS_REGION)

destroy:
	aws cloudformation delete-stack --stack-name $(STACK_NAME)

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

test_funcs := $(FUNCS:%=test-%)

$(test_funcs):
	go vet ./$(@:test-%=%)
	go test -v -cover ./$(@:test-%=%)
