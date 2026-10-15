#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${1:?usage: scripts/release.sh <version>}"
DIST_DIR="${R2_RELEASE_DIST_DIR:-$ROOT/dist}"
OUT="$DIST_DIR/$VERSION"
BUCKET="${R2_RELEASE_BUCKET:-transcript1-downloads}"
BASE_URL="${R2_RELEASE_BASE_URL:-https://downloads.transcript1.com}"
ENDPOINT="${R2_RELEASE_ENDPOINT:-}"
DRY_RUN="${R2_RELEASE_DRY_RUN:-0}"
UPLOADER="${R2_RELEASE_UPLOADER:-aws}"
CURL_RESOLVE="${R2_RELEASE_CURL_RESOLVE:-}"
RESUME="${R2_RELEASE_RESUME:-0}"
SKIP_GIT_CHECKS="${RELEASE_SKIP_GIT_CHECKS:-0}"
KEY_FILE="${TRANSCRIPT1_RELEASE_KEY_FILE:-}"
PUB_FILE="${TRANSCRIPT1_RELEASE_PUB_FILE:-}"
IMMUTABLE_CACHE='public, max-age=31536000, immutable'
CURRENT_CACHE='public, max-age=300'
RELEASE_PREFIX="cli/releases/$VERSION"
TARGETS='darwin/arm64 darwin/amd64 linux/amd64 linux/arm64 windows/amd64'
LDFLAGS="-s -w -X github.com/transcript1/cli/internal/version.Version=$VERSION"
CHECK_DIR=''

die() {
  echo "$*" >&2
  exit 1
}

cleanup() {
  [[ -z "$CHECK_DIR" ]] || rm -rf "$CHECK_DIR"
}
trap cleanup EXIT

check_pub_file() {
  [[ -z "$PUB_FILE" ]] && return 0
  [[ "$DRY_RUN" == 1 ]] && return 0
  die 'TRANSCRIPT1_RELEASE_PUB_FILE is only for dry runs;' \
    'a real release checks the key against internal/update/release-key.pub'
}

check_settings() {
  [[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || die "invalid version: $VERSION (expected x.y.z)"
  check_pub_file
  [[ -n "$KEY_FILE" ]] || die 'TRANSCRIPT1_RELEASE_KEY_FILE is required (path of the private release key)'
  [[ -f "$KEY_FILE" ]] || die "release key not found: $KEY_FILE"
  command -v go >/dev/null || die 'go is required'
  case "$UPLOADER" in
    aws | wrangler) ;;
    *) die "unsupported R2_RELEASE_UPLOADER: $UPLOADER" ;;
  esac

  [[ "$DRY_RUN" == 1 ]] && return 0
  command -v curl >/dev/null || die 'curl is required'
  if [[ "$UPLOADER" == wrangler ]]; then
    command -v npx >/dev/null || die 'npx is required for Wrangler uploads'
    return 0
  fi

  : "${AWS_ACCESS_KEY_ID:?AWS_ACCESS_KEY_ID is required}"
  : "${AWS_SECRET_ACCESS_KEY:?AWS_SECRET_ACCESS_KEY is required}"
  : "${ENDPOINT:?R2_RELEASE_ENDPOINT is required}"
  command -v aws >/dev/null || die 'aws CLI is required'
}

check_repo() {
  [[ "$SKIP_GIT_CHECKS" == 1 ]] && return 0
  [[ -z "$(git -C "$ROOT" status --porcelain)" ]] || die 'working tree is not clean'
  git -C "$ROOT" rev-parse -q --verify "refs/tags/v$VERSION" >/dev/null && die "tag v$VERSION already exists"
  (cd "$ROOT" && go vet ./... && go test ./...)
}

build_binaries() {
  local target os arch binary
  rm -rf "$OUT"
  for target in $TARGETS; do
    os="${target%/*}"
    arch="${target#*/}"
    binary=transcript1
    [[ "$os" == windows ]] && binary=transcript1.exe
    echo "building $os/$arch"
    (cd "$ROOT" && GOOS="$os" GOARCH="$arch" CGO_ENABLED=0 go build -trimpath -ldflags "$LDFLAGS" \
      -o "$OUT/bin/${os}_${arch}/$binary" ./cmd/transcript1)
  done
}

