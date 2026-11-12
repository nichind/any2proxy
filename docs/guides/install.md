# install

### release binary

unix:

```sh
curl -fsSL https://raw.githubusercontent.com/nichind/any2proxy/main/packaging/install.sh | sh
```

windows:

```powershell
irm https://raw.githubusercontent.com/nichind/any2proxy/main/packaging/install.ps1 | iex
```

### from source

go 1.25+:

```sh
go install ./cmd/a2p
a2p version
```

```text
a2p 0.1.0-dev
```

if `a2p` is missing, add `$(go env GOPATH)/bin` or `%USERPROFILE%\go\bin` to `PATH`.
