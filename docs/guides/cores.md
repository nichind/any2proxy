# cores

```sh
a2p cores install sing-box
a2p cores install sing-box --version v1.14.4
a2p cores update sing-box
a2p cores update --all
a2p cores versions sing-box
a2p cores list
a2p cores verify sing-box
a2p cores import sing-box ./sing-box
a2p cores remove sing-box
```

`install` and `update` pull from github releases, check sha256 and unpack zip, tar.gz or gz.
`--os` and `--arch` pick another target.

forks and direct links:

```sh
a2p cores install fork --repo owner/sing-box --bin-name sing-box
a2p cores install custom --url https://example.com/core.zip --sha256 <hex> --bin-name core
a2p run --to sing-box --core fork ./links.txt
```

`--core` picks the installed binary. `--to` picks the config format.
