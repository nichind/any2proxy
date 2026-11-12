import json
import subprocess
import threading


class Client:
    def __init__(self, binary="a2pd", args=None):
        self._process = subprocess.Popen(
            [binary] + list(args or []),
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.DEVNULL,
            text=True,
            bufsize=1,
        )
        self._lock = threading.Lock()
        self._next = 0

    def call(self, method, params=None):
        with self._lock:
            self._next += 1
            request = {"id": self._next, "method": method, "params": params or {}}
            assert self._process.stdin is not None
            assert self._process.stdout is not None
            self._process.stdin.write(json.dumps(request) + "\n")
            self._process.stdin.flush()
            line = self._process.stdout.readline()
            if not line:
                raise RuntimeError("a2pd closed the stream")
            response = json.loads(line)
            error = response.get("error")
            if error:
                raise RuntimeError(f"{error['code']}: {error['message']}")
            return response.get("result")

    def version(self):
        return self.call("version")["version"]

    def parse(self, body, format="plain"):
        return self.call("parse", {"body": body, "format": format})

    def compile(self, body, format="plain", target="sing-box", port=1080, strict=False):
        return self.call(
            "compile",
            {"body": body, "format": format, "target": target, "port": port, "strict": strict},
        )

    def run(self, body, format="plain", target="sing-box", binary="sing-box", port=0, strict=False):
        return self.call(
            "run",
            {
                "body": body,
                "format": format,
                "target": target,
                "binary": binary,
                "port": port,
                "strict": strict,
            },
        )

    def stop(self, session_id):
        return self.call("stop", {"session_id": session_id})

    def close(self):
        if self._process.stdin:
            self._process.stdin.close()
        if self._process.stdout:
            self._process.stdout.close()
        self._process.wait(timeout=5)
