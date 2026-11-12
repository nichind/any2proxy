import json
import os
import shutil
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parent
REPO = ROOT.parent.parent

TARGETS = {
    "win32-x64": ("windows", "amd64", ".exe"),
    "win32-arm64": ("windows", "arm64", ".exe"),
    "linux-x64": ("linux", "amd64", ""),
    "linux-arm64": ("linux", "arm64", ""),
    "darwin-x64": ("darwin", "amd64", ""),
    "darwin-arm64": ("darwin", "arm64", ""),
}


def version():
    data = json.loads((ROOT / "package.json").read_text(encoding="utf-8"))
    return data["version"]


def main():
    release = version()
    for name, (goos, goarch, ext) in TARGETS.items():
        package = ROOT / "npm" / name
        manifest = json.loads((package / "package.json").read_text(encoding="utf-8"))
        if manifest["version"] != release:
            raise SystemExit(f"{name}: version {manifest['version']} != {release}")
        bins = package / "bin"
        shutil.rmtree(bins, ignore_errors=True)
        bins.mkdir(parents=True)
        env = {**os.environ, "GOOS": goos, "GOARCH": goarch, "CGO_ENABLED": "0"}
        ldflags = f"-s -w -X github.com/nichind/any2proxy/internal/version.Version={release}"
        for binary in ("a2p", "a2pd"):
            subprocess.run(
                [
                    "go",
                    "build",
                    "-trimpath",
                    f"-ldflags={ldflags}",
                    "-o",
                    str(bins / (binary + ext)),
                    f"github.com/nichind/any2proxy/cmd/{binary}",
                ],
                cwd=REPO,
                env=env,
                check=True,
            )
        print(f"{name} -> {bins}")


if __name__ == "__main__":
    main()
