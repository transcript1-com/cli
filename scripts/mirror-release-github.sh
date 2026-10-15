#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION="${1:?usage: scripts/mirror-release-github.sh <version>}"
OUT="${R2_RELEASE_DIST_DIR:-$ROOT/dist}/$VERSION"
REPO="${GITHUB_MIRROR_REPO:-transcript1/cli}"

create_release() {
  gh auth status >/dev/null 2>&1 || return 1
  gh release create "v$VERSION" --verify-tag --repo "$REPO" --notes "transcript1 $VERSION" "$@"
}

if ! command -v gh >/dev/null; then
  echo 'gh not found; skipping the GitHub mirror'
  exit 0
fi

assets=("$OUT"/transcript1_"$VERSION"_*.tar.gz "$OUT"/transcript1_"$VERSION"_*.zip "$OUT/checksums.txt")
for file in "${assets[@]}"; do
  if [[ ! -f "$file" ]]; then
    echo "missing release file: $file" >&2
    exit 1
  fi
done

if create_release "${assets[@]}"; then
  echo "Mirrored transcript1 $VERSION to GitHub"
  exit 0
fi

echo 'warning: GitHub mirror failed; the R2 release is not affected. Retry with:' >&2
echo "  scripts/mirror-release-github.sh $VERSION" >&2
