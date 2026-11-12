# Benchmarks

Windows amd64, Go 1.25, single run.

| Test | Result |
|---|---|
| parse 1000 nodes | 81 ms |
| compile 1000 nodes to sing-box | 147 ms |
| sing-box start to endpoint | 714 ms |
| mihomo start to endpoint | 1376 ms |
| Xray start to endpoint | 1619 ms |
| sing-box RSS | 33.4 MB |
| mihomo RSS | 25.0 MB |
| Xray RSS | 21.7 MB |
| local SS loopback, 10 MiB | 107.6 MB/s, 0.097 s |

Loopback test: sing-box SS server and client on `127.0.0.1`, HTTP file transfer through
`a2p` SOCKS. Not a WAN throughput test.
