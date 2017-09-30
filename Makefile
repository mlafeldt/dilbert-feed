ENV = staging

deploy:
	@up deploy $(ENV) -v
	@up url $(ENV)

stage: ENV=staging
stage: deploy

prod: ENV=production
prod: deploy

destroy:
	@up stack delete --async
