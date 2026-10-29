#!/bin/sh
# else.sh installer for macOS, Linux, FreeBSD and WSL.
#
#   curl -fsSL https://cristi-boboc.github.io/else.sh/install.sh | sh
#
# Environment:
#   ELSE_SH_VERSION       tag to install (default: latest release)
#   ELSE_SH_INSTALL_DIR   where to put the binary (default: ~/.local/bin)
#   ELSE_SH_NO_HOOK=1     only install the binary, don't touch shell rc files
set -eu

repo="cristi-boboc/else.sh"
dir="${ELSE_SH_INSTALL_DIR:-$HOME/.local/bin}"

say() { printf '\033[1m==>\033[0m %s\n' "$*"; }
die() { printf '\033[31merror:\033[0m %s\n' "$*" >&2; exit 1; }
need() { command -v "$1" >/dev/null 2>&1 || die "$1 is required"; }

case "$(uname -s)" in
  Linux) os=linux ;;
  Darwin) os=darwin ;;
  FreeBSD) os=freebsd ;;
  MINGW* | MSYS* | CYGWIN*) die "on Windows run: irm https://cristi-boboc.github.io/else.sh/install.ps1 | iex" ;;
  *) die "unsupported OS: $(uname -s)" ;;
esac
case "$(uname -m)" in
  x86_64 | amd64) arch=amd64 ;;
  arm64 | aarch64) arch=arm64 ;;
  *) die "unsupported architecture: $(uname -m)" ;;
esac

if command -v curl >/dev/null 2>&1; then
  fetch() { curl -fsSL "$1" -o "$2"; }
  latest() { curl -fsSLI -o /dev/null -w '%{url_effective}' "https://github.com/$repo/releases/latest"; }
else
  need wget
  fetch() { wget -qO "$2" "$1"; }
  latest() { wget -S --spider "https://github.com/$repo/releases/latest" 2>&1 | sed -n 's/^ *Location: //p' | tail -1; }
fi
need tar

version="${ELSE_SH_VERSION:-}"
[ -n "$version" ] || version="$(latest | sed 's#.*/tag/##' | tr -d '\r')"
[ -n "$version" ] || die "could not determine the latest release"

asset="else.sh_${os}_${arch}.tar.gz"
base="https://github.com/$repo/releases/download/$version"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

say "downloading else.sh $version ($os/$arch)"
fetch "$base/$asset" "$tmp/$asset" || die "download failed: $base/$asset"
fetch "$base/checksums.txt" "$tmp/checksums.txt" || die "could not download checksums"

want="$(grep " $asset\$" "$tmp/checksums.txt" | cut -d' ' -f1)"
if command -v sha256sum >/dev/null 2>&1; then got="$(sha256sum "$tmp/$asset" | cut -d' ' -f1)"
else got="$(shasum -a 256 "$tmp/$asset" | cut -d' ' -f1)"; fi
[ -n "$want" ] && [ "$want" = "$got" ] || die "checksum mismatch for $asset"

tar -xzf "$tmp/$asset" -C "$tmp" else.sh
mkdir -p "$dir"
mv "$tmp/else.sh" "$dir/else.sh"
chmod +x "$dir/else.sh"
say "installed $dir/else.sh"

case ":$PATH:" in
  *":$dir:"*) ;;
  *) printf '\n  %s is not on your PATH. Add it, e.g.:\n    export PATH="%s:$PATH"\n\n' "$dir" "$dir" ;;
esac

[ "${ELSE_SH_NO_HOOK:-}" = 1 ] && exit 0

# This script runs in a child shell; the shell the user typed into is our parent.
parent="$(ps -o comm= -p "$PPID" 2>/dev/null | sed 's#.*/##; s/^-//' || true)"
case "$parent" in
  bash | zsh | fish | pwsh) "$dir/else.sh" install "$parent" ;;
  *) "$dir/else.sh" install "$(basename "${SHELL:-sh}")" || "$dir/else.sh" install ;;
esac
