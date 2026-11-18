#!/bin/sh
# Prints a checksum of the Go sources. run.sh stores it next to the binary
# it builds (bin/.src-hash) and rebuilds when it no longer matches, so a
# stale bin/ copied in by a plugin install never outlives a source change.
root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root" || exit 1
{ find cmd internal -name '*.go' -type f | LC_ALL=C sort; echo go.mod; } |
  while read -r f; do echo "$f"; cat "$f"; done | cksum | cut -d' ' -f1
