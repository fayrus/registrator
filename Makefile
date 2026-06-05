NAME=development/registrator
DEV_RUN_OPTS ?= consul:
LINT_OUTPUT ?= golangci-lint.out
COVERAGE_OUT ?= coverage.out

local:
	docker build --no-cache -t $(NAME):local .

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

test:
	go test ./...

coverage:
	go test ./... -coverprofile=$(COVERAGE_OUT)
	go tool cover -func=$(COVERAGE_OUT)

coverage-html:
	go test ./... -coverprofile=$(COVERAGE_OUT)
	go tool cover -html=$(COVERAGE_OUT)

tidy:
	go mod tidy

docs-lock:
	pip-compile --generate-hashes --output-file docs/requirements.txt docs/requirements.in

.PHONY: local dev lint lint-output test coverage coverage-html tidy docs-lock
