# any2proxy

local proxy from links, subscriptions and clash configs. runs sing-box, mihomo or xray underneath.

### install

with go 1.25+:

```sh
go install ./cmd/a2p
```

or grab a release binary:

```sh
curl -fsSL https://raw.githubusercontent.com/nichind/any2proxy/main/packaging/install.sh | sh
```

### use

```sh
a2p parse ./links.txt
a2p links ./config.yaml
a2p compile ./links.txt
a2p run --binary ./sing-box ./links.txt
```

`a2p run` prints one line:

```text
socks5://127.0.0.1:1080
```

### cores

```sh
a2p cores install sing-box
a2p cores install mihomo
a2p cores install xray
a2p cores versions sing-box
a2p cores update --all
```

forks and direct links:

```sh
a2p cores install fork --repo owner/sing-box --bin-name sing-box
a2p cores install custom --url https://example.com/core.zip --sha256 <hex> --bin-name core
```

`--core` picks an installed binary, `--to` picks the config format.

### subscriptions

```sh
a2p sub add main https://example.com/sub
a2p sub refresh main
a2p run --binary ./sing-box sub:main
```

### modes

```sh
a2p run --mode auto --binary ./sing-box ./links.txt
a2p run --mode pool --binary ./sing-box ./links.txt
a2p run --mode balance --to mihomo ./config.yaml
```

`balance` needs mihomo or sing-box. xray has no groups.

### protocols

`a2p` = parse, links and config emit. other columns = core support.

| protocol | a2p | sing-box | mihomo | xray |
|---|:---:|:---:|:---:|:---:|
| vless | yes | yes | yes | yes |
| vmess | yes | yes | yes | yes |
| trojan | yes | yes | yes | yes |
| shadowsocks | yes | yes | yes | yes |
| hysteria2 | yes | yes | yes | no |
| tuic | yes | yes | yes | no |
| anytls | yes | yes | yes | no |
| hysteria | no | yes | yes | yes |
| shadowtls | no | yes | no | no |
| snell | no | yes | yes | no |
| ssh | no | yes | yes | no |
| tor | no | yes | no | no |
| wireguard | no | yes | yes | yes |
| naive | no | yes | no | no |
| shadowsocksr | no | no | yes | no |
| mieru | no | no | yes | no |
| sudoku | no | no | yes | no |
| shadowquic | no | no | yes | no |
| trusttunnel | no | no | yes | no |
| masque | no | no | yes | yes |
| zerotier | no | no | yes | no |
| openvpn | no | no | yes | no |
| easytier | no | no | yes | no |
| tailscale | no | no | yes | no |
| tailcat | no | yes | no | no |
| socks | no | yes | yes | yes |
| http | no | yes | yes | yes |

built-in outbounds, not proxy nodes: `direct`, `block`, `dns`, `freedom`, `blackhole`, `loopback`, `bridge`, `rematch`.

### groups

| group | a2p | mihomo | sing-box |
|---|:---:|:---:|:---:|
| select | yes | yes | yes |
| url-test | yes | yes | yes |
| fallback | yes | yes | partial |
| load-balance | yes | yes | partial |
| relay | yes | yes | no |

`fallback` and `load-balance` are passed through to mihomo. sing-box maps them to `selector`
because it has no direct equivalent. `relay` is not compiled.

### test

```sh
a2p test --to mihomo ./config.yaml
a2p test --to sing-box --concurrency 4 ./config.yaml
```

mihomo tests run sequentially. `--concurrency` is ignored for mihomo.

### sdk

all sdks talk to `a2pd` over json-rpc.

- go: [sdk/go](sdk/go)
- python: [sdk/python](sdk/python)
- node: [sdk/node](sdk/node)
- rust: [sdk/rust](sdk/rust)

```sh
go install ./cmd/a2pd
printf '{"id":1,"method":"version"}\n' | a2pd
```

### c abi

```sh
CGO_ENABLED=1 go build -buildmode=c-shared -o dist/libany2proxy.dll ./cmd/libany2proxy
python cabi/test.py dist/libany2proxy.dll
```

### docs

- [install](docs/guides/install.md)
- [quickstart](docs/guides/quickstart.md)
- [troubleshooting](docs/guides/troubleshooting.md)
- [cores](docs/guides/cores.md)
- [security](docs/security.md)

### license

see [LICENSE](LICENSE) and [NOTICE](NOTICE).
