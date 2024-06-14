test:
	go test ./...

watch_test:
	find . | entr -c -r make test

lint:
	golangci-lint run ./...
	regal lint .

watch_lint:
	find . | entr -c -r make lint
