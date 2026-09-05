#!/bin/sh
set -eu

repo="${SQUIRREL_INSTALL_REPO:-janiorvalle/squirrel}"
install_dir="${SQUIRREL_INSTALL_DIR:-$HOME/.local/bin}"
base_url="${SQUIRREL_INSTALL_BASE_URL:-}"
version="${SQUIRREL_INSTALL_VERSION:-}"
archive_name="${SQUIRREL_INSTALL_ARCHIVE:-}"

fail() {
  printf 'squirrel installer: %s\n' "$*" >&2
  exit 1
}

command -v curl >/dev/null 2>&1 || fail "curl is required"

case "$(uname -s)" in
  Darwin) os=darwin ;;
  Linux) os=linux ;;
  *) fail "unsupported operating system; on Windows run the PowerShell installer: irm https://raw.githubusercontent.com/janiorvalle/squirrel/main/install.ps1 | iex" ;;
esac
case "$(uname -m)" in
  x86_64|amd64) arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  *) fail "unsupported architecture: $(uname -m)" ;;
esac

if [ -z "$version" ]; then
  release_json=$(curl -fsSL "https://api.github.com/repos/$repo/releases/latest") || fail "no published release found for $repo; see https://github.com/$repo/releases or run go install github.com/$repo/cmd/squirrel@latest"
  version=$(printf '%s\n' "$release_json" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"v\{0,1\}\([^"]*\)".*/\1/p' | head -n 1)
  [ -n "$version" ] || fail "latest release did not include a tag_name"
fi
version=${version#v}

if [ -z "$archive_name" ]; then
  archive_name="squirrel_${version}_${os}_${arch}.tar.gz"
fi
if [ -z "$base_url" ]; then
  base_url="https://github.com/$repo/releases/download/v$version"
fi
base_url=${base_url%/}

tmp_dir=$(mktemp -d 2>/dev/null || mktemp -d -t squirrel-install)
stage_squirrel=""
dest_squirrel="$install_dir/squirrel"
backup_squirrel="$tmp_dir/previous-squirrel"
had_squirrel=false
transaction_active=false

restore_install() {
  transaction_active=false
  if [ "$had_squirrel" = true ]; then
    mv -f "$backup_squirrel" "$dest_squirrel"
  else
    rm -f "$dest_squirrel"
  fi
}
cleanup() {
  [ "$transaction_active" = false ] || restore_install
  rm -rf "$tmp_dir"
  [ -z "$stage_squirrel" ] || rm -f "$stage_squirrel"
}
trap cleanup EXIT HUP INT TERM

archive="$tmp_dir/$archive_name"
checksums="$tmp_dir/checksums.txt"
printf 'Downloading squirrel %s for %s/%s...\n' "$version" "$os" "$arch"
curl -fsSL "$base_url/$archive_name" -o "$archive" || fail "could not download $archive_name"
curl -fsSL "$base_url/checksums.txt" -o "$checksums" || fail "could not download checksums.txt"

expected=$(awk -v file="$archive_name" '$2 == file || $2 == "*" file { print $1; exit }' "$checksums")
[ -n "$expected" ] || fail "checksums.txt has no entry for $archive_name"
if command -v sha256sum >/dev/null 2>&1; then
  actual=$(sha256sum "$archive" | awk '{print $1}')
elif command -v shasum >/dev/null 2>&1; then
  actual=$(shasum -a 256 "$archive" | awk '{print $1}')
else
  fail "sha256sum or shasum is required to verify the download"
fi
[ "$actual" = "$expected" ] || fail "checksum mismatch for $archive_name"

tar -xzf "$archive" -C "$tmp_dir"
[ -x "$tmp_dir/squirrel" ] || fail "archive did not contain squirrel"
"$tmp_dir/squirrel" --version >/dev/null || fail "release squirrel failed its version smoke test"

mkdir -p "$install_dir"
stage_squirrel="$install_dir/.squirrel.new.$$"
install -m 0755 "$tmp_dir/squirrel" "$stage_squirrel"
if [ -e "$dest_squirrel" ] || [ -L "$dest_squirrel" ]; then
  cp -p "$dest_squirrel" "$backup_squirrel"
  had_squirrel=true
fi
transaction_active=true
mv -f "$stage_squirrel" "$dest_squirrel" || fail "could not replace squirrel"
stage_squirrel=""

"$dest_squirrel" --version >/dev/null || fail "installed squirrel failed its version smoke test"
transaction_active=false
printf 'Installed squirrel to %s\n' "$install_dir"
case ":$PATH:" in
  *":$install_dir:"*) ;;
  *) printf 'Add %s to PATH to run squirrel from any directory: put this line in your shell profile, ~/.zshrc or ~/.bashrc (fish: fish_add_path %s), then open a new terminal.\n  export PATH="%s:$PATH"\n' "$install_dir" "$install_dir" "$install_dir" ;;
esac

# Piped through sh, stdin is the script itself. The terminal is still there as
# /dev/tty, so setup can ask its questions. Without one, setup prints the plan
# and the flags and changes nothing.
if ( : </dev/tty ) 2>/dev/null; then
  "$dest_squirrel" setup </dev/tty
else
  "$dest_squirrel" setup
fi
