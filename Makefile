ENV     = dev
FUNCS   = $(subst /,,$(dir $(wildcard */main.go)))
SERVICE = $(shell awk '/^service:/ {print $$2}' serverless.yml)

dev: ENV=dev
dev: deploy

prod: ENV=prod
prod: deploy

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
