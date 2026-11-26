# Prints the processes on NVIDIA GPUs for gpuprocs.go: nvidia-smi's two views
# of them (memory per compute process; load, type and graphics processes from
# pmon), then the sections of procs.sh for just their PIDs. POSIX sh and base
# tools only: nothing is installed on the server.
export LC_ALL=C
gpus=
apps=
pmon=
if command -v nvidia-smi >/dev/null 2>&1; then
	gpus=$(nvidia-smi --query-gpu=index,uuid,memory.total --format=csv,noheader,nounits 2>/dev/null)
	apps=$(nvidia-smi --query-compute-apps=gpu_uuid,pid,used_gpu_memory,process_name \
		--format=csv,noheader,nounits 2>/dev/null)
	# pmon samples for about a second; some drivers and MIG mode lack it.
	pmon=$(nvidia-smi pmon -c 1 -s um 2>/dev/null)
fi
# Every PID either view names, once; an idle GPU's pmon row has "-".
pids=$({
	printf '%s\n' "$apps" | cut -d, -f2
	printf '%s\n' "$pmon" | awk '$1 !~ /^#/ { print $2 }'
} | tr -d ' ' | grep -E '^[0-9]+$' | sort -un)
echo '@@gpus'
[ -n "$gpus" ] && printf '%s\n' "$gpus"
echo '@@apps'
[ -n "$apps" ] && printf '%s\n' "$apps"
echo '@@pmon'
[ -n "$pmon" ] && printf '%s\n' "$pmon"
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
