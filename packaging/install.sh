#!/bin/sh
set -eu

repo="${A2P_REPO:-nichind/any2proxy}"
version="${A2P_VERSION:-latest}"
os="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$os" in
  linux|darwin) ;;
  *) echo "unsupported OS: $os" >&2; exit 1 ;;
esac

arch="$(uname -m)"
case "$arch" in
  x86_64|amd64) arch="amd64" ;;
  aarch64|arm64) arch="arm64" ;;
  *) echo "unsupported arch: $arch" >&2; exit 1 ;;
esac

if [ "$version" = "latest" ]; then
  base="https://github.com/$repo/releases/latest/download"
else
  base="https://github.com/$repo/releases/download/$version"
fi

archive="a2p_${os}_${arch}.tar.gz"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

curl -fsSL "$base/$archive" -o "$tmp/$archive"
curl -fsSL "$base/checksums.txt" -o "$tmp/checksums.txt"
(cd "$tmp" && grep " $archive$" checksums.txt | sha256sum -c -)

tar -xzf "$tmp/$archive" -C "$tmp"
mkdir -p "$HOME/.local/bin"
install -m 755 "$tmp/a2p" "$HOME/.local/bin/a2p"
install -m 755 "$tmp/a2pd" "$HOME/.local/bin/a2pd"
echo "installed a2p and a2pd to $HOME/.local/bin"
