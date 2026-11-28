# Prints a metrics snapshot every $every seconds for parse.go, one @@section
# per source. Go sets every and healthEvery above this script. POSIX sh and base tools only: nothing is installed on the server.
export LC_ALL=C
# Asking for GPUs only makes sense where the NVIDIA driver tools exist.
gpu=
command -v nvidia-smi >/dev/null 2>&1 && gpu=1
# AMD and Intel GPUs have no tool to ask: the kernel shows them under sysfs.
# DAVISH_DRM is for the tests, which point it at a fake tree.
drm=${DAVISH_DRM:-/sys/class/drm}
# Marketing names come from lspci when it is there; one call, not one per snapshot.
pcinames=
command -v lspci >/dev/null 2>&1 && pcinames=$(lspci -D -mm 2>/dev/null | grep -E '"(VGA compatible|3D|Display)')
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
	# One line per card: slot|vendor|driver|busy %|vram used|vram total|temp m°C|power µW|product name
	echo '@@gpudrm'
	for c in "$drm"/card[0-9]*; do
		case "$c" in *-*) continue ;; esac # card0-DP-1 and the like are connectors
		d=$c/device
		[ -r "$d/vendor" ] || continue
		vendor= slot= driver= busy= used= total= temp= power= pname=
		{ read -r vendor < "$d/vendor"; } 2>/dev/null
		# NVIDIA cards are nvidia-smi's, unless it is missing.
		[ "$vendor" = 0x10de ] && [ -n "$gpu" ] && continue
		while IFS='=' read -r k v; do
			case $k in PCI_SLOT_NAME) slot=$v ;; DRIVER) driver=$v ;; esac
		done < "$d/uevent" 2>/dev/null
		{ read -r busy < "$d/gpu_busy_percent"; } 2>/dev/null
		{ read -r used < "$d/mem_info_vram_used"; } 2>/dev/null
		{ read -r total < "$d/mem_info_vram_total"; } 2>/dev/null
		{ read -r pname < "$d/product_name"; } 2>/dev/null
		for h in "$d"/hwmon/hwmon*; do
			{ read -r temp < "$h/temp1_input"; } 2>/dev/null
			{ read -r power < "$h/power1_average" || read -r power < "$h/power1_input"; } 2>/dev/null
			break
		done
		echo "$slot|$vendor|$driver|$busy|$used|$total|$temp|$power|$pname"
	done
	echo '@@gpuname'
	[ -n "$pcinames" ] && printf '%s\n' "$pcinames"
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
