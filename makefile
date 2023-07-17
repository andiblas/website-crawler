build_and_run:
	go build ./cmd/crawler
	./crawler --url $(URL)

tests:
	go test ./... -v