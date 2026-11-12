import { Client } from "./index.js";

const client = new Client("go", ["run", "../../cmd/a2pd"]);
try {
  const version = await client.version();
  if (!version) throw new Error("empty version");

  const profile = await client.parse("vless://u@example.com:443#one");
  if (profile.nodes[0].tag !== "one") throw new Error("unexpected parse result");

  const compiled = await client.compile("vless://u@example.com:443#one", "plain", "mihomo", 1080);
  if (!compiled.config.includes("socks-port")) throw new Error("unexpected compile result");
} finally {
  await client.close();
}
