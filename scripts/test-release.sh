#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
VERSION=9.9.9
PREFIX="cli/releases/$VERSION"
IMMUTABLE='public, max-age=31536000, immutable'
CURRENT='public, max-age=300'

fail() {
  echo "FAIL: $1" >&2
  if [[ -f "${2:-}" ]]; then
    sed 's/^/  | /' "$2" >&2
  fi

  exit 1
}

find_line() {
  local n
  n=$(grep -n -F -m 1 -- "$1" "$2" | cut -d: -f1) || true
  [[ -n "$n" ]] || fail "missing output: $1" "$2"
  echo "$n"
}

run_release() {
  (
    cd "$ROOT"
    PATH="$TMP/bin:$PATH" \
    R2_RELEASE_DRY_RUN=1 \
    RELEASE_SKIP_GIT_CHECKS=1 \
    R2_RELEASE_DIST_DIR="$TMP/dist" \
    TRANSCRIPT1_RELEASE_KEY_FILE="$1" \
    TRANSCRIPT1_RELEASE_PUB_FILE="$TMP/a.pub" \
    bash scripts/release.sh "$VERSION"
  )
}

mkdir -p "$TMP/bin"
printf '#!/bin/sh\nexit 1\n' > "$TMP/bin/gh"
chmod +x "$TMP/bin/gh"

(cd "$ROOT" && go run ./cmd/release-tool keygen --pub "$TMP/a.pub" "$TMP/a.key" > /dev/null)
(cd "$ROOT" && go run ./cmd/release-tool keygen --pub "$TMP/b.pub" "$TMP/b.key" > /dev/null)

run_release "$TMP/a.key" > "$TMP/out.txt" 2>&1 || fail 'release exited non-zero in dry run' "$TMP/out.txt"

sig_line=$(find_line "UPLOAD cli/latest.json.sig " "$TMP/out.txt")
json_line=$(find_line "UPLOAD cli/latest.json " "$TMP/out.txt")
(( sig_line < json_line )) || fail 'latest.json.sig must upload before latest.json' "$TMP/out.txt"

for name in \
  "transcript1_${VERSION}_darwin_arm64.tar.gz" \
  "transcript1_${VERSION}_darwin_amd64.tar.gz" \
  "transcript1_${VERSION}_linux_amd64.tar.gz" \
  "transcript1_${VERSION}_linux_arm64.tar.gz" \
  "transcript1_${VERSION}_windows_amd64.zip" \
  checksums.txt
do
  absent=$(find_line "VERIFY ABSENT s3://transcript1-downloads/$PREFIX/$name" "$TMP/out.txt")
  upload=$(find_line "UPLOAD $PREFIX/$name " "$TMP/out.txt")
  (( absent < upload )) || fail "$name must be checked absent before it uploads" "$TMP/out.txt"
  (( upload < sig_line )) || fail "$name must upload before latest.json.sig" "$TMP/out.txt"
  grep -F -- "UPLOAD $PREFIX/$name " "$TMP/out.txt" | grep -F -q "$IMMUTABLE" ||
    fail "$name must use the immutable cache header" "$TMP/out.txt"
done

grep -F "UPLOAD cli/latest.json.sig " "$TMP/out.txt" | grep -F -q "$CURRENT" ||
  fail 'latest.json.sig must use the 5 minute cache header' "$TMP/out.txt"
grep -F "UPLOAD cli/latest.json " "$TMP/out.txt" | grep -F -q "$CURRENT" ||
  fail 'latest.json must use the 5 minute cache header' "$TMP/out.txt"

grep -F -q 'warning: GitHub mirror failed' "$TMP/out.txt" || fail 'missing the mirror warning' "$TMP/out.txt"
grep -F -q 'gh release create v9.9.9 --repo transcript1/cli' "$TMP/out.txt" ||
  fail 'missing the mirror retry command' "$TMP/out.txt"

if run_release "$TMP/b.key" > "$TMP/mismatch.txt" 2>&1; then
  fail 'a mismatched key must fail the release' "$TMP/mismatch.txt"
fi

grep -F -q 'release key does not match' "$TMP/mismatch.txt" || fail 'missing the key mismatch error' "$TMP/mismatch.txt"
if grep -q -E '^(UPLOAD|VERIFY ABSENT) ' "$TMP/mismatch.txt"; then
  fail 'a mismatched key must fail before any upload line' "$TMP/mismatch.txt"
fi

echo 'release script contract passed'
