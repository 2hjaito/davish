#!/bin/sh
# Measures how fast the built app starts and how much memory it holds when
# idle, for the numbers asked in PRs (docs/PLAN.md M21). Runs with an empty
# HOME so no saved server is connected: never touches a real server.
set -eu

bin=${1:-}
if [ -z "$bin" ]; then
  case "$(uname)" in
    Darwin) bin=build/bin/Davish.app/Contents/MacOS/Davish ;;
    *) bin=build/bin/Davish ;;
  esac
fi
[ -x "$bin" ] || { echo "build first: wails build" >&2; exit 1; }

home=$(mktemp -d)
log=$(mktemp)
trap 'kill "$pid" 2>/dev/null || true; rm -rf "$home" "$log"' EXIT

HOME=$home DAVISH_BENCH=1 "$bin" 2>"$log" &
pid=$!

i=0
until grep -q 'davish-bench ready_ms=' "$log"; do
  i=$((i + 1))
  [ "$i" -lt 300 ] || { echo "no ready line after 30s" >&2; exit 1; }
  sleep 0.1
done
ready=$(sed -n 's/.*davish-bench ready_ms=\([0-9]*\).*/\1/p' "$log" | head -n 1)

# Idle: let the first screen settle, then read resident memory.
sleep 10
rss=$(ps -o rss= -p "$pid" | tr -d ' ')
echo "start to first screen: ${ready} ms"
echo "idle RSS (app process): $((rss / 1024)) MB"
if [ "$(uname)" = Darwin ]; then
  echo "note: WebKit renders in separate WebContent processes, not counted here"
fi
