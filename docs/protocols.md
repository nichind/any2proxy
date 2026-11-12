# Protocols

| Protocol | parse | links | sing-box | mihomo | Xray |
|---|:---:|:---:|:---:|:---:|:---:|
| vless | yes | yes | yes | yes | yes |
| vmess | yes | yes | yes | yes | yes |
| trojan | yes | yes | yes | yes | yes |
| ss | yes | yes | yes | yes | yes |
| hysteria2 | yes | yes | yes | yes | no |
| tuic | yes | yes | yes | yes | no |
| anytls | yes | yes | yes | yes | no |

Xray does not implement hysteria2, tuic or anytls. Those nodes are dropped from Xray
configs and reported as unsupported; use sing-box or mihomo for them.

## Groups

`select` and `url-test` are emitted for mihomo and sing-box from Clash `proxy-groups`.
Xray has no equivalent and drops groups. `--mode balance` starts one core with the full group
set; `--to xray` rejects this mode.
