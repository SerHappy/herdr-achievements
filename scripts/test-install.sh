#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
installer="$root_dir/scripts/install.sh"
test_dir="$(mktemp -d "${TMPDIR:-/tmp}/herdr-achievements-install-test.XXXXXX")"
trap 'rm -rf "$test_dir"' EXIT HUP INT TERM

version="$(awk -F '"' '/^version[[:space:]]*=/ { print $2; exit }' "$root_dir/herdr-plugin.toml")"
[[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]
release_dir="$test_dir/releases/v$version"
fake_bin="$test_dir/fake-bin"
mkdir -p "$release_dir" "$fake_bin"

printf '%s\n' '#!/usr/bin/env bash' 'shasum -a 256 "$1"' > "$fake_bin/sha256sum"
chmod 0755 "$fake_bin/sha256sum"

for asset in \
	herdr-achievements-darwin-arm64 \
	herdr-achievements-darwin-amd64 \
	herdr-achievements-linux-arm64 \
	herdr-achievements-linux-amd64; do
	printf 'fixture for %s\n' "$asset" > "$release_dir/$asset"
done
(cd "$release_dir" && shasum -a 256 herdr-achievements-* > SHA256SUMS)

run_install() {
	local kernel=$1 machine=$2 destination=$3
	TEST_UNAME_S="$kernel" TEST_UNAME_M="$machine" \
	PATH="$fake_bin:$PATH" \
	HERDR_ACHIEVEMENTS_RELEASE_BASE_URL="file://$test_dir/releases" \
	HERDR_ACHIEVEMENTS_BIN_DIR="$destination" \
	"$installer"
}

# Make uname return its requested value for both -s and -m.
printf '%s\n' '#!/usr/bin/env bash' 'case "$1" in -s) printf "%s\\n" "${TEST_UNAME_S:?}" ;; -m) printf "%s\\n" "${TEST_UNAME_M:?}" ;; esac' > "$fake_bin/uname"
chmod 0755 "$fake_bin/uname"

assert_mapping() {
	local kernel=$1 machine=$2 expected=$3 destination
	destination="$test_dir/$kernel-$machine/bin"
	run_install "$kernel" "$machine" "$destination"
	cmp "$release_dir/$expected" "$destination/herdr-achievements"
}

assert_mapping Darwin arm64 herdr-achievements-darwin-arm64
assert_mapping Darwin x86_64 herdr-achievements-darwin-amd64
assert_mapping Linux aarch64 herdr-achievements-linux-arm64
assert_mapping Linux x86_64 herdr-achievements-linux-amd64

if run_install FreeBSD x86_64 "$test_dir/unsupported-os/bin" 2>&1; then
	echo "expected unsupported OS to fail" >&2
	exit 1
fi
if run_install Linux riscv64 "$test_dir/unsupported-arch/bin" 2>&1; then
	echo "expected unsupported architecture to fail" >&2
	exit 1
fi

printf '%s\n' '0000000000000000000000000000000000000000000000000000000000000000  herdr-achievements-linux-amd64' > "$release_dir/SHA256SUMS"
if run_install Linux x86_64 "$test_dir/bad-checksum/bin" 2>&1; then
	echo "expected checksum mismatch to fail" >&2
	exit 1
fi

echo "install.sh tests passed"
