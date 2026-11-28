# Prints the processes on GPUs for gpuprocs.go: nvidia-smi's two views of the
# NVIDIA ones (memory per compute process; load, type and graphics processes
# from pmon), the AMD and Intel cards and the DRM file descriptors each
# process holds open (the kernel counts their GPU time and memory there, as
# nvtop reads them), then the sections of procs.sh for just those PIDs.
# POSIX sh and base tools only: nothing is installed on the server.
# DAVISH_DRM and DAVISH_PROC are for the tests, which point them at fake trees.
export LC_ALL=C
gpus=
apps=
pmon=
smi=
if command -v nvidia-smi >/dev/null 2>&1; then
	smi=1
	gpus=$(nvidia-smi --query-gpu=index,uuid,memory.total --format=csv,noheader,nounits 2>/dev/null)
	apps=$(nvidia-smi --query-compute-apps=gpu_uuid,pid,used_gpu_memory,process_name \
		--format=csv,noheader,nounits 2>/dev/null)
	# pmon samples for about a second; some drivers and MIG mode lack it.
	pmon=$(nvidia-smi pmon -c 1 -s um 2>/dev/null)
fi
drm=${DAVISH_DRM:-/sys/class/drm}
fdroot=${DAVISH_PROC:-/proc}
# One line per card, in the order collect.sh numbers them: slot|vendor|vram total.
cards=
for c in "$drm"/card[0-9]*; do
	case "$c" in *-*) continue ;; esac
	d=$c/device
	[ -r "$d/vendor" ] || continue
	vendor= slot= total=
	{ read -r vendor < "$d/vendor"; } 2>/dev/null
	# NVIDIA cards are nvidia-smi's, unless it is missing (as in collect.sh).
	[ "$vendor" = 0x10de ] && [ -n "$smi" ] && continue
	while IFS='=' read -r k v; do
		case $k in PCI_SLOT_NAME) slot=$v ;; esac
	done < "$d/uevent" 2>/dev/null
	{ read -r total < "$d/mem_info_vram_total"; } 2>/dev/null
	cards="$cards$slot|$vendor|$total
"
done
# What each process has open on a card. grep -H names the file, so the PID is
# in the path. Only a process of the user (or all, for root) can be read.
fds=
if [ -n "$cards" ]; then
	fds=$(find "$fdroot"/[0-9]*/fdinfo -type f -exec grep -H -E \
		'^drm-(driver|pdev|client-id|engine-[a-z0-9_-]+|memory-vram|total-vram[0-9]*|total-local[0-9]*):' {} + 2>/dev/null)
fi
# Every PID any view names, once; an idle GPU's pmon row has "-".
pids=$({
	printf '%s\n' "$apps" | cut -d, -f2
	printf '%s\n' "$pmon" | awk '$1 !~ /^#/ { print $2 }'
	printf '%s\n' "$fds" | sed -n 's#^.*/\([0-9][0-9]*\)/fdinfo/[^:]*:.*#\1#p'
} | tr -d ' ' | grep -E '^[0-9]+$' | sort -un)
echo '@@gpus'
[ -n "$gpus" ] && printf '%s\n' "$gpus"
echo '@@apps'
[ -n "$apps" ] && printf '%s\n' "$apps"
echo '@@pmon'
[ -n "$pmon" ] && printf '%s\n' "$pmon"
echo '@@drmcards'
[ -n "$cards" ] && printf '%s' "$cards"
echo '@@drmfd'
[ -n "$fds" ] && printf '%s\n' "$fds"
echo '@@uptime'
cat /proc/uptime 2>/dev/null
echo '@@clk'
getconf CLK_TCK 2>/dev/null || echo 100
echo '@@pagesize'
getconf PAGESIZE 2>/dev/null || echo 4096
echo '@@meminfo'
head -n 1 /proc/meminfo 2>/dev/null
echo '@@passwd'
cut -d: -f1,3 /etc/passwd 2>/dev/null
echo '@@ps'
# Word splitting of $pids is wanted: one PID per word.
# shellcheck disable=SC2086
[ -n "$pids" ] && ps -ww -o pid=,uid=,args= -p "$(echo $pids | tr ' ' ',')" 2>/dev/null
echo '@@stat'
for p in $pids; do
	cat "/proc/$p/stat" 2>/dev/null
done
echo '@@end'
