# quickstart

put one link per line:

```text
vless://uuid@example.com:443#demo
```

parse:

```sh
a2p parse ./links.txt
```

```json
{"nodes":[{"tag":"demo","protocol":"vless","server":"example.com","port":443,"transport":"tcp","account":"uuid"}]}
```

run:

```sh
a2p run --binary ./sing-box ./links.txt
```

```text
socks5://127.0.0.1:1080
```

export links:

```sh
a2p links ./config.yaml
```

test every node:

```sh
a2p test --to mihomo ./config.yaml
```

mihomo and xray:

```sh
a2p compile --to mihomo ./links.txt
a2p run --binary ./mihomo --to mihomo ./links.txt

a2p compile --to xray ./links.txt
a2p run --binary ./xray --to xray ./links.txt
```

subscriptions:

```sh
a2p sub add main https://example.com/sub
a2p sub refresh main
a2p run --binary ./sing-box sub:main
```

next: [troubleshooting](troubleshooting.md).
