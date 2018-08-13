ENV     = staging
FUNCS   = $(subst /,,$(dir $(wildcard */main.go)))
SERVICE = $(shell awk '/^service:/ {print $$2}' serverless.yml)

staging: ENV=staging
staging: deploy

production: ENV=production
production: deploy

deploy: test build
	serverless deploy --stage $(ENV) --verbose

deploy_funcs = $(FUNCS:%=deploy-%)

$(deploy_funcs): deploy-%: test-% build-%
	serverless deploy function --function $(@:deploy-%=%) --stage $(ENV) --verbose

destroy:
	serverless remove --stage $(ENV) --verbose

logs_funcs = $(FUNCS:%=logs-%)

$(logs_funcs):
	serverless logs --function $(@:logs-%=%) --stage $(ENV) --tail --no-color

url:
	@aws cloudformation describe-stacks --stack-name $(SERVICE)-$(ENV) \
		--query "Stacks[0].Outputs[?OutputKey == 'ServiceEndpoint'].OutputValue" \
		--output text

build_funcs = $(FUNCS:%=build-%)

build: $(build_funcs)

$(build_funcs):
	GOOS=linux GOARCH=amd64 go build -o bin/$(@:build-%=%) ./$(@:build-%=%)

test:
	go vet ./...
	go test -v -cover -count=1 ./...

test_funcs = $(FUNCS:%=test-%)

$(test_funcs):
	go vet ./$(@:test-%=%)
	go test -v -cover ./$(@:test-%=%)
