# contributing

```sh
go test ./...
go vet ./...
gofmt -l ./cmd ./internal ./tests
buf lint
buf generate
```

generated code is rebuilt with `buf generate`.
