.PHONY: build_and_run tests

URL_PARAMETER := $(if $(URL), --url $(URL),)
DEPTH_PARAMETER := $(if $(DEPTH), --depth $(DEPTH),)
MAX_CONCURRENCY_PARAMETER := $(if $(MAX_CONCURRENCY), --max_concurrency $(MAX_CONCURRENCY),)
TIMEOUT_PARAMETER := $(if $(TIMEOUT), --timeout $(TIMEOUT),)
RETRIES_PARAMETER := $(if $(RETRIES), --retries $(RETRIES),)

build_and_run:
	go build ./cmd/crawler
	./crawler $(URL_PARAMETER) $(DEPTH_PARAMETER) $(MAX_CONCURRENCY_PARAMETER) $(TIMEOUT_PARAMETER) $(RETRIES_PARAMETER)

tests:
	go test ./... -v