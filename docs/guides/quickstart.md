# Quickstart

Save links, one per line:

```text
vless://uuid@example.com:443#demo
```

Parse:

```sh
a2p parse ./links.txt
```

Output:

```json
{"nodes":[{"tag":"demo","protocol":"vless","server":"example.com","port":443,"transport":"tcp","account":"uuid"}]}
```

Compile and run:

```sh
a2p compile ./links.txt
a2p run --binary ./sing-box ./links.txt
```

Endpoint:

```text
socks5://127.0.0.1:1080
```

Mihomo and Xray:

```sh
a2p compile --to mihomo ./links.txt
a2p run --binary ./mihomo --to mihomo ./links.txt

a2p compile --to xray ./links.txt
a2p run --binary ./xray --to xray ./links.txt
```

Subscriptions:

```sh
a2p sub add main https://example.com/sub
a2p sub refresh main
a2p run --binary ./sing-box sub:main
```

Next: [Troubleshooting](troubleshooting.md).
