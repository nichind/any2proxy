# any2proxy

Local proxy from links, subscriptions and Clash configs. Backends: sing-box, mihomo, Xray.

## Install

Go 1.25 or newer:

```sh
go install ./cmd/a2p
```

## Use

```sh
a2p parse ./links.txt
a2p compile ./links.txt
a2p run --binary ./sing-box ./links.txt
```

`a2p run` prints one endpoint:

```text
socks5://127.0.0.1:1080
```

Compile for another core:

```sh
a2p compile --to mihomo ./links.txt
a2p compile --to xray ./links.txt
a2p run --binary ./mihomo --to mihomo ./links.txt
```

Cores:

```sh
a2p cores install sing-box
a2p cores list
a2p cores verify sing-box
a2p run --to sing-box ./links.txt
a2p run --to sing-box --core fork ./links.txt
```

Forks and direct links:

```sh
a2p cores install fork --repo owner/sing-box --bin-name sing-box
a2p cores install custom --url https://example.com/core.zip --sha256 <hex> --bin-name core
```

Subscriptions:

```sh
a2p sub add main https://example.com/sub
a2p sub refresh main
a2p run --binary ./sing-box sub:main
```

Modes:

```sh
a2p run --mode auto --binary ./sing-box ./links.txt
a2p run --mode pool --binary ./sing-box ./links.txt
```

Environment and endpoint checks:

```sh
a2p doctor
a2p check socks5://127.0.0.1:1080
a2p check --url https://example.com socks5://127.0.0.1:1080
```

## SDKs

All SDKs speak JSON-RPC to `a2pd`.

- Go: `sdk/go`
- Python: `sdk/python`
- Node: `sdk/node`
- Rust: `sdk/rust`

```sh
go install ./cmd/a2pd
```

```sh
printf '{"id":1,"method":"version"}\n' | a2pd
```

## C ABI

```sh
CGO_ENABLED=1 go build -buildmode=c-shared -o dist/libany2proxy.dll ./cmd/libany2proxy
python cabi/test.py dist/libany2proxy.dll
```

Functions: `a2p_new`, `a2p_call`, `a2p_version`, `a2p_free`, `a2p_close`.

## Status

- `parse` handles plain link lists, Clash YAML and subscription URLs.
- `compile` emits sing-box JSON, mihomo YAML and Xray JSON.
- `run` starts a core, prints endpoints and fails over in `auto` mode.
- `cores` installs from GitHub Releases, verifies sha256, imports local binaries and resolves `--to` automatically.
- Subscription URLs are DPAPI-encrypted on Windows and mode 0600 elsewhere.

## Docs

- [Install](docs/guides/install.md)
- [Quickstart](docs/guides/quickstart.md)
- [Troubleshooting](docs/guides/troubleshooting.md)
- [Cores](docs/guides/cores.md)
- [Security](docs/security.md)
- [Licensing](docs/licensing.md)

## License

MIT. Cores: sing-box GPL-3.0, mihomo GPL-3.0, Xray MPL-2.0. See `NOTICE`.
