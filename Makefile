protobuf:
	go run -mod vendor build-tools/gremlin/gremlin.go

checks:
	go vet -mod=vendor ./...

tests:
	go test -failfast ./...

full-tests:
	go test -tags integration ./...