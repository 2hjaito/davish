# Prints a metrics snapshot every $every seconds for parse.go, one @@section
# per source. Go sets every and healthEvery above this script. POSIX sh and base tools only: nothing is installed on the server.
export LC_ALL=C
# Asking for GPUs only makes sense where the NVIDIA driver tools exist.
gpu=
command -v nvidia-smi >/dev/null 2>&1 && gpu=1
docker=
command -v docker >/dev/null 2>&1 && docker=1
# Failed services and container restarts change slowly: every healthEvery-th
# snapshot (30 seconds), starting with the first.
n=0
while :; do
	echo '@@stat'
	grep '^cpu' /proc/stat
	echo '@@meminfo'
	cat /proc/meminfo
	echo '@@loadavg'
	cat /proc/loadavg
	echo '@@uptime'
	cat /proc/uptime
	echo '@@netdev'
	cat /proc/net/dev
	echo '@@df'
	df -kP 2>/dev/null
	echo '@@diskstats'
	cat /proc/diskstats 2>/dev/null
	echo '@@block'
	ls /sys/block 2>/dev/null
	echo '@@gpu'
	[ -n "$gpu" ] && nvidia-smi --query-gpu=index,name,utilization.gpu,memory.used,memory.total,temperature.gpu,power.draw \
		--format=csv,noheader,nounits 2>/dev/null
	if [ $((n % healthEvery)) -eq 0 ]; then
		echo '@@failed'
		systemctl list-units --type=service --state=failed --no-legend --plain 2>/dev/null | cut -d' ' -f1
		echo '@@containers'
		if [ -n "$docker" ]; then
			ids=$(docker ps -aq 2>/dev/null)
			# Word splitting of $ids is wanted: one argument per container.
			[ -n "$ids" ] && docker inspect --format '{{.Name}} {{.RestartCount}} {{.State.Status}}' $ids 2>/dev/null
		fi
	fi
	n=$((n + 1))
	echo '@@end'
	sleep "$every"
done
