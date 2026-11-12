# Node SDK

```js
import { Client } from "any2proxy";

const client = new Client("a2pd");
console.log(await client.version());
console.log(await client.parse("vless://uuid@example.com:443#demo"));
await client.close();
```

`a2pd` must be on `PATH`, or pass a path as the first argument.
