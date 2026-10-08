#!/bin/sh
# Runs the claude-colorizer binary: the plugin's own build, then one on PATH,
# otherwise builds it once with `go build`. Never fails the calling hook.
root="$(cd "$(dirname "$0")/.." && pwd)"
for bin in "$root/bin/claude-colorizer" "$root/bin/claude-colorizer.exe"; do
  [ -x "$bin" ] && exec "$bin" "$@"
done
if command -v claude-colorizer >/dev/null 2>&1; then
  exec claude-colorizer "$@"
fi
if command -v go >/dev/null 2>&1 && (cd "$root" && go build -o bin/ ./cmd/claude-colorizer) >/dev/null 2>&1; then
  for bin in "$root/bin/claude-colorizer" "$root/bin/claude-colorizer.exe"; do
    [ -x "$bin" ] && exec "$bin" "$@"
  done
fi
exit 0