run_release_tool() {
  (cd "$ROOT" && go run ./cmd/release-tool "$@")
}

package_release() {
  run_release_tool package --version "$VERSION" --dist "$OUT" --root "$ROOT"
}

write_manifest() {
  local args=(--version "$VERSION" --dist "$OUT" --base-url "$BASE_URL" --key "$KEY_FILE")
  [[ -z "$PUB_FILE" ]] || args+=(--pub "$PUB_FILE")
  run_release_tool manifest "${args[@]}"
}

run_curl() {
  if [[ -n "$CURL_RESOLVE" ]]; then
    curl --resolve "$CURL_RESOLVE" "$@"
    return
  fi

  curl "$@"
}

get_content_type() {
  case "$1" in
    *.tar.gz) echo application/gzip ;;
    *.zip) echo application/zip ;;
    *.json) echo application/json ;;
    *) echo 'text/plain; charset=utf-8' ;;
  esac
}

ensure_absent() {
  local key="$1"
  local result
  local code
  if [[ "$DRY_RUN" == 1 ]]; then
    echo "VERIFY ABSENT s3://$BUCKET/$key"
    return
  fi

  set +e
  if [[ "$UPLOADER" == wrangler ]]; then
    result=$(npx --yes wrangler r2 object get "$BUCKET/$key" --remote --pipe 2>&1 >/dev/null)
  else
    result=$(AWS_DEFAULT_REGION=auto aws --endpoint-url "$ENDPOINT" s3api head-object \
      --bucket "$BUCKET" --key "$key" 2>&1)
  fi
  code=$?
  set -e
  [[ "$code" -ne 0 ]] || die "refusing to overwrite immutable release object: s3://$BUCKET/$key;" \
    "if this version's files are already uploaded, re-run with R2_RELEASE_RESUME=1 to publish latest.json"
  is_absent "$UPLOADER" "$result" || die "could not verify immutable release key: $result"
}

is_absent() {
  local uploader="$1"
  local result="$2"
  case "$result" in
    *NoSuchKey*) return 0 ;;
  esac

  if [[ "$uploader" == wrangler ]]; then
    case "$result" in
      *'The specified key does not exist'*) return 0 ;;
      *'(404)'*[Oo]bject* | *[Oo]bject*'(404)'*) return 0 ;;
    esac

    return 1
  fi

  case "$result" in
    *'(404)'*HeadObject*) return 0 ;;
  esac

  return 1
}

upload() {
  local file="$1"
  local key="$2"
  local type="$3"
  local cache="$4"
  if [[ "$DRY_RUN" == 1 ]]; then
    echo "UPLOAD $key $type $cache"
    return
  fi

  echo "uploading $key"
  if [[ "$UPLOADER" == wrangler ]]; then
    npx --yes wrangler r2 object put "$BUCKET/$key" --remote --file "$file" \
      --content-type "$type" --cache-control "$cache" --force
    return
  fi

  AWS_DEFAULT_REGION=auto aws --endpoint-url "$ENDPOINT" s3 cp "$file" "s3://$BUCKET/$key" \
    --content-type "$type" --cache-control "$cache" --no-progress
}

verify_length() {
  local file="$1"
  local url="$2"
  local expected
  local headers
  local actual
  expected=$(wc -c < "$file" | tr -d ' ')
  headers=$(run_curl --fail --silent --show-error --location --head "$url") || die "could not read $url"
  actual=$(echo "$headers" | tr -d '\r' | awk 'tolower($1) == "content-length:" { value = $2 } END { print value }')
  [[ "$actual" == "$expected" ]] || die "content length mismatch for $url: expected $expected, got ${actual:-missing}"
}

