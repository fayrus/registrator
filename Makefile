NAME=registrator
DEV_RUN_OPTS ?= consul:

local:
	docker build -t $(NAME):local .

dev:
	docker build -t $(NAME):dev .
	docker run --rm \
		-v /var/run/docker.sock:/tmp/docker.sock \
		$(NAME):dev /bin/registrator $(DEV_RUN_OPTS)

.PHONY: local dev
