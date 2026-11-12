# go sdk

```go
client, err := any2proxy.Start("a2pd")
if err != nil {
    panic(err)
}
defer client.Close()

version, _ := client.Version(context.Background())
profile, _ := client.Parse(context.Background(), "vless://uuid@example.com:443#demo", "plain")
```

`a2pd` must be on `path`, or pass a path to `start`.
