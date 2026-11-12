import os
import sys
from pathlib import Path


def _exec(name):
    exe = name + (".exe" if os.name == "nt" else "")
    path = Path(__file__).resolve().parent / "_bin" / exe
    if not path.is_file():
        raise SystemExit(f"{name}: binary missing; install a platform wheel")
    os.execv(str(path), [str(path), *sys.argv[1:]])


def a2p():
    _exec("a2p")


def a2pd():
    _exec("a2pd")
