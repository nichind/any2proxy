import os
import re
import shutil
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent
REPO = ROOT.parent.parent
BIN = ROOT / "any2proxy" / "_bin"
DIST = ROOT / "dist"

TARGETS = [
    ("windows", "amd64", "win_amd64"),
    ("windows", "arm64", "win_arm64"),
    ("linux", "amd64", "manylinux_2_17_x86_64"),
    ("linux", "arm64", "manylinux_2_17_aarch64"),
    ("linux", "amd64", "musllinux_1_2_x86_64"),
    ("linux", "arm64", "musllinux_1_2_aarch64"),
    ("darwin", "amd64", "macosx_11_0_x86_64"),
    ("darwin", "arm64", "macosx_11_0_arm64"),
]


def version():
    text = (ROOT / "pyproject.toml").read_text(encoding="utf-8")
    match = re.search(r'^version = "(.+)"$', text, re.MULTILINE)
    if not match:
        raise SystemExit("version not found in pyproject.toml")
    return match.group(1)


def build(goos, goarch, release):
    shutil.rmtree(BIN, ignore_errors=True)
    BIN.mkdir(parents=True)
    ext = ".exe" if goos == "windows" else ""
    env = {**os.environ, "GOOS": goos, "GOARCH": goarch, "CGO_ENABLED": "0"}
    ldflags = f"-s -w -X github.com/nichind/any2proxy/internal/version.Version={release}"
    for name in ("a2p", "a2pd"):
        subprocess.run(
            [
                "go",
                "build",
                "-trimpath",
                f"-ldflags={ldflags}",
                "-o",
                str(BIN / (name + ext)),
                f"github.com/nichind/any2proxy/cmd/{name}",
            ],
            cwd=REPO,
            env=env,
            check=True,
        )


def main():
    if shutil.which("go") is None:
        raise SystemExit("go not found")
    release = version()
    shutil.rmtree(DIST, ignore_errors=True)
    for goos, goarch, tag in TARGETS:
        print(f"==> {goos}/{goarch} -> {tag}", file=sys.stderr)
        build(goos, goarch, release)
        shutil.rmtree(ROOT / "build", ignore_errors=True)
        for egg in ROOT.glob("*.egg-info"):
            shutil.rmtree(egg, ignore_errors=True)
        subprocess.run(
            [sys.executable, "-m", "build", "--wheel", "--no-isolation"],
            cwd=ROOT,
            check=True,
        )
        wheel = max(DIST.glob("*.whl"), key=lambda path: path.stat().st_mtime)
        subprocess.run(
            [
                sys.executable,
                "-m",
                "wheel",
                "tags",
                "--python-tag",
                "py3",
                "--abi",
                "none",
                "--platform-tag",
                tag,
                "--remove",
                str(wheel),
            ],
            check=True,
        )
    shutil.rmtree(BIN, ignore_errors=True)


if __name__ == "__main__":
    main()
