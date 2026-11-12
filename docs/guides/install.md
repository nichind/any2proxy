# Install

## Release binary

Unix:

```sh
curl -fsSL https://raw.githubusercontent.com/nichind/any2proxy/main/packaging/install.sh | sh
```

Windows:

```powershell
irm https://raw.githubusercontent.com/nichind/any2proxy/main/packaging/install.ps1 | iex
```

## From source

Go 1.25 or newer.

```sh
go install ./cmd/a2p
a2p version
```


```text
a2p 0.1.0-dev
```

`PATH`:
- Unix: `$(go env GOPATH)/bin`
- Windows: `%USERPROFILE%\go\bin`

From source:

```sh
git clone https://github.com/nichind/any2proxy
cd any2proxy
go build ./cmd/a2p
```
