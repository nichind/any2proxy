# Troubleshooting

`a2p: command not found`
- Unix: add `$(go env GOPATH)/bin` to `PATH`
- Windows: add `%USERPROFILE%\go\bin` to `PATH`

`cannot open` / `cannot parse`
```sh
a2p parse ./links.txt
cat links.txt | a2p parse -
a2p parse https://example.com/sub
```
Check the `warnings` array for line numbers.

`no core binary`
```sh
a2p run --binary ./sing-box ./links.txt
```
Or set `A2P_CORE`.

`core did not start`
```sh
a2p run --dry-run --binary ./sing-box ./links.txt
```
Run the same config with the core directly.

`private subscription blocked`
```sh
a2p parse --allow-private http://127.0.0.1:8080/sub
```
