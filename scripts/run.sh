#!/bin/sh
# Runs the claude-colorizer binary: the plugin's own build, then one on PATH,
# otherwise builds it once with `go build`. Never fails the calling hook.
#
# The plugin's build is rebuilt when the Go sources change (see src-hash.sh):
# installing the plugin copies bin/ from the checkout, which may predate the
# sources copied with it. Without Go, the existing binary is used as is.
root="$(cd "$(dirname "$0")/.." && pwd)"
stamp="$root/bin/.src-hash"

build() {
  command -v go >/dev/null 2>&1 || return 1
  hash="$("$root/scripts/src-hash.sh")" || return 1
  (cd "$root" && go build -o bin/ ./cmd/claude-colorizer) >/dev/null 2>&1 || return 1
  printf '%s\n' "$hash" >"$stamp"
}

stale() {
  command -v go >/dev/null 2>&1 || return 1
  [ "$(cat "$stamp" 2>/dev/null)" != "$("$root/scripts/src-hash.sh")" ]
}

for bin in "$root/bin/claude-colorizer" "$root/bin/claude-colorizer.exe"; do
  if [ -x "$bin" ]; then
    stale && build
    exec "$bin" "$@"
  fi
done
if command -v claude-colorizer >/dev/null 2>&1; then
  exec claude-colorizer "$@"
fi
if build; then
  for bin in "$root/bin/claude-colorizer" "$root/bin/claude-colorizer.exe"; do
    [ -x "$bin" ] && exec "$bin" "$@"
  done
fi
exit 0
