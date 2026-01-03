#!/usr/bin/env bash
set -euo pipefail

: "${REPO_NAME:?set REPO_NAME}"
: "${MINIO_ALIAS:?set MINIO_ALIAS (mc alias name)}"
: "${MINIO_BUCKET:?set MINIO_BUCKET}"
: "${REMOTE_PREFIX:?set REMOTE_PREFIX}"

TARBALL_NAME="${TARBALL_NAME:-repo.tgz}"
DEBUG="${DEBUG:-0}"
DRY_RUN="${DRY_RUN:-0}"

# --- alias helpers ---
alias_exists() {
  # prints 0 if exists, 1 otherwise
  mc alias list 2>/dev/null | awk '{print $1}' | grep -qx "${MINIO_ALIAS}"
}

ensure_alias() {
  if alias_exists; then
    [ "$DEBUG" = "1" ] && echo "DEBUG=1 -> mc alias '${MINIO_ALIAS}' exists"
    return 0
  fi

  echo "mc alias '${MINIO_ALIAS}' not found. Creating it..."

  : "${MINIO_ENDPOINT:?set MINIO_ENDPOINT (e.g. http://host.docker.internal:9000)}"
  : "${MINIO_ACCESS_KEY:?set MINIO_ACCESS_KEY}"
  : "${MINIO_SECRET_KEY:?set MINIO_SECRET_KEY}"

  # Create alias (official pattern)
  mc alias set "${MINIO_ALIAS}" "${MINIO_ENDPOINT}" "${MINIO_ACCESS_KEY}" "${MINIO_SECRET_KEY}" >/dev/null

  # Verify
  if ! alias_exists; then
    echo "ERROR: failed to create mc alias '${MINIO_ALIAS}'"
    exit 1
  fi

  echo "mc alias '${MINIO_ALIAS}' created."
}

# always exclude these
EXCLUDES=(
  "--exclude=.git"
  "--exclude=.terraform"
  "--exclude=.terragrunt-cache"
  "--exclude=*.tfstate"
  "--exclude=*.tfstate.*"
  "--exclude=*.gitignore"
  "--exclude=Makefile"
  "--exclude=*.sh"
  "--exclude=*.tgz"
  "--exclude=*.yaml"
  "--exclude=.DS_Store"
)

# optional extra excludes (space separated patterns)
EXTRA_EXCLUDES="${EXTRA_EXCLUDES:-}"
for pat in $EXTRA_EXCLUDES; do
  EXCLUDES+=("--exclude=$pat")
done

if [ "$DRY_RUN" = "1" ]; then
  echo "DRY_RUN=1 -> listing files that would be included (approx)"
  find . -type f \
    -not -path './.git/*' \
    -not -path '*/.terraform/*' \
    -not -path '*/.terragrunt-cache/*' \
    -not -path '*.sh' \
    -not -path '*.yaml' \
    -not -path '*.tgz' \
    -not -name '*.tfstate' \
    -not -name '*.tfstate.*' \
    -not -name '*.gitignore' \
    -not -name 'Makefile' \
    -not -name '.DS_Store' \
    | sed 's|^\./||' | sort
  exit 0
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

echo "Packing ${REPO_NAME} -> ${tmp}/${TARBALL_NAME}"
tar -czf "${tmp}/${TARBALL_NAME}" "${EXCLUDES[@]}" .

if [ "$DEBUG" = "1" ]; then
  echo "DEBUG=1 -> tar contents (first 200)"
  tar -tzf "${tmp}/${TARBALL_NAME}" | head -n 200
  echo "DEBUG=1 -> tar size:"
  ls -lah "${tmp}/${TARBALL_NAME}"
fi

# Ensure mc alias before upload
ensure_alias

dest="${MINIO_ALIAS}/${MINIO_BUCKET}/${REMOTE_PREFIX}/${TARBALL_NAME}"
echo "Uploading -> ${dest}"
mc cp "${tmp}/${TARBALL_NAME}" "${dest}"

echo "Done."
echo "S3 key: ${REMOTE_PREFIX}/${TARBALL_NAME}"
