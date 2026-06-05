#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  bash scripts/update_install_cli_version.sh v0.13.0

Updates content/getting_started/install_cli.md download URLs for the given release.
EOF
}

if [[ $# -ne 1 ]]; then
  usage >&2
  exit 2
fi

version="$1"

if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z]+)*$ ]]; then
  echo "version must look like v0.13.0" >&2
  exit 2
fi

release="${version#v}"
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"
file="${INSTALL_CLI_MD:-$repo_root/content/getting_started/install_cli.md}"

if [[ ! -f "$file" ]]; then
  echo "file not found: $file" >&2
  exit 1
fi

tmp="$(mktemp "${TMPDIR:-/tmp}/install_cli.XXXXXX")"
trap 'rm -f "$tmp"' EXIT

sed -E \
  -e "s#releases/download/v[0-9][0-9A-Za-z.+-]*#releases/download/${version}#g" \
  -e "s#pdfcpu_[0-9][0-9A-Za-z.+-]*_#pdfcpu_${release}_#g" \
  "$file" > "$tmp"

if cmp -s "$file" "$tmp"; then
  echo "$file already uses $version"
  exit 0
fi

mv "$tmp" "$file"
trap - EXIT

echo "updated $file to $version"
