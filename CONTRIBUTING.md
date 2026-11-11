# Contributing

```sh
go test ./...
go vet ./...
gofmt -l ./cmd ./internal ./tests
buf lint
buf generate
```

Rules:
- comments: only why, never what
- generated code: not edited by hand
- one change per PR
