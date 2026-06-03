NAME=registrator
DEV_RUN_OPTS ?= consul:

local:
	docker build -t $(NAME):local .

dev:
	docker build -t $(NAME):dev .
	docker run --rm \
		-v /var/run/docker.sock:/tmp/docker.sock \
		$(NAME):dev /bin/registrator $(DEV_RUN_OPTS)

lint:
	golangci-lint run ./...

docs-lock:
	pip-compile --generate-hashes --output-file docs/requirements.txt docs/requirements.in

.PHONY: local dev lint docs-lock