upload_versioned() {
  local file
  local key
  if [[ "$RESUME" == 1 ]]; then
    echo 'RESUME skipping the versioned uploads'
    return 0
  fi

  for file in "$@"; do
    key="$RELEASE_PREFIX/$(basename "$file")"
    ensure_absent "$key"
    upload "$file" "$key" "$(get_content_type "$file")" "$IMMUTABLE_CACHE"
  done
}

publish_versioned() {
  local file
  local files=("$OUT"/transcript1_"$VERSION"_*.tar.gz "$OUT"/transcript1_"$VERSION"_*.zip "$OUT/checksums.txt")
  upload_versioned "${files[@]}"
  [[ "$DRY_RUN" == 1 ]] && return 0
  for file in "${files[@]}"; do
    verify_length "$file" "$BASE_URL/$RELEASE_PREFIX/$(basename "$file")"
  done
}

# latest.json is the commit point, so it goes up last
publish_current() {
  upload "$OUT/latest.json.sig" cli/latest.json.sig 'text/plain; charset=utf-8' "$CURRENT_CACHE"
  upload "$OUT/latest.json" cli/latest.json application/json "$CURRENT_CACHE"
}

verify_public() {
  local url
  local urls
  [[ "$DRY_RUN" == 1 ]] && return 0
  CHECK_DIR=$(mktemp -d)
  run_curl --fail --silent --show-error --location -o "$CHECK_DIR/latest.json" "$BASE_URL/cli/latest.json?v=$VERSION"
  run_curl --fail --silent --show-error --location -o "$CHECK_DIR/latest.json.sig" "$BASE_URL/cli/latest.json.sig?v=$VERSION"
  cmp -s "$CHECK_DIR/latest.json" "$OUT/latest.json" || die 'public latest.json differs from the local file'
  cmp -s "$CHECK_DIR/latest.json.sig" "$OUT/latest.json.sig" || die 'public latest.json.sig differs from the local file'
  grep -q "\"version\": \"$VERSION\"" "$CHECK_DIR/latest.json" || die "public latest.json is not version $VERSION"
  urls=$(sed -n 's/.*"url": *"\([^"]*\)".*/\1/p' "$OUT/latest.json")
  [[ -n "$urls" ]] || die 'latest.json has no asset URLs'
  for url in $urls; do
    run_curl --fail --silent --show-error --location --head "$url" >/dev/null
  done
}

tag_release() {
  [[ "$DRY_RUN" == 1 ]] && return 0
  git -C "$ROOT" tag "v$VERSION"
}

mirror_github() {
  if [[ "$DRY_RUN" == 1 ]]; then
    echo "MIRROR (dry run) scripts/mirror-release-github.sh $VERSION"
    return 0
  fi

  bash "$ROOT/scripts/mirror-release-github.sh" "$VERSION"
}

list_release_names() {
  local target os arch ext
  for target in $TARGETS; do
    os="${target%/*}"
    arch="${target#*/}"
    ext=tar.gz
    [[ "$os" == windows ]] && ext=zip
    echo "transcript1_${VERSION}_${os}_${arch}.$ext"
  done

  echo checksums.txt
}

check_unpublished() {
  local name
  [[ "$DRY_RUN" == 1 ]] && return 0
  for name in $(list_release_names); do
    ensure_absent "$RELEASE_PREFIX/$name"
  done
}

build_or_reuse() {
  if [[ "$RESUME" != 1 ]]; then
    check_unpublished
    build_binaries
    package_release
    return 0
  fi

  [[ -f "$OUT/checksums.txt" ]] || die "nothing to resume: $OUT/checksums.txt is missing"
}

main() {
  check_settings
  check_repo
  build_or_reuse
  write_manifest
  publish_versioned
  publish_current
  verify_public
  tag_release
  mirror_github
  if [[ "$DRY_RUN" == 1 ]]; then
    echo "Dry run: built and signed transcript1 $VERSION, uploaded nothing"
    return 0
  fi

  echo "Published transcript1 $VERSION to $BASE_URL/cli/"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  main
fi
