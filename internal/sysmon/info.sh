# Prints what the server is, for info.go. Every source is optional.
export LC_ALL=C
echo '@@os'
cat /etc/os-release 2>/dev/null
echo '@@uname'
uname -r
uname -m
echo '@@hostname'
cat /proc/sys/kernel/hostname 2>/dev/null || hostname
echo '@@cpu'
m=$(grep -m 1 '^model name' /proc/cpuinfo 2>/dev/null | cut -d: -f2-)
# ARM kernels often leave the model out of /proc/cpuinfo; lscpu knows it.
[ -z "$m" ] && m=$(lscpu 2>/dev/null | grep -m 1 '^Model name' | cut -d: -f2-)
echo "$m"
echo '@@virt'
systemd-detect-virt 2>/dev/null
# Hardware, network and system facts for Monitor's Info page. All read only.
echo '@@cpucount'
grep -c '^processor' /proc/cpuinfo 2>/dev/null
grep '^physical id' /proc/cpuinfo 2>/dev/null | sort -u | wc -l
grep -m 1 '^cpu cores' /proc/cpuinfo 2>/dev/null | cut -d: -f2
echo '@@mem'
grep -E '^(MemTotal|SwapTotal):' /proc/meminfo 2>/dev/null
echo '@@boot'
grep '^btime' /proc/stat 2>/dev/null
echo '@@tz'
timedatectl show -p Timezone --value 2>/dev/null || readlink /etc/localtime 2>/dev/null
echo '@@users'
who 2>/dev/null | awk '{print $1}' | sort -u
echo '@@systemd'
systemctl --version 2>/dev/null | head -n 1
echo '@@lsblk'
lsblk -J -b -o NAME,TYPE,SIZE,MODEL,ROTA,TRAN,MOUNTPOINT,FSTYPE 2>/dev/null
echo '@@ip'
ip -j addr 2>/dev/null
echo '@@route'
ip -j route show default 2>/dev/null
echo '@@dns'
grep '^nameserver' /etc/resolv.conf 2>/dev/null | awk '{print $2}'
echo '@@end'
