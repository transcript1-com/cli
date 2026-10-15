#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
VERSION=9.9.9
PREFIX="cli/releases/$VERSION"
IMMUTABLE='public, max-age=31536000, immutable'
CURRENT='public, max-age=300'
ARCHIVES="transcript1_${VERSION}_darwin_arm64.tar.gz transcript1_${VERSION}_darwin_amd64.tar.gz"
ARCHIVES="$ARCHIVES transcript1_${VERSION}_linux_amd64.tar.gz transcript1_${VERSION}_linux_arm64.tar.gz"
ARCHIVES="$ARCHIVES transcript1_${VERSION}_windows_amd64.zip"

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
    GH_LOG="$TMP/gh-release.log" \
    R2_RELEASE_DRY_RUN=1 \
    RELEASE_SKIP_GIT_CHECKS=1 \
    R2_RELEASE_DIST_DIR="$TMP/dist" \
    TRANSCRIPT1_RELEASE_KEY_FILE="$1" \
    TRANSCRIPT1_RELEASE_PUB_FILE="$TMP/a.pub" \
    bash scripts/release.sh "$VERSION"
  )
}

run_mirror() {
  (
    cd "$ROOT"
    PATH="$TMP/bin:$PATH" GH_LOG="$TMP/gh-mirror.log" GH_EXIT="$1" R2_RELEASE_DIST_DIR="$TMP/mirror-dist" \
      bash scripts/mirror-release-github.sh "$VERSION"
  )
}

classify() {
  (source "$ROOT/scripts/release.sh" "$VERSION" && is_absent "$1" "$2")
}

mkdir -p "$TMP/bin"
printf '#!/bin/sh\necho "$*" >> "$GH_LOG"\nexit "${GH_EXIT:-0}"\n' > "$TMP/bin/gh"
chmod +x "$TMP/bin/gh"

(cd "$ROOT" && go run ./cmd/release-tool keygen --pub "$TMP/a.pub" "$TMP/a.key" > /dev/null)
(cd "$ROOT" && go run ./cmd/release-tool keygen --pub "$TMP/b.pub" "$TMP/b.key" > /dev/null)

run_release "$TMP/a.key" > "$TMP/out.txt" 2>&1 || fail 'release exited non-zero in dry run' "$TMP/out.txt"

sig_line=$(find_line "UPLOAD cli/latest.json.sig " "$TMP/out.txt")
json_line=$(find_line "UPLOAD cli/latest.json " "$TMP/out.txt")
(( sig_line < json_line )) || fail 'latest.json.sig must upload before latest.json' "$TMP/out.txt"

for name in $ARCHIVES checksums.txt; do
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

grep -F -q "MIRROR (dry run) scripts/mirror-release-github.sh $VERSION" "$TMP/out.txt" ||
  fail 'a dry run must print the mirror line' "$TMP/out.txt"
[[ ! -e "$TMP/gh-release.log" ]] || fail 'a dry run must never call gh' "$TMP/gh-release.log"

if run_release "$TMP/b.key" > "$TMP/mismatch.txt" 2>&1; then
  fail 'a mismatched key must fail the release' "$TMP/mismatch.txt"
fi

grep -F -q 'release key does not match' "$TMP/mismatch.txt" || fail 'missing the key mismatch error' "$TMP/mismatch.txt"
if grep -q -E '^(UPLOAD|VERIFY ABSENT) ' "$TMP/mismatch.txt"; then
  fail 'a mismatched key must fail before any upload line' "$TMP/mismatch.txt"
fi

R2_RELEASE_RESUME=1 run_release "$TMP/a.key" > "$TMP/resume.txt" 2>&1 || fail 'resume exited non-zero' "$TMP/resume.txt"
if grep -q -E '^VERIFY ABSENT |^UPLOAD cli/releases/' "$TMP/resume.txt"; then
  fail 'resume must skip the absent checks and the versioned uploads' "$TMP/resume.txt"
fi

resume_sig=$(find_line 'UPLOAD cli/latest.json.sig ' "$TMP/resume.txt")
resume_json=$(find_line 'UPLOAD cli/latest.json ' "$TMP/resume.txt")
(( resume_sig < resume_json )) || fail 'resume must upload latest.json.sig before latest.json' "$TMP/resume.txt"

mkdir -p "$TMP/mirror-dist/$VERSION"
for name in $ARCHIVES checksums.txt; do
  : > "$TMP/mirror-dist/$VERSION/$name"
done

run_mirror 1 > "$TMP/mirror-fail.txt" 2>&1 || fail 'a mirror failure must not fail the script' "$TMP/mirror-fail.txt"
grep -F -q 'warning: GitHub mirror failed' "$TMP/mirror-fail.txt" || fail 'missing the mirror warning' "$TMP/mirror-fail.txt"
grep -F -q "scripts/mirror-release-github.sh $VERSION" "$TMP/mirror-fail.txt" ||
  fail 'missing the mirror retry command' "$TMP/mirror-fail.txt"

rm -f "$TMP/gh-mirror.log"
run_mirror 0 > "$TMP/mirror-ok.txt" 2>&1 || fail 'mirror exited non-zero' "$TMP/mirror-ok.txt"
create=$(grep -F 'release create' "$TMP/gh-mirror.log") || fail 'gh release create was not called' "$TMP/gh-mirror.log"
for want in "v$VERSION" '--repo transcript1/cli' "--notes transcript1 $VERSION" checksums.txt $ARCHIVES; do
  [[ "$create" == *"$want"* ]] || fail "gh release create is missing: $want" "$TMP/gh-mirror.log"
done

is_expected() {
  local want="$1"
  local result="$2"
  case "$want" in
    absent) [[ "$result" -eq 0 ]] ;;
    other) [[ "$result" -ne 0 ]] ;;
    *) return 1 ;;
  esac
}

expect() {
  local want="$1"
  local name="$2"
  local result
  result=0
  classify "$3" "$4" || result=$?
  is_expected "$want" "$result" || fail "$name: expected $want"
}

expect absent 'aws 404 head-object' aws 'An error occurred (404) when calling the HeadObject operation: Not Found'
expect absent 'aws NoSuchKey' aws 'An error occurred (NoSuchKey) when calling the HeadObject operation'
expect other 'aws forbidden' aws 'An error occurred (403) when calling the HeadObject operation: Forbidden'
expect other 'aws missing bucket' aws 'An error occurred (NoSuchBucket) when calling the HeadObject operation'
expect other 'aws bare not found' aws 'Not Found'
expect absent 'wrangler missing key' wrangler 'The specified key does not exist. [code: 10007]'
expect absent 'wrangler 404 object' wrangler 'R2 object not found (404)'
expect other 'wrangler missing token' wrangler 'CLOUDFLARE_API_TOKEN not found'
expect other 'wrangler missing bucket' wrangler 'The specified bucket does not exist. [code: 10006]'

echo 'release script contract passed'
