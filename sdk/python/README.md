# python sdk

```python
from any2proxy import Client

client = Client("a2pd")
print(client.version())
print(client.parse("vless://uuid@example.com:443#demo"))
client.close()
```

`a2pd` must be on `path`, or pass a path as the first argument.
