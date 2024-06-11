watch_test:
	find . | entr -c -r make test

test:
	go test ./...
