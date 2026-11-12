# install

```sh
pip install any2proxy
```

puts `a2p` and `a2pd` on `path`.

with go 1.25+:

```sh
go install ./cmd/a2p
a2p version
```

```text
a2p 0.2.0-dev
```

if `a2p` is missing, add `$(go env GOPATH)/bin` or `%USERPROFILE%\go\bin` to `PATH`.

then install a core:

```sh
a2p cores install sing-box
```
