# python sdk

wheel includes `a2p` and `a2pd`.

```python
from any2proxy import Client

client = Client("a2pd")
print(client.version())
print(client.parse("vless://uuid@example.com:443#demo"))
client.close()
```

`a2pd` is installed on `path`, or pass a path as the first argument.
