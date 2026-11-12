# protocols

| protocol | parse | links | sing-box | mihomo | xray |
|---|:---:|:---:|:---:|:---:|:---:|
| vless | yes | yes | yes | yes | yes |
| vmess | yes | yes | yes | yes | yes |
| trojan | yes | yes | yes | yes | yes |
| ss | yes | yes | yes | yes | yes |
| hysteria2 | yes | yes | yes | yes | no |
| tuic | yes | yes | yes | yes | no |
| anytls | yes | yes | yes | yes | no |

xray has no hysteria2, tuic or anytls. those nodes are dropped from xray configs and reported
as unsupported. use sing-box or mihomo.

`select` and `url-test` groups are emitted for mihomo and sing-box. xray has no groups.
`--mode balance` runs one core with the full group set; `--to xray` rejects it.
