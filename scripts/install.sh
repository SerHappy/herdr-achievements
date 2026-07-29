#!/usr/bin/env bash
# Downloads the release binary selected by the manifest version and host platform.
set -euo pipefail

readonly REPOSITORY="SerHappy/herdr-achievements"
readonly DEFAULT_RELEASE_BASE_URL="https://github.com/${REPOSITORY}/releases/download"
CLEANUP_TEMP_DIR=""
CLEANUP_INSTALL_TMP=""

die() {
	echo "herdr-achievements installer: $*" >&2
	exit 1
}

cleanup() {
	[[ -z "$CLEANUP_TEMP_DIR" ]] || rm -rf "$CLEANUP_TEMP_DIR"
	[[ -z "$CLEANUP_INSTALL_TMP" ]] || rm -f "$CLEANUP_INSTALL_TMP"
}

detect_platform() {
	local kernel machine os arch
	kernel="$(uname -s)"
	machine="$(uname -m)"

	case "$kernel" in
	Darwin) os="darwin" ;;
	Linux) os="linux" ;;
	*) die "unsupported operating system: $kernel" ;;
	esac

	case "$machine" in
	arm64 | aarch64) arch="arm64" ;;
	x86_64 | amd64) arch="amd64" ;;
	*) die "unsupported architecture: $machine" ;;
	esac

	printf '%s-%s\n' "$os" "$arch"
}

sha256_file() {
	local target=$1
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$target" | awk '{print $1}'
	elif command -v shasum >/dev/null 2>&1; then
		shasum -a 256 "$target" | awk '{print $1}'
	else
		die "no SHA-256 utility found; install sha256sum or shasum"
	fi
}

main() {
	local script_dir repo_root manifest version platform os arch asset release_base asset_url sums_url
	local temp_dir downloaded_binary downloaded_sums expected actual bin_dir

	script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
	repo_root="$(cd -- "$script_dir/.." && pwd)"
	manifest="$repo_root/herdr-plugin.toml"
	version="$(awk -F '"' '/^version[[:space:]]*=/ { print $2; exit }' "$manifest")"
	[[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || die "invalid or missing version in $manifest"

	platform="$(detect_platform)"
	os="${platform%-*}"
	arch="${platform#*-}"
	asset="herdr-achievements-${os}-${arch}"
	release_base="${HERDR_ACHIEVEMENTS_RELEASE_BASE_URL:-$DEFAULT_RELEASE_BASE_URL}"
	asset_url="${release_base%/}/v${version}/${asset}"
	sums_url="${release_base%/}/v${version}/SHA256SUMS"

	temp_dir="$(mktemp -d "${TMPDIR:-/tmp}/herdr-achievements-install.XXXXXX")"
	CLEANUP_TEMP_DIR="$temp_dir"
	trap cleanup EXIT HUP INT TERM
	downloaded_binary="$temp_dir/$asset"
	downloaded_sums="$temp_dir/SHA256SUMS"

	if ! curl --fail --location --silent --show-error --retry 3 --retry-delay 1 \
		-o "$downloaded_binary" "$asset_url"; then
		die "download failed (version=$version platform=$platform asset_url=$asset_url)"
	fi
	if ! curl --fail --location --silent --show-error --retry 3 --retry-delay 1 \
		-o "$downloaded_sums" "$sums_url"; then
		die "checksum download failed (version=$version platform=$platform asset_url=$asset_url)"
	fi

	expected="$(awk -v asset="$asset" '$2 == asset || $2 == "*" asset { print $1; exit }' "$downloaded_sums")"
	[[ -n "$expected" ]] || die "no checksum found for $asset"
	actual="$(sha256_file "$downloaded_binary")"
	[[ "$actual" == "$expected" ]] || die "SHA-256 mismatch for $asset"

	bin_dir="${HERDR_ACHIEVEMENTS_BIN_DIR:-$repo_root/bin}"
	mkdir -p "$bin_dir"
	CLEANUP_INSTALL_TMP="$(mktemp "$bin_dir/.herdr-achievements.XXXXXX")"
	install -m 0755 "$downloaded_binary" "$CLEANUP_INSTALL_TMP"
	mv -f "$CLEANUP_INSTALL_TMP" "$bin_dir/herdr-achievements"
	CLEANUP_INSTALL_TMP=""
}

main "$@"
