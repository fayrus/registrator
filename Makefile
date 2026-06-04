NAME=registrator
DEV_RUN_OPTS ?= consul:
LINT_OUTPUT ?= golangci-lint.out

local:
	docker build -t $(NAME):local .

dev:
	docker build -t $(NAME):dev .
	docker run --rm \
		-v /var/run/docker.sock:/tmp/docker.sock \
		$(NAME):dev /bin/registrator $(DEV_RUN_OPTS)

lint:
	golangci-lint run ./...

lint-output:
	@golangci-lint run ./... > $(LINT_OUTPUT) 2>&1; status=$$?; \
	echo "golangci-lint output written to $(LINT_OUTPUT)"; \
	exit $$status

tidy:
	go mod tidy

docs-lock:
	pip-compile --generate-hashes --output-file docs/requirements.txt docs/requirements.in

.PHONY: local dev lint lint-output tidy docs-lock
