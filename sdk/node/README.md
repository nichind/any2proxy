# node sdk

package includes `a2p` and `a2pd` for the current platform.

```js
import { Client } from "any2proxy";

const client = new Client();
console.log(await client.version());
console.log(await client.parse("vless://uuid@example.com:443#demo"));
await client.close();
```

`a2p` and `a2pd` are on `path` after a global install. pass a path to `new Client(...)` to use another binary.
