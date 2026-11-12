# contributing

before a pr:

```sh
go test ./...
go vet ./...
gofmt -l ./cmd ./internal ./tests
buf lint
buf generate
```

keep comments rare: explain why, not what. don't edit generated code. keep prs small.
