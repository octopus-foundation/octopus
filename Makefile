protobuf:
	go run -mod vendor build-tools/gremlin/gremlin.go

binaries:
	go run -mod vendor build-tools/bin-maker/bin-maker.go

binary-only:
	go run build-tools/bin-maker/bin-maker.go -root ${BIN_PATH} -excludes="${EXCLUDES}"

checks:
	go vet -mod=vendor ./...

tests:
	go test -failfast ./...

full-tests:
	go test -tags integration ./...