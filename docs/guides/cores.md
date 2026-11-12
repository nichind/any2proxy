# Cores

```sh
a2p cores install sing-box
a2p cores install sing-box --version v1.14.4
a2p cores update sing-box
a2p cores list
a2p cores verify sing-box
a2p cores import sing-box ./sing-box
a2p cores remove sing-box
```

`install` and `update` download from GitHub Releases, verify sha256 and unpack zip,
tar.gz or gz. `--os` and `--arch` select another target. `--binary` overrides the imported
core at run time.

Custom fork or direct link:

```sh
a2p cores install fork --repo owner/sing-box --bin-name sing-box
a2p cores install custom --url https://example.com/core.zip --sha256 <hex> --bin-name core
a2p run --to sing-box --core fork ./links.txt
```

`--core` selects the installed binary; `--to` selects the config format.
