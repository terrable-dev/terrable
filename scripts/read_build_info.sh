#!/usr/bin/env bash

set -euo pipefail

file="${1:-terrable_build}"

read_key() {
  local key="$1"

  awk -F= -v key="$key" '
    $0 ~ /^[[:space:]]*#/ || $0 ~ /^[[:space:]]*$/ { next }
    {
      current_key = $1
      sub(/^[[:space:]]+/, "", current_key)
      sub(/[[:space:]]+$/, "", current_key)

      if (current_key != key) {
        next
      }

      value = substr($0, index($0, "=") + 1)
      sub(/^[[:space:]]+/, "", value)
      sub(/[[:space:]]+$/, "", value)

      if ((value ~ /^".*"$/) || (value ~ /^'\''.*'\''$/)) {
        value = substr(value, 2, length(value) - 2)
      }

      print value
      exit
    }
  ' "$file"
}

version="$(read_key version)"
preview_tag="$(read_key preview-tag)"

if [[ -z "$version" || ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "terrable_build must include version = X.Y.Z" >&2
  exit 1
fi

if [[ -n "$preview_tag" && ! "$preview_tag" =~ ^[0-9A-Za-z][0-9A-Za-z.-]*$ ]]; then
  echo "terrable_build preview-tag must match ^[0-9A-Za-z][0-9A-Za-z.-]*$" >&2
  exit 1
fi

echo "version=$version"
echo "preview_tag=$preview_tag"
