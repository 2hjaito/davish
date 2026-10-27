# Prints a metrics snapshot every 2 seconds for parse.go, one @@section per
# source. POSIX sh and base tools only: nothing is installed on the server.
export LC_ALL=C
while :; do
	echo '@@stat'
	head -n 1 /proc/stat
	echo '@@meminfo'
	cat /proc/meminfo
	echo '@@loadavg'
	cat /proc/loadavg
	echo '@@uptime'
	cat /proc/uptime
	echo '@@netdev'
	cat /proc/net/dev
	echo '@@df'
	df -kP / 2>/dev/null
	echo '@@end'
	sleep 2
done
