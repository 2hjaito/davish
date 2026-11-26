#!/bin/sh
# Measures how fast the built app starts and how much memory and CPU it uses
# (.claude/docs/PLAN.md M21). Never touches a real server.
#
#   scripts/bench.sh [binary]            start with an empty HOME (no saved
#                                        server), report start time and idle use
#   scripts/bench.sh --attach [seconds] [pid]
#                                        attach to a running Davish and sample
#                                        it (default 30s) while you set up a
#                                        scenario by hand, on the test sshd only.
#                                        Pass the pid when several are running.
#   scripts/bench.sh --render [runs] [binary]
#                                        the render bench: a fresh process per
#                                        preset (Beautiful, Balanced, Light), each
#                                        timing idle, a long list scrolling and a
#                                        window dragged on the demo's sample data;
#                                        prints the median of [runs] (default 3)
#   scripts/bench.sh --wallpaper [runs] [binary]
#                                        the same, for what a picture as the wallpaper
#                                        costs: Beautiful and Balanced with 0 widgets,
#                                        and with 3 widgets without and with the
#                                        picture, and Light with 3 without and with it
#   scripts/bench.sh --widgets [runs] [binary]
#                                        the same, for what widgets cost: Beautiful
#                                        and Balanced with 0, 1 and 3 widgets on the
#                                        desktop, and Light with 0 and 3 (hidden)
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
  [ "$(echo "$after" | cut -d' ' -f3)" -gt 0 ] || { echo "pid $2 exited while sampling; no result" >&2; exit 1; }
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
  if [ -n "${3:-}" ]; then
    echo "$pids" | grep -qx "$3" || { echo "pid $3 is not a running Davish" >&2; exit 1; }
    pids=$3
  elif [ "$(echo "$pids" | wc -l)" -ne 1 ]; then
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

