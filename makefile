URL_PARAMETER=
TIMEOUT_PARAMETER=
RETRIES_PARAMETER=

ifneq ($(URL),)
	URL_PARAMETER=--url $(URL)
endif

ifneq ($(TIMEOUT),)
	TIMEOUT_PARAMETER=--timeout $(TIMEOUT)
endif

ifneq ($(RETRIES),)
	RETRIES_PARAMETER=--retries $(RETRIES)
endif

build_and_run:
	go build ./cmd/crawler
	./crawler $(URL_PARAMETER) $(TIMEOUT_PARAMETER) $(RETRIES_PARAMETER)

tests:
	go test ./... -v