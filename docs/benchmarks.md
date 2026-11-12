# benchmarks

windows amd64, go 1.25, single run.

| test | result |
|---|---|
| parse 1000 nodes | 81 ms |
| compile 1000 nodes to sing-box | 147 ms |
| sing-box start to endpoint | 714 ms |
| mihomo start to endpoint | 1376 ms |
| xray start to endpoint | 1619 ms |
| sing-box rss | 33.4 mb |
| mihomo rss | 25.0 mb |
| xray rss | 21.7 mb |
| local ss loopback, 10 mib | 107.6 mb/s, 0.097 s |

loopback: sing-box ss server and client on `127.0.0.1`, http file through `a2p` socks.
not a wan throughput test.
