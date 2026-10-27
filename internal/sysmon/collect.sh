# Prints a metrics snapshot every 2 seconds for parse.go, one @@section per
# source. POSIX sh and base tools only: nothing is installed on the server.
export LC_ALL=C
# Asking for GPUs only makes sense where the NVIDIA driver tools exist.
gpu=
command -v nvidia-smi >/dev/null 2>&1 && gpu=1
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
	echo '@@end'
	sleep 2
done