# render_bench <runs> <binary> <preset:widgets[:picture]>...: one fresh process per
# combination and run, each on the demo's sample data, then the median of the
# runs. The CPU of the app and its WebKit processes is read around each
# scenario, from the marks the app prints, so what a widget costs is not lost
# in a scroll or a drag.
render_bench() {
  runs=$1
  bin=$2
  shift 2
  [ -x "$bin" ] || { echo "build first: wails build" >&2; exit 1; }
  results=$(mktemp)
  envline=$(mktemp)
  pid=""
  trap 'kill "$pid" 2>/dev/null || true; rm -f "$results" "$envline"' EXIT
  # read_marks: reads the marks of $log not yet seen and takes the CPU of the
  # process tree at each: at a start for the baseline, at an end for the share.
  read_marks() {
    n=$(grep -c 'davish-bench mark ' "$log" || true)
    while [ "$seen" -lt "$n" ]; do
      seen=$((seen + 1))
      mark=$(grep 'davish-bench mark ' "$log" | sed -n "${seen}p")
      cpu=$(snapshot "$pid" | cut -d' ' -f2)
      scen=$(echo "$mark" | sed -n 's/.*scenario=\([a-z]*\).*/\1/p')
      case "$mark" in
        *phase=start*) cpu0=$cpu ;;
        *phase=end*)
          ms=$(echo "$mark" | sed -n 's/.*ms=\([0-9]*\).*/\1/p')
          echo "scenario=cpu_$scen preset=$preset widgets=$widgets picture=$picture cpu_pct=$(awk -v a="$cpu" -v b="$cpu0" -v ms="$ms" 'BEGIN { printf "%.1f", (a - b) / (ms / 1000) * 100 }')" >>"$results"
          ;;
      esac
    done
  }
  for run in $(seq 1 "$runs"); do
    for combo in "$@"; do
      preset=${combo%%:*}
      rest=${combo#*:}
      widgets=${rest%%:*}
      picture=0
      case "$rest" in *:*) picture=${rest#*:} ;; esac
      home=$(mktemp -d)
      log=$(mktemp)
      HOME=$home DAVISH_BENCH=render DAVISH_BENCH_PRESET=$preset DAVISH_BENCH_WIDGETS=$widgets DAVISH_BENCH_PICTURE=$picture "$bin" 2>"$log" &
      pid=$!
      seen=0
      cpu0=0
      i=0
      until grep -q 'davish-bench \(done\|error=\)' "$log"; do
        i=$((i + 1))
        [ "$i" -lt 1200 ] || { echo "run $run, $preset $widgets picture=$picture: no result after 120s" >&2; break; }
        read_marks
        sleep 0.1
      done
      # The last scenario ends just before "done": its mark is read here, while
      # the app still waits a second before it quits.
      read_marks
      kill "$pid" 2>/dev/null || true
      wait "$pid" 2>/dev/null || true
      grep 'davish-bench scenario=' "$log" | sed 's/^davish-bench //' >>"$results" || true
      grep 'davish-bench env' "$log" | sed 's/^davish-bench env //' | tr '\n' ' ' >"$envline" || true
      grep 'davish-bench error=' "$log" | sed "s/^/run $run, $preset $widgets: /" >&2 || true
      placed=$(sed -n 's/.*davish-bench env placed=\([0-9]*\).*/\1/p' "$log" | head -n 1)
      [ "$widgets" = 0 ] || [ "${placed:-0}" = "$widgets" ] || echo "run $run, $preset: only ${placed:-0} of $widgets widgets found room, so its numbers are not for $widgets" >&2
      rm -rf "$home" "$log"
      echo "run $run/$runs, $preset with $widgets widget(s), picture $picture, done" >&2
    done
  done
  [ -s "$results" ] || { echo "no results" >&2; exit 1; }
  echo "$(cat "$envline")"
  echo "median of $runs run(s). over_ms: time past 16.7 ms of the frames over 25 ms, summed over 3 s."
  echo "cpu_pct: CPU of the app and its WebKit processes over the scenario, % of one core; cpu_vs0: the same less the run with 0 widgets and no picture."
  awk '
    function med(key, n,  a, i, j, t) {
      for (i = 1; i <= n; i++) a[i] = v[key, i]
      for (i = 2; i <= n; i++) { t = a[i]; for (j = i - 1; j >= 1 && a[j] > t; j--) a[j + 1] = a[j]; a[j + 1] = t }
      return (n % 2) ? a[(n + 1) / 2] : (a[n / 2] + a[n / 2 + 1]) / 2
    }
    {
      for (f = 1; f <= NF; f++) { split($f, kv, "="); x[kv[1]] = kv[2] }
      sc = x["scenario"]
      isCpu = (sc ~ /^cpu_/)
      if (isCpu) sc = substr(sc, 5)
      pic = (x["picture"] == "") ? 0 : x["picture"]
      k = sc SUBSEP x["preset"] SUBSEP x["widgets"] SUBSEP pic
      if (!(k in seen)) { seen[k] = 1; order[++m] = k; cnt[k] = 0; ccnt[k] = 0 }
      if (isCpu) { n = ++ccnt[k]; v[k SUBSEP "cpu", n] = x["cpu_pct"]; next }
      n = ++cnt[k]
      split("mean_ms p95_ms max_ms over_ms", names, " ")
      for (q = 1; q <= 4; q++) v[k SUBSEP names[q], n] = x[names[q]]
      lt[k] = x["long_tasks"]
    }
    END {
      for (o = 1; o <= m; o++) { k = order[o]; if (ccnt[k] > 0) cpu[k] = med(k SUBSEP "cpu", ccnt[k]) }
      printf "%-8s %-10s %7s %7s %5s %8s %7s %7s %8s %8s %8s %s\n", "scenario", "preset", "widgets", "picture", "runs", "mean_ms", "p95_ms", "max_ms", "over_ms", "cpu_pct", "cpu_vs0", "long_tasks"
      nsc = split("idle scroll drag", scs, " ")
      for (si = 1; si <= nsc; si++) {
        for (o = 1; o <= m; o++) {
          k = order[o]; split(k, kk, SUBSEP)
          if (kk[1] != scs[si] || cnt[k] == 0) continue
          base = kk[1] SUBSEP kk[2] SUBSEP 0 SUBSEP 0
          vs = ((kk[3] > 0 || kk[4] > 0) && (k in cpu) && (base in cpu)) ? sprintf("%+.1f", cpu[k] - cpu[base]) : "-"
          c = (k in cpu) ? sprintf("%.1f", cpu[k]) : "-"
          printf "%-8s %-10s %7s %7s %5d %8.1f %7.1f %7.1f %8.1f %8s %8s %s\n", kk[1], kk[2], kk[3], kk[4], cnt[k], med(k SUBSEP "mean_ms", cnt[k]), med(k SUBSEP "p95_ms", cnt[k]), med(k SUBSEP "max_ms", cnt[k]), med(k SUBSEP "over_ms", cnt[k]), c, vs, lt[k]
        }
        if (si < nsc) print ""
      }
    }' "$results"
}

# The default binary of this system.
default_bin() {
  case "$(uname)" in
    Darwin) echo build/bin/Davish.app/Contents/MacOS/Davish ;;
    *) echo build/bin/Davish ;;
  esac
}

if [ "${1:-}" = "--render" ]; then
  render_bench "${2:-3}" "${3:-$(default_bin)}" beautiful:0 balanced:0 light:0
  exit 0
fi

if [ "${1:-}" = "--wallpaper" ]; then
  echo "A picture drawn by the bench (2560 x 1440, JPEG) is the wallpaper; Light hides it by its knob:" >&2
  echo "light 3 with a picture against light 3 without shows nothing is drawn or kept there." >&2
  render_bench "${2:-3}" "${3:-$(default_bin)}" \
    beautiful:0:0 beautiful:3:0 beautiful:3:1 balanced:0:0 balanced:3:0 balanced:3:1 light:3:0 light:3:1
  exit 0
fi

if [ "${1:-}" = "--widgets" ]; then
  echo "Docker widgets read sample data here, so no docker command runs. Light hides widgets:" >&2
  echo "light 3 against light 0 shows what the knob removes." >&2
  render_bench "${2:-3}" "${3:-$(default_bin)}" \
    beautiful:0 beautiful:1 beautiful:3 balanced:0 balanced:1 balanced:3 light:0 light:3
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
