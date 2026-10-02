#!/usr/bin/env bash
set -euo pipefail

repo="Shooa/ccauth"
install_dir="${CCAUTH_INSTALL_DIR:-/usr/local/bin}"

if [[ -n "${CCAUTH_INSTALL_DIR:-}" ]]; then
  mkdir -p "${install_dir}"
elif [[ ! -w "${install_dir}" ]]; then
  install_dir="${HOME}/.local/bin"
  mkdir -p "${install_dir}"
fi

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"
case "${arch}" in
  x86_64|amd64) arch="amd64" ;;
  aarch64|arm64) arch="arm64" ;;
  *) echo "ccauth: unsupported architecture: ${arch}" >&2; exit 1 ;;
esac

latest="$(curl -fsSL "https://api.github.com/repos/${repo}/releases/latest" |
  sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -1)"
if [[ -z "${latest}" ]]; then
  echo "ccauth: failed to determine latest release" >&2
  exit 1
fi

version="${latest#v}"
archive="ccauth_${version}_${os}_${arch}.tar.gz"
url="https://github.com/${repo}/releases/download/${latest}/${archive}"

echo "Installing ccauth ${version} (${os}/${arch}) -> ${install_dir}"
tmp="$(mktemp -d)"
trap 'rm -rf "${tmp}"' EXIT
curl -fsSL "${url}" | tar -C "${tmp}" -xzf -
install -m 0755 "${tmp}/ccauth" "${install_dir}/ccauth"

echo "Installed: $(command -v ccauth || echo "${install_dir}/ccauth")"
ccauth version 2>/dev/null || "${install_dir}/ccauth" version
