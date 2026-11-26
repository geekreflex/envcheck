#!/bin/sh
# Install envcheck from a GitHub release, verifying its SHA-256 checksum.
#
#   curl -sSfL https://raw.githubusercontent.com/geekreflex/envcheck/main/install.sh | sh
#
# Environment:
#   ENVCHECK_VERSION      release tag to install, e.g. v0.1.0 (default: latest)
#   ENVCHECK_INSTALL_DIR  where to put the binary (default: /usr/local/bin if
#                         writable, otherwise ~/.local/bin)
#
# Supports Linux and macOS on amd64 and arm64. On Windows, download the zip
# from the releases page or use `go install`.

set -eu

REPO="geekreflex/envcheck"
BINARY="envcheck"

say() { printf '%s\n' "$*" >&2; }
fail() {
	say "envcheck install: $*"
	exit 1
}
need() { command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"; }

need curl
need tar
need uname

case "$(uname -s)" in
Linux) os=linux ;;
Darwin) os=darwin ;;
*) fail "unsupported OS $(uname -s); download a release manually from https://github.com/$REPO/releases" ;;
esac

case "$(uname -m)" in
x86_64 | amd64) arch=amd64 ;;
arm64 | aarch64) arch=arm64 ;;
*) fail "unsupported architecture $(uname -m)" ;;
esac

if command -v sha256sum >/dev/null 2>&1; then
	sha256() { sha256sum "$1" | cut -d' ' -f1; }
elif command -v shasum >/dev/null 2>&1; then
	sha256() { shasum -a 256 "$1" | cut -d' ' -f1; }
else
	fail "need sha256sum or shasum to verify the download"
fi

tag="${ENVCHECK_VERSION:-}"
if [ -z "$tag" ]; then
	# /releases/latest redirects to /releases/tag/<tag>; no API token needed.
	url=$(curl -sSfL -o /dev/null -w '%{url_effective}' "https://github.com/$REPO/releases/latest") ||
		fail "could not reach GitHub to find the latest release"
	tag="${url##*/}"
	case "$tag" in
	v*) ;;
	*) fail "no published release found at https://github.com/$REPO/releases" ;;
	esac
fi
version="${tag#v}"

archive="${BINARY}_${version}_${os}_${arch}.tar.gz"
base="https://github.com/$REPO/releases/download/$tag"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT INT TERM

say "Downloading $BINARY $tag for $os/$arch"
curl -sSfL -o "$tmp/$archive" "$base/$archive" || fail "download failed: $base/$archive"
curl -sSfL -o "$tmp/checksums.txt" "$base/checksums.txt" || fail "download failed: $base/checksums.txt"

want=$(awk -v f="$archive" '$2 == f { print $1 }' "$tmp/checksums.txt")
[ -n "$want" ] || fail "$archive is not listed in checksums.txt"
got=$(sha256 "$tmp/$archive")
[ "$want" = "$got" ] || fail "checksum mismatch for $archive (expected $want, got $got)"

tar -xzf "$tmp/$archive" -C "$tmp" "$BINARY"

dir="${ENVCHECK_INSTALL_DIR:-}"
if [ -z "$dir" ]; then
	if [ -w /usr/local/bin ]; then
		dir=/usr/local/bin
	else
		dir="$HOME/.local/bin"
	fi
fi
mkdir -p "$dir"

# Install via a temp name and rename, so a running envcheck is never left
# half-written.
cp "$tmp/$BINARY" "$dir/.$BINARY.new"
chmod 0755 "$dir/.$BINARY.new"
mv -f "$dir/.$BINARY.new" "$dir/$BINARY"

say "Installed $dir/$BINARY"
case ":$PATH:" in
*":$dir:"*) ;;
*) say "Note: $dir is not on your PATH. Add it, e.g.: export PATH=\"$dir:\$PATH\"" ;;
esac
