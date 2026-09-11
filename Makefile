APP_NAME ?= asana-extractor
IMAGE ?= $(APP_NAME):latest
CONTAINER_NAME ?= $(APP_NAME)
CONFIG ?= config/config.yaml
CONFIG_EXAMPLE ?= config/configExample.yaml
OUTPUT_DIR ?= output

.PHONY: config check-config build test vet run docker-build docker-run docker-run-foreground docker-stop docker-logs clean

config:
	@test -f $(CONFIG) || cp $(CONFIG_EXAMPLE) $(CONFIG)

check-config:
	@test -f $(CONFIG) || (echo "Missing $(CONFIG). Run 'make config' and set http.access_token first."; exit 1)
	@! grep -q "put_personal_access_token_here" $(CONFIG) || (echo "Set http.access_token in $(CONFIG) before running."; exit 1)

build:
	@mkdir -p bin
	go build -o bin/$(APP_NAME) ./cmd

test:
	go test ./...

vet:
	go vet ./...

run: check-config
	go run ./cmd --config $(CONFIG)

docker-build:
	docker build -f infra/Dockerfile -t $(IMAGE) .

docker-run: check-config docker-build
	@mkdir -p $(OUTPUT_DIR)
	docker run --rm -d \
		--name $(CONTAINER_NAME) \
		-v "$(PWD)/$(CONFIG):/app/config/config.yaml:ro" \
		-v "$(PWD)/$(OUTPUT_DIR):/app/output" \
		$(IMAGE)

docker-run-foreground: check-config docker-build
	@mkdir -p $(OUTPUT_DIR)
	docker run --rm \
		--name $(CONTAINER_NAME) \
		-v "$(PWD)/$(CONFIG):/app/config/config.yaml:ro" \
		-v "$(PWD)/$(OUTPUT_DIR):/app/output" \
		$(IMAGE)

docker-stop:
	-docker rm -f $(CONTAINER_NAME)

docker-logs:
	docker logs -f $(CONTAINER_NAME)

clean:
	rm -rf bin
