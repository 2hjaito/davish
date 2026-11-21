#!/bin/sh
# Measures how fast the built app starts and how much memory and CPU it uses
# (.claude/docs/PLAN.md M21). Never touches a real server.
#
#   scripts/bench.sh [binary]            start with an empty HOME (no saved
#                                        server), report start time and idle use
#   scripts/bench.sh --attach [seconds]  attach to a running Davish and sample
#                                        it (default 30s) while you set up a
#                                        scenario by hand, on the test sshd only
#
# Totals count the app and every child it spawned (WebKitWebProcess on Linux).
# On macOS the WebKit WebContent, Networking and GPU services are not children,
# so those started no earlier than the app are added by name. Another app that
# opens a web view while you measure would be counted too: keep others closed.
set -eu

# Prints "<rss_kb> <cpu_seconds> <process_count>" for the app tree rooted at $1.
snapshot() {
  ps -axo pid=,ppid=,rss=,time=,etime=,comm= | awk -v root="$1" -v mac="$([ "$(uname)" = Darwin ] && echo 1 || echo 0)" '
    function secs(t,  d, p, n, s) {
      d = 0
      if (t ~ /-/) { split(t, a, "-"); d = a[1]; t = a[2] }
      n = split(t, p, ":")
      s = (n == 3) ? p[1] * 3600 + p[2] * 60 + p[3] : p[1] * 60 + p[2]
      return s + d * 86400
    }
    { pid[NR] = $1; ppid[NR] = $2; rss[NR] = $3; cpu[NR] = secs($4); age[NR] = secs($5)
      $1 = $2 = $3 = $4 = $5 = ""; comm[NR] = $0
      if (pid[NR] == root) rootage = age[NR] }
    END {
      keep[root] = 1
      do {
        grew = 0
        for (i = 1; i <= NR; i++)
          if (!(pid[i] in keep) && (ppid[i] in keep)) { keep[pid[i]] = 1; grew = 1 }
      } while (grew)
      for (i = 1; i <= NR; i++)
        if ((pid[i] in keep) || (mac == 1 && age[i] <= rootage + 2 && comm[i] ~ /com\.apple\.WebKit\.(WebContent|Networking|GPU)/)) {
          r += rss[i]; c += cpu[i]; n++
        }
      printf "%d %.2f %d\n", r, c, n
    }'
}

# report <label> <pid> <seconds>: memory now, and CPU averaged over the window.
report() {
  set -- "$1" "$2" "$3" "$(snapshot "$2")"
  before_cpu=$(echo "$4" | cut -d' ' -f2)
  sleep "$3"
  after=$(snapshot "$2")
  rss=$(echo "$after" | cut -d' ' -f1)
  after_cpu=$(echo "$after" | cut -d' ' -f2)
  procs=$(echo "$after" | cut -d' ' -f3)
  awk -v l="$1" -v r="$rss" -v b="$before_cpu" -v a="$after_cpu" -v s="$3" -v n="$procs" \
    'BEGIN { printf "%s: RSS %d MB in %d processes, CPU %.1f%% of one core over %ds\n", l, r / 1024, n, (a - b) / s * 100, s }'
}

if [ "${1:-}" = "--attach" ]; then
  secs=${2:-30}
  case "$(uname)" in
    Darwin) pids=$(pgrep -f 'Davish.app/Contents/MacOS/Davish' || true) ;;
    *) pids=$(pgrep -x Davish || true) ;;
  esac
  [ -n "$pids" ] || { echo "no running Davish found" >&2; exit 1; }
  if [ "$(echo "$pids" | wc -l)" -ne 1 ]; then
    echo "more than one Davish is running; quit the others so the numbers are the app's own:" >&2
    for p in $pids; do ps -o pid=,command= -p "$p" >&2; done
    exit 1
  fi
  pid=$pids
  ps -o command= -p "$pid" | sed 's/^/attached to: /'
  echo "leave the scenario untouched while sampling"
  report "scenario" "$pid" "$secs"
  exit 0
fi

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

# Idle: let the first screen settle, then sample.
sleep 10
echo "start to first screen: ${ready} ms"
report "idle" "$pid" 10
