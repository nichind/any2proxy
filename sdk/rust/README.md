# rust sdk

```rust
use any2proxy::Client;

let mut client = Client::start("a2pd", &[]).unwrap();
println!("{}", client.version().unwrap());
let profile = client.parse("vless://uuid@example.com:443#demo", "plain").unwrap();
client.close().unwrap();
```

`a2pd` must be on `path`, or pass a path as the first argument.
