.PHONY: build
build:
	sam build

build-CobblerUsersFunction:
	GOOS=linux CGO_ENABLED=0 go build -tags lambda.norpc -o $(ARTIFACTS_DIR)/bootstrap .

.PHONY: init
init: build
	sam deploy --guided

.PHONY: deploy
deploy: build
	sam deploy

.PHONY: delete
delete:
	sam delete
