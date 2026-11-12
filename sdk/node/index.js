import { spawn } from "node:child_process";
import { createInterface } from "node:readline";
import { resolveBinary } from "./bin/locate.js";

export class Client {
  constructor(binary = resolveBinary("a2pd"), args = []) {
    this.process = spawn(binary, args, { stdio: ["pipe", "pipe", "inherit"] });
    this.next = 0;
    this.pending = new Map();

    this.reader = createInterface({ input: this.process.stdout });
    this.reader.on("line", (line) => {
      const message = JSON.parse(line);
      const pending = this.pending.get(message.id);
      if (!pending) return;
      this.pending.delete(message.id);
      if (message.error) {
        pending.reject(new Error(`${message.error.code}: ${message.error.message}`));
      } else {
        pending.resolve(message.result);
      }
    });
    this.process.on("exit", () => {
      for (const pending of this.pending.values()) {
        pending.reject(new Error("a2pd exited"));
      }
      this.pending.clear();
    });
  }

  call(method, params = {}) {
    return new Promise((resolve, reject) => {
      const id = ++this.next;
      this.pending.set(id, { resolve, reject });
      this.process.stdin.write(JSON.stringify({ id, method, params }) + "\n");
    });
  }

  async version() {
    const result = await this.call("version");
    return result.version;
  }

  parse(body, format = "plain") {
    return this.call("parse", { body, format });
  }

  compile(body, format = "plain", target = "sing-box", port = 1080, strict = false) {
    return this.call("compile", { body, format, target, port, strict });
  }

  run(body, format = "plain", target = "sing-box", binary = "sing-box", port = 0, strict = false) {
    return this.call("run", { body, format, target, binary, port, strict });
  }

  stop(sessionId) {
    return this.call("stop", { session_id: sessionId });
  }

  close() {
    if (this.process.exitCode !== null) return Promise.resolve();
    return new Promise((resolve) => {
      this.process.once("exit", resolve);
      this.process.stdin.end();
    });
  }
}
