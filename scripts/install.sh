#!/bin/sh
# Install icons8-mcp for any MCP host.
#
#   curl -fsSL https://raw.githubusercontent.com/torkay/better-icons8-mcp/main/scripts/install.sh | sh
#
# Downloads the release binary for this platform, checks it against the
# published sha256, and puts it on PATH. Claude Code users do not need this;
# the plugin carries its own launcher.
#
# Override with ICONS8_MCP_VERSION (a tag) or ICONS8_INSTALL_DIR.
set -eu

REPO="torkay/better-icons8-mcp"
dir="${ICONS8_INSTALL_DIR:-$HOME/.local/bin}"

case "$(uname -s)" in
    Linux)   os=linux ;;
    Darwin)  os=darwin ;;
    MINGW*|MSYS*|CYGWIN*) os=windows ;;
    *) echo "icons8-mcp: unsupported OS $(uname -s)" >&2; exit 1 ;;
esac

case "$(uname -m)" in
    x86_64|amd64)  arch=amd64 ;;
    arm64|aarch64) arch=arm64 ;;
    *) echo "icons8-mcp: unsupported architecture $(uname -m)" >&2; exit 1 ;;
esac

if command -v curl >/dev/null 2>&1; then
    get() { curl -fsSL "$1" -o "$2"; }
    resolve() { curl -fsSLI -o /dev/null -w '%{url_effective}' "$1"; }
elif command -v wget >/dev/null 2>&1; then
    get() { wget -qO "$2" "$1"; }
    resolve() { wget -qS --spider "$1" 2>&1 | sed -n 's/^ *Location: *//p' | tail -1; }
else
    echo "icons8-mcp: need curl or wget" >&2
    exit 1
fi

# Read the version off the /releases/latest redirect rather than the API, which
# rate-limits unauthenticated callers.
version="${ICONS8_MCP_VERSION:-}"
if [ -z "$version" ]; then
    version="$(resolve "https://github.com/$REPO/releases/latest" | sed 's#.*/##')"
fi
case "$version" in
    v*) ;;
    *) echo "icons8-mcp: could not work out the latest version; set ICONS8_MCP_VERSION" >&2; exit 1 ;;
esac

name="icons8-mcp_${version}_${os}_${arch}"
base="https://github.com/$REPO/releases/download/$version"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

echo "icons8-mcp: fetching $name" >&2
get "$base/$name.tar.gz" "$tmp/pkg.tar.gz"
get "$base/checksums.txt" "$tmp/checksums.txt"

want=$(awk -v f="$name.tar.gz" '$2 == f || $2 == "*"f {print $1}' "$tmp/checksums.txt")
if command -v sha256sum >/dev/null 2>&1; then
    got=$(sha256sum "$tmp/pkg.tar.gz" | cut -d' ' -f1)
elif command -v shasum >/dev/null 2>&1; then
    got=$(shasum -a 256 "$tmp/pkg.tar.gz" | cut -d' ' -f1)
else
    got=""
fi
if [ -n "$want" ] && [ -n "$got" ] && [ "$want" != "$got" ]; then
    echo "icons8-mcp: checksum mismatch for $name.tar.gz" >&2
    echo "  expected $want" >&2
    echo "  got      $got" >&2
    exit 1
fi

tar -xzf "$tmp/pkg.tar.gz" -C "$tmp"
src="$tmp/$name/icons8-mcp"
[ "$os" = windows ] && src="$src.exe"
mkdir -p "$dir"
mv "$src" "$dir/icons8-mcp"
chmod +x "$dir/icons8-mcp"

echo "icons8-mcp: installed $version to $dir/icons8-mcp" >&2
case ":$PATH:" in
    *":$dir:"*) echo "icons8-mcp: run \`icons8-mcp auth\` to sign in" >&2 ;;
    *) echo "icons8-mcp: $dir is not on PATH; add it, then run \`icons8-mcp auth\`" >&2 ;;
esac
